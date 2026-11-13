package database

import (
	"context"
	"encoding/json"
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

// UpdateColumn quoted the column name only in ALTER ... TYPE and COMMENT and
// interpolated it raw in NOT NULL, DEFAULT and RENAME: a mixed-case or
// reserved name failed there, and any name was spliced into the DDL.
func TestAdminUpdateColumnQuoting(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_columns")
	lane6Exec(t, ctx, `CREATE TABLE lane6_t ("Mixed Case" int, "order" text, id int)`)

	yes := true
	def := "7"
	for _, name := range []string{"Mixed Case", "order"} {
		if err := UpdateColumn(ctx, "lane6_t", name, &ColumnUpdate{NotNull: &yes}); err != nil {
			t.Fatalf("%s NOT NULL: %v", name, err)
		}
	}
	if err := UpdateColumn(ctx, "lane6_t", "Mixed Case", &ColumnUpdate{Default: &def}); err != nil {
		t.Fatalf("SET DEFAULT: %v", err)
	}
	empty := ""
	if err := UpdateColumn(ctx, "lane6_t", "Mixed Case", &ColumnUpdate{Default: &empty}); err != nil {
		t.Fatalf("DROP DEFAULT: %v", err)
	}
	newName := "Renamed Col"
	if err := UpdateColumn(ctx, "lane6_t", "Mixed Case", &ColumnUpdate{Name: &newName}); err != nil {
		t.Fatalf("RENAME: %v", err)
	}
	// A name carrying SQL is an identifier, never statement text.
	evil := `id DROP NOT NULL; DROP TABLE lane6_t; --`
	if err := UpdateColumn(ctx, "lane6_t", evil, &ColumnUpdate{NotNull: &yes}); err == nil {
		t.Errorf("a column named %q should not exist", evil)
	}

	var cols []string
	rows, err := GetConn(ctx).Query(ctx, `SELECT attname || ':' || attnotnull || ':' || atthasdef
		FROM pg_attribute WHERE attrelid = 'lane6_t'::regclass AND attnum > 0 ORDER BY attnum`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"Renamed Col:true:false", "order:true:false", "id:false:false"}
	if strings.Join(cols, ",") != strings.Join(want, ",") {
		t.Errorf("columns of lane6_t: %v, want %v", cols, want)
	}
}

// UpdateTable renamed the table and then ran OWNER TO with the old name, so a
// PATCH with both name and owner always failed with 42P01.
func TestAdminUpdateTableRenameAndOwner(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_tables")
	lane6Exec(t, ctx, `CREATE TABLE lane6_t (id int)`, `CREATE SCHEMA lane6_s`)

	newName, owner := "lane6_t2", "postgres"
	if err := UpdateTable(ctx, "lane6_t", &TableUpdate{Name: &newName, Owner: &owner}); err != nil {
		t.Fatalf("rename and owner: %v", err)
	}
	// schema, name, owner and comment together
	schema, name3, comment := "lane6_s", "lane6_t3", "c"
	yes := true
	if err := UpdateTable(ctx, "lane6_t2", &TableUpdate{Schema: &schema, Name: &name3,
		Owner: &owner, Comment: &comment, RowSecurity: &yes}); err != nil {
		t.Fatalf("schema, rename and owner: %v", err)
	}
	var n int
	if err := GetConn(ctx).QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n
		ON n.oid = c.relnamespace WHERE relname = 'lane6_t3' AND nspname = 'lane6_s'
		AND relrowsecurity AND obj_description(c.oid) = 'c'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("lane6_s.lane6_t3 missing after the update")
	}
	// UpdateSchema already renames last: owner and name together work.
	schema2 := "lane6_s2"
	if err := UpdateSchema(ctx, "lane6_s", &SchemaUpdate{Name: &schema2, Owner: &owner}); err != nil {
		t.Fatalf("schema rename and owner: %v", err)
	}
}

// The columns were listed with the pg_type name only, so numeric(10,2) read
// as numeric: the typmod was lost. typedef carries format_type.
func TestAdminColumnTypedef(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_typedef")
	lane6Exec(t, ctx, `CREATE DOMAIN lane6_d AS varchar(5)`,
		`CREATE TABLE lane6_t (a numeric(10,2), b varchar(20), c text[], d int, e lane6_d)`)

	want := map[string][2]string{
		"a": {"numeric", "numeric(10,2)"},
		"b": {"varchar", "character varying(20)"},
		"c": {"_text", "text[]"},
		"d": {"int4", "integer"},
		"e": {"varchar", "lane6_d"},
	}
	columns, err := GetColumns(ctx, "lane6_t")
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != len(want) {
		t.Fatalf("columns: %+v", columns)
	}
	for _, c := range columns {
		if got := [2]string{c.Type, c.Typedef}; got != want[c.Name] {
			t.Errorf("column %s: type, typedef = %v, want %v", c.Name, got, want[c.Name])
		}
	}
	c, err := GetColumn(ctx, "lane6_t", "a")
	if err != nil {
		t.Fatal(err)
	}
	if c.Typedef != "numeric(10,2)" {
		t.Errorf("GetColumn typedef = %q", c.Typedef)
	}
}

// The functions API wrote the arguments with Go field names and a byte mode,
// listed a zero-argument function with one empty argument, did not quote the
// argument names, broke on a body with $$, could not drop one overload,
// listed procedures and had no replace, volatility, strict or security.
func TestAdminFunctions(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_fn")
	lane6Exec(t, ctx, `CREATE PROCEDURE lane6_p() LANGUAGE sql AS 'SELECT 1'`,
		`CREATE FUNCTION lane6_o(int4) RETURNS int LANGUAGE sql AS 'SELECT 1'`,
		`CREATE FUNCTION lane6_o(int4, int4) RETURNS int LANGUAGE sql AS 'SELECT 2'`)

	var in Function
	if err := json.Unmarshal([]byte(`{"name": "lane6_f", "returns": "text",
		"arguments": [{"name": "My Arg", "type": "int4"}, {"name": "o", "type": "text", "mode": "out"}],
		"definition": "SELECT (\"My Arg\" + length($$a$$))::text || $fn$x$fn$",
		"volatility": "immutable", "strict": true, "security": "definer"}`), &in); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFunction(ctx, &in); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := CreateFunction(ctx, &in); err == nil {
		t.Errorf("a second create without replace should fail")
	}
	in.Replace = true
	if _, err := CreateFunction(ctx, &in); err != nil {
		t.Fatalf("create or replace: %v", err)
	}
	if _, err := CreateFunction(ctx, &Function{Name: "lane6_z", Returns: "int", Definition: "SELECT 1"}); err != nil {
		t.Fatalf("create zero-argument: %v", err)
	}
	for _, bad := range []Function{
		{Name: "lane6_bad", Definition: "SELECT 1", Volatility: "immutable; DROP TABLE x"},
		{Name: "lane6_bad", Definition: "SELECT 1", Security: "nobody"},
	} {
		if _, err := CreateFunction(ctx, &bad); err == nil {
			t.Errorf("%+v should be refused", bad)
		} else if _, ok := err.(*BuildError); !ok {
			t.Errorf("%+v: %T %v, want a BuildError", bad, err, err)
		}
	}

	functions, err := ListFunctions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string][]Function{}
	for _, f := range functions {
		byName[f.Name] = append(byName[f.Name], f)
	}
	if len(byName["lane6_p"]) != 0 {
		t.Errorf("the procedure lane6_p is listed")
	}
	if len(byName["lane6_o"]) != 2 {
		t.Errorf("lane6_o overloads: %+v", byName["lane6_o"])
	}
	f := byName["lane6_f"]
	if len(f) != 1 {
		t.Fatalf("lane6_f: %+v", f)
	}
	got, _ := json.Marshal(f[0].Arguments)
	if want := `[{"name":"My Arg","type":"integer","mode":"in","typeid":23},{"name":"o","type":"text","mode":"out","typeid":25}]`; string(got) != want {
		t.Errorf("lane6_f arguments: %s, want %s", got, want)
	}
	if f[0].Volatility != "i" || !f[0].Strict || f[0].Security != "definer" {
		t.Errorf("lane6_f: volatility %q strict %v security %q", f[0].Volatility, f[0].Strict, f[0].Security)
	}
	z := byName["lane6_z"]
	if len(z) != 1 {
		t.Fatalf("lane6_z: %+v", z)
	}
	got, _ = json.Marshal(z[0])
	if !strings.Contains(string(got), `"arguments":[]`) || !strings.Contains(string(got), `"strict":false`) ||
		!strings.Contains(string(got), `"security":"invoker"`) {
		t.Errorf("lane6_z: %s", got)
	}
	var out string
	if err := GetConn(ctx).QueryRow(ctx, `SELECT lane6_f(2)`).Scan(&out); err != nil || out != "3x" {
		t.Errorf("calling lane6_f: %q %v", out, err)
	}
	var n int
	if err := GetConn(ctx).QueryRow(ctx, `SELECT count(*) FROM pg_proc WHERE proname = 'lane6_f'`).Scan(&n); err != nil {
		t.Fatal(err)
	}

	// drop one overload by its argument types, then the zero-argument function
	if err := DeleteFunctionSignature(ctx, "lane6_o", []string{"int4", "integer"}); err != nil {
		t.Fatalf("drop lane6_o(int4, int4): %v", err)
	}
	if err := DeleteFunctionSignature(ctx, "lane6_z", []string{}); err != nil {
		t.Fatalf("drop lane6_z(): %v", err)
	}
	if err := DeleteFunctionSignature(ctx, "lane6_o", []string{"no_such_type"}); err == nil {
		t.Errorf("an unknown argument type should be refused")
	}
	if err := GetConn(ctx).QueryRow(ctx,
		`SELECT count(*) FROM pg_proc WHERE proname IN ('lane6_o', 'lane6_z')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("%d functions left, want lane6_o(int4) only", n)
	}
}

// DeleteView always ran DROP VIEW, so a materialized view could not be
// dropped; POST had no replace and there was no PATCH.
func TestAdminViewsUpdateReplaceDelete(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_views2")
	lane6Exec(t, ctx, `CREATE TABLE lane6_t (id int, n int)`, `CREATE SCHEMA lane6_s`)

	for _, materialized := range []bool{false, true} {
		name := "lane6_v"
		if materialized {
			name = "lane6_mv"
		}
		if _, err := CreateView(ctx, &View{Name: name, Materialized: materialized,
			Definition: "SELECT id FROM lane6_t"}); err != nil {
			t.Fatal(err)
		}
		_, err := CreateView(ctx, &View{Name: name, Materialized: materialized, Replace: true,
			Definition: "SELECT id, n FROM lane6_t"})
		if materialized {
			if _, ok := err.(*BuildError); !ok {
				t.Errorf("replace of a materialized view: %T %v, want a BuildError", err, err)
			}
		} else if err != nil {
			t.Errorf("replace of a view: %v", err)
		}
		newName, owner, schema := name+"2", "postgres", "lane6_s"
		if err := UpdateView(ctx, name, &ViewUpdate{Name: &newName, Owner: &owner, Schema: &schema}); err != nil {
			t.Fatalf("update %s: %v", name, err)
		}
		var n int
		if err := GetConn(ctx).QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n
			ON n.oid = c.relnamespace WHERE relname = $1 AND nspname = 'lane6_s'`, newName).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("lane6_s.%s missing after the update", newName)
		}
	}
	GetQueryOptions(ctx).Schema = "lane6_s"
	defer func() { GetQueryOptions(ctx).Schema = "public" }()
	for _, name := range []string{"lane6_v2", "lane6_mv2"} {
		if err := DeleteView(ctx, name); err != nil {
			t.Errorf("drop %s: %v", name, err)
		}
	}
}

// The policies took the schema from a dot in the table name, ignoring the
// profile header, concatenated command with no whitelist (an empty one gave
// "FOR  USING") and could not be altered.
func TestAdminPolicies(t *testing.T) {
	ctx := lane6DB(t, "lane6_admin_policies")
	lane6Exec(t, ctx, `CREATE SCHEMA lane6_s`, `CREATE TABLE lane6_s.lane6_t (id int)`,
		`CREATE ROLE lane6_pol_role`)
	t.Cleanup(func() { lane6Exec(t, ctx, `DROP OWNED BY lane6_pol_role`, `DROP ROLE lane6_pol_role`) })
	GetQueryOptions(ctx).Schema = "lane6_s"
	defer func() { GetQueryOptions(ctx).Schema = "public" }()

	using := "id > 0"
	if _, err := CreatePolicy(ctx, &Policy{Name: "lane6_all", Table: "lane6_t", Using: &using}); err != nil {
		t.Fatalf("empty command: %v", err)
	}
	if _, err := CreatePolicy(ctx, &Policy{Name: "lane6_sel", Table: "lane6_t", Command: "select", Using: &using}); err != nil {
		t.Fatalf("select: %v", err)
	}
	_, err := CreatePolicy(ctx, &Policy{Name: "lane6_bad", Table: "lane6_t", Command: "SELECT USING (true); --"})
	if _, ok := err.(*BuildError); !ok {
		t.Errorf("a bad command: %T %v, want a BuildError", err, err)
	}

	newName, using2, check := "lane6_all2", "id > 1", "id < 10"
	if err := UpdatePolicy(ctx, "lane6_t", "lane6_all", &PolicyUpdate{Name: &newName,
		Roles: []string{"lane6_pol_role"}, Using: &using2, Check: &check}); err != nil {
		t.Fatalf("update: %v", err)
	}
	policies, err := GetPolicies(ctx, "lane6_t")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, p := range policies {
		got[p.Name] = p.Command + "|" + strings.Join(p.Roles, ",") + "|" + *p.Using
		if p.Check != nil {
			got[p.Name] += "|" + *p.Check
		}
	}
	want := map[string]string{
		"lane6_all2": "ALL|lane6_pol_role|(id > 1)|(id < 10)",
		"lane6_sel":  "SELECT|public|(id > 0)",
	}
	if len(got) != len(want) || got["lane6_all2"] != want["lane6_all2"] || got["lane6_sel"] != want["lane6_sel"] {
		t.Errorf("policies: %v, want %v", got, want)
	}
	for _, name := range []string{"lane6_all2", "lane6_sel"} {
		if err := DeletePolicy(ctx, "lane6_t", name); err != nil {
			t.Errorf("drop %s: %v", name, err)
		}
	}
}

// A column grant became a table grant (columns was ignored), and GET of the
// grants of a schema or a function always returned [].
func TestAdminGrantColumnsSchemaFunction(t *testing.T) {
	got, err := buildGrantRevoke(&Privilege{Types: []string{"UPDATE", "select"}, TargetType: "table",
		TargetName: "public.t", Columns: []string{"a", "B c"}, Grantee: "r"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := `GRANT UPDATE ("a", "B c"), select ("a", "B c") ON table "public"."t" TO "r"`; got != want {
		t.Errorf("column grant\n want: %s\n  got: %s", want, got)
	}
	for _, typ := range []string{"DELETE", "ALL"} {
		_, err = buildGrantRevoke(&Privilege{Types: []string{typ}, TargetType: "table",
			TargetName: "t", Columns: []string{"a"}, Grantee: "r"}, true)
		if _, ok := err.(*BuildError); !ok {
			t.Errorf("%s with columns: %T %v, want a BuildError", typ, err, err)
		}
	}

	ctx := lane6DB(t, "lane6_admin_grants")
	lane6Exec(t, ctx, `CREATE SCHEMA lane6_s`, `CREATE TABLE lane6_s.lane6_t (a int, b int)`,
		`CREATE FUNCTION lane6_s.lane6_f() RETURNS int LANGUAGE sql AS 'SELECT 1'`,
		`CREATE ROLE lane6_grantee`)
	t.Cleanup(func() { lane6Exec(t, ctx, `DROP OWNED BY lane6_grantee`, `DROP ROLE lane6_grantee`) })

	for _, p := range []Privilege{
		{Types: []string{"USAGE"}, TargetType: "schema", TargetName: "lane6_s", Grantee: "lane6_grantee"},
		{Types: []string{"EXECUTE"}, TargetType: "function", TargetName: "lane6_s.lane6_f", Grantee: "lane6_grantee"},
		{Types: []string{"UPDATE"}, TargetType: "table", TargetName: "lane6_s.lane6_t", Columns: []string{"b"}, Grantee: "lane6_grantee"},
	} {
		if _, err := CreatePrivilege(ctx, &p); err != nil {
			t.Fatalf("grant %+v: %v", p, err)
		}
	}
	var colPriv, tablePriv bool
	if err := GetConn(ctx).QueryRow(ctx, `SELECT has_column_privilege('lane6_grantee', 'lane6_s.lane6_t', 'b', 'UPDATE'),
		has_table_privilege('lane6_grantee', 'lane6_s.lane6_t', 'UPDATE')`).Scan(&colPriv, &tablePriv); err != nil {
		t.Fatal(err)
	}
	if !colPriv || tablePriv {
		t.Errorf("column grant: column privilege %v, table privilege %v", colPriv, tablePriv)
	}
	for _, c := range []struct{ typ, name, want string }{
		{"schema", "lane6_s", "USAGE"},
		{"function", "lane6_s.lane6_f", "EXECUTE"},
	} {
		privileges, err := GetPrivileges(ctx, c.typ, c.name)
		if err != nil {
			t.Fatalf("GetPrivileges(%s): %v", c.typ, err)
		}
		found := false
		for _, p := range privileges {
			if p.Grantee == "lane6_grantee" && strings.Join(p.Types, ",") == c.want && p.TargetType == c.typ {
				found = true
			}
		}
		if !found {
			t.Errorf("GetPrivileges(%s, %s): %+v", c.typ, c.name, privileges)
		}
	}
}
