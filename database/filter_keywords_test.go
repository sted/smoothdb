package database

import (
	"reflect"
	"testing"
)

// Card 35450 filter-keywords. A bare null, true, false, unknown or not_null
// operand was lower-cased and written into the SQL for every operator, so
// eq.true on a text column was `= true` (42883), eq.NULL was `= null` (never
// true) and eq.not_null was `= NOT NULL` (42601). As in PostgREST 16.4
// (SqlFragment.hs pgFmtFilter), every operand is bound as a parameter except
// the operand of is, which is a keyword. In an any/all list an unquoted null
// is a SQL NULL element, as in the array literal PostgREST binds.
func TestFilterKeywordsAreBound(t *testing.T) {
	tests := []struct {
		query  string
		sql    string
		values []any
	}{
		{"?name=eq.true", `SELECT * FROM "table" WHERE "table"."name" = $1`, []any{"true"}},
		{"?name=eq.null", `SELECT * FROM "table" WHERE "table"."name" = $1`, []any{"null"}},
		{"?name=eq.NULL", `SELECT * FROM "table" WHERE "table"."name" = $1`, []any{"NULL"}},
		{"?name=neq.False", `SELECT * FROM "table" WHERE "table"."name" <> $1`, []any{"False"}},
		{"?name=eq.not_null", `SELECT * FROM "table" WHERE "table"."name" = $1`, []any{"not_null"}},
		{"?name=eq.unknown", `SELECT * FROM "table" WHERE "table"."name" = $1`, []any{"unknown"}},
		{"?name=in.(a,null)", `SELECT * FROM "table" WHERE "table"."name" IN ($1, $2)`, []any{"a", "null"}},
		{"?name=in.(TRUE,false)", `SELECT * FROM "table" WHERE "table"."name" IN ($1, $2)`, []any{"TRUE", "false"}},
		{"?name=fts.null", `SELECT * FROM "table" WHERE "table"."name" @@ to_tsquery($1)`, []any{"null"}},
		{"?name=fts(english).true", `SELECT * FROM "table" WHERE "table"."name" @@ to_tsquery('english', $1)`, []any{"true"}},
		{"?name=plfts.null", `SELECT * FROM "table" WHERE "table"."name" @@ plainto_tsquery($1)`, []any{"null"}},
		{"?name=gt.false", `SELECT * FROM "table" WHERE "table"."name" > $1`, []any{"false"}},
		// any/all: an unquoted null is a NULL element, a quoted one the string
		{"?name=eq(any).{a,null}", `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"a", nil}},
		{"?name=eq(any).{a,NULL}", `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"a", nil}},
		{"?name=eq(any).{a,%20null}", `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"a", nil}},
		{`?name=eq(any).{a,"null"}`, `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"a", "null"}},
		{"?name=like(all).{true,a*}", `SELECT * FROM "table" WHERE ("table"."name" LIKE $1 AND "table"."name" LIKE $2)`, []any{"true", "a%"}},
		{"?name=eq(any).{nullx,true}", `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"nullx", "true"}},
		// logic trees bind them too
		{"?or=(name.eq.null,name.eq.true)", `SELECT * FROM "table" WHERE ("table"."name" = $1 OR "table"."name" = $2)`, []any{"null", "true"}},
		{"?or=(name.eq(any).{x,null})", `SELECT * FROM "table" WHERE (("table"."name" = $1 OR "table"."name" = $2))`, []any{"x", nil}},
		// controls: the operand of is stays a keyword, any case
		{"?a=is.NULL", `SELECT * FROM "table" WHERE "table"."a" IS null`, nil},
		{"?a=is.null", `SELECT * FROM "table" WHERE "table"."a" IS null`, nil},
		{"?a=is.TRUE", `SELECT * FROM "table" WHERE "table"."a" IS true`, nil},
		{"?a=is.Not_Null", `SELECT * FROM "table" WHERE "table"."a" IS NOT NULL`, nil},
		{"?a=not.is.Unknown", `SELECT * FROM "table" WHERE NOT "table"."a" IS unknown`, nil},
		{"?or=(a.is.NULL,b.eq.null)", `SELECT * FROM "table" WHERE ("table"."a" IS null OR "table"."b" = $1)`, []any{"null"}},
	}
	for _, test := range tests {
		sql, values := buildFromQuery(t, test.query)
		if sql != test.sql {
			t.Errorf("%s\n want: %s\n  got: %s", test.query, test.sql, sql)
		}
		if len(values) == 0 && len(test.values) == 0 {
			continue
		}
		if !reflect.DeepEqual(values, test.values) {
			t.Errorf("%s: values %#v, want %#v", test.query, values, test.values)
		}
	}
}
