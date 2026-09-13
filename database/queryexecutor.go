package database

import (
	"context"
	"strings"
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

func querySerialize(ctx context.Context, query string, values []any) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.ContentType == "unknown/unknown" {
		return nil, 0, &ContentTypeError{msg: "Content type not available"}
	}
	if options.RangeMin > options.RangeMax && options.RangeMin != -1 && options.RangeMax != -1 {
		return nil, 0, &RangeError{msg: "Requested range not satisfiable"}
	}
	info := gi.Db.info.Load()
	rows, err := gi.Conn.Query(ctx, query, values...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var serializer TextSerializer
	switch options.ContentType {
	case "text/csv":
		serializer = &CSVSerializer{}
	case "application/octet-stream":
		serializer = &BinarySerializer{}
	default:
		serializer = gi.QueryBuilder.preferredSerializer()
	}
	return serializer.Serialize(rows, false, options.Singular, info)
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
	if err := checkSchema(ctx); err != nil {
		return nil, 0, err
	}
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := &gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return querySerialize(ctx, insert, values)
	} else {
		tag, err := gi.Conn.Exec(ctx, insert, values...)
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
	update, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return querySerialize(ctx, update, values)
	} else {
		tag, err := gi.Conn.Exec(ctx, update, values...)
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
	delete, values, err := gi.QueryBuilder.BuildDelete(table, parts, options, gi.Db.info.Load())
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		return querySerialize(ctx, delete, values)
	} else {
		tag, err := gi.Conn.Exec(ctx, delete, values...)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}

func Execute(ctx context.Context, function string, record Record, filters Filters, readonly bool) ([]byte, int64, error) {
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
		if len(record) != 0 {

		}
		record = make(map[string]any)
		for k, vv := range params {
			for _, v := range vv {
				record[k] = v
			}
		}
	}
	info := gi.Db.info.Load()
	f := info.GetFunction(_s(function, options.Schema))
	// A call to a STABLE or IMMUTABLE function runs READ ONLY whatever the
	// method, as in PostgREST (Plan.hs callReadPlan: Inv + Stable/Immutable ->
	// SQL.Read); a GET already does. The marker is a promise PostgreSQL does not
	// check: a function that writes despite it fails with 25006 (405).
	if !readonly && f != nil && f.Volatility != "v" {
		if err := SetReadOnly(ctx); err != nil {
			return nil, 0, err
		}
	}
	exec, values, err := gi.QueryBuilder.BuildExecute(function, record, parts, options, info)
	if err != nil {
		return nil, 0, err
	}
	rows, err := gi.Conn.Query(ctx, exec, values...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var scalar bool
	if f != nil {
		rettype := info.GetTypeById(f.ReturnTypeId)
		if rettype != nil {
			scalar = !rettype.IsComposite && !rettype.IsTable && !f.HasOut
		}
	}
	single := f != nil && !f.ReturnIsSet
	var serializer TextSerializer
	switch options.ContentType {
	case "text/csv":
		serializer = &CSVSerializer{}
	case "application/octet-stream":
		serializer = &BinarySerializer{}
	default:
		serializer = gi.QueryBuilder.preferredSerializer()
	}
	return serializer.Serialize(rows, scalar, single, info)
}
