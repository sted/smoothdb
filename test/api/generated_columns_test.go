package test_api

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// serverVersionNum reads the server_version_num of the test database server.
func serverVersionNum(t *testing.T) int {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("cannot connect to the test database: %v", err)
	}
	defer conn.Close(ctx)
	var version int
	if err := conn.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatalf("cannot read server_version_num: %v", err)
	}
	return version
}

// A generated column cannot be written. Like PostgREST, smoothdb sends what the
// client sent and reports PostgreSQL's refusal (SQLSTATE 428C9) as a 400 with
// PostgreSQL's message, instead of silently dropping the column: a select=* (or
// CSV export) → POST round trip over such a table is refused, and $info marks
// the column read-only so a client can know without discovering it by 428C9.
// PostgreSQL 18 adds VIRTUAL generated columns (its default kind); the table
// gets one only when the server supports them.
func TestGeneratedColumns(t *testing.T) {

	virtual := serverVersionNum(t) >= 180000

	columns := `{"name": "a", "type": "text"},
				{"name": "b", "type": "text", "constraints": ["GENERATED ALWAYS AS (a || '!') STORED"]}`
	row := `{"a":"x","b":"x!"}`
	updatedRow := `{"a":"y","b":"y!"}`
	csvRows := "a,b\nx,x!"
	infoColumns := `{"name":"a","type":"text","notnull":false,"default":null,"readonly":false,"comment":null,"constraints":null,"table":"generated_cols","schema":"public"},` +
		`{"name":"b","type":"text","notnull":false,"default":null,"generated":"stored","readonly":true,"comment":null,"constraints":null,"table":"generated_cols","schema":"public"}`
	if virtual {
		columns += `,
				{"name": "c", "type": "text", "constraints": ["GENERATED ALWAYS AS (a || '?') VIRTUAL"]}`
		row = `{"a":"x","b":"x!","c":"x?"}`
		updatedRow = `{"a":"y","b":"y!","c":"y?"}`
		csvRows = "a,b,c\nx,x!,x?"
		infoColumns += `,{"name":"c","type":"text","notnull":false,"default":null,"generated":"virtual","readonly":true,"comment":null,"constraints":null,"table":"generated_cols","schema":"public"}`
	}

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// drop table generated_cols
		{
			Method: "DELETE",
			Query:  "/dbtest/tables/generated_cols",
		},
		// create table generated_cols
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "generated_cols",
				"columns": [
				` + columns + `
				]}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	const insertRefused = `{"subsystem":"database","message":"cannot insert a non-DEFAULT value into column \"b\"","code":"428C9","hint":"","details":"Column \"b\" is a generated column.","position":0}`
	const updateRefused = `{"subsystem":"database","message":"column \"b\" can only be updated to DEFAULT","code":"428C9","hint":"","details":"Column \"b\" is a generated column.","position":0}`

	tests := []test.Test{
		// positive control: without the generated column the insert succeeds and
		// the generated value comes back
		{
			Description: "insert without the generated column succeeds and returns the generated value",
			Method:      "POST",
			Query:       "/generated_cols",
			Body:        `{"a": "x"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[` + row + `]`,
			Status:      201,
		},
		{
			Description: "select=* returns the generated column",
			Query:       "/generated_cols?select=*",
			Expected:    `[` + row + `]`,
			Status:      200,
		},
		// the round trip: what select=* returned, posted back
		{
			Description: "posting a select=* row back is refused with 400 and 428C9",
			Method:      "POST",
			Query:       "/generated_cols",
			Body:        `[` + row + `]`,
			Expected:    insertRefused,
			Status:      400,
		},
		{
			Description: "csv export returns the generated column",
			Query:       "/generated_cols",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected:    csvRows,
			Status:      200,
		},
		{
			Description: "posting the csv export back is refused with 400 and 428C9",
			Method:      "POST",
			Query:       "/generated_cols",
			Body:        csvRows,
			Headers:     test.Headers{"Content-Type": {"text/csv"}},
			Expected:    insertRefused,
			Status:      400,
		},
		{
			Description: "updating the generated column is refused with 400 and 428C9",
			Method:      "PATCH",
			Query:       "/generated_cols?a=eq.x",
			Body:        `{"b": "y"}`,
			Expected:    updateRefused,
			Status:      400,
		},
		// positive control: updating the source column recomputes the generated one
		{
			Description: "updating the source column succeeds and returns the recomputed value",
			Method:      "PATCH",
			Query:       "/generated_cols?a=eq.x",
			Body:        `{"a": "y"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[` + updatedRow + `]`,
			Status:      200,
		},
		// $info reports the generated columns read-only (with no default) and the others not
		{
			Description: "$info reports generated columns as read-only",
			Query:       "/$info/generated_cols",
			Expected: `{"name":"generated_cols","schema":"public","owner":"admin","comment":null,"rowsecurity":false,"columns":[` +
				infoColumns + `],"constraints":null,"hasindexes":false,"hastriggers":false,"ispartition":false}`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
