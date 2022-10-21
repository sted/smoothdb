package database

import (
	"context"
)

type Record = map[string]any

func (db *Database) GetRecords(ctx context.Context, table string, filters Filters) ([]byte, error) {
	return db.exec.Select(ctx, table, filters)
}

func (db *Database) CreateRecords(ctx context.Context, table string, records []Record) ([]byte, int64, error) {
	return db.exec.Insert(ctx, table, records)
}

func (db *Database) UpdateRecords(ctx context.Context, table string, record Record, filters Filters) ([]byte, int64, error) {
	return db.exec.Update(ctx, table, record, filters)
}

func (db *Database) DeleteRecords(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	return db.exec.Delete(ctx, table, filters)
}
