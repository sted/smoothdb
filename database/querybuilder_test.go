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
			// jsonb: quoted string value on a json-typed path (issue #19)
			`?payload->status=eq."red"`,
			`SELECT * FROM "table" WHERE ("table"."payload"->'status') = $1`,
			[]any{`"red"`},
		},
		{
			// jsonb: string array containment on a json-typed path (issue #19)
			`?payload->tags=cs.["red"]`,
			`SELECT * FROM "table" WHERE ("table"."payload"->'tags') @> $1`,
			[]any{`["red"]`},
		},
		{
			// jsonb: whole-column equality keeps quoted strings and bare booleans (issue #19)
			`?payload=eq.{"nums":[1,2,3],"tags":["red","blue"],"mixed":["red",1,true]}`,
			`SELECT * FROM "table" WHERE "table"."payload" = $1`,
			[]any{`{"nums":[1,2,3],"tags":["red","blue"],"mixed":["red",1,true]}`},
		},
		{
			// jsonb: quoted "true" is a JSON string, not a boolean
			`?payload->flag=eq."true"`,
			`SELECT * FROM "table" WHERE ("table"."payload"->'flag') = $1`,
			[]any{`"true"`},
		},
		{
			// jsonb: bare boolean on a json-typed path stays bare
			`?payload->flag=eq.true`,
			`SELECT * FROM "table" WHERE ("table"."payload"->'flag') = $1`,
			[]any{"true"},
		},
		{
			// text result with ->> keeps stripping quotes
			`?payload->>status=eq."red"`,
			`SELECT * FROM "table" WHERE ("table"."payload"->>'status') = $1`,
			[]any{"red"},
		},
		// --- IS on JSON paths: the keyword operand can never be a bind parameter
		//     (`x IS $1` is a syntax error), whatever the path type. ---
		{
			// is.null on a ->> path
			"?payload->>k=is.null",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'k') IS null`,
			nil,
		},
		{
			// is.not_null on a ->> path
			"?payload->>k=is.not_null",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'k') IS NOT NULL`,
			nil,
		},
		{
			// not.is.null on a ->> path
			"?payload->>k=not.is.null",
			`SELECT * FROM "table" WHERE NOT ("table"."payload"->>'k') IS null`,
			nil,
		},
		{
			// is.true / is.false / is.unknown on a ->> path
			"?payload->>i=is.true&payload->>j=is.false&payload->>k=is.unknown",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'i') IS true AND ("table"."payload"->>'j') IS false AND ("table"."payload"->>'k') IS unknown`,
			nil,
		},
		{
			// is.null on a -> (json-typed) path
			"?payload->k=is.null",
			`SELECT * FROM "table" WHERE ("table"."payload"->'k') IS null`,
			nil,
		},
		{
			// is.not_null on a -> path
			"?payload->k=is.not_null",
			`SELECT * FROM "table" WHERE ("table"."payload"->'k') IS NOT NULL`,
			nil,
		},
		{
			// not.is.null on a -> path
			"?payload->k=not.is.null",
			`SELECT * FROM "table" WHERE NOT ("table"."payload"->'k') IS null`,
			nil,
		},
		{
			// is.true on a -> path
			"?payload->k=is.true",
			`SELECT * FROM "table" WHERE ("table"."payload"->'k') IS true`,
			nil,
		},
		{
			// plain column: IS keywords stay verbatim (unchanged)
			"?a=is.null&b=not.is.null&c=is.not_null",
			`SELECT * FROM "table" WHERE "table"."a" IS null AND NOT "table"."b" IS null AND "table"."c" IS NOT NULL`,
			nil,
		},
		{
			// every other operator on a JSON path keeps binding the value: with ->>
			// eq.null / eq.true compare against the strings 'null' / 'true' (PostgREST)
			"?payload->>j=eq.null&payload->>k=eq.true",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'j') = $1 AND ("table"."payload"->>'k') = $2`,
			[]any{"null", "true"},
		},
		{
			// jsonb containment on a -> path still binds
			"?payload->k=cs.{a:1}",
			`SELECT * FROM "table" WHERE ("table"."payload"->'k') @> $1`,
			[]any{`{"a":1}`},
		},
		{
			// isdistinct is not IS: PostgREST parses its operand as a plain value
			// and binds it (IS DISTINCT FROM $1 is valid SQL), so null is bound
			// on a JSON path
			"?payload->>k=isdistinct.null",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'k') IS DISTINCT FROM $1`,
			[]any{"null"},
		},
		{
			// isdistinct.null on a plain column: unchanged (keyword)
			"?a=isdistinct.null",
			`SELECT * FROM "table" WHERE "table"."a" IS DISTINCT FROM null`,
			nil,
		},
		{
			// IN on a ->> path keeps binding keyword-looking values
			"?payload->>k=in.(null,true)",
			`SELECT * FROM "table" WHERE ("table"."payload"->>'k') IN ($1, $2)`,
			[]any{"null", "true"},
		},
		{
			// jsonb: IN with quoted values on a json-typed path
			`?payload->status=in.("red","blue")`,
			`SELECT * FROM "table" WHERE ("table"."payload"->'status') IN ($1, $2)`,
			[]any{`"red"`, `"blue"`},
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
		// --- Filter values: a top-level value runs to the end of the parameter
		//     (PostgREST pSingleVal); inside in.(), any/all lists and logic
		//     trees it ends at the next separator unless quoted (pListElement,
		//     pLogicSingleVal). Before the fix at most one extra dot survived
		//     (completeIfFloat) and the rest was silently dropped. ---
		{
			// more than one dot
			"?ver=eq.1.2.3",
			`SELECT * FROM "table" WHERE "table"."ver" = $1`,
			[]any{"1.2.3"},
		},
		{
			// dots and an at sign
			"?mail=eq.a.b@c.com",
			`SELECT * FROM "table" WHERE "table"."mail" = $1`,
			[]any{"a.b@c.com"},
		},
		{
			// comma
			"?name=eq.a,b",
			`SELECT * FROM "table" WHERE "table"."name" = $1`,
			[]any{"a,b"},
		},
		{
			// one dot (worked before: positive control)
			"?name=eq.autoexec.bat",
			`SELECT * FROM "table" WHERE "table"."name" = $1`,
			[]any{"autoexec.bat"},
		},
		{
			// negated operator
			"?ver=not.eq.1.2.3",
			`SELECT * FROM "table" WHERE NOT "table"."ver" = $1`,
			[]any{"1.2.3"},
		},
		{
			// like: the star rewrite applies to the whole value
			"?name=like.*a.b*",
			`SELECT * FROM "table" WHERE "table"."name" LIKE $1`,
			[]any{"%a.b%"},
		},
		{
			// regex: the spec case "imatch..*YY.*" used to bind "."
			"?k=imatch..*YY.*",
			`SELECT * FROM "table" WHERE "table"."k" ~* $1`,
			[]any{".*YY.*"},
		},
		{
			// full text search with a language
			"?x=fts(english).a.b",
			`SELECT * FROM "table" WHERE "table"."x" @@ to_tsquery('english', $1)`,
			[]any{"a.b"},
		},
		{
			// colon, space after a dot, apostrophe, unbalanced double quote,
			// arrow, equals sign: all ordinary characters in a value
			"?a=eq.10:30&b=eq.Sidney K. Meier&c=eq.O'Brien&d=eq.5\" display&e=eq.a->b&f=eq.a=b",
			`SELECT * FROM "table" WHERE "table"."a" = $1 AND "table"."b" = $2 AND "table"."c" = $3 AND "table"."d" = $4 AND "table"."e" = $5 AND "table"."f" = $6`,
			[]any{"10:30", "Sidney K. Meier", "O'Brien", "5\" display", "a->b", "a=b"},
		},
		{
			// a bracket or a quote that does not span the whole value is literal
			"?a=eq.[draft] v2&b=eq.(1,2)x&c=eq.\"a\"b",
			`SELECT * FROM "table" WHERE "table"."a" = $1 AND "table"."b" = $2 AND "table"."c" = $3`,
			[]any{"[draft] v2", "(1,2)x", "\"a\"b"},
		},
		{
			// keyword followed by more: not a keyword
			"?x=eq.null.x",
			`SELECT * FROM "table" WHERE "table"."x" = $1`,
			[]any{"null.x"},
		},
		{
			// empty value (PostgREST: searching for an empty string)
			"?x=eq.",
			`SELECT * FROM "table" WHERE "table"."x" = $1`,
			[]any{""},
		},
		{
			// in: dotted elements, quoted element with a comma, empty set
			"?a=in.(a.b,c.d)&b=in.(1.2.3,x)&c=in.(\"a,b\",c.d)&d=in.()",
			`SELECT * FROM "table" WHERE "table"."a" IN ($1, $2) AND "table"."b" IN ($3, $4) AND "table"."c" IN ($5, $6) AND "table"."d" = ANY('{}')`,
			[]any{"a.b", "c.d", "1.2.3", "x", "a,b", "c.d"},
		},
		{
			// any/all list: dotted elements
			"?x=eq(any).{a.b,c.d.e}",
			`SELECT * FROM "table" WHERE ("table"."x" = $1 OR "table"."x" = $2)`,
			[]any{"a.b", "c.d.e"},
		},
		{
			// logic tree: a value ends at the next separator unless quoted
			"?or=(x.eq.a.b.c,y.eq.\"c,d\",z.in.(1.2.3,e.f))",
			`SELECT * FROM "table" WHERE ("table"."x" = $1 OR "table"."y" = $2 OR "table"."z" IN ($3, $4))`,
			[]any{"a.b.c", "c,d", "1.2.3", "e.f"},
		},
		{
			// logic tree: an empty value
			"?or=(x.eq.,y.eq.1)",
			`SELECT * FROM "table" WHERE ("table"."x" = $1 OR "table"."y" = $2)`,
			[]any{"", "1"},
		},
		{
			// a quoted value closes on the same quote character; a quote not at
			// the start of a value, or not followed by a separator, or left
			// open, is an ordinary character (PostgREST pQuotedValue)
			"?a=eq.\"O'Brien\"&b=in.(O'Brien,Smith)&c=in.(\")&d=eq.\"foo&e=in.(\"a\"b,c)&f=in.(\"a,b\"c,d)",
			`SELECT * FROM "table" WHERE "table"."a" = $1 AND "table"."b" IN ($2, $3) AND "table"."c" IN ($4) AND "table"."d" = $5 AND "table"."e" IN ($6, $7) AND "table"."f" IN ($8, $9, $10)`,
			[]any{"O'Brien", "O'Brien", "Smith", "\"", "\"foo", "\"a\"b", "c", "\"a", "b\"c", "d"},
		},
		{
			// a backslash escapes only inside quotes
			"?a=in.(\\)&b=eq.a\\.b&c=eq.\"a\\\"b\"",
			`SELECT * FROM "table" WHERE "table"."a" IN ($1) AND "table"."b" = $2 AND "table"."c" = $3`,
			[]any{"\\", "a\\.b", "a\"b"},
		},
		{
			// spaces: stripped after the parenthesis of a list (in.(  ) is the
			// empty set), kept anywhere else; an empty element is a value
			"?a=in.( \"a\")&b=in.(a, \"b\")&c=in.(  )&d=in.(a,)&e=in.( ,3)&f=eq. a",
			`SELECT * FROM "table" WHERE "table"."a" IN ($1) AND "table"."b" IN ($2, $3) AND "table"."c" = ANY('{}') AND "table"."d" IN ($4, $5) AND "table"."e" IN ($6, $7) AND "table"."f" = $8`,
			[]any{"a", "a", " \"b\"", "a", "", "", "3", " a"},
		},
		// --- Recursive queries ---
		{
			// basic recursive query
			"?id=start.5&manager_id=recurse.3",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive".* FROM "__recursive"`,
			[]any{"5", 3},
		},
		{
			// recursive with select and order
			"?id=start.1&parent_id=recurse.all&select=id,name&order=name.asc",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."parent_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive"."id", "__recursive"."name" FROM "__recursive" ORDER BY "__recursive"."name"`,
			[]any{"1", 100},
		},
		{
			// plain filters are RESULT filters — applied to the outer query, NOT
			// inside the CTE (the prune-the-walk behavior moved under the walk. prefix).
			"?id=start.5&manager_id=recurse.3&is_active=is.true",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive".* FROM "__recursive" WHERE "__recursive"."is_active" IS true`,
			[]any{"5", 3},
		},
		{
			// walk.* filters prune the traversal — injected inside BOTH CTE arms
			// (the sibling of the result-filter case above).
			"?id=start.5&manager_id=recurse.3&walk.is_active=is.true",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 AND "table"."is_active" IS true UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path) AND "table"."is_active" IS true) SELECT "__recursive".* FROM "__recursive"`,
			[]any{"5", 3},
		},
		{
			// walk-prune AND result-filter together. The result filter (mainWhere) is
			// built in BuildSelect first, so it claims $1; then walk $2, start $3, depth $4.
			"?id=start.5&manager_id=recurse.3&walk.name=eq.x&is_active=eq.y",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $3 AND "table"."name" = $2 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $4 AND NOT "table"."id" = ANY("__recursive".__path) AND "table"."name" = $2) SELECT "__recursive".* FROM "__recursive" WHERE "__recursive"."is_active" = $1`,
			[]any{"y", "x", "5", 3},
		},
		{
			// recursive with limit and offset
			"?id=start.5&manager_id=recurse.3&limit=10&offset=5",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive".* FROM "__recursive" LIMIT $3 OFFSET $4`,
			[]any{"5", 3, int64(10), int64(5)},
		},
		{
			// after operator — excludes the seed row
			"?id=after.5&manager_id=recurse.3",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive".* FROM "__recursive" WHERE __depth > 0`,
			[]any{"5", 3},
		},
		// --- Via (multi-table) recursive queries ---
		// The CTE carries only (node key, depth) and dedups with UNION: a whole-row CTE with
		// a per-path cycle guard enumerates every simple path of the graph, exponential in
		// depth. The seed is excluded from re-entry (any walk through it has a shorter
		// suffix, and `after` relies on it being at depth 0 only); a "__dedup" CTE keeps one
		// row per node at its shallowest depth (DISTINCT ON the key — no whole-row equality,
		// which json, xml or point columns lack); the table is joined back onto it. The edge
		// table is wrapped in a derived table of (__from, __to) pairs so each arm can use an
		// index on the known node.
		{
			// basic via — base case is the start node, edges followed in the recursive step
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table".* FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0`,
			[]any{"1", 100},
		},
		{
			// via with edge filter — inside the edge derived table, where the edge table is
			// in scope; via values shifted by offset
			"?id=after.1&id=recurse.3&edge=via(src_id,dst_id)&edge.rel_type=eq.contains",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $2 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge" WHERE "edge"."rel_type" = $1) "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $3 AND "table"."id" <> $2), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table".* FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0`,
			[]any{"contains", "1", 3},
		},
		{
			// via + user order + limit, no __depth: ORDER BY and LIMIT apply to the joined
			// result, after the dedup
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)&select=id,name&order=name.desc&limit=10",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table"."id", "table"."name" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0 ORDER BY "table"."name" DESC LIMIT $3`,
			[]any{"1", 100, int64(10)},
		},
		{
			// via + walk-prune filter (both CTE arms) + result filter (outer query). The
			// result filter is built first in BuildSelect and claims $1; then start $2, depth $3.
			"?id=start.1&id=recurse.all&edge=via(src_id,dst_id)&walk.is_active=is.true&name=eq.x",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $2 AND "table"."is_active" IS true UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $3 AND "table"."id" <> $2 AND "table"."is_active" IS true), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table".* FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "table"."name" = $1`,
			[]any{"x", "1", 100},
		},
		// --- __depth selectable + via min-depth dedup ---
		{
			// __depth is a selectable pseudo-column; single-table mode keeps a plain SELECT
			"?id=start.1&parent_id=recurse.all&select=id,name,__depth",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."parent_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive"."id", "__recursive"."name", "__recursive"."__depth" FROM "__recursive"`,
			[]any{"1", 100},
		},
		{
			// via + __depth: read from the "__dedup" CTE, which keeps the min depth per node
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)&select=id,__depth",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table"."id", "__dedup"."__depth" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0`,
			[]any{"1", 100},
		},
		{
			// via + __depth + user order: the ORDER BY targets the joined table directly
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)&select=id,__depth&order=id",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table"."id", "__dedup"."__depth" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0 ORDER BY "table"."id"`,
			[]any{"1", 100},
		},
		{
			// via ordered by an unselected __depth: a plain ORDER BY on the "__dedup" column,
			// nothing surfaces in the projection (as in single-table mode)
			"?id=start.1&id=recurse.all&edge=via(src_id,dst_id)&select=id&order=__depth.desc",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table"."id" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node ORDER BY "__dedup"."__depth" DESC`,
			[]any{"1", 100},
		},
		// --- __path: the path array, on request only ---
		{
			// single-table: the CTE always carries __path (one path per node on an FK tree),
			// selecting it just projects the column
			"?id=start.1&parent_id=recurse.all&select=id,__path",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."parent_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT "__recursive"."id", "__recursive"."__path" FROM "__recursive"`,
			[]any{"1", 100},
		},
		{
			// via + __path: the path-carrying shape — UNION ALL with the per-path cycle guard,
			// which enumerates every simple path (exponential in depth) — and the dedup keeps
			// the shallowest path per node, ties broken by the path itself
			"?id=start.1&id=recurse.all&edge=via(src_id,dst_id)&select=id,__depth,__path",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table"."id", "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth, __path FROM "__recursive" ORDER BY __node, __depth, __path) SELECT "table"."id", "__dedup"."__depth", "__dedup"."__path" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node`,
			[]any{"1", 100},
		},
		{
			// via ordered by __path without selecting it also carries the path
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)&select=id&order=__path",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table"."id", "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth, __path FROM "__recursive" ORDER BY __node, __depth, __path) SELECT "table"."id" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0 ORDER BY "__dedup"."__path"`,
			[]any{"1", 100},
		},
		{
			// a result filter on __path reads it from the CTE, so it carries the path too
			// (found in review: without it the outer WHERE named a column the CTE lacked)
			"?id=start.1&id=recurse.all&edge=via(src_id,dst_id)&select=id&__path=cs.{1,2}",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $2 UNION ALL SELECT "table"."id", "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $3 AND NOT "table"."id" = ANY("__recursive".__path)), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth, __path FROM "__recursive" ORDER BY __node, __depth, __path) SELECT "table"."id" FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup"."__path" @> $1`,
			[]any{"{1,2}", "1", 100},
		},
		// --- bidirectional via!both ---
		{
			// via!both: the edge derived table carries both orientations (UNION ALL), so
			// each arm is driven by an index on the known node instead of an OR join
			"?id=after.1&id=recurse.all&edge=via!both(src_id,dst_id)",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $1 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge" UNION ALL SELECT "edge"."dst_id", "edge"."src_id" FROM "edge") "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $2 AND "table"."id" <> $1), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table".* FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0`,
			[]any{"1", 100},
		},
		{
			// via!both with an edge filter: the filter is repeated in both arms (same marker)
			"?id=after.1&id=recurse.all&edge=via!both(src_id,dst_id)&edge.rel_type=eq.contains",
			`WITH RECURSIVE "__recursive" AS (SELECT "table"."id" AS __node, 0 AS __depth FROM "table" WHERE "table"."id" = $2 UNION SELECT "table"."id", "__recursive".__depth + 1 FROM "__recursive" INNER JOIN (SELECT "edge"."src_id" AS __from, "edge"."dst_id" AS __to FROM "edge" WHERE "edge"."rel_type" = $1 UNION ALL SELECT "edge"."dst_id", "edge"."src_id" FROM "edge" WHERE "edge"."rel_type" = $1) "__edge" ON "__edge".__from = "__recursive".__node INNER JOIN "table" ON "table"."id" = "__edge".__to WHERE "__recursive".__depth < $3 AND "table"."id" <> $2), "__dedup" AS (SELECT DISTINCT ON (__node) __node, __depth FROM "__recursive" ORDER BY __node, __depth) SELECT "table".* FROM "__dedup" INNER JOIN "table" ON "table"."id" = "__dedup".__node WHERE "__dedup".__depth > 0`,
			[]any{"contains", "1", 100},
		},
	}

	for i, test := range tests {
		url, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("table", url.Query())
		if err != nil {
			t.Errorf("\n%d. Unexpected parse error %q \n(query string -> \"%v\")", i, err, test.query)
			continue
		}
		query, values, err := DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
		if err != nil {
			t.Errorf("\n%d. Unexpected build error %q \n(query string -> \"%v\")", i, err, test.query)
			continue
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

func TestRecursiveParserErrors(t *testing.T) {
	errorTests := []struct {
		query string
		errMsg string
	}{
		{"?id=start.5", "'start' requires a 'recurse' operator"},
		{"?manager_id=recurse.3", "'recurse' requires a 'start' or 'after' operator"},
		{"?id=start.5&manager_id=recurse.0", "recurse depth must be a positive integer or 'all'"},
		{"?id=start.5&manager_id=recurse.abc", "recurse depth must be a positive integer or 'all'"},
		{"?id=start.5&manager_id=recurse.-1", "recurse depth must be a positive integer or 'all'"},
		{"?id=start.1&manager_id=recurse.3&edge=via(src_id,dst_id)", "'via' requires 'start' and 'recurse' to use the same field"},
		{"?id=start.1&id=recurse.3&edge=via(src_id)", "via requires two columns: via(from_col,to_col)"},
		{"?id=start.1&id=recurse.3&edge=via(,dst_id)", "via requires two columns: via(from_col,to_col)"},
		{"?id=start.1&id=recurse.3&edge=via!sideways(src_id,dst_id)", "via direction must be 'both': via!both(from_col,to_col)"},
	}

	for i, test := range errorTests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		_, err = PostgRestParser{}.parse("table", u.Query())
		if err == nil {
			t.Errorf("%d. Expected error for %q, got nil", i, test.query)
			continue
		}
		if err.Error() != test.errMsg {
			t.Errorf("%d. Expected error %q, got %q", i, test.errMsg, err.Error())
		}
	}
}

// TestFilterValueErrors: a comma inside a logic tree separates filters and a
// filter operator needs its delimiter, as in PostgREST; the operand of IS is
// checked whole.
func TestFilterValueErrors(t *testing.T) {
	errorTests := []struct {
		query  string
		errMsg string
	}{
		{"?or=(x.eq.a,b)", "'=' expected"},
		{"?x=eq", "'.' expected"},
		{"?x=in", "'.' expected"},
		{"?x=is.null.x", "IS operator requires null, not_null, true, false or unknown"},
		{"?x=in.(a", "')' expected"},
	}

	for i, test := range errorTests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		_, err = PostgRestParser{}.parse("table", u.Query())
		if err == nil {
			t.Errorf("%d. Expected error for %q, got nil", i, test.query)
			continue
		}
		if err.Error() != test.errMsg {
			t.Errorf("%d. Expected error %q, got %q", i, test.errMsg, err.Error())
		}
	}
}

// TestRecursiveBuildErrors covers errors raised while building the recursive
// SELECT (not during parsing) — e.g. aggregating over a walk.
func TestRecursiveBuildErrors(t *testing.T) {
	errorTests := []struct {
		query  string
		errMsg string
	}{
		{"?id=start.1&parent_id=recurse.all&select=parent_id,id.count()", "aggregate functions cannot be used with recursive queries"},
	}

	for i, test := range errorTests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("table", u.Query())
		if err != nil {
			t.Fatalf("%d. unexpected parse error for %q: %v", i, test.query, err)
		}
		_, _, err = DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
		if err == nil {
			t.Errorf("%d. Expected error for %q, got nil", i, test.query)
			continue
		}
		if err.Error() != test.errMsg {
			t.Errorf("%d. Expected error %q, got %q", i, test.errMsg, err.Error())
		}
	}
}

// TestBuildExecuteDeterministicOrder guards against the pg_stat_statements
// fragmentation bug (see CollHub doc 27565): named RPC parameters must be
// emitted in a stable order regardless of Go map iteration so identical calls
// hash to the same queryid.
func TestBuildExecuteDeterministicOrder(t *testing.T) {
	record := Record{
		"p_a": 1,
		"p_b": "hello",
		"p_c": 2,
		"p_d": 3.14,
		"p_e": true,
	}

	// Without SchemaInfo: fall back to alphabetical key order.
	t.Run("no schema info", func(t *testing.T) {
		var first string
		for i := 0; i < 50; i++ {
			q, _, err := CommonBuilder{}.BuildExecute("fn", record, &QueryParts{}, &QueryOptions{}, nil)
			if err != nil {
				t.Fatalf("BuildExecute error: %v", err)
			}
			if i == 0 {
				first = q
				continue
			}
			if q != first {
				t.Fatalf("non-deterministic SQL across runs:\n  first: %s\n  got:   %s", first, q)
			}
		}
		// Alphabetical: p_a, p_b, p_c, p_d, p_e
		want := `SELECT * FROM "fn"("p_a" := $1, "p_b" := $2, "p_c" := $3, "p_d" := $4, "p_e" := $5) t `
		if first != want {
			t.Errorf("expected alphabetical order\n  want: %s\n  got:  %s", want, first)
		}
	})

	// With SchemaInfo: use declared signature order.
	t.Run("with signature", func(t *testing.T) {
		info := &SchemaInfo{
			cachedTypes: map[uint32]Type{0: {}},
			cachedFunctions: map[string]Function{
				"fn": {
					Name:   "fn",
					Schema: "",
					Arguments: []Argument{
						{Name: "p_c", Mode: 'i'},
						{Name: "p_a", Mode: 'i'},
						{Name: "p_e", Mode: 'i'},
						{Name: "p_b", Mode: 'i'},
						{Name: "p_d", Mode: 'i'},
					},
				},
			},
		}
		var first string
		var firstValues []any
		for i := 0; i < 50; i++ {
			q, v, err := CommonBuilder{}.BuildExecute("fn", record, &QueryParts{}, &QueryOptions{}, info)
			if err != nil {
				t.Fatalf("BuildExecute error: %v", err)
			}
			if i == 0 {
				first = q
				firstValues = v
				continue
			}
			if q != first {
				t.Fatalf("non-deterministic SQL across runs:\n  first: %s\n  got:   %s", first, q)
			}
			if !compareValues(v, firstValues) {
				t.Fatalf("non-deterministic values across runs:\n  first: %v\n  got:   %v", firstValues, v)
			}
		}
		// Signature order: p_c, p_a, p_e, p_b, p_d
		want := `SELECT * FROM "fn"("p_c" := $1, "p_a" := $2, "p_e" := $3, "p_b" := $4, "p_d" := $5) t `
		if first != want {
			t.Errorf("expected signature order\n  want: %s\n  got:  %s", want, first)
		}
		wantValues := []any{2, 1, true, "hello", 3.14}
		if !compareValues(firstValues, wantValues) {
			t.Errorf("values must follow key order\n  want: %v\n  got:  %v", wantValues, firstValues)
		}
	})

	// Extra record keys not in signature come after, in alphabetical order.
	t.Run("extra keys after signature", func(t *testing.T) {
		info := &SchemaInfo{
			cachedTypes: map[uint32]Type{0: {}},
			cachedFunctions: map[string]Function{
				"fn": {
					Name:   "fn",
					Schema: "",
					Arguments: []Argument{
						{Name: "p_a", Mode: 'i'},
						{Name: "p_b", Mode: 'i'},
					},
				},
			},
		}
		rec := Record{"p_b": 2, "p_a": 1, "z_extra": 99, "a_extra": 98}
		var first string
		for i := 0; i < 50; i++ {
			q, _, err := CommonBuilder{}.BuildExecute("fn", rec, &QueryParts{}, &QueryOptions{}, info)
			if err != nil {
				t.Fatalf("BuildExecute error: %v", err)
			}
			if i == 0 {
				first = q
				continue
			}
			if q != first {
				t.Fatalf("non-deterministic SQL:\n  first: %s\n  got:   %s", first, q)
			}
		}
		want := `SELECT * FROM "fn"("p_a" := $1, "p_b" := $2, "a_extra" := $3, "z_extra" := $4) t `
		if first != want {
			t.Errorf("unexpected order\n  want: %s\n  got:  %s", want, first)
		}
	})

	// OUT/TABLE-mode arguments must not be emitted as input params.
	t.Run("skip out mode args", func(t *testing.T) {
		info := &SchemaInfo{
			cachedTypes: map[uint32]Type{0: {}},
			cachedFunctions: map[string]Function{
				"fn": {
					Name:   "fn",
					Schema: "",
					Arguments: []Argument{
						{Name: "p_in", Mode: 'i'},
						{Name: "p_out", Mode: 'o'},
						{Name: "p_tbl", Mode: 't'},
					},
				},
			},
		}
		rec := Record{"p_in": 1}
		q, _, err := CommonBuilder{}.BuildExecute("fn", rec, &QueryParts{}, &QueryOptions{}, info)
		if err != nil {
			t.Fatalf("BuildExecute error: %v", err)
		}
		want := `SELECT * FROM "fn"("p_in" := $1) t `
		if q != want {
			t.Errorf("OUT/TABLE args should be skipped\n  want: %s\n  got:  %s", want, q)
		}
	})
}

// TestBuildInsertKeys guards the column list of a bulk INSERT. Without
// ?columns= it is the key set shared by every object, so an array whose objects
// differ is refused ("All object keys must match", as PostgREST) instead of
// being built from the first object alone, which dropped the keys the others
// added. With ?columns= it is exactly the listed set for every row: an absent
// key is a NULL parameter, a key not listed is ignored. Columns are emitted in
// alphabetical order so the SQL text is stable across runs.
func TestBuildInsertKeys(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		records []Record
		wantSQL string
		wantVal []any
		wantErr string
	}{
		{
			name:    "key present only in a later object",
			records: []Record{{"id": 1}, {"id": 2, "body": "y"}},
			wantErr: "All object keys must match",
		},
		{
			name:    "key missing from a later object",
			records: []Record{{"id": 1, "body": "x"}, {"id": 2}},
			wantErr: "All object keys must match",
		},
		{
			name:    "a null-valued key is still a key",
			records: []Record{{"id": 1, "body": nil}, {"id": 2}},
			wantErr: "All object keys must match",
		},
		{
			name:    "an empty object among non-empty ones",
			records: []Record{{}, {"id": 1}},
			wantErr: "All object keys must match",
		},
		{
			name:    "same keys in a different order",
			records: []Record{{"id": 1, "body": "x"}, {"body": "y", "id": 2}},
			wantSQL: `INSERT INTO "t" ("body", "id") VALUES ($1, $2), ($3, $4)`,
			wantVal: []any{"x", 1, "y", 2},
		},
		{
			name:    "?columns= inserts the listed columns for every row",
			query:   "?columns=id,body",
			records: []Record{{"id": 1}, {"id": 2, "body": "y", "extra": true}},
			wantSQL: `INSERT INTO "t" ("body", "id") VALUES ($1, $2), ($3, $4)`,
			wantVal: []any{nil, 1, "y", 2},
		},
		{
			name:    "?columns= ignores the keys not listed",
			query:   "?columns=id",
			records: []Record{{"id": 1, "body": "x"}, {"id": 2}},
			wantSQL: `INSERT INTO "t" ("id") VALUES ($1), ($2)`,
			wantVal: []any{1, 2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u, err := url.Parse(test.query)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := PostgRestParser{}.parse("t", u.Query())
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			q, v, err := CommonBuilder{}.BuildInsert("t", test.records, parts, &QueryOptions{}, nil)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil (SQL: %s)", test.wantErr, q)
				}
				if _, ok := err.(*BuildError); !ok || err.Error() != test.wantErr {
					t.Fatalf("expected *BuildError %q, got %T %q", test.wantErr, err, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildInsert error: %v", err)
			}
			if q != test.wantSQL {
				t.Errorf("SQL\n  want: %s\n  got:  %s", test.wantSQL, q)
			}
			if !compareValues(v, test.wantVal) {
				t.Errorf("values\n  want: %v\n  got:  %v", test.wantVal, v)
			}
		})
	}
}

// TestBuildMutationOrder: on POST/PATCH/DELETE `order=` orders the returned
// representation (PostgREST 13.0.0, #3013 "Fix order= with POST, PATCH, PUT
// and DELETE requests"): the mutation is wrapped in the _source CTE and the
// outer select carries the ORDER BY. `limit`/`offset` are ignored on
// mutations, as PostgREST does since the same release dropped limited
// updates/deletes: every matching row is written and the representation is
// never cut. TestQueryBuilder is the positive control for the LIMIT/OFFSET
// emission on a SELECT.
func TestBuildMutationOrder(t *testing.T) {
	tests := []struct {
		method         string
		query          string
		representation bool
		expectedSQL    string
		values         []any
	}{
		{
			// order applies to the representation: CTE + outer ORDER BY
			"PATCH", "?id=gt.1&order=id.desc", true,
			`WITH _source AS (UPDATE "table" SET "body" = $1 WHERE "table"."id" > $2 RETURNING *) SELECT * FROM _source ORDER BY "_source"."id" DESC`,
			[]any{"x", "1"},
		},
		{
			// with a select the outer select carries the formatting, the
			// RETURNING exposes the raw columns
			"PATCH", "?select=id,n:name::text&order=name.asc.nullsfirst", true,
			`WITH _source AS (UPDATE "table" SET "body" = $1 RETURNING "table"."id", "table"."name") SELECT "_source"."id", "_source"."name"::text AS "n" FROM _source ORDER BY "_source"."name" NULLS FIRST`,
			[]any{"x"},
		},
		{
			// an order column outside the select is added to the RETURNING so
			// the outer ORDER BY can see it (it is not part of the output)
			"PATCH", "?select=name&order=id", true,
			`WITH _source AS (UPDATE "table" SET "body" = $1 RETURNING "table"."name", "table"."id") SELECT "_source"."name" FROM _source ORDER BY "_source"."id"`,
			[]any{"x"},
		},
		{
			// no representation: nothing to order, no CTE
			"PATCH", "?id=gt.1&order=id.desc", false,
			`UPDATE "table" SET "body" = $1 WHERE "table"."id" > $2`,
			[]any{"x", "1"},
		},
		{
			// no order: the plain RETURNING form is kept
			"PATCH", "?id=gt.1&select=id", true,
			`UPDATE "table" SET "body" = $1 WHERE "table"."id" > $2 RETURNING "table"."id"`,
			[]any{"x", "1"},
		},
		{
			// limit/offset are ignored on mutations (#3013): no LIMIT, no OFFSET,
			// no range values, every matching row is updated
			"PATCH", "?id=gt.1&order=id&limit=1&offset=1", true,
			`WITH _source AS (UPDATE "table" SET "body" = $1 WHERE "table"."id" > $2 RETURNING *) SELECT * FROM _source ORDER BY "_source"."id"`,
			[]any{"x", "1"},
		},
		{
			"PATCH", "?id=gt.1&limit=1", false,
			`UPDATE "table" SET "body" = $1 WHERE "table"."id" > $2`,
			[]any{"x", "1"},
		},
		{
			"DELETE", "?id=lt.3&order=id.desc", true,
			`WITH _source AS (DELETE FROM "table" WHERE "table"."id" < $1 RETURNING *) SELECT * FROM _source ORDER BY "_source"."id" DESC`,
			[]any{"3"},
		},
		{
			"DELETE", "?id=lt.3&select=id&limit=1&offset=1", true,
			`DELETE FROM "table" WHERE "table"."id" < $1 RETURNING "table"."id"`,
			[]any{"3"},
		},
		{
			"DELETE", "?id=lt.3&order=id&limit=1", false,
			`DELETE FROM "table" WHERE "table"."id" < $1`,
			[]any{"3"},
		},
		{
			"POST", "?select=id,body&order=id.desc", true,
			`WITH _source AS (INSERT INTO "table" ("body") VALUES ($1) RETURNING "table"."id", "table"."body") SELECT "_source"."id", "_source"."body" FROM _source ORDER BY "_source"."id" DESC`,
			[]any{"x"},
		},
		{
			"POST", "?order=id&limit=1", false,
			`INSERT INTO "table" ("body") VALUES ($1)`,
			[]any{"x"},
		},
	}

	for i, test := range tests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("table", u.Query())
		if err != nil {
			t.Fatalf("%d. unexpected parse error for %q: %v", i, test.query, err)
		}
		options := &QueryOptions{ReturnRepresentation: test.representation}
		record := Record{"body": "x"}
		var query string
		var values []any
		switch test.method {
		case "PATCH":
			query, values, err = CommonBuilder{}.BuildUpdate("table", record, parts, options, nil)
		case "DELETE":
			query, values, err = CommonBuilder{}.BuildDelete("table", parts, options, nil)
		case "POST":
			query, values, err = CommonBuilder{}.BuildInsert("table", []Record{record}, parts, options, nil)
		}
		if err != nil {
			t.Errorf("%d. unexpected build error for %s %q: %v", i, test.method, test.query, err)
			continue
		}
		if query != test.expectedSQL {
			t.Errorf("\n%d. Expected \n\t\"%v\", \ngot \n\t\"%v\" \n(%s %q)", i, test.expectedSQL, query, test.method, test.query)
			continue
		}
		if !compareValues(values, test.values) {
			t.Errorf("\n%d. Expected values\n\t\"%v\", \ngot \n\t\"%v\" \n(%s %q)", i, test.values, values, test.method, test.query)
		}
	}
}

// A json path on an array or composite column needs its to_jsonb wrapper
// also when the representation of a mutation is read through the _source CTE
// (an order= or an embed puts it there): the column type is looked up on the
// real table, not on the alias. Found in review.
func TestBuildMutationJsonPathThroughSource(t *testing.T) {
	info := &SchemaInfo{
		cachedTypes:       map[uint32]Type{0: {}},
		cachedColumnTypes: map[string]map[string]ColumnType{"table": {"tags": {Name: "tags", IsArray: true}}},
	}
	tests := []struct{ query, want string }{
		{
			"?select=tags->0",
			`UPDATE "table" SET "body" = $1 RETURNING (to_jsonb("table"."tags")->0) AS "tags"`,
		},
		{
			"?select=tags->0&order=id",
			`WITH _source AS (UPDATE "table" SET "body" = $1 RETURNING "table"."tags", "table"."id") SELECT (to_jsonb("_source"."tags")->0) AS "tags" FROM _source ORDER BY "_source"."id"`,
		},
	}
	for _, test := range tests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("table", u.Query())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		sql, _, err := CommonBuilder{}.BuildUpdate("table", Record{"body": "x"}, parts, &QueryOptions{ReturnRepresentation: true}, info)
		if err != nil {
			t.Fatalf("BuildUpdate error: %v", err)
		}
		if sql != test.want {
			t.Errorf("%s\n  want: %s\n  got:  %s", test.query, test.want, sql)
		}
	}
}
