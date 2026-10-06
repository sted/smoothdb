package database

import (
	"context"
)

type Record = map[string]any

func GetRecords(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	return Select(ctx, table, filters)
}

func CreateRecords(ctx context.Context, table string, records []Record, filters Filters) ([]byte, int64, error) {
	return Insert(ctx, table, records, filters)
}

// CreateRecordsFromJSON inserts the records decoded from body, a JSON object
// or array of objects, sending body itself as the rows, as PostgREST does.
func CreateRecordsFromJSON(ctx context.Context, table string, records []Record, body []byte, filters Filters) ([]byte, int64, error) {
	return InsertJSON(ctx, table, records, body, filters)
}

func UpdateRecords(ctx context.Context, table string, record Record, filters Filters) ([]byte, int64, error) {
	return Update(ctx, table, record, filters)
}

func DeleteRecords(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	return Delete(ctx, table, filters)
}
