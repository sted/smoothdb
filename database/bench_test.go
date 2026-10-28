package database

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"testing"
)

// parse consumes the control parameters of the values it is given (select,
// order, limit...), so every iteration parses a copy.

// BenchmarkParse parses a typical query string: a select with embeds, filters,
// a logic tree, a list and an order.
func BenchmarkParse(b *testing.B) {
	u, err := url.Parse(`?select=id,name,title,author:authors(id,name),tags:tags(name)&age=gte.18&or=(status.eq.active,status.eq.pending)&name=in.(a,b,"c,d",e)&order=name.desc,id&limit=10`)
	if err != nil {
		b.Fatal(err)
	}
	q := u.Query()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (PostgRestParser{}).parse("t", maps.Clone(q)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseQuotedList parses an in.() list whose every element ends
// inside a quoted token, so that each one makes the parser rescan from its
// stop byte (see rawValue). The time per op must grow linearly with n.
func BenchmarkParseQuotedList(b *testing.B) {
	for _, n := range []int{1000, 4000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			q := url.Values{"a": {"in.(" + strings.Repeat(`x.",`, n) + "x)"}}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := (PostgRestParser{}).parse("t", maps.Clone(q)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkBuildSelectEmbed parses and builds a select with a labelled embed,
// a filter on it and an order. It parses at every iteration because building
// marks the parsed filters.
func BenchmarkBuildSelectEmbed(b *testing.B) {
	u, err := url.Parse(`?select=id,name,children:web_content(id,name)&children.name=eq.x&id=gt.10&order=id.desc`)
	if err != nil {
		b.Fatal(err)
	}
	q := u.Query()
	info := selfEmbedInfo()
	b.ReportAllocs()
	for b.Loop() {
		parts, err := PostgRestParser{}.parse("web_content", maps.Clone(q))
		if err != nil {
			b.Fatal(err)
		}
		if _, _, err := (DirectQueryBuilder{}).BuildSelect("web_content", parts, &QueryOptions{}, info); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerialize serializes 10000 rows of text, integers, timestamps
// before and after 2000, floats, numerics and dates, with a new serializer at
// every iteration (BenchmarkSerializer reuses one, whose output keeps growing).
// The rows are read once and copied, so the database is out of the loop.
func BenchmarkSerialize(b *testing.B) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		b.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	gi := GetSmoothContext(ctx)
	info := gi.Db.info.Load()
	rows, err := gi.Conn.Query(ctx, `select 'name '||i as name, i as number,
		timestamp '1985-06-01 12:00:00.25' + i * interval '1 hour' as ts,
		i::float8 / 3 as f, (i::numeric / 7)::numeric(12,4) as n, date '2000-01-01' + i as d
		from generate_series(1, 10000) i`)
	if err != nil {
		b.Fatal(err)
	}
	copied, err := CopyRows(rows)
	rows.Close()
	if err != nil {
		b.Fatal(err)
	}
	for _, s := range []struct {
		name string
		new  func() TextSerializer
	}{
		{"JSON", func() TextSerializer { return &JSONSerializer{} }},
		{"CSV", func() TextSerializer { return &CSVSerializer{} }},
	} {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				copied.CurrentRow = -1
				if _, _, err := s.new().Serialize(copied, false, false, info); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
