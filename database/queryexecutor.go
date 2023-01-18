package database

import "context"

type QueryExecutor struct{}

func (QueryExecutor) Select(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gi := GetGreenContext(ctx)
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, err
	}
	options := gi.QueryOptions
	schema := options.Schema
	rels := gi.Db.GetRelationships(_s(table, schema))
	query, err := gi.QueryBuilder.BuildSelect(table, parts, options, rels)
	if err != nil {
		return nil, err
	}
	rows, err := gi.Conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
}

func (QueryExecutor) Insert(ctx context.Context, table string, records []Record) ([]byte, int64, error) {
	gi := GetGreenContext(ctx)
	options := gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildInsert(table, records, options)
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
	gi := GetGreenContext(ctx)
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	insert, values, err := gi.QueryBuilder.BuildUpdate(table, record, parts, options)
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

func (QueryExecutor) Delete(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	gi := GetGreenContext(ctx)
	parts, err := gi.RequestParser.parseQuery(table, filters)
	if err != nil {
		return nil, 0, err
	}
	options := gi.QueryOptions
	delete, err := gi.QueryBuilder.BuildDelete(table, parts, options)
	if err != nil {
		return nil, 0, err
	}
	if options.ReturnRepresentation {
		rows, err := gi.Conn.Query(ctx, delete)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		data, err := gi.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
		return data, 0, err
	} else {
		tag, err := gi.Conn.Exec(ctx, delete)
		if err != nil {
			return nil, 0, err
		}
		return nil, tag.RowsAffected(), nil
	}
}