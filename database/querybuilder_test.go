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
			// recursive with user filters (applied to both base case and recursive step)
			"?id=start.5&manager_id=recurse.3&is_active=is.true",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 AND "table"."is_active" IS true UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "__recursive" ON "table"."manager_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path) AND "table"."is_active" IS true) SELECT "__recursive".* FROM "__recursive"`,
			[]any{"5", 3},
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
		{
			// basic via — base case is start node, edges followed in recursive step
			"?id=after.1&id=recurse.all&edge=via(src_id,dst_id)",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $1 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "edge" ON "edge"."dst_id" = "table"."id" INNER JOIN "__recursive" ON "edge"."src_id" = "__recursive"."id" WHERE "__recursive".__depth < $2 AND NOT "table"."id" = ANY("__recursive".__path)) SELECT DISTINCT "__recursive".* FROM "__recursive" WHERE __depth > 0`,
			[]any{"1", 100},
		},
		{
			// via with edge filter — via values shifted by offset
			"?id=after.1&id=recurse.3&edge=via(src_id,dst_id)&edge.rel_type=eq.contains",
			`WITH RECURSIVE "__recursive" AS (SELECT "table".*, 0 AS __depth, ARRAY["table"."id"] AS __path FROM "table" WHERE "table"."id" = $2 UNION ALL SELECT "table".*, "__recursive".__depth + 1, "__recursive".__path || "table"."id" FROM "table" INNER JOIN "edge" ON "edge"."dst_id" = "table"."id" INNER JOIN "__recursive" ON "edge"."src_id" = "__recursive"."id" WHERE "__recursive".__depth < $3 AND NOT "table"."id" = ANY("__recursive".__path) AND "edge"."rel_type" = $1) SELECT DISTINCT "__recursive".* FROM "__recursive" WHERE __depth > 0`,
			[]any{"contains", "1", 3},
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
