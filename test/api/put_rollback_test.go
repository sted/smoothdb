package test_api

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// TestPutRollsBackStatementTrigger: with TransactionMode "none" (the suite's
// mode) a PUT whose body has not the key of its URL writes no row, but its
// statement still fires a FOR EACH STATEMENT trigger. The PUT answers 400
// (PGRST115) and the effect of the trigger is rolled back, as in PostgREST
// (MainTx.hs, failPut: SQL.condemn). A PUT that succeeds keeps it: the
// positive control that the trigger fires.
func TestPutRollsBackStatementTrigger(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Prepare(cmdConfig, []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/putaudited",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "putaudited",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true, "constraints": ["PRIMARY KEY"]},
					{"name": "v", "type": "int4"}
				]}`,
		},
	})

	ctx := context.Background()
	dbtest, err := pgx.ParseConfig(testDatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	dbtest.Database = "dbtest"
	conn, err := pgx.ConnectConfig(ctx, dbtest)
	if err != nil {
		t.Fatalf("cannot connect to dbtest: %v", err)
	}
	defer conn.Close(ctx)
	// the fixtures go away also when the test fails
	t.Cleanup(func() {
		c, err := pgx.ConnectConfig(ctx, dbtest)
		if err != nil {
			t.Errorf("cleanup: cannot connect to dbtest: %v", err)
			return
		}
		defer c.Close(ctx)
		if _, err := c.Exec(ctx, `DROP TABLE IF EXISTS putaudited, putaudit; DROP FUNCTION IF EXISTS putaudit_log()`); err != nil {
			t.Errorf("cleanup: %v", err)
		}
		// the later tests see the schema without the fixtures
		rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := srv.GetDBE().ReloadDatabaseSchema(rctx, "dbtest"); err != nil {
			t.Errorf("cleanup: cannot reload the schema cache of dbtest: %v", err)
		}
	})
	_, err = conn.Exec(ctx, `
		DROP TABLE IF EXISTS putaudit;
		CREATE TABLE putaudit (at timestamptz DEFAULT now());
		GRANT ALL ON putaudit TO PUBLIC;
		CREATE OR REPLACE FUNCTION putaudit_log() RETURNS trigger LANGUAGE plpgsql AS
			$$ BEGIN INSERT INTO putaudit DEFAULT VALUES; RETURN NULL; END $$;
		CREATE TRIGGER putaudit_log AFTER INSERT ON putaudited
			FOR EACH STATEMENT EXECUTE FUNCTION putaudit_log();`)
	if err != nil {
		t.Fatal(err)
	}
	audits := func() int {
		var n int
		if err := conn.QueryRow(ctx, "SELECT count(*) FROM putaudit").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	// The schema cache learns the key of the new table from a reload: until
	// then a PUT answers 405. The reload is synchronous, with a deadline.
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = srv.GetDBE().ReloadDatabaseSchema(rctx, "dbtest")
	cancel()
	if err != nil {
		t.Fatalf("cannot reload the schema cache of dbtest: %v", err)
	}
	test.Execute(t, testConfig, []test.Test{{
		Description: "a PUT whose body has not the key of its URL",
		Method:      "PUT",
		Query:       "/putaudited?id=eq.1",
		Body:        `{"id": 2, "v": 2}`,
		Status:      400,
	}})
	if n := audits(); n != 0 {
		t.Errorf("a refused PUT kept the effect of its statement trigger: %d audit rows, want 0", n)
	}
	test.Execute(t, testConfig, []test.Test{{
		Description: "a PUT of the row of its URL",
		Method:      "PUT",
		Query:       "/putaudited?id=eq.1",
		Body:        `{"id": 1, "v": 1}`,
		Status:      204,
	}})
	if n := audits(); n != 1 {
		t.Errorf("the statement trigger of a PUT: %d audit rows, want 1", n)
	}
}
