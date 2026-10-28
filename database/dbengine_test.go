package database

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestEngine(t *testing.T) {

	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	dbe.DeleteDatabase(ctx, "test_base")
	dbe.DeleteDatabase(ctx, "test_base_2")

	t.Run("GetOrCreateActiveDatabase: get", func(t *testing.T) {
		_, err = dbe.GetOrCreateActiveDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("DeleteDatabase", func(t *testing.T) {
		err = dbe.DeleteDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GetOrCreateActiveDatabase: create", func(t *testing.T) {
		_, err = dbe.GetOrCreateActiveDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
		err = dbe.DeleteDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("CreateDatabase", func(t *testing.T) {
		_, err = dbe.CreateDatabase(ctx, "test_base", "", true)
		if err != nil {
			t.Fatal(err)
		}
		err = dbe.DeleteDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("CloneDatabase", func(t *testing.T) {
		_, err = dbe.CreateDatabase(ctx, "test_base", "", true)
		if err != nil {
			t.Fatal(err)
		}
		// nobody is connected to test_base, so we do not need to forse sessions termination on it
		_, err = dbe.CloneDatabase(ctx, "test_base_2", "test_base", false)
		if err != nil {
			t.Fatal(err)
		}
		err = dbe.DeleteDatabase(ctx, "test_base_2")
		if err != nil {
			t.Fatal(err)
		}
		db, err := dbe.GetActiveDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
		// create a connection to test_base, so we will need to force deleting sessions to clone it
		nctx, nconn, _ := ContextWithDb(ctx, db, "test")

		admin_ctx, admin_conn, _ := ContextWithDb(ctx, nil, "postgres")
		defer ReleaseConn(admin_ctx, admin_conn)
		_, err = dbe.CloneDatabase(admin_ctx, "test_base_2", "test_base", true)
		if err != nil {
			t.Fatal(err)
		}
		err = dbe.DeleteDatabase(admin_ctx, "test_base_2")
		if err != nil {
			t.Fatal(err)
		}
		ReleaseConn(nctx, nconn)
		err = dbe.DeleteDatabase(admin_ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})
}

// Concurrent requests for a database that does not exist: the first caller
// fails in GetDatabase and wakes the waiters, which return the database when
// no activation error was recorded. That failure path recorded none, so a
// waiter got a *Database with no pool, and its first AcquireConnection
// panicked. Every caller must get the error.
func TestGetActiveDatabaseConcurrentFailure(t *testing.T) {
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		var wg sync.WaitGroup
		var unexpected atomic.Int32
		for g := 0; g < 32; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				db, err := dbe.GetActiveDatabase(ctx, "no_such_database_for_race")
				if err == nil || db != nil {
					unexpected.Add(1)
				}
			}()
		}
		wg.Wait()
		if n := unexpected.Load(); n > 0 {
			t.Fatalf("round %d: %d callers got a database without an error", round, n)
		}
	}
}
