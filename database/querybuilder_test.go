package database

import (
	"net/url"
	"testing"
)

func TestQueryBuilder(t *testing.T) {

	tests := []struct {
		query       string
		expectedSQL string
	}{
		{
			// basic column selection
			"?select=a,b,c",
			`SELECT "table"."a", "table"."b", "table"."c" FROM "table"`,
		},
		{
			// with labels
			"?select=Aa:a,b,Cc:c",
			`SELECT "table"."a" AS "Aa", "table"."b", "table"."c" AS "Cc" FROM "table"`,
		},
		{
			// with casts and labels
			"?select=a::text,bbb:b::integer,c",
			`SELECT "table"."a"::text, "table"."b"::integer AS "bbb", "table"."c" FROM "table"`,
		},
		{
			// skipping a column
			"?select=a,,c",
			`SELECT "table"."a", "table"."c" FROM "table"`,
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
		},
		{
			// complex order by
			"?order=a.desc,b.asc,c.nullslast,d.desc.nullslast,e.asc.nullsfirst",
			`SELECT * FROM "table" ORDER BY "table"."a" DESC, "table"."b", "table"."c", "table"."d" DESC NULLS LAST, "table"."e" NULLS FIRST`,
		},
		{
			// limit and offest
			"?order=a,b&limit=20&offset=100",
			`SELECT * FROM "table" ORDER BY "table"."a", "table"."b" LIMIT 20 OFFSET 100`,
		},
		{
			// simple where
			"?age=gte.12&age=lte.18&name=eq.pippo",
			`SELECT * FROM "table" WHERE "table"."age" >= '12' AND "table"."age" <= '18' AND "table"."name" = 'pippo'`,
		},
		{
			// complex where
			"?grade=gte.90&student=is.true&or=(age.eq.14,not.and(age.gte.11,age.lte.17))",
			`SELECT * FROM "table" WHERE "table"."grade" >= '90' AND ("table"."age" = '14' OR NOT ("table"."age" >= '11' AND "table"."age" <= '17')) AND "table"."student" IS true`,
		},
		{
			// complex where 2
			"?not.or=(age.not.eq.14,and(age.gte.11,age.lte.17))&city=eq.milano",
			`SELECT * FROM "table" WHERE "table"."city" = 'milano' AND NOT (NOT "table"."age" = '14' OR "table"."age" >= '11' AND "table"."age" <= '17')`,
		},
		{
			// quotes
			"?&name=eq.\"Stefano,DelliPonti\"&zbackslash=eq.\"\\\\bs\\\"\"",
			`SELECT * FROM "table" WHERE "table"."name" = 'Stefano,DelliPonti' AND "table"."zbackslash" = '\bs"'`,
		},
		{
			// in
			"?age=in.(10,20,30)",
			`SELECT * FROM "table" WHERE "table"."age" IN ('10', '20', '30')`,
		},
		{
			// range 1
			"?period=ov.[2017-01-01,2017-06-30]",
			`SELECT * FROM "table" WHERE "table"."period" && '[2017-01-01,2017-06-30]'`,
		},
		{
			// range 2
			"?period=cd.(2017-01-01,2017-06-30]",
			`SELECT * FROM "table" WHERE "table"."period" <@ '(2017-01-01,2017-06-30]'`,
		},
		{
			// range 3
			"?period=adj.(2017-01-01,2017-06-30)",
			`SELECT * FROM "table" WHERE "table"."period" -|- '(2017-01-01,2017-06-30)'`,
		},
		{
			// array
			"?tags=cd.{cool,swag}",
			`SELECT * FROM "table" WHERE "table"."tags" <@ '{"cool","swag"}'`,
		},
		{
			// json
			"?select=a->b->c,b->>c->d->e,pippo:c->d->e::int&jsondata->a->b=eq.{e:{f:2,g:[1,2]}}",
			`SELECT ("table"."a"->'b'->'c') AS "c", ("table"."b"->>'c'->'d'->'e') AS "e", ("table"."c"->'d'->'e')::int AS "pippo" FROM "table" WHERE "table"."jsondata"->'a'->'b' = '{"e":{"f":2,"g":[1,2]}}'`,
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
		query, _, err := DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
		if err != nil {
			t.Error(err)
		}
		if query != test.expectedSQL {
			t.Errorf("\n%d. Expected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.expectedSQL, query, test.query)
		}

	}
}
