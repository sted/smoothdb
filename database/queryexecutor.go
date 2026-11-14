package database

import (
	"context"
	"slices"
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
	switch {
	case options.locate:
		serializer = LocationSerializer{}
	case options.ContentType == "text/csv":
		serializer = &CSVSerializer{}
	case options.ContentType == "application/octet-stream":
		serializer = &BinarySerializer{}
	default:
		serializer = gi.QueryBuilder.preferredSerializer()
	}
	// the request's variables ride in front of the statements when they are
	// still pending (see requestVars)
	batch, err := gi.newBatch(ctx, options.totalPrelude, query, options.totalQuery, options.insertedQuery)
	if err != nil {
		return nil, 0, err
	}
	if options.totalQuery == "" && options.insertedQuery == "" && batch.lead == 0 {
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
	if options.insertedQuery != "" {
		batch.Queue(options.insertedQuery)
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
	if options.insertedQuery != "" {
		if err := br.QueryRow().Scan(&options.Inserted); err != nil {
			return nil, 0, err
		}
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
	// The resolution applies when the insert has a conflict target: the
	// primary key or ?on_conflict= (Response.hs, responsePreferences).
	options.ResolutionApplied = (options.MergeDuplicates || options.IgnoreDuplicates) &&
		(len(parts.conflictFields) > 0 || (info != nil && info.GetPrimaryKey(_s(table, options.Schema)) != nil))
	options.Inserted = -1
	if options.MergeDuplicates {
		options.insertedQuery = insertedQuery
		defer func() { options.insertedQuery = "" }()
	}
	switch {
	case options.ReturnRepresentation:
		return writeRead(ctx, insert, values)
	case options.Return == "headers-only":
		options.locate = true
		defer func() { options.locate = false }()
		if options.Singular {
			// a refused row count rolls the insert back
			return writeRead(ctx, insert, values)
		}
		// no rows to refuse: no transaction of the request, no COMMIT
		return runRead(ctx, insert, values, false, false)
	default:
		n, err := writeExec(ctx, insert, values)
		return nil, n, err
	}
}

// PutError is a PUT that PostgREST refuses before or after its write
// (ApiRequest.hs, getRanges; Plan.hs, mutatePlan; MainTx.hs, failPut).
type PutError struct {
	Status  int
	Code    string
	Message string
}

func (e *PutError) Error() string { return e.Message }

var (
	errPutLimit   = &PutError{400, "PGRST114", "limit/offset querystring parameters are not allowed for PUT"}
	errPutFilters = &PutError{405, "PGRST105", "Filters must include all and only primary key columns with 'eq' operators"}
	errPutPayload = &PutError{400, "PGRST115", "Payload values do not match URL in primary key column(s)"}
)

// putCondition is a primary key column of a PUT and the value its filter
// gives.
type putCondition struct {
	column, value string
}

// putConditions reads the primary key of a PUT from its filters: an eq
// filter on each column of the key, and no other root filter. A logic
// filter (and, or) is refused at any level. ok is false when the filters do
// not address one row, or when the relation has no key.
func putConditions(root *WhereConditionNode, keys []string) (conds []putCondition, ok bool) {
	if len(keys) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	if root != nil {
		if root.not || root.operator != "" {
			return nil, false
		}
		for _, n := range root.children {
			if n.isContainer() {
				return nil, false
			}
			if len(n.field.relPath) > 0 {
				// a filter on an embedded resource
				continue
			}
			if n.operator != "=" || n.not || n.opModifier != "" || n.field.jsonPath != "" ||
				!slices.Contains(keys, n.field.name) || len(n.values) != 1 {
				return nil, false
			}
			seen[n.field.name] = true
			conds = append(conds, putCondition{n.field.name, n.values[0]})
		}
	}
	if len(seen) != len(keys) {
		return nil, false
	}
	return conds, true
}

// Upsert writes the row that the filters of a PUT address, as PostgREST
// does (MutationSingleUpsert): an insert of the body rows with that key,
// merged on the key. The filters must give the whole key and nothing else
// (PGRST105, 405). limit and offset are refused (PGRST114, 400), a Range
// header is ignored. When the body has no row with that key, nothing is
// written (PGRST115, 400). Inserted tells a created row from an updated one.
func Upsert(ctx context.Context, table string, records []Record, body []byte, filters Filters) ([]byte, int64, error) {
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	if parts.limit != "" || parts.offset != "" {
		return nil, 0, errPutLimit
	}
	options := &gi.QueryOptions
	ignoreRange(options)
	info := gi.Db.info.Load()
	if body == nil {
		if info, err = info.withColumnTypesOf(ctx, table, options.Schema); err != nil {
			return nil, 0, err
		}
	}
	var keys []string
	if info != nil {
		keys = info.GetKeyColumns(_s(table, options.Schema))
	}
	conds, ok := putConditions(parts.whereConditionsTree, keys)
	if !ok {
		return nil, 0, errPutFilters
	}
	parts.putConditions = conds
	parts.conflictFields = map[string]struct{}{}
	for _, k := range keys {
		parts.conflictFields[k] = struct{}{}
	}
	options.MergeDuplicates, options.IgnoreDuplicates = true, false
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, body, parts, options, info)
	if err != nil {
		return nil, 0, err
	}
	options.Inserted = -1
	options.insertedQuery = insertedQuery
	defer func() { options.insertedQuery = "" }()
	if options.ReturnRepresentation {
		data, n, err := writeRead(ctx, insert, values)
		if _, single := err.(*SerializeError); single && options.Singular {
			// the representation of no row: the body has not the key
			return nil, 0, errPutPayload
		}
		if err == nil && n != 1 {
			err = errPutPayload
		}
		return data, n, err
	}
	// one row or none: nothing to roll back, and no single object to check
	singular := options.Singular
	options.Singular = false
	n, err := writeExec(ctx, insert, values)
	options.Singular = singular
	if err == nil && n != 1 {
		err = errPutPayload
	}
	return nil, n, err
}

// writeExec runs the write query of a request that asks for no rows back,
// and answers the number of rows it wrote. With a single object asked
// (vnd.pgrst.object), a write of more or less than one row fails and rolls
// back, as in PostgREST, whatever the return preference: the write runs in
// a transaction of the request, as in writeRead. With a merge-duplicates
// insert, the count of the created rows follows in the same batch (see
// insertedCounter). Otherwise it is one statement, without rows.
func writeExec(ctx context.Context, query string, values []any) (int64, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.Singular && gi.vars != nil && dbe.config.TransactionMode == "none" {
		gi.vars.writeTx = true
	}
	var n int64
	if options.insertedQuery == "" {
		tag, err := gi.exec(ctx, query, values...)
		if err != nil {
			return 0, err
		}
		n = tag.RowsAffected()
	} else {
		tag, err := gi.execInserted(ctx, query, values)
		if err != nil {
			return 0, err
		}
		n = tag.RowsAffected()
	}
	if options.Singular && n != 1 {
		return 0, notSingleError(n)
	}
	return n, nil
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
	}
	n, err := writeExec(ctx, update, values)
	return nil, n, err
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
	}
	n, err := writeExec(ctx, delete, values)
	return nil, n, err
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
	if oneValue && options.ContentType != "application/octet-stream" {
		// the page has one row, for Content-Range, also when the body is
		// null. With no separate count the total is that count, as in
		// PostgREST (countF); otherwise it stays the count of the rows.
		n = 1
		if options.Count != "" && options.totalQuery == "" {
			options.Total = 1
		}
	}
	if data == nil {
		// an empty value (an empty bytea) is content: nil means void
		data = []byte{}
	}
	return data, n, nil
}
