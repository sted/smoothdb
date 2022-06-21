package database

import "context"

type QueryExecutor struct{}

func (QueryExecutor) Select(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gctx := GetGreenContext(ctx)
	query, err := gctx.QueryBuilder.BuildSelect(gctx.QueryStringParser, table, filters)
	if err != nil {
		return nil, err
	}
	rows, err := gctx.Conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return gctx.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
}

func (QueryExecutor) Insert(ctx context.Context, table string, records []Record) ([]byte, error) {
	gctx := GetGreenContext(ctx)
	insert, values, err := gctx.QueryBuilder.BuildInsert(table, records)
	if err != nil {
		return nil, err
	}
	rows, err := gctx.Conn.Query(ctx, insert, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return gctx.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
}

func (QueryExecutor) Delete(ctx context.Context, table string, filters Filters) ([]byte, error) {
	gctx := GetGreenContext(ctx)
	query, err := gctx.QueryBuilder.BuildDelete(gctx.QueryStringParser, table, filters)
	if err != nil {
		return nil, err
	}
	rows, err := gctx.Conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return gctx.QueryBuilder.preferredSerializer().Serialize(ctx, rows)
}
