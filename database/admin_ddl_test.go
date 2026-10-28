package database

import (
	"context"
	"strings"
	"testing"
)

// The /admin DDL helpers had no test: these call each one directly. Every
// object they create carries the lane6_ prefix and is dropped at the end.

// GetUsers ran usersQuery, which filters on the authenticator role as $1,
// without passing it, so every GET /admin/users failed.
func TestAdminGetUsers(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	DeleteUser(ctx, "lane6_user")
	if _, err := CreateUser(ctx, &User{Name: "lane6_user"}); err != nil {
		t.Fatal(err)
	}
	defer DeleteUser(ctx, "lane6_user")

	users, err := GetUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range users {
		if u.Name == "lane6_user" {
			found = true
		}
	}
	if !found {
		t.Errorf("lane6_user missing from GetUsers: %+v", users)
	}
}

// lane6DB activates a fresh database named name and returns a context bound to
// it as postgres; the database is dropped when the test ends.
func lane6DB(t *testing.T, name string) context.Context {
	t.Helper()
	mctx, mconn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(mctx, name)
	db, err := dbe.GetOrCreateActiveDatabase(mctx, name)
	if err != nil {
		ReleaseConn(mctx, mconn)
		t.Fatal(err)
	}
	ctx, conn, err := ContextWithDb(context.Background(), db, "postgres")
	if err != nil {
		ReleaseConn(mctx, mconn)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ReleaseConn(ctx, conn)
		if err := dbe.DeleteDatabase(mctx, name); err != nil {
			t.Errorf("dropping %s: %v", name, err)
		}
		ReleaseConn(mctx, mconn)
	})
	return ctx
}

func lane6Exec(t *testing.T, ctx context.Context, sqls ...string) {
	t.Helper()
	for _, sql := range sqls {
		if _, err := GetConn(ctx).Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

// GetView selected five columns and scanned four, so pgx refused every read
// of a single view.
func TestAdminGetView(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_views")
	lane6Exec(t, ctx, `CREATE TABLE lane6_t (id int)`)

	for _, materialized := range []bool{false, true} {
		name := "lane6_v"
		if materialized {
			name = "lane6_mv"
		}
		if _, err := CreateView(ctx, &View{Name: name, Materialized: materialized,
			Definition: "SELECT id FROM lane6_t"}); err != nil {
			t.Fatal(err)
		}
		view, err := GetView(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if view.Name != name || view.Schema != "public" || view.Owner != "postgres" ||
			view.Materialized != materialized || !strings.Contains(view.Definition, "lane6_t") {
			t.Errorf("GetView(%s): %+v", name, view)
		}
	}
}
