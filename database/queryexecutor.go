package database

import (
	"context"
)

func querySerialize(ctx context.Context, query string, values []any) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	options := gi.QueryOptions
	info := gi.Db.info
	rows, err := gi.Conn.Query(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	serializer := gi.QueryBuilder.preferredSerializer()
	var data []byte
	if options.Singular {
		data, err = serializer.SerializeSingle(ctx, rows, false, info)
	} else {
		data, err = serializer.Serialize(ctx, rows, false, info)
	}
	return data, err
}

func Select(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, err
	}
	options := gi.QueryOptions
	query, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, gi.Db.info)
	if err != nil {
		return nil, err
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
		var data []byte
		data, err = querySerialize(ctx, insert, values)
		return data, 0, err
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
		var data []byte
		data, err = querySerialize(ctx, update, values)
		return data, 0, err
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
		var data []byte
		data, err = querySerialize(ctx, delete, values)
		return data, 0, err
	} else {
		tag, err := gi.Conn.Exec(ctx, delete, values...)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}

func Execute(ctx context.Context, function string, record Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(function, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
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
	var data []byte
	if f != nil && !f.ReturnIsSet {
		data, err = gi.QueryBuilder.preferredSerializer().SerializeSingle(ctx, rows, scalar, info)
	} else {
		data, err = gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows, scalar, info)
	}
	return data, 0, err
}
