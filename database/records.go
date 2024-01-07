package database

import (
	"context"
)

type Record = map[string]any

func GetRecords(ctx context.Context, table string, filters Filters) ([]byte, error) {
	return Select(ctx, table, filters)
}

func CreateRecords(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	return Insert(ctx, table, records, filters)
}

func UpdateRecords(ctx context.Context, table string, record Record, filters Filters) ([]byte, int64, error) {
	return Update(ctx, table, record, filters)
}

func DeleteRecords(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	return Delete(ctx, table, filters)
}
