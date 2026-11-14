package main

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// item is a case or a leftover of an it block, in the order of the source.
type item struct {
	first, last int
	c           *Case
	todo        string
}

// SkipReason, when set, is the Skip of every case Emit writes (-skip).
// A request that the tool cannot convert then becomes a skipped case with
// its method and path only, so that it is counted.
var SkipReason string

// Emit writes the Go cases of f: each case has its hspec text quoted above
// it. A case or a part of an it that the tool cannot convert is written as
// a TODO comment with its hspec text.
func Emit(w io.Writer, f *SpecFile, indent string) {
	quotedCtx := map[int]bool{}
	ci, li := 0, 0
	for _, it := range f.Its {
		var items []item
		for ci < len(f.Cases) && f.Cases[ci].It.Line == it.It.Line {
			c := &f.Cases[ci]
			items = append(items, item{first: c.FirstLine, last: c.LastLine, c: c, todo: c.TODO})
			ci++
		}
		for li < len(f.Leftovers) && f.Leftovers[li].It.Line == it.It.Line {
			l := f.Leftovers[li]
			items = append(items, item{first: l.FirstLine, last: l.LastLine,
				todo: "not a request with shouldRespondWith (assertions in Haskell code)"})
			li++
		}
		sort.SliceStable(items, func(a, b int) bool { return items[a].first < items[b].first })
		for _, p := range it.Path {
			if !quotedCtx[p.Line] {
				quotedCtx[p.Line] = true
				fmt.Fprintf(w, "%s// %s\n", indent, f.quoteLine(p.Line))
			}
		}
		if len(items) == 0 {
			f.quote(w, indent, it.First, it.Last)
			fmt.Fprintf(w, "%s// TODO(hspec2go): no request in this it\n", indent)
			f.emitSkeleton(w, &it, it.First, it.Last, indent)
			continue
		}
		next := it.First
		for _, x := range items {
			from := x.first
			if next < from {
				from = next
			}
			if x.last >= from {
				f.quote(w, indent, from, x.last)
			}
			if x.last+1 > next {
				next = x.last + 1
			}
			if x.todo != "" {
				fmt.Fprintf(w, "%s// TODO(hspec2go): %s\n", indent, oneLine(x.todo))
				f.emitSkeleton(w, &it, x.first, x.last, indent)
				continue
			}
			emitCase(w, x.c, indent)
		}
	}
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

func (f *SpecFile) quoteLine(n int) string {
	l := f.Lines[n-1]
	ind := len(l) - len(strings.TrimLeft(l, " "))
	if ind >= f.BaseInd {
		return strings.TrimRight(l[f.BaseInd:], " ")
	}
	return strings.TrimSpace(l)
}

func (f *SpecFile) quote(w io.Writer, indent string, from, to int) {
	for n := from; n <= to; n++ {
		if strings.TrimSpace(f.Lines[n-1]) == "" {
			fmt.Fprintf(w, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(w, "%s// %s\n", indent, f.quoteLine(n))
	}
}

func goString(s string) string {
	if !strings.ContainsAny(s, "`\r") && strconv.CanBackquote(strings.ReplaceAll(s, "\n", "")) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func emitCase(w io.Writer, c *Case, indent string) {
	in := indent + "\t"
	fmt.Fprintf(w, "%s{\n", indent)
	fmt.Fprintf(w, "%sDescription: %s,\n", in, strconv.Quote(c.It.Title))
	fmt.Fprintf(w, "%sMethod: %q,\n", in, c.Method)
	fmt.Fprintf(w, "%sQuery: %s,\n", in, strconv.Quote(c.Query))
	if len(c.Headers) > 0 {
		fmt.Fprintf(w, "%sHeaders: test.Headers{%s},\n", in, headersLit(c.Headers))
	}
	if c.Body != "" {
		fmt.Fprintf(w, "%sBody: %s,\n", in, goString(c.Body))
	}
	if c.HasExpected {
		fmt.Fprintf(w, "%sExpected: %s,\n", in, goString(c.Expected))
	}
	if c.ExpEmpty {
		fmt.Fprintf(w, "%sExpectedEmpty: true,\n", in)
	}
	if len(c.ExpHeaders) > 0 {
		var parts []string
		seen := map[string]bool{}
		for _, h := range c.ExpHeaders {
			if seen[strings.ToLower(h[0])] {
				// test.Test checks one value per header name
				continue
			}
			seen[strings.ToLower(h[0])] = true
			parts = append(parts, fmt.Sprintf("%q: %q", h[0], h[1]))
		}
		fmt.Fprintf(w, "%sExpectedHeaders: map[string]string{%s},\n", in, strings.Join(parts, ", "))
	}
	fmt.Fprintf(w, "%sStatus: %d,\n", in, c.Status)
	if SkipReason != "" {
		fmt.Fprintf(w, "%sSkip: %q,\n", in, SkipReason)
	}
	fmt.Fprintf(w, "%s},\n", indent)
}

var methodRE = regexp.MustCompile(`\bmethod(Get|Post|Patch|Put|Delete|Head|Options)\b|\b(get|post|patch|put|delete)\s+"`)
var pathRE = regexp.MustCompile(`"(/[^"]*)"`)

// emitSkeleton writes, in -skip mode, a skipped case for the first request
// in lines [from, to] that the tool did not convert.
func (f *SpecFile) emitSkeleton(w io.Writer, it *ItBlock, from, to int, indent string) {
	if SkipReason == "" {
		return
	}
	src := strings.Join(f.Lines[from-1:to], "\n")
	p := pathRE.FindStringSubmatch(src)
	if p == nil {
		return
	}
	method := "GET"
	if m := methodRE.FindStringSubmatch(src); m != nil {
		method = strings.ToUpper(m[1] + m[2])
	}
	in := indent + "\t"
	fmt.Fprintf(w, "%s{\n", indent)
	fmt.Fprintf(w, "%sDescription: %s,\n", in, strconv.Quote(it.It.Title))
	fmt.Fprintf(w, "%sMethod: %q,\n", in, method)
	fmt.Fprintf(w, "%sQuery: %s,\n", in, strconv.Quote(p[1]))
	fmt.Fprintf(w, "%sSkip: %q,\n", in, SkipReason+"; the expectations are not converted")
	fmt.Fprintf(w, "%s},\n", indent)
}

func headersLit(h [][2]string) string {
	var names []string
	vals := map[string][]string{}
	for _, x := range h {
		if _, ok := vals[x[0]]; !ok {
			names = append(names, x[0])
		}
		vals[x[0]] = append(vals[x[0]], x[1])
	}
	var parts []string
	for _, n := range names {
		var vs []string
		for _, v := range vals[n] {
			vs = append(vs, strconv.Quote(v))
		}
		parts = append(parts, fmt.Sprintf("%q: {%s}", n, strings.Join(vs, ", ")))
	}
	return strings.Join(parts, ", ")
}
