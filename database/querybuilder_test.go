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
			// complex booleans
			"?or=(or(and(a.eq.1,b.eq.93,c.eq.apple),and(e.eq.1,f.eq.93,g.eq.apple)),or(and(a.eq.1,b.eq.35,c.eq.apple),and(e.eq.1,f.eq.35,g.eq.apple)),or(and(a.eq.1,b.eq.25978,c.eq.apple),and(e.eq.1,f.eq.25978,g.eq.apple)))",
			`SELECT * FROM "table" WHERE (("table"."a" = $1 AND "table"."b" = $2 AND "table"."c" = $3 OR "table"."e" = $4 AND "table"."f" = $5 AND "table"."g" = $6) OR ("table"."a" = $7 AND "table"."b" = $8 AND "table"."c" = $9 OR "table"."e" = $10 AND "table"."f" = $11 AND "table"."g" = $12) OR ("table"."a" = $13 AND "table"."b" = $14 AND "table"."c" = $15 OR "table"."e" = $16 AND "table"."f" = $17 AND "table"."g" = $18))`,
			[]any{"1", "93", "apple", "1", "93", "apple", "1", "35", "apple", "1", "35", "apple", "1", "25978", "apple", "1", "25978", "apple"},
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
		{
			// simple aggregate function
			"?select=amount.sum()",
			`SELECT SUM("table"."amount") AS "sum" FROM "table"`,
			nil,
		},
		{
			// multiple aggregate functions
			"?select=amount.sum(),amount.avg(),count:id.count()",
			`SELECT SUM("table"."amount") AS "sum", AVG("table"."amount") AS "avg", COUNT("table"."id") AS "count" FROM "table"`,
			nil,
		},
		{
			// aggregate with grouping
			"?select=amount.sum(),customer_id",
			`SELECT SUM("table"."amount") AS "sum", "table"."customer_id" FROM "table" GROUP BY "table"."customer_id"`,
			nil,
		},
		{
			// aggregate with cast
			"?select=amount.avg()::int",
			`SELECT AVG("table"."amount")::int AS "avg" FROM "table"`,
			nil,
		},
		{
			// aggregate with where clause
			"?select=amount.sum()&status=eq.completed",
			`SELECT SUM("table"."amount") AS "sum" FROM "table" WHERE "table"."status" = $1`,
			[]any{"completed"},
		},
		{
			// count() without field
			"?select=count()",
			`SELECT COUNT(*) AS "count" FROM "table"`,
			nil,
		},
		{
			// count() with alias
			"?select=cnt:count()",
			`SELECT COUNT(*) AS "cnt" FROM "table"`,
			nil,
		},
		{
			// count() with cast
			"?select=count()::text",
			`SELECT COUNT(*)::text AS "count" FROM "table"`,
			nil,
		},
		{
			// count() with other fields (grouping)
			"?select=count(),customer_id",
			`SELECT COUNT(*) AS "count", "table"."customer_id" FROM "table" GROUP BY "table"."customer_id"`,
			nil,
		},
		{
			// JSON field with cast before aggregation
			"?select=jsonb_col->>key::integer.sum()",
			`SELECT SUM(("table"."jsonb_col"->>'key')::integer) AS "sum" FROM "table"`,
			nil,
		},
		{
			// JSON field with cast before and after aggregation
			"?select=s:jsonb_col->>key::integer.sum()::text",
			`SELECT SUM(("table"."jsonb_col"->>'key')::integer)::text AS "s" FROM "table"`,
			nil,
		},
		{
			// Regular field with cast before aggregation
			"?select=price::numeric.avg()",
			`SELECT AVG(("table"."price")::numeric) AS "avg" FROM "table"`,
			nil,
		},
		{
			// Regular field with cast before aggregation and after
			"?select=total:price::numeric.sum()::text",
			`SELECT SUM(("table"."price")::numeric)::text AS "total" FROM "table"`,
			nil,
		},
		{
			// Complex JSON aggregation with grouping
			"?select=project_id,total:invoice_total::numeric.sum(),count()",
			`SELECT "table"."project_id", SUM(("table"."invoice_total")::numeric) AS "total", COUNT(*) AS "count" FROM "table" GROUP BY "table"."project_id"`,
			nil,
		},
		{
			// Multiple JSON aggregations
			"?select=data->>value::integer.sum(),data->>count::integer.avg()",
			`SELECT SUM(("table"."data"->>'value')::integer) AS "sum", AVG(("table"."data"->>'count')::integer) AS "avg" FROM "table"`,
			nil,
		},
		{
			// Regular field cast without aggregation
			"?select=name::text,age::integer",
			`SELECT "table"."name"::text, "table"."age"::integer FROM "table"`,
			nil,
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
