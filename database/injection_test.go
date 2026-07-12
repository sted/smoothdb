package database

import (
	"net/url"
	"testing"
)

// These are regression tests for SQL-injection sinks where user-controlled
// tokens were interpolated into SQL text with bare quotes instead of being
// escaped/parameterized. The request parser lets a client smuggle a literal
// ' or " (and even separators) into a token via a backslash escape, so every
// interpolation site must quote/escape its input.
//
// Each case asserts the SAFE (properly escaped) SQL: the doubled quote is the
// evidence that the breakout attempt was neutralized.

func buildFromQuery(t *testing.T, query string) (string, []any) {
	t.Helper()
	u, err := url.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := PostgRestParser{}.parse("table", u.Query())
	if err != nil {
		t.Fatalf("parse error for %q: %v", query, err)
	}
	sql, values, err := DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
	if err != nil {
		t.Fatalf("build error for %q: %v", query, err)
	}
	return sql, values
}

// Finding 4: the select alias (AS "...") was built with raw quotes, so a "
// smuggled into the label broke out of the identifier into the SELECT list.
func TestInjectionSelectAliasQuoting(t *testing.T) {
	// label token becomes: x"    (the \" is an escaped literal double-quote)
	sql, _ := buildFromQuery(t, `?select=x\":a`)
	want := `SELECT "table"."a" AS "x""" FROM "table"`
	if sql != want {
		t.Errorf("alias not escaped\n want: %s\n  got: %s", want, sql)
	}
}

// Finding 3: JSON-path members were wrapped in bare single quotes, so a '
// smuggled into a path member broke out of the string literal.
func TestInjectionJSONPathQuoting(t *testing.T) {
	// path member token becomes: x'y
	sql, _ := buildFromQuery(t, `?select=data->>x\'y`)
	want := `SELECT ("table"."data"->>'x''y') AS "x'y" FROM "table"`
	if sql != want {
		t.Errorf("json path member not escaped\n want: %s\n  got: %s", want, sql)
	}
}

// Finding 2: full-text-search config arguments were interpolated with bare
// single quotes, so a ' in the fts(config) argument broke out of the literal.
func TestInjectionFTSConfigQuoting(t *testing.T) {
	// fts config token becomes: en'x
	sql, values := buildFromQuery(t, `?body=fts(en\'x).cat`)
	want := `SELECT * FROM "table" WHERE "table"."body" @@ to_tsquery('en''x', $1)`
	if sql != want {
		t.Errorf("fts config not escaped\n want: %s\n  got: %s", want, sql)
	}
	if len(values) != 1 || values[0] != "cat" {
		t.Errorf("fts value should be parameterized, got %v", values)
	}
}

// SET ROLE takes the role verbatim from the JWT claim. It is built by
// concatenation (PrepareConnection needs a live connection, so this locks in the
// quoting contract that line relies on): a malicious role must stay a single
// quoted identifier and cannot break out into additional statements.
func TestInjectionSetRoleQuoting(t *testing.T) {
	cases := map[string]string{
		`postgres; DROP TABLE users; --`: `SET ROLE "postgres; DROP TABLE users; --"`,
		`admin" ; DROP`:                  `SET ROLE "admin"" ; DROP"`,
	}
	for role, want := range cases {
		got := "SET ROLE " + quote(role)
		if got != want {
			t.Errorf("role %q not neutralized\n want: %s\n  got: %s", role, want, got)
		}
	}
}

// Finding 1: embedded-resource filter values take the nmarker == -1 branch of
// appendValue, which interpolated the value with bare single quotes instead of
// using a $N placeholder. This exercises that branch directly.
func TestInjectionEmbeddedValueEscaping(t *testing.T) {
	where, _, _ := appendValue("", `x' OR '1'='1`, nil, -1, false)
	want := `'x'' OR ''1''=''1'`
	if where != want {
		t.Errorf("interpolated value not escaped\n want: %s\n  got: %s", want, where)
	}
}

// Finding 5: a ::cast is interpolated verbatim into the SELECT list because a
// cast cannot be a bind parameter. The scanner lets a quoted token carry commas,
// spaces and parentheses, so an unvalidated cast can add columns or otherwise
// break out of the select list. Every cast must be rejected unless it is a
// syntactically valid type name.
func TestInjectionCastRejected(t *testing.T) {
	bad := []string{
		`?select=id::"int4, secret"`,           // smuggles an extra select item
		`?select=id::"int4 from pg_authid --"`, // breaks out of the select list
		`?select=id::"text)"`,                  // unbalanced parenthesis
		`?select=amount.sum()::"x, y"`,         // same, on the aggregate cast
	}
	for _, q := range bad {
		u, err := url.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (PostgRestParser{}).parse("table", u.Query()); err == nil {
			t.Errorf("expected parse error for malicious cast %q, got none", q)
		}
	}
}

// The counterpart to TestInjectionCastRejected: legitimate casts (including
// schema-qualified, array and multi-word SQL types) must still be accepted.
func TestValidCastsAccepted(t *testing.T) {
	good := []string{
		`?select=id::int4`,
		`?select=id::int8`,
		`?select=amount::numeric`,
		`?select=tags::text[]`,
		`?select=d::double precision`,
		`?select=t::public.mytype`,
		`?select=amount.sum()::int8`,
	}
	for _, q := range good {
		u, err := url.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (PostgRestParser{}).parse("table", u.Query()); err != nil {
			t.Errorf("valid cast %q rejected: %v", q, err)
		}
	}
}

// Finding (DDL sinks): database names and roles were interpolated into DDL with
// bare quotes (or none), so a " in a name broke out of the identifier. Every
// database-level DDL builder must emit the name as a single quoted identifier.
func TestInjectionDatabaseDDLQuoting(t *testing.T) {
	evil := `db"; DROP DATABASE x; --`
	q := `"db""; DROP DATABASE x; --"` // evil, neutralized as one identifier
	cases := []struct{ got, want string }{
		{buildCreateDatabase(evil, ""), `CREATE DATABASE ` + q},
		{buildCreateDatabase("d", evil), `CREATE DATABASE "d" OWNER ` + q},
		{buildAlterDatabaseOwner("d", evil), `ALTER DATABASE "d" OWNER TO ` + q},
		{buildAlterDatabaseRename("d", evil), `ALTER DATABASE "d" RENAME TO ` + q},
		{buildAlterDatabaseAllowConnections(evil, false), `ALTER DATABASE ` + q + ` WITH ALLOW_CONNECTIONS false`},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("DDL not neutralized\n want: %s\n  got: %s", c.want, c.got)
		}
	}
}

// Finding (grant sinks): privilege verbs and the target object type came from
// client JSON and were interpolated verbatim. They must be whitelisted, while
// the target name and grantee are quoted.
func TestInjectionGrantWhitelist(t *testing.T) {
	// Invalid input must be a *BuildError so the API maps it to 400, not 500.
	if _, err := buildGrantRevoke(&Privilege{
		Types: []string{"SELECT; DROP TABLE x"}, TargetType: "table", TargetName: "t", Grantee: "r",
	}, true); err == nil {
		t.Error("expected error for injected privilege verb")
	} else if _, ok := err.(*BuildError); !ok {
		t.Errorf("expected *BuildError, got %T", err)
	}
	if _, err := buildGrantRevoke(&Privilege{
		Types: []string{"SELECT"}, TargetType: "table; DROP", TargetName: "t", Grantee: "r",
	}, true); err == nil {
		t.Error("expected error for injected target type")
	} else if _, ok := err.(*BuildError); !ok {
		t.Errorf("expected *BuildError, got %T", err)
	}
	got, err := buildGrantRevoke(&Privilege{
		Types: []string{"SELECT", "INSERT"}, TargetType: "table", TargetName: "public.t", Grantee: "r",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := `GRANT SELECT, INSERT ON table "public"."t" TO "r"`; got != want {
		t.Errorf("unexpected grant\n want: %s\n  got: %s", want, got)
	}
}
