package main

import (
	"fmt"
	"go/format"
	"os"
	"regexp"
	"sort"
	"strconv"

	"strings"
)

// fixItem is a port case matched to a spec request, for -fix.
type fixItem struct {
	pc   *PortCase
	c    *Case
	spec *SpecFile
	less [][3]string // checks the port does not do: field, spec value, port value
}

type edit struct {
	at, del int
	text    string
}

// Manual lists the cases whose stale quote -fix leaves to a person.
var Manual []string

var proseRE = regexp.MustCompile(`^\s*[A-Z][a-z]+\s+[a-z]`)
var ctxLineRE = regexp.MustCompile(`^\s*(context|describe)\s+"`)
var itLineRE = regexp.MustCompile(`^\s*(it|specify)\s+"`)
var noteWordRE = regexp.MustCompile(`^[A-Za-z][\w'-]*\s`)
var hspecWords = map[string]bool{"get": true, "post": true, "patch": true, "put": true, "delete": true, "request": true,
	"it": true, "context": true, "describe": true, "let": true, "in": true, "do": true, "when": true, "where": true}

// isNote tells a prose note of the port from quoted hspec text.
func isNote(l string) bool {
	t := strings.TrimSpace(l)
	if strings.HasPrefix(t, "(") || strings.HasPrefix(t, "@@") {
		return true
	}
	if !noteWordRE.MatchString(t) || strings.ContainsAny(t, "\"[]{}$=|`<\\") {
		return false
	}
	w := strings.Fields(t)[0]
	return !hspecWords[w] && !strings.HasPrefix(w, "method") && !strings.HasPrefix(w, "match")
}

// applyFixes edits the port files: it adds the checks the port does not do,
// and it replaces quoted hspec text that 16.4 changed with the 16.4 text.
// It keeps the notes of the port and never changes a value the port checks.
func applyFixes(items []fixItem, up string) (int, int, error) {
	byFile := map[string][]edit{}
	nChecks, nQuotes := 0, 0
	for _, it := range items {
		src, err := os.ReadFile(it.pc.File)
		if err != nil {
			return 0, 0, err
		}
		// the indentation of the literal
		ls := strings.LastIndexByte(string(src[:it.pc.LitOpen]), '\n') + 1
		indent := string(src[ls:it.pc.LitOpen])

		// 1. quoted text
		stale := false
		for _, q := range it.pc.Quoted {
			if isHspec(q) && !strings.Contains(up, squeeze(q)) {
				stale = true
			}
			// the quote names another it than the one of the request in 16.4
			if m := itQuoteRE.FindStringSubmatch(q); m != nil && itLineRE.MatchString(q) {
				if t, err := strconv.Unquote(`"` + m[1] + `"`); err == nil && t != it.c.It.Title {
					stale = true
				}
			}
		}
		// only a comment with nothing but hspec text: a note of the port can
		// be about another case
		pure := true
		for _, q := range it.pc.Quoted {
			if isNote(q) || proseRE.MatchString(q) {
				pure = false
			}
		}
		if stale && !pure {
			Manual = append(Manual, fmt.Sprintf("%s:%d", it.pc.File, it.pc.Line))
		}
		if stale && pure && it.pc.CmtStart >= 0 {
			var out []string
			hadIt := false
			for _, q := range it.pc.Quoted {
				if ctxLineRE.MatchString(q) {
					if strings.Contains(up, squeeze(q)) {
						out = append(out, q)
					} else if n := len(it.c.Path); n > 0 {
						// a context that 16.4 renamed: its innermost block
						out = append(out, it.spec.quoteLine(it.c.Path[n-1].Line))
					}
				}
				if itLineRE.MatchString(q) {
					hadIt = true
				}
			}
			if hadIt {
				out = append(out, it.spec.quoteLine(it.c.It.Line))
				if it.c.Index == 0 {
					// the bindings between the it and its first request
					for n := it.c.It.Line + 1; n < it.c.FirstLine; n++ {
						out = append(out, it.spec.quoteLine(n))
					}
				}
			}
			for n := it.c.FirstLine; n <= it.c.LastLine; n++ {
				out = append(out, it.spec.quoteLine(n))
			}
			for _, q := range it.pc.Quoted {
				if isNote(q) {
					out = append(out, q)
				}
			}
			var b strings.Builder
			for i, l := range out {
				if i > 0 {
					b.WriteString("\n" + indent)
				}
				if strings.TrimSpace(l) == "" {
					b.WriteString("//")
				} else {
					b.WriteString("// " + l)
				}
			}
			byFile[it.pc.File] = append(byFile[it.pc.File], edit{at: it.pc.CmtStart, del: it.pc.CmtEnd - it.pc.CmtStart, text: b.String()})
			nQuotes++
		}

		// 2. checks, but not those an "@@ not asserted" note of the port
		// names: a header by its name, the body by "body" or "message"
		var noted string
		for _, q := range it.pc.Quoted {
			if t := strings.TrimSpace(q); strings.HasPrefix(t, "@@ not asserted") || strings.HasPrefix(t, "@@ partly asserted") {
				noted += strings.ToLower(t) + " "
			}
		}
		var fields, hdrs []string
		for _, d := range it.less {
			if noted != "" {
				name := strings.ToLower(strings.SplitN(d[1], ":", 2)[0])
				if (d[0] == "ExpectedHeaders" && strings.Contains(noted, name)) ||
					(d[0] == "Expected" && (strings.Contains(noted, "body") || strings.Contains(noted, "message"))) {
					continue
				}
			}
			switch d[0] {
			case "Status":
				if !it.pc.HasStatus {
					fields = append(fields, "Status: "+d[1])
				}
			case "Expected":
				if it.pc.ExpVal[0] >= 0 {
					// an empty Expected: its value changes
					byFile[it.pc.File] = append(byFile[it.pc.File], edit{at: it.pc.ExpVal[0], del: it.pc.ExpVal[1] - it.pc.ExpVal[0], text: goString(it.c.Expected)})
				} else {
					fields = append(fields, "Expected: "+goString(it.c.Expected))
				}
			case "ExpectedEmpty":
				fields = append(fields, "ExpectedEmpty: true")
			case "ExpectedHeaders":
				for _, h := range it.c.ExpHeaders {
					if strings.HasPrefix(d[1], h[0]+": ") {
						hdrs = append(hdrs, fmt.Sprintf("%q: %q", h[0], h[1]))
						break
					}
				}
			}
		}
		hdrs = uniq(hdrs)
		if len(hdrs) > 0 && it.pc.HdrClose >= 0 {
			byFile[it.pc.File] = append(byFile[it.pc.File], insertBefore(src, it.pc.HdrClose, hdrs, indent+"\t"))
		} else if len(hdrs) > 0 {
			fields = append(fields, "ExpectedHeaders: map[string]string{"+strings.Join(hdrs, ", ")+"}")
		}
		if len(fields) > 0 {
			byFile[it.pc.File] = append(byFile[it.pc.File], insertBefore(src, it.pc.LitClose, fields, indent))
		}
		if len(fields)+len(hdrs) > 0 {
			nChecks++
		}
	}
	for fn, eds := range byFile {
		src, err := os.ReadFile(fn)
		if err != nil {
			return 0, 0, err
		}
		sort.SliceStable(eds, func(a, b int) bool { return eds[a].at > eds[b].at })
		for _, e := range eds {
			src = append(src[:e.at], append([]byte(e.text), src[e.at+e.del:]...)...)
		}
		out, err := format.Source(src)
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %v", fn, err)
		}
		if err := os.WriteFile(fn, out, 0o644); err != nil {
			return 0, 0, err
		}
	}
	return nChecks, nQuotes, nil
}

// insertBefore adds elements to a composite literal before its closing
// brace at offset close. On a brace of its own line, each element goes on
// a line of its own.
func insertBefore(src []byte, close int, elems []string, indent string) edit {
	ls := strings.LastIndexByte(string(src[:close]), '\n') + 1
	if strings.TrimSpace(string(src[ls:close])) == "" {
		var b strings.Builder
		for _, e := range elems {
			b.WriteString(indent + "\t" + e + ",\n")
		}
		return edit{at: ls, text: b.String()}
	}
	k := close - 1
	for k >= 0 && (src[k] == ' ' || src[k] == '\t' || src[k] == '\n') {
		k--
	}
	text := strings.Join(elems, ", ")
	if src[k] != ',' && src[k] != '{' {
		text = ", " + text
	}
	return edit{at: close, text: text}
}
