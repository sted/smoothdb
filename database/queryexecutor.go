package database

import (
	"context"
)

type RangeError struct {
	msg string // description of error
}

func (e RangeError) Error() string { return e.msg }

type ContentTypeError struct {
	msg string // description of error
}

func (e ContentTypeError) Error() string { return e.msg }

func querySerialize(ctx context.Context, query string, values []any) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	options := gi.QueryOptions
	if options.ContentType == "unknown/unknown" {
		return nil, 0, &ContentTypeError{msg: "Content type not available"}
	}
	if options.RangeMin > options.RangeMax && options.RangeMin != -1 && options.RangeMax != -1 {
		return nil, 0, &RangeError{msg: "Requested range not satisfiable"}
	}
	info := gi.Db.info
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
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	query, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, gi.Db.info)
	if err != nil {
		return nil, 0, err
	}
	return querySerialize(ctx, query, values)
}

func Insert(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, parts, options, gi.Db.info)
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
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	update, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options, gi.Db.info)
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
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	delete, values, err := gi.QueryBuilder.BuildDelete(table, parts, options, gi.Db.info)
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
	gi := GetSmoothContext(ctx)
	options := gi.QueryOptions
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
	info := gi.Db.info
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
	f := info.GetFunction(_s(function, options.Schema))
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
