package database

import "context"

type QueryExecutor struct{}

func querySerialize(ctx context.Context, query string, values []any) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	options := gi.QueryOptions
	rows, err := gi.Conn.Query(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	serializer := gi.QueryBuilder.preferredSerializer()
	var data []byte
	if options.Singular {
		data, err = serializer.SerializeSingle(ctx, rows, false)
	} else {
		data, err = serializer.Serialize(ctx, rows)
	}
	return data, err
}

func (QueryExecutor) Select(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, err
	}
	options := gi.QueryOptions
	query, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, err
	}
	return querySerialize(ctx, query, values)
}

func (QueryExecutor) Insert(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, parts, options, &gi.Db.DbInfo)
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

func (QueryExecutor) Update(ctx context.Context, table string, record Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	update, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options, &gi.Db.DbInfo)
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

func (QueryExecutor) Delete(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	delete, values, err := gi.QueryBuilder.BuildDelete(table, parts, options, &gi.Db.DbInfo)
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

func (QueryExecutor) Execute(ctx context.Context, function string, record Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(function, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	exec, values, err := gi.QueryBuilder.BuildExecute(function, record, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, 0, err
	}
	rows, err := gi.Conn.Query(ctx, exec, values...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var data []byte
	if function != "getallprojects" && function != "ret_setof_integers" { //@@
		data, err = gi.QueryBuilder.preferredSerializer().SerializeSingle(ctx, rows, true)
	} else {
		data, err = gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
	}
	return data, 0, err
}
