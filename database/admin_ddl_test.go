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

// DeleteFunction quoted the name twice ("""public"""."""f"""), so dropping a
// function always failed with 42883.
func TestAdminDeleteFunction(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_functions")
	lane6Exec(t, ctx, `CREATE FUNCTION lane6_f() RETURNS int LANGUAGE sql AS 'SELECT 1'`)

	if err := DeleteFunction(ctx, "lane6_f"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := GetConn(ctx).QueryRow(ctx,
		`SELECT count(*) FROM pg_proc WHERE proname = 'lane6_f'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("lane6_f still exists after DeleteFunction")
	}
}

// CreateConstraint took the table name from the constraint name, so it
// altered a table that does not exist; the name itself was dropped.
func TestAdminCreateConstraint(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_constraints")
	lane6Exec(t, ctx, `CREATE TABLE lane6_t (id int, n int)`)

	if _, err := CreateConstraint(ctx, &Constraint{Name: "lane6_positive", Table: "lane6_t",
		Definition: "CHECK (id > 0)"}); err != nil {
		t.Fatal(err)
	}
	// Without a name PostgreSQL chooses one.
	if _, err := CreateConstraint(ctx, &Constraint{Table: "lane6_t",
		Definition: "UNIQUE (n)"}); err != nil {
		t.Fatal(err)
	}
	constraints, err := GetConstraints(ctx, "lane6_t")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, c := range constraints {
		names[c.Name] = c.Definition
	}
	if names["lane6_positive"] != "CHECK (id > 0)" || names["lane6_t_n_key"] != "UNIQUE (n)" || len(names) != 2 {
		t.Fatalf("constraints of lane6_t: %v", names)
	}
	if err := DeleteConstraint(ctx, "lane6_t", "lane6_positive"); err != nil {
		t.Fatal(err)
	}
	constraints, err = GetConstraints(ctx, "lane6_t")
	if err != nil {
		t.Fatal(err)
	}
	if len(constraints) != 1 || constraints[0].Name != "lane6_t_n_key" {
		t.Errorf("after DeleteConstraint: %+v", constraints)
	}
}

// UpdateRole ran a bare ALTER ROLE "x" when the update set no attribute, a
// syntax error, so a rename alone (or an empty update) always failed.
func TestAdminUpdateRole(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	for _, name := range []string{"lane6_role", "lane6_role2"} {
		DeleteRole(ctx, name)
		defer DeleteRole(ctx, name)
	}
	if _, err := CreateRole(ctx, &Role{Name: "lane6_role"}); err != nil {
		t.Fatal(err)
	}

	if err := UpdateRole(ctx, "lane6_role", &RoleUpdate{}); err != nil {
		t.Fatalf("empty update: %v", err)
	}
	newName := "lane6_role2"
	if err := UpdateRole(ctx, "lane6_role", &RoleUpdate{Name: &newName}); err != nil {
		t.Fatalf("rename only: %v", err)
	}
	yes := true
	if err := UpdateRole(ctx, "lane6_role2", &RoleUpdate{CanLogin: &yes}); err != nil {
		t.Fatalf("attribute only: %v", err)
	}
	role, err := GetRole(ctx, "lane6_role2")
	if err != nil {
		t.Fatal(err)
	}
	if !role.CanLogin {
		t.Errorf("lane6_role2 after update: %+v", role)
	}
}
