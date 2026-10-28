package database

import (
	"context"
	"testing"
)

// When preparing the connection fails (here SET ROLE to a role that does not
// exist), ContextWithDb returned the error without releasing the connection it
// had acquired, so every such call kept one out of the pool.
func TestContextWithDbReleasesOnPrepareFailure(t *testing.T) {
	db, err := dbe.GetMainDatabase(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	acquired := func() int32 { return db.pool.Stat().AcquiredConns() }
	before := acquired()
	for i := 0; i < 3; i++ {
		if _, _, err := ContextWithDb(context.Background(), nil, "no_such_role_for_leak"); err == nil {
			t.Fatal("expected SET ROLE to a missing role to fail")
		}
	}
	if after := acquired(); after != before {
		t.Errorf("acquired connections went from %d to %d after failed ContextWithDb calls", before, after)
	}
}

// pgx reports an error the server raises while executing a cached statement
// through rows.Err(), not Query. GetTypes checked it but returned the outer
// err, still nil, so a failed type query filled the schema cache with a
// partial list. The first call caches the statement; in an aborted
// transaction the second one fails at execution.
func TestGetTypesReportsQueryError(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	gi := GetSmoothContext(ctx)
	if _, err := GetTypes(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := gi.Conn.Exec(ctx, "BEGIN"); err != nil {
		t.Fatal(err)
	}
	defer gi.Conn.Exec(ctx, "ROLLBACK")
	if _, err := gi.Conn.Exec(ctx, "SELECT 1/0"); err == nil {
		t.Fatal("expected the division by zero to abort the transaction")
	}
	types, err := GetTypes(ctx)
	if err == nil {
		t.Errorf("expected an error from GetTypes in an aborted transaction, got %d types and nil", len(types))
	}
}
