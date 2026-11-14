package database

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type RangeError struct {
	msg string // description of error
}

func (e RangeError) Error() string { return e.msg }

type ContentTypeError struct {
	msg string // description of error
}

func (e ContentTypeError) Error() string { return e.msg }

// SchemaError: the Accept-Profile or Content-Profile header names a schema
// that is not exposed (406, PostgREST PGRST106)
type SchemaError struct {
	msg  string
	Hint string
}

func (e SchemaError) Error() string { return e.msg }

// schemaExposed refuses a schema outside the exposed list, with the list as
// a hint; an empty list exposes every schema.
func schemaExposed(schema string, exposed []string) *SchemaError {
	if len(exposed) == 0 {
		return nil
	}
	for _, s := range exposed {
		if s == schema {
			return nil
		}
	}
	return &SchemaError{"Invalid schema: " + schema, "Only the following schemas are exposed: " + strings.Join(exposed, ", ")}
}

// checkSchema refuses a request whose schema, from the Accept-Profile or
// Content-Profile header, is not in Database.ExposedSchemas, as PostgREST
// refuses one outside db-schemas: before any SQL, with the exposed list as a
// hint. With no ExposedSchemas configured every schema is reachable: the
// search path (SchemaSearchPath) says nothing about exposure — a deployment
// resolves its extension types through it while its clients select the data
// schemas by header.
func checkSchema(ctx context.Context) error {
	if dbe == nil {
		return nil
	}
	if err := schemaExposed(GetSmoothContext(ctx).QueryOptions.Schema, dbe.config.ExposedSchemas); err != nil {
		return err
	}
	return nil
}

// ignoreRange drops the Range header of a table write (insert, update or
// delete), as PostgREST does: the header pages reads (GET, and a function
// call whatever its method), never a mutation. The write and its
// representation take every row, and neither the 416 of an unsatisfiable
// range nor a Content-Range derived from it can follow. The values are the
// ones the request parser leaves when no Range header is given.
func ignoreRange(options *QueryOptions) {
	options.HasRange = false
	options.RangeMin = -1
	options.RangeMax = -1
}

func querySerialize(ctx context.Context, query string, values []any) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.ContentType == "unknown/unknown" {
		return nil, 0, &ContentTypeError{msg: "Content type not available"}
	}
	return runRead(ctx, query, values, false, options.Singular)
}

// runRead runs a query and serializes its rows in the content type of the
// request. It answers the serialized rows and their number; with a count
// it sets options.Total, the number of rows read or, for a ranged count, the
// answer of options.totalQuery, which it sends in the same batch after the
// query (and options.totalPrelude, if any, before it): one round trip, and
// the page needs no row of its own to carry the total (see buildAfterSelect).
func runRead(ctx context.Context, query string, values []any, scalar, single bool) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.RangeMin > options.RangeMax && options.RangeMin != -1 && options.RangeMax != -1 {
		return nil, 0, &RangeError{msg: "Requested range not satisfiable"}
	}
	info := gi.Db.info.Load()
	var serializer TextSerializer
	switch options.ContentType {
	case "text/csv":
		serializer = &CSVSerializer{}
	case "application/octet-stream":
		serializer = &BinarySerializer{}
	default:
		serializer = gi.QueryBuilder.preferredSerializer()
	}
	// the request's variables ride in front of the statements when they are
	// still pending (see requestVars)
	batch, err := gi.newBatch(ctx, options.totalPrelude, query, options.totalQuery)
	if err != nil {
		return nil, 0, err
	}
	if options.totalQuery == "" && batch.lead == 0 {
		rows, err := gi.Conn.Query(ctx, query, values...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		out, n, err := serializer.Serialize(rows, scalar, single, info)
		if err == nil {
			gi.remember1(query)
			if options.Count != "" {
				options.Total = n
			}
		}
		return out, n, err
	}
	if options.totalPrelude != "" {
		batch.Queue(options.totalPrelude)
	}
	batch.Queue(query, values...)
	if options.totalQuery != "" {
		batch.Queue(options.totalQuery, options.totalValues...)
	}
	br, err := gi.send(ctx, batch)
	if err != nil {
		return nil, 0, err
	}
	// with autocommit the end of the batch is the commit, and its error (a
	// deferred constraint) is the request's: end reads it
	out, n, err := readBatch(br, options, serializer, scalar, single, info)
	if err := gi.end(ctx, br, batch, err); err != nil {
		return nil, 0, err
	}
	return out, n, nil
}

// readBatch reads the results of runRead's batch: the total's prelude, the
// rows, the total.
func readBatch(br pgx.BatchResults, options *QueryOptions, serializer TextSerializer, scalar, single bool, info *SchemaInfo) ([]byte, int64, error) {
	if options.totalPrelude != "" {
		if _, err := br.Exec(); err != nil {
			return nil, 0, err
		}
	}
	rows, err := br.Query()
	if err != nil {
		return nil, 0, err
	}
	out, n, err := serializer.Serialize(rows, scalar, single, info)
	rows.Close()
	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		return nil, 0, err
	}
	if options.totalQuery == "" {
		if options.Count != "" {
			options.Total = n
		}
	} else if err := br.QueryRow().Scan(&options.Total); err != nil {
		return nil, 0, err
	}
	return out, n, nil
}

func Select(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := &gi.QueryOptions
	query, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	return querySerialize(ctx, query, values)
}

func Insert(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	return InsertJSON(ctx, table, records, nil, filters)
}

// InsertJSON is Insert for records decoded from body, a JSON object or array
// of objects: body itself is sent as the rows (see BuildInsert).
func InsertJSON(ctx context.Context, table string, records []Record, body []byte, filters Filters) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := &gi.QueryOptions
	ignoreRange(options)
	info := gi.Db.info.Load()
	if body == nil {
		// Go values need the column types (see insertRows), also for a
		// relation created since the schema cache was loaded
		if info, err = info.withColumnTypesOf(ctx, table, options.Schema); err != nil {
			return nil, 0, err
		}
	}
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, body, parts, options, info)
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return writeRead(ctx, insert, values)
	} else {
		tag, err := gi.exec(ctx, insert, values...)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}

func Update(ctx context.Context, table string, record Record, filters Filters) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := &gi.QueryOptions
	ignoreRange(options)
	update, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return writeRead(ctx, update, values)
	} else {
		tag, err := gi.exec(ctx, update, values...)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}

func Delete(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := &gi.QueryOptions
	ignoreRange(options)
	delete, values, err := gi.QueryBuilder.BuildDelete(table, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return writeRead(ctx, delete, values)
	} else {
		tag, err := gi.exec(ctx, delete, values...)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}

// FunctionNotFoundError: no function of the schema cache takes the
// arguments of a call (404, PostgREST PGRST202)
type FunctionNotFoundError struct {
	msg string
}

func (e FunctionNotFoundError) Error() string { return e.msg }

// OIDs of the pseudo-types a function can return
const (
	recordOID = 2249
	voidOID   = 2278
)

// Execute calls the function named function. A GET (readonly) takes the
// arguments from the query string, a POST from record, decoded from the
// body, or from raw, the body itself (see RawBody). A function that returns
// void answers nil data: the handler answers 204 with no body.
func Execute(ctx context.Context, function string, record Record, raw *RawBody, filters Filters, readonly bool) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.ContentType == "unknown/unknown" {
		return nil, 0, &ContentTypeError{msg: "Content type not available"}
	}
	var params Filters
	if readonly {
		params = gi.RequestParser.filterParameters(filters)
	}
	parts, err := gi.RequestParser.parse(function, filters)
	if err != nil {
		return nil, 0, err
	}
	if readonly {
		record = make(Record, len(params))
		for k, vv := range params {
			record[k] = QueryValues(vv)
		}
	}
	var rawType string
	if raw != nil {
		rawType = raw.Type
		if rawType != "json" {
			// a text, xml or bytea body is no list of arguments
			record = nil
		}
	}
	info := gi.Db.info.Load()
	// The overload the call invokes, by the names of the arguments it passes
	f, err := info.FindFunctionFor(_s(function, options.Schema), orderedRecordKeys(record, parts.columnFields, nil), rawType)
	if err != nil {
		return nil, 0, err
	}
	if raw != nil {
		switch {
		case f != nil && f.singleUnnamed() != nil:
			// the body goes whole to the single unnamed parameter
			record = Record{"": raw}
		case rawType == "json":
		case f == nil:
			// a text, xml or bytea body fills only a single unnamed
			// parameter of its type (PGRST202)
			return nil, 0, &FunctionNotFoundError{"Could not find the function " + _s(function, options.Schema) +
				" with a single unnamed " + rawType + " parameter in the schema cache"}
		default:
			// a function whose parameters all have a default: called without
			// arguments, as PostgREST does
			record = Record{}
		}
	}
	// A call to a STABLE or IMMUTABLE function runs READ ONLY whatever the
	// method, as in PostgREST (Plan.hs callReadPlan: Inv + Stable/Immutable ->
	// SQL.Read); a GET already does. The marker is a promise PostgreSQL does not
	// check: a function that writes despite it fails with 25006 (405).
	if !readonly && f != nil && f.Volatility != "v" {
		if err := SetReadOnly(ctx); err != nil {
			return nil, 0, err
		}
	}
	exec, values, err := gi.QueryBuilder.BuildExecute(function, f, record, parts, options, info)
	if err != nil {
		return nil, 0, err
	}
	var scalar bool
	if f != nil {
		if f.returnsRecord() {
			// the column pgrst_scalar of each row, or the rows (see
			// recordExtracts)
			scalar = recordExtracts(f, options)
		} else if rettype := info.GetTypeById(f.ReturnTypeId); rettype != nil {
			scalar = !rettype.IsComposite && !rettype.IsTable && !f.HasOut
		}
	}
	// vnd.pgrst.object asks for one row: zero or many fail with 406, as in
	// PostgREST (PGRST116). Only it checks the number of rows (MainTx.hs).
	// A VOLATILE call can write before the check: with TransactionMode
	// "none" it runs in a transaction of its own, which a 406 rolls back
	// (see writeRead). A GET runs READ ONLY and writes nothing.
	if options.Singular && !readonly && (f == nil || f.Volatility == "v") && gi.vars != nil && dbe.config.TransactionMode == "none" {
		gi.vars.writeTx = true
	}
	// A set of scalars as one object keeps its column (PostgREST's
	// asJsonSingleF): {"pgrst_scalar": 1}. A set of records is a row already.
	setOfScalarObject := options.Singular && scalar && f != nil && f.ReturnIsSet && !f.returnsRecord()
	// A function that returns one value counts one row whatever it answers
	// (PostgREST's pageCountSelectF): vnd.pgrst.object never refuses it.
	oneValue := f != nil && !f.ReturnIsSet
	data, n, err := runRead(ctx, exec, values, scalar, options.Singular && !oneValue)
	if err != nil {
		return nil, 0, err
	}
	if f != nil && f.ReturnTypeId == voidOID && !f.ReturnIsSet {
		return nil, n, nil
	}
	// the JSON serializer writes every content type but CSV and binary (see runRead)
	isJSON := options.ContentType != "text/csv" && options.ContentType != "application/octet-stream"
	switch {
	case setOfScalarObject && isJSON:
		data = append(append([]byte(`{"pgrst_scalar":`), data...), '}')
	case oneValue && isJSON:
		// a function that returns one value answers that value, not an
		// array, and null for no row (a limit of 0)
		if n == 0 {
			data = []byte("null")
		} else if len(data) >= 2 {
			data = data[1 : len(data)-1]
		}
	}
	if data == nil {
		// an empty value (an empty bytea) is content: nil means void
		data = []byte{}
	}
	return data, n, nil
}
