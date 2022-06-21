package database

import (
	"context"
)

func (db *Database) GetRecords(ctx context.Context, table string, filters Filters) ([]byte, error) {
	return db.exec.Select(ctx, table, filters)
}

func (db *Database) CreateRecords(ctx context.Context, table string, records []Record) ([]byte, error) {
	return db.exec.Insert(ctx, table, records)
}

func (db *Database) DeleteRecords(ctx context.Context, table string, filters Filters) ([]byte, error) {
	return db.exec.Delete(ctx, table, filters)
}
