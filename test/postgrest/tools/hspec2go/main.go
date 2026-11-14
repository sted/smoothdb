// Command hspec2go converts the hspec cases of a PostgREST spec file into
// Go cases of the ported suite, and compares the ported suite with the
// spec files (drift mode). See README.md.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fn := flag.String("func", "", "write a whole Go file with a test function of this name")
	drift := flag.Bool("drift", false, "compare the Go files given as arguments with the spec files")
	exclude := flag.String("exclude", "", "for -drift: spec files (comma-separated base names) left out of the coverage total")
	specDir := flag.String("spec", os.ExpandEnv("$HOME/Devel/postgrest-v16.4/test/spec/Feature"), "the spec directory, for -drift")
	flag.BoolVar(&Fix, "fix", false, "with -drift: add the checks the port does not do and refresh changed quotes, in place")
	flag.StringVar(&SkipReason, "skip", "", "port every case as skipped with this reason (a planned feature)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: hspec2go [-func Name] [-skip reason] File.hs\n       hspec2go -drift [-spec dir] port_test.go...\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *drift {
		ex := map[string]bool{}
		for _, f := range strings.Split(*exclude, ",") {
			if f != "" {
				ex[f] = true
			}
		}
		if err := runDrift(os.Stdout, *specDir, flag.Args(), ex); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	src, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f := ParseSpec(filepath.Base(flag.Arg(0)), string(src))
	indent := "\t\t"
	var out bytes.Buffer
	if *fn != "" {
		fmt.Fprintf(&out, "package postgrest\n\nimport (\n\t\"testing\"\n\n\t\"github.com/sted/smoothdb/test\"\n)\n\n")
		fmt.Fprintf(&out, "// %s holds the cases of %s (PostgREST 16.4), converted by hspec2go.\n", *fn, f.Name)
		fmt.Fprintf(&out, "func %s(t *testing.T) {\n\ttests := []test.Test{\n", *fn)
	}
	Emit(&out, f, indent)
	if *fn != "" {
		fmt.Fprintf(&out, "\t}\n\ttest.Execute(t, testConfig, tests)\n}\n")
		// a whole file can go through gofmt
		if src, err := format.Source(out.Bytes()); err == nil {
			out.Reset()
			out.Write(src)
		} else {
			fmt.Fprintln(os.Stderr, "gofmt:", err)
		}
	}
	os.Stdout.Write(out.Bytes())
	todo := 0
	for _, c := range f.Cases {
		if c.TODO != "" {
			todo++
		}
	}
	fmt.Fprintf(os.Stderr, "%s: %d it, %d requests, %d converted, %d TODO requests, %d TODO parts\n",
		f.Name, len(f.Its), len(f.Cases), len(f.Cases)-todo, todo, len(f.Leftovers))
}
