package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// PortCase is a test.Test literal of the port, with the hspec text quoted
// above it.
type PortCase struct {
	File      string
	Line      int
	ItTitle   string // the it quoted above, or the last one quoted before
	Quoted    []string
	Fields    map[string]string // evaluated string fields
	Headers   [][2]string
	ExpHdrs   map[string]string
	ExpHdrsOK bool
	Status    int
	ExpEmpty  bool
	Unevaled  []string // fields with an expression the tool cannot evaluate
}

type driftRow struct {
	where, it, req, field, spec, port, skip string
}

// kind is "less" when the port checks less than the spec (a status, body
// or header it does not check), and "conflict" when the values differ.
func (r driftRow) kind() string {
	if strings.HasPrefix(r.port, "(not checked)") || strings.HasPrefix(r.port, "(body not checked)") {
		return "less"
	}
	return "conflict"
}

var itQuoteRE = regexp.MustCompile(`\bit "((?:[^"\\]|\\.)*)"`)

// a quoted comment line that matches is hspec text, not a note of the port
var hspecRE = regexp.MustCompile(`^\s*(it|context|describe|get|post|patch|put|delete) "|shouldRespondWith|^\s*request\b|\bmethod[A-Z][a-z]+\b|\[json\||matchStatus|matchHeaders|<:>`)

// stripComment removes a Haskell -- comment that is not inside a string.
func stripComment(s string) string {
	in := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && in:
			i++
		case s[i] == '"':
			in = !in
		case !in && strings.HasPrefix(s[i:], "--") && (i == 0 || s[i-1] == ' '):
			return s[:i]
		}
	}
	return s
}

func runDrift(w io.Writer, specDir string, files []string) error {
	// the upstream spec
	var specs []*SpecFile
	err := filepath.Walk(specDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, "Spec.hs") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(specDir, p)
		specs = append(specs, ParseSpec(rel, string(src)))
		return nil
	})
	if err != nil {
		return err
	}
	if len(specs) == 0 {
		return fmt.Errorf("no spec file under %s", specDir)
	}
	byTitle := map[string][]*Case{}
	itTitles := map[string]bool{}
	caseFile := map[*Case]string{}
	var upText strings.Builder // the spec without whitespace
	for _, s := range specs {
		for i := range s.Cases {
			c := &s.Cases[i]
			byTitle[c.It.Title] = append(byTitle[c.It.Title], c)
			caseFile[c] = s.Name
		}
		for _, it := range s.Its {
			itTitles[it.It.Title] = true
		}
		for _, l := range s.Lines {
			upText.WriteString(squeeze(l))
		}
	}

	var rows, quoteRows []driftRow
	up := upText.String()
	used := map[*Case]bool{}
	unquoted := map[string]int{}
	portCount := map[string]int{}
	for _, fn := range files {
		pcs, err := readPort(fn)
		if err != nil {
			return err
		}
		portCount[filepath.Base(fn)] = len(pcs)
		// the spec file of this port file: the one with most title hits
		votes := map[string]int{}
		for _, pc := range pcs {
			for _, c := range byTitle[pc.ItTitle] {
				votes[caseFile[c]]++
			}
		}
		home := ""
		for f, n := range votes {
			if n > votes[home] || (n == votes[home] && f < home) {
				home = f
			}
		}
		for _, pc := range pcs {
			where := fmt.Sprintf("%s:%d", filepath.Base(pc.File), pc.Line)
			skip := pc.Fields["Skip"]
			for _, q := range pc.Quoted {
				if isHspec(q) && !strings.Contains(up, squeeze(q)) {
					quoteRows = append(quoteRows, driftRow{where: where, it: pc.ItTitle, spec: "", port: strings.TrimSpace(q)})
				}
			}
			own := false
			for _, q := range pc.Quoted {
				if isHspec(q) {
					own = true
				}
			}
			// a case without hspec text of its own is an added case, even
			// when it follows a quoted it
			if pc.ItTitle == "" || !own {
				unquoted[filepath.Base(pc.File)]++
				continue
			}
			cands := byTitle[pc.ItTitle]
			if len(cands) == 0 && itTitles[pc.ItTitle] {
				// the spec it has no request the tool converts: nothing to compare
				continue
			}
			if len(cands) == 0 {
				rows = append(rows, driftRow{where, pc.ItTitle, pc.req(), "it", "(no it with this title in 16.4)", "", skip})
				continue
			}
			c := pickCase(cands, pc, used, caseFile, home)
			if c == nil {
				var reqs []string
				for _, x := range cands {
					reqs = append(reqs, x.Method+" "+x.Query)
				}
				rows = append(rows, driftRow{where, pc.ItTitle, pc.req(), "request", strings.Join(uniq(reqs), "; "), pc.req(), skip})
				continue
			}
			used[c] = true
			if c.TODO != "" {
				continue // the spec case is not convertible: nothing to compare
			}
			for _, d := range compare(c, pc) {
				rows = append(rows, driftRow{where, pc.ItTitle, pc.req(), d[0], d[1], d[2], skip})
			}
		}
	}

	// report
	fmt.Fprintf(w, "# Drift of the port against PostgREST 16.4\n\n")
	for _, kind := range []string{"conflict", "less"} {
		title := "Values that differ"
		if kind == "less" {
			title = "What the port does not check"
		}
		fmt.Fprintf(w, "## %s\n\n| port | it | request | field | spec 16.4 | port | skip |\n|---|---|---|---|---|---|---|\n", title)
		for _, r := range rows {
			if r.kind() == kind {
				fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s |\n", r.where, cell(r.it), cell(r.req), r.field, cell(r.spec), cell(r.port), cell(r.skip))
			}
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "\n## Quoted hspec lines not in 16.4\n\n| port | it | quoted line |\n|---|---|---|\n")
	for _, r := range quoteRows {
		fmt.Fprintf(w, "| %s | %s | %s |\n", r.where, cell(r.it), cell(r.port))
	}
	fmt.Fprintf(w, "\n## Spec cases missing from the port\n\n")
	fmt.Fprintf(w, "| spec file | requests | matched | missing | missing, convertible | TODO in spec |\n|---|---|---|---|---|---|\n")
	type miss struct{ file, it, req, todo string }
	var misses []miss
	tot := [5]int{}
	for _, s := range specs {
		n, m, conv, todo := len(s.Cases), 0, 0, 0
		for i := range s.Cases {
			c := &s.Cases[i]
			if c.TODO != "" {
				todo++
			}
			if used[c] {
				m++
				continue
			}
			if c.TODO == "" {
				conv++
			}
			misses = append(misses, miss{s.Name, c.It.Title, c.Method + " " + c.Query, c.TODO})
		}
		fmt.Fprintf(w, "| %s | %d | %d | %d | %d | %d |\n", s.Name, n, m, n-m, conv, todo)
		tot[0] += n
		tot[1] += m
		tot[2] += n - m
		tot[3] += conv
		tot[4] += todo
	}
	fmt.Fprintf(w, "| total | %d | %d | %d | %d | %d |\n", tot[0], tot[1], tot[2], tot[3], tot[4])
	fmt.Fprintf(w, "\n### The missing requests\n\n| spec file | it | request | note |\n|---|---|---|---|\n")
	for _, m := range misses {
		note := ""
		if m.todo != "" {
			note = "TODO: " + oneLine(m.todo)
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", m.file, cell(m.it), cell(m.req), cell(note))
	}
	fmt.Fprintf(w, "\n## Port cases without a quoted it\n\n| port file | cases | without hspec text |\n|---|---|---|\n")
	var names []string
	for f := range portCount {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		fmt.Fprintf(w, "| %s | %d | %d |\n", f, portCount[f], unquoted[f])
	}
	return nil
}

func uniq(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func cell(s string) string {
	s = strings.ReplaceAll(oneLine(s), "|", `\|`)
	return s
}

// squeeze removes all whitespace: a quote may wrap lines another way
func squeeze(s string) string { return strings.Join(strings.Fields(stripComment(s)), "") }

func isHspec(s string) bool { return hspecRE.MatchString(s) }

func (pc *PortCase) req() string {
	m := pc.Fields["Method"]
	if m == "" {
		m = "GET"
	}
	return m + " " + pc.Fields["Query"]
}

func sameQuery(a, b string) bool {
	return a == b || unescape(strings.TrimPrefix(a, "/")) == unescape(strings.TrimPrefix(b, "/"))
}

var pctRE = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

// unescape decodes the %XX sequences of a path and leaves a lone % as it
// is: the spec writes some paths half encoded.
func unescape(s string) string {
	b := pctRE.ReplaceAllStringFunc(s, func(m string) string {
		v, _ := strconv.ParseUint(m[1:], 16, 8)
		return string([]byte{byte(v)})
	})
	return strings.ReplaceAll(b, "+", " ")
}

// pickCase returns the spec case of the port case: same it title, method
// and path, not used yet, in the home spec file first.
func pickCase(cands []*Case, pc *PortCase, used map[*Case]bool, caseFile map[*Case]string, home string) *Case {
	m := pc.Fields["Method"]
	if m == "" {
		m = "GET"
	}
	var best *Case
	for pass := 0; pass < 4 && best == nil; pass++ {
		for _, c := range cands {
			if pass%2 == 0 && caseFile[c] != home {
				continue
			}
			if pass < 2 && used[c] {
				continue
			}
			if c.Method == m && sameQuery(c.Query, pc.Fields["Query"]) {
				best = c
				break
			}
		}
	}
	if best == nil {
		// a spec request the tool did not convert has no method and path
		for _, c := range cands {
			if c.TODO != "" && !used[c] {
				return c
			}
		}
	}
	return best
}

func normJSON(s string) (string, bool) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return strings.TrimSpace(s), false
	}
	b, _ := json.Marshal(v)
	return string(b), true
}

// compare returns the differences between a spec case and a port case:
// field, spec value, port value.
func compare(c *Case, pc *PortCase) [][3]string {
	var d [][3]string
	switch {
	case pc.Status == 0 && !pc.unevaled("Status"):
		// test.Execute does not check a status of 0
		d = append(d, [3]string{"Status", strconv.Itoa(c.Status), "(not checked)"})
	case c.Status != pc.Status && !pc.unevaled("Status"):
		d = append(d, [3]string{"Status", strconv.Itoa(c.Status), strconv.Itoa(pc.Status)})
	}
	if c.HasExpected {
		pe, ok := pc.Fields["Expected"]
		if !ok || pe == "" {
			if pc.unevaled("Expected") {
				d = append(d, [3]string{"Expected", c.Expected, "(not checked: not evaluated by the tool)"})
			} else {
				d = append(d, [3]string{"Expected", c.Expected, "(body not checked)"})
			}
		} else {
			a, _ := normJSON(c.Expected)
			b, _ := normJSON(pe)
			if a != b {
				d = append(d, [3]string{"Expected", c.Expected, pe})
			}
		}
	}
	if c.ExpEmpty && !pc.ExpEmpty {
		if pc.Fields["Expected"] == "" && !pc.unevaled("Expected") {
			d = append(d, [3]string{"ExpectedEmpty", "true", "(body not checked)"})
		} else {
			d = append(d, [3]string{"ExpectedEmpty", "true", "false (" + pc.Fields["Expected"] + ")"})
		}
	}
	for _, h := range c.ExpHeaders {
		v, ok := lookupHdr(pc.ExpHdrs, h[0])
		switch {
		case !ok && !pc.unevaled("ExpectedHeaders"):
			d = append(d, [3]string{"ExpectedHeaders", h[0] + ": " + showHdr(h[1]), "(not checked)"})
		case ok && v != h[1]:
			d = append(d, [3]string{"ExpectedHeaders", h[0] + ": " + showHdr(h[1]), h[0] + ": " + showHdr(v)})
		}
	}
	// request body
	if !pc.unevaled("Body") {
		a, _ := normJSON(c.Body)
		b, _ := normJSON(pc.Fields["Body"])
		if a != b {
			d = append(d, [3]string{"Body", c.Body, pc.Fields["Body"]})
		}
	}
	// request headers
	if !pc.unevaled("Headers") {
		sh := hdrSet(c.Headers)
		ph := hdrSet(pc.Headers)
		if sh != ph {
			d = append(d, [3]string{"Headers", sh, ph})
		}
	}
	return d
}

func showHdr(v string) string {
	if v == "" {
		return "(absent)"
	}
	return v
}

func lookupHdr(m map[string]string, k string) (string, bool) {
	for n, v := range m {
		if strings.EqualFold(n, k) {
			return v, true
		}
	}
	return "", false
}

func hdrSet(h [][2]string) string {
	var s []string
	for _, x := range h {
		if strings.EqualFold(x[0], "Range-Unit") {
			continue
		}
		// a list header is the same as one header per element
		for _, v := range strings.Split(x[1], ",") {
			s = append(s, strings.ToLower(x[0])+": "+strings.TrimSpace(v))
		}
	}
	sort.Strings(s)
	return strings.Join(s, "; ")
}

func (pc *PortCase) unevaled(f string) bool {
	for _, x := range pc.Unevaled {
		if x == f {
			return true
		}
	}
	return false
}

// readPort returns the test.Test literals of a Go file, in source order,
// with the comments above each one.
func readPort(fn string) ([]*PortCase, error) {
	fset := gotoken.NewFileSet()
	file, err := parser.ParseFile(fset, fn, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var out []*PortCase
	var lists []*ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if at, ok := cl.Type.(*ast.ArrayType); ok && isTestType(at.Elt) {
			lists = append(lists, cl)
			return false
		}
		if isTestType(cl.Type) {
			lists = append(lists, &ast.CompositeLit{Elts: []ast.Expr{cl}, Lbrace: cl.Pos()})
			return false
		}
		return true
	})
	for _, l := range lists {
		prev := l.Lbrace
		title := ""
		for _, e := range l.Elts {
			cl, ok := e.(*ast.CompositeLit)
			if !ok {
				continue
			}
			var quoted []string
			for _, cg := range file.Comments {
				if cg.Pos() > prev && cg.End() < cl.Pos() {
					for _, c := range cg.List {
						quoted = append(quoted, strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), " "))
					}
				}
			}
			for _, q := range quoted {
				if m := itQuoteRE.FindStringSubmatch(q); m != nil {
					if t, err := strconv.Unquote(`"` + m[1] + `"`); err == nil {
						title = t
					} else {
						title = m[1]
					}
				}
			}
			pc := &PortCase{File: fn, Line: fset.Position(cl.Pos()).Line, ItTitle: title, Quoted: quoted, Fields: map[string]string{}}
			evalTest(cl, pc)
			out = append(out, pc)
			prev = cl.End()
		}
	}
	return out, nil
}

func isTestType(e ast.Expr) bool {
	se, ok := e.(*ast.SelectorExpr)
	if ok {
		return se.Sel.Name == "Test"
	}
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "Test"
}

func evalString(e ast.Expr) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind == gotoken.STRING {
			s, err := strconv.Unquote(x.Value)
			return s, err == nil
		}
	case *ast.BinaryExpr:
		if x.Op == gotoken.ADD {
			a, ok1 := evalString(x.X)
			b, ok2 := evalString(x.Y)
			return a + b, ok1 && ok2
		}
	case *ast.ParenExpr:
		return evalString(x.X)
	}
	return "", false
}

func evalTest(cl *ast.CompositeLit, pc *PortCase) {
	for _, e := range cl.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key := kv.Key.(*ast.Ident).Name
		switch key {
		case "Status":
			if bl, ok := kv.Value.(*ast.BasicLit); ok {
				pc.Status, _ = strconv.Atoi(bl.Value)
			} else {
				pc.Unevaled = append(pc.Unevaled, key)
			}
		case "ExpectedEmpty":
			if id, ok := kv.Value.(*ast.Ident); ok {
				pc.ExpEmpty = id.Name == "true"
			}
		case "Headers":
			m, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				if id, ok := kv.Value.(*ast.Ident); !ok || id.Name != "nil" {
					pc.Unevaled = append(pc.Unevaled, key)
				}
				continue
			}
			for _, x := range m.Elts {
				kv2 := x.(*ast.KeyValueExpr)
				name, ok1 := evalString(kv2.Key)
				vals, ok2 := kv2.Value.(*ast.CompositeLit)
				if !ok1 || !ok2 {
					pc.Unevaled = append(pc.Unevaled, key)
					break
				}
				for _, v := range vals.Elts {
					s, ok := evalString(v)
					if !ok {
						pc.Unevaled = append(pc.Unevaled, key)
					}
					pc.Headers = append(pc.Headers, [2]string{name, s})
				}
			}
		case "ExpectedHeaders":
			m, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				pc.Unevaled = append(pc.Unevaled, key)
				continue
			}
			pc.ExpHdrs = map[string]string{}
			for _, x := range m.Elts {
				kv2 := x.(*ast.KeyValueExpr)
				name, ok1 := evalString(kv2.Key)
				v, ok2 := evalString(kv2.Value)
				if !ok1 || !ok2 {
					pc.Unevaled = append(pc.Unevaled, key)
					continue
				}
				pc.ExpHdrs[name] = v
			}
		default:
			s, ok := evalString(kv.Value)
			if !ok {
				pc.Unevaled = append(pc.Unevaled, key)
				continue
			}
			pc.Fields[key] = s
		}
	}
}
