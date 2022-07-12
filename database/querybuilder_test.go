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
			"SELECT a, b, c FROM table",
		},
		{
			// with labels
			"?select=a:Aa,b,c:Cc",
			"SELECT a AS Aa, b, c AS Cc FROM table",
		},
		{
			// with casts and labels
			"?select=a::text,b:bbb::integer,c",
			"SELECT a::text, b::integer AS bbb, c FROM table",
		},
		{
			// skipping a column
			"?select=a,,c",
			"SELECT a, c FROM table",
		},
		{
			// order by
			"?order=a,b",
			"SELECT * FROM table ORDER BY a, b",
		},
		{
			// complex order by
			"?order=a.desc,b.asc,c.nullslast,d.desc.nullslast,e.asc.nullsfirst",
			"SELECT * FROM table ORDER BY a DESC, b, c, d DESC NULLS LAST, e NULLS FIRST",
		},
		{
			// limit and offest
			"?order=a,b&limit=20&offset=100",
			"SELECT * FROM table ORDER BY a, b LIMIT 20 OFFSET 100",
		},
		{
			// simple where
			"?age=gte.12&age=lte.18&name=eq.pippo",
			"SELECT * FROM table WHERE age >= '12' AND age <= '18' AND name = 'pippo'",
		},
		{
			// complex where
			"?grade=gte.90&student=is.true&or=(age.eq.14,not.and(age.gte.11,age.lte.17))",
			"SELECT * FROM table WHERE grade >= '90' AND (age = '14' OR NOT (age >= '11' AND age <= '17')) AND student IS 'true'",
		},
		{
			// complex where 2
			"?not.or=(age.not.eq.14,and(age.gte.11,age.lte.17))&city=eq.milano",
			"SELECT * FROM table WHERE city = 'milano' AND NOT (NOT age = '14' OR age >= '11' AND age <= '17')",
		},
	}

	for i, test := range tests {
		url, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		query, err := DirectQueryBuilder{}.BuildSelect(PostgRestParser{}, "table", url.Query())
		if err != nil {
			t.Error(err)
		}
		if query != test.expectedSQL {
			t.Errorf("\n%d. Expected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.expectedSQL, query, test.query)
		}

	}
}
