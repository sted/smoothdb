package database

import "context"

type QueryExecutor struct{}

func (QueryExecutor) Select(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, err
	}
	options := gi.QueryOptions
	query, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, err
	}
	rows, err := gi.Conn.Query(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
}

func (QueryExecutor) Insert(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		rows, err := gi.Conn.Query(ctx, insert, values...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		data, err := gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
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
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	update, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		rows, err := gi.Conn.Query(ctx, update, values...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		data, err := gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
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
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	delete, values, err := gi.QueryBuilder.BuildDelete(table, parts, options, &gi.Db.DbInfo)
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		rows, err := gi.Conn.Query(ctx, delete, values...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		data, err := gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
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
	parts, err := gi.RequestParser.parseQuery(function, filters)
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
	if function != "getallprojects" && function != "ret_setof_integers" {
		data, err = gi.QueryBuilder.preferredSerializer().SerializeSingle(ctx, rows)
	} else {
		data, err = gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
	}
	return data, 0, err
}
