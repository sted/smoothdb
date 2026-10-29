# The ported PostgREST spec suite

The tests in this directory are PostgREST's own spec cases, ported to Go and run
against smoothdb. They target **PostgREST 16.4** (`PostgRESTVersion` in
`main_test.go`): the cases come from `test/spec` of the PostgREST repository at
tag `v16.4`, and where a card and the spec disagree, PostgREST 16.4 run live is
the reference.

## How a case is written

Each case quotes its hspec block as a comment, followed by the Go `test.Test`
that ports it. The fixtures in `fixtures/` are PostgREST's, copied verbatim.

A case smoothdb does not pass is **not** left as a bare comment. It is ported
with a `Skip` reason naming the missing behaviour and the card that tracks it:

```go
// it "fails with 405 on DELETE" $
//   request methodDelete "/rpc/getitemrange" ...
{
	Description: "DELETE on an RPC answers 405",
	Method:      "DELETE",
	Query:       "/rpc/getitemrange",
	Status:      405,
	Skip:        "PostgREST 16.4: 405 on DELETE /rpc; card 32490 (rpc details)",
},
```

A case that needs a fixture the copied fixtures lack names it in the reason
("needs fixture <name>"). A case that is deliberately different from PostgREST
says so ("deliberate: ...") and stays skipped.

## The compatibility number

At the end of a run the suite prints (go test shows it with `-v`, or when the
package fails)

    PostgREST 16.4 spec cases: N run, M skipped (x% run)

`go test -v ./test/postgrest` also lists every skipped case with its reason
(`SKIP ...` lines). Implementing a behaviour removes its skips and moves the
number. Run with `-run`, the number covers only the selected tests.

## Moving the target

To move to a newer PostgREST: compare its `test/spec` with the tag above, port
the new and changed cases (live or skipped), refresh `fixtures/` from its
`test/spec/fixtures`, and update `PostgRESTVersion`, this file and the README
of the project.
