package database

import (
	"context"
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
		_, err = dbe.CreateDatabase(ctx, "test_base", true)
		if err != nil {
			t.Fatal(err)
		}
		err = dbe.DeleteDatabase(ctx, "test_base")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("CloneDatabase", func(t *testing.T) {
		_, err = dbe.CreateDatabase(ctx, "test_base", true)
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

		admin_ctx, admin_conn, _ := ContextWithDb(ctx, nil, "admin")
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
