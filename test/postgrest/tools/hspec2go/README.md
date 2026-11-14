# hspec2go

hspec2go reads the hspec cases of a PostgREST spec file (`test/spec/Feature/**/*Spec.hs`
in the PostgREST repository) and writes them as Go cases of the ported suite in
`test/postgrest`. It also compares the ported suite with the spec files (drift mode).

It is a small tool, not a Haskell parser. It knows the forms that most spec cases use
and writes a `TODO(hspec2go)` comment with the hspec text for everything else.
It never drops a part of the spec without a trace.

## Convert a spec file

```sh
go run ./test/postgrest/tools/hspec2go -func TestPostgREST_QueryLimited \
    ~/Devel/postgrest-v16.4/test/spec/Feature/Query/QueryLimitedSpec.hs > test/postgrest/query_limited_test.go
```

Without `-func`, the tool writes only the elements of the `[]test.Test` literal, to paste
into an existing file. The summary line on stderr gives the number of `it` blocks,
requests, converted requests and TODO parts.

Each case quotes its hspec text as a comment above the `test.Test`, as the rest of the
port does. An `it` with several requests gives several cases.

With `-skip "reason"`, every case gets that `Skip` reason: use it for a planned feature
("PostgREST 16.4: ...; card N (label)"). In this mode, a request that the tool cannot
convert also becomes a skipped case with its method and path, so that it is counted.

The tool converts these forms:

- `get`, `post`, `patch`, `put`, `delete`, `options`, and `request methodX path headers body`.
- Request headers: `[("Name", "value")]`, `[]`, `mempty`, `acceptHdrs`, `rangeHdrs` and
  `rangeHdrsWithCount` (without `Range-Unit`, as in the rest of the port), `authHeaderJWT`,
  `authHeader`, `planHdr`, `hContentType` and the other `hName` constants, joined with `++`.
- Bodies: strings, `[json|...|]` (bare object keys are quoted) and `[str|...|]`.
- `shouldRespondWith` with a JSON block, a string (`""` gives `ExpectedEmpty`) or a status,
  and the record `{ matchStatus = N, matchHeaders = [...] }`.
- `"Name" <:> "value"`, `matchContentTypeJson`, `matchContentTypeSingular`, `matchCTArrayStrip`, `matchCTSingularStrip` and
  `matchHeaderAbsent hName` (an empty value in `ExpectedHeaders`).
- Simple `let` and `where` bindings: the tool puts the expression in place of the name.

It writes a TODO for `liftIO`, `<-`, `shouldBe`, `shouldSatisfy`, JWTs that Haskell code
generates, `matchHeaderValuePresent`, `#{...}` interpolations and the helpers it does not
know. Review the output before you commit it. A converted case is not a verified case:
run it, and give a case that fails a `Skip` reason with the card that tracks it.

## Drift mode

```sh
cd test/postgrest
go run ./tools/hspec2go -drift \
    -exclude CustomMediaSpec.hs,PostGISSpec.hs,TimezoneSpec.hs,NoSuperuserSpec.hs \
    *_test.go ../../server/*_test.go > drift.md
```

The tool reads every spec file under `-spec` (default `~/Devel/postgrest-v16.4/test/spec/Feature`)
and every `test.Test` literal in the Go files. It matches a port case to a spec request
by the quoted `it` title, the method and the path. A port case without hspec text of its
own is an added case and is only counted. The report has these parts:

- **Values that differ**: status, body, headers or request of the port differ from the spec,
  or the quoted `it` title is not in the spec.
- **What the port does not check**: the spec checks a status, a body or a header that the
  port does not check.
- **Quoted hspec lines not in 16.4**: the quoted text changed upstream. The tool removes
  all whitespace and `--` comments before it compares.
- **Spec cases missing from the port**, for each spec file, and the list of missing requests.
- **Coverage of the `it` blocks**: for each spec file, the `it` blocks whose port cases all
  run, partly run, are all skipped, or are missing. `-exclude A.hs,B.hs` leaves the spec
  files that smoothdb does not want out of the total.

A port case without quoted hspec text still counts when its method and path are a spec
request (`aggregates_test.go`). The tool does not follow cases that Go helpers build
(`rollback_test.go`): count those by hand.

With `-fix`, the drift mode also edits the port files in place:

- It adds the checks the spec does and the port does not (status, body, `ExpectedEmpty`,
  headers). It never changes a value the port checks, and it leaves out a check that an
  "@@ not asserted" note of the case names.
- It replaces quoted hspec text that 16.4 changed, or that names another `it`, with the
  16.4 text. It does this only for a comment with nothing but hspec text, and lists the
  others ("quote to refresh by hand").

Run the suite after `-fix`. A new check that smoothdb fails becomes an "@@ not asserted"
note with its reason, or a skip when the case cannot pass at all. The port marks with
"@@ added" the cases that are not 16.4 cases (from an older spec, or smoothdb's own): the
drift mode counts them as added, up to the next `it` or `context` that 16.4 has.

A difference is not always an error. Some cases are deliberate (`deliberate: ...` skips)
or come from live runs of PostgREST 16.4. Read the case before you change an expectation.
