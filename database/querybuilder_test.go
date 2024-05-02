package database

import (
	"net/url"
	"testing"
)

func compareValues(v1 []any, v2 []any) bool {
	if len(v1) != len(v2) {
		return false
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			return false
		}
	}
	return true
}

func TestQueryBuilder(t *testing.T) {

	tests := []struct {
		query       string
		expectedSQL string
		values      []any
	}{
		{
			// basic column selection
			"?select=a,b,c",
			`SELECT "table"."a", "table"."b", "table"."c" FROM "table"`,
			nil,
		},
		{
			// with labels
			"?select=Aa:a,b,Cc:c",
			`SELECT "table"."a" AS "Aa", "table"."b", "table"."c" AS "Cc" FROM "table"`,
			nil,
		},
		{
			// with casts and labels
			"?select=a::text,bbb:b::integer,c",
			`SELECT "table"."a"::text, "table"."b"::integer AS "bbb", "table"."c" FROM "table"`,
			nil,
		},
		{
			// skipping a column
			"?select=a,,c",
			`SELECT "table"."a", "table"."c" FROM "table"`,
			nil,
		},
		// {
		// 	// for resource embedding
		// 	"?select=a,table2(b,c)",
		// 	`SELECT a, c FROM "table"`,
		// },
		{
			// order by
			"?order=a,b",
			`SELECT * FROM "table" ORDER BY "table"."a", "table"."b"`,
			nil,
		},
		{
			// complex order by
			"?order=a.desc,b.asc,c.nullslast,d.desc.nullslast,e.asc.nullsfirst",
			`SELECT * FROM "table" ORDER BY "table"."a" DESC, "table"."b", "table"."c", "table"."d" DESC NULLS LAST, "table"."e" NULLS FIRST`,
			nil,
		},
		{
			// limit and offest
			"?order=a,b&limit=20&offset=100",
			`SELECT * FROM "table" ORDER BY "table"."a", "table"."b" LIMIT $1 OFFSET $2`,
			[]any{int64(20), int64(100)},
		},
		{
			// simple where
			"?age=gte.12&age=lte.18&name=eq.pippo",
			`SELECT * FROM "table" WHERE "table"."age" >= $1 AND "table"."age" <= $2 AND "table"."name" = $3`,
			[]any{"12", "18", "pippo"},
		},
		{
			// complex where
			"?grade=gte.90&student=is.true&or=(age.eq.14,not.and(age.gte.11,age.lte.17))",
			`SELECT * FROM "table" WHERE "table"."grade" >= $1 AND ("table"."age" = $2 OR NOT ("table"."age" >= $3 AND "table"."age" <= $4)) AND "table"."student" IS true`,
			[]any{"90", "14", "11", "17"},
		},
		{
			// complex where 2
			"?not.or=(age.not.eq.14,and(age.gte.11,age.lte.17))&city=eq.milano",
			`SELECT * FROM "table" WHERE "table"."city" = $1 AND NOT (NOT "table"."age" = $2 OR "table"."age" >= $3 AND "table"."age" <= $4)`,
			[]any{"milano", "14", "11", "17"},
		},
		{
			// quotes
			"?&name=eq.\"Stefano,DelliPonti\"&zbackslash=eq.\"\\\\bs\\\"\"",
			`SELECT * FROM "table" WHERE "table"."name" = $1 AND "table"."zbackslash" = $2`,
			[]any{"Stefano,DelliPonti", "\\bs\""},
		},
		{
			// in
			"?age=in.(10,20,30)",
			`SELECT * FROM "table" WHERE "table"."age" IN ($1, $2, $3)`,
			[]any{"10", "20", "30"},
		},
		{
			// range 1
			"?period=ov.[2017-01-01,2017-06-30]",
			`SELECT * FROM "table" WHERE "table"."period" && $1`,
			[]any{"[2017-01-01,2017-06-30]"},
		},
		{
			// range 2
			"?period=cd.(2017-01-01,2017-06-30]",
			`SELECT * FROM "table" WHERE "table"."period" <@ $1`,
			[]any{"(2017-01-01,2017-06-30]"},
		},
		{
			// range 3
			"?period=adj.(2017-01-01,2017-06-30)",
			`SELECT * FROM "table" WHERE "table"."period" -|- $1`,
			[]any{"(2017-01-01,2017-06-30)"},
		},
		{
			// array
			"?tags=cd.{cool,swag}",
			`SELECT * FROM "table" WHERE "table"."tags" <@ $1`,
			[]any{"{\"cool\",\"swag\"}"},
		},
		{
			// json
			"?select=a->b->c,b->>c->d->e,pippo:c->d->e::int&jsondata->a->b=eq.{e:{f:2,g:[1,2]}}",
			`SELECT ("table"."a"->'b'->'c') AS "c", ("table"."b"->>'c'->'d'->'e') AS "e", ("table"."c"->'d'->'e')::int AS "pippo" FROM "table" WHERE ("table"."jsondata"->'a'->'b') = $1`,
			[]any{"{\"e\":{\"f\":2,\"g\":[1,2]}}"},
		},
	}

	for i, test := range tests {
		url, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("table", url.Query())
		if err != nil {
			t.Error(err)
		}
		query, values, err := DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
		if err != nil {
			t.Error(err)
		}
		if query != test.expectedSQL {
			t.Errorf("\n%d. Expected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.expectedSQL, query, test.query)
			continue
		}
		if !compareValues(values, test.values) {
			t.Errorf("\n%d. Expected values\n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.values, values, test.query)
		}
	}
}
