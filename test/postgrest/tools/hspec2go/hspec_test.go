package main

import (
	"bytes"
	"strings"
	"testing"
)

const sample = `spec :: SpecWithConfig
spec = describe "things" $ do
  let single = ("Accept", "application/vnd.pgrst.object+json")
  context "reading" $ do
    it "reads twice" $ do
      get "/items?id=eq.1"
        ` + "`shouldRespondWith`" + ` [json|[{"id":1}]|]
          { matchHeaders = [matchContentTypeJson] }
      request methodPatch "/items?id=eq.1" [single, ("Prefer", "return=representation")] [json| { id: 2 } |]
        ` + "`shouldRespondWith`" + ` ""
          { matchStatus = 204
          , matchHeaders = [matchHeaderAbsent hContentType, "Content-Range" <:> "0-0/*"]
          }
    it "has haskell code" $ do
      r <- request methodGet "/items" (rangeHdrs $ ByteRangeFromTo 0 1) ""
      liftIO $ simpleStatus r ` + "`shouldBe`" + ` ok200
    it "gives a status" $
      request methodDelete "/items" (rangeHdrsWithCount (ByteRangeFrom 2)) mempty ` + "`shouldRespondWith`" + ` 404
`

func TestParseSpec(t *testing.T) {
	f := ParseSpec("Sample.hs", sample)
	if len(f.Its) != 3 {
		t.Fatalf("want 3 it blocks, got %d", len(f.Its))
	}
	if len(f.Cases) != 3 {
		t.Fatalf("want 3 cases, got %d: %+v", len(f.Cases), f.Cases)
	}
	c := f.Cases[0]
	if c.Method != "GET" || c.Query != "/items?id=eq.1" || c.Expected != `[{"id":1}]` || c.Status != 200 ||
		len(c.ExpHeaders) != 1 || c.ExpHeaders[0][1] != "application/json; charset=utf-8" {
		t.Errorf("case 0: %+v", c)
	}
	c = f.Cases[1]
	if c.TODO != "" || c.Method != "PATCH" || c.Body != `{ "id": 2 }` || !c.ExpEmpty || c.Status != 204 ||
		len(c.Headers) != 2 || c.Headers[0] != [2]string{"Accept", "application/vnd.pgrst.object+json"} ||
		len(c.ExpHeaders) != 2 || c.ExpHeaders[0] != [2]string{"Content-Type", ""} {
		t.Errorf("case 1: %+v", c)
	}
	c = f.Cases[2]
	if c.Method != "DELETE" || c.Status != 404 || len(c.Headers) != 2 || c.Headers[1] != [2]string{"Range", "2-"} {
		t.Errorf("case 2: %+v", c)
	}
	// the it with Haskell code is a leftover, not dropped
	if len(f.Leftovers) != 1 || f.Leftovers[0].It.Title != "has haskell code" {
		t.Errorf("leftovers: %+v", f.Leftovers)
	}
}

func TestEmit(t *testing.T) {
	var b bytes.Buffer
	Emit(&b, ParseSpec("Sample.hs", sample), "\t\t")
	out := b.String()
	for _, want := range []string{
		`// spec = describe "things" $ do`,
		`//   context "reading" $ do`,
		`//     it "reads twice" $ do`,
		`Method: "PATCH",`,
		`ExpectedEmpty: true,`,
		`// TODO(hspec2go): not a request with shouldRespondWith`,
		`//       liftIO $ simpleStatus r`,
		`Status: 404,`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}
