package postgrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sted/smoothdb/test"
)

// The plan of a request (Accept: application/vnd.pgrst.plan), from
// PostgREST's test/spec/Feature/Query/PlanSpec.hs. PostgREST explains the
// whole statement it runs: a CTE that aggregates the rows into JSON, with
// the count in the same statement. smoothdb serializes the rows in Go and
// counts in a statement of its own, so its plan has no json_agg node and no
// count. The cases that assert that shape are skipped as deliberate; the
// contract (media type, options, formats, status, headers, errors) is the
// same.

const (
	planJSONForJSON = `application/vnd.pgrst.plan+json; for="application/json"; charset=utf-8`
	planTextForJSON = `application/vnd.pgrst.plan+text; for="application/json"; charset=utf-8`
)

// planBody decodes a plan in the JSON format: an array with one object.
func planBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var plans []map[string]any
	if err := json.Unmarshal(body, &plans); err != nil || len(plans) != 1 {
		t.Fatalf("not a JSON plan: %v\n%s", err, body)
	}
	return plans[0]
}

// planCost is the total cost of the root node, as PostgREST's planCost.
func planCost(t *testing.T, body []byte) float64 {
	t.Helper()
	plan, _ := planBody(t, body)["Plan"].(map[string]any)
	cost, ok := plan["Total Cost"].(float64)
	if !ok {
		t.Fatalf("no Total Cost in the plan:\n%s", body)
	}
	return cost
}

func costAbove(min float64) func(*testing.T, []byte, http.Header) {
	return func(t *testing.T, body []byte, _ http.Header) {
		if c := planCost(t, body); c <= min {
			t.Errorf("total cost %v, want more than %v", c, min)
		}
	}
}

func costBelow(max float64) func(*testing.T, []byte, http.Header) {
	return func(t *testing.T, body []byte, _ http.Header) {
		if c := planCost(t, body); c >= max {
			t.Errorf("total cost %v, want less than %v", c, max)
		}
	}
}

// writtenPlan is the check of a write's plan: a cost and a Content-Length
// that is not zero (PostgREST's notZeroContentLength).
func writtenPlan(t *testing.T, body []byte, h http.Header) {
	if c := planCost(t, body); c <= 0 {
		t.Errorf("total cost %v, want more than 0", c)
	}
	if cl := h.Get("Content-Length"); cl == "" || cl == "0" {
		t.Errorf("Content-Length %q, want a length", cl)
	}
}

func bodyHas(s string) func(*testing.T, []byte, http.Header) {
	return func(t *testing.T, body []byte, _ http.Header) {
		if !bytes.Contains(body, []byte(s)) {
			t.Errorf("the body has no %q:\n%s", s, body)
		}
	}
}

func bodyHasNot(s string) func(*testing.T, []byte, http.Header) {
	return func(t *testing.T, body []byte, _ http.Header) {
		if bytes.Contains(body, []byte(s)) {
			t.Errorf("the body has %q:\n%s", s, body)
		}
	}
}

// textPlan is the check of a plan in the text format: the lines of EXPLAIN,
// each one ending with a newline, as PostgREST's planRow (BS.unlines).
func textPlan(t *testing.T, body []byte, _ http.Header) {
	if !bytes.Contains(body, []byte("(cost=")) || !bytes.HasSuffix(body, []byte("\n")) {
		t.Errorf("not a text plan:\n%s", body)
	}
}

// tiobeRows is PostgREST's getInsertDataForTiobePlsTable: the languages
// "Lang 20" to "Lang <rows+20>".
func tiobeRows(rows int) string {
	var b strings.Builder
	b.WriteString("[")
	for i := 20; i <= rows+20; i++ {
		if i > 20 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"name":"Lang %d","rank":%d}`, i, i)
	}
	b.WriteString("]")
	return b.String()
}

func accept(v string) test.Headers { return test.Headers{"Accept": {v}} }

func TestPlan(t *testing.T) {
	tests := []test.Test{
		// describe "read table/view plan"
		// it "outputs the total cost for a single filter on a table"
		{
			Description:     "the total cost for a single filter on a table",
			Query:           "/projects?id=in.(1,2,3)",
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for a single filter on a view"
		{
			Description:     "the total cost for a single filter on a view",
			Query:           "/projects_view?id=gt.2",
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           costAbove(0),
		},
		// it "outputs blocks info when using the buffers option"
		{
			Description:     "blocks info with the buffers option",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+json; options=buffers"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=buffers; charset=utf-8`},
			Check:           bodyHas("Shared Hit Blocks"),
		},
		// it "outputs the search path when using the settings option": smoothdb
		// sets Database.SchemaSearchPath as written, PostgREST quotes each
		// schema of db-schemas; the path is the same
		{
			Description:     "the search path with the settings option",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+json; options=settings"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=settings; charset=utf-8`},
			Check: func(t *testing.T, body []byte, _ http.Header) {
				settings, _ := planBody(t, body)["Settings"].(map[string]any)
				if settings["search_path"] != "test" {
					t.Errorf(`Settings %v, want {"search_path": "test"}`, settings)
				}
			},
		},
		// it "outputs WAL info when using the wal option"
		{
			Description:     "WAL info with the wal option",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+json; options=analyze|wal"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=analyze|wal; charset=utf-8`},
			Check: func(t *testing.T, body []byte, _ http.Header) {
				plan, _ := planBody(t, body)["Plan"].(map[string]any)
				if plan["WAL Records"] != float64(0) {
					t.Errorf("WAL Records %v, want 0:\n%s", plan["WAL Records"], body)
				}
			},
		},
		// it "outputs columns info when using the verbose option"
		// PostgREST reads the Output of the scan under its aggregate
		// (projects.id, ...); in smoothdb the scan is the root, and a query
		// of one relation names its columns without it.
		{
			Description:     "columns info with the verbose option",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+json; options=verbose"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=verbose; charset=utf-8`},
			Check:           bodyHas(`"Output": ["id", "name", "client_id"]`),
		},
		// it "includes the Query Identifier field when using the verbose option"
		{
			Description:     "the Query Identifier with the verbose option",
			Skip:            "needs compute_query_id on, a setting of the server: PostgreSQL computes no Query Identifier with its default (auto) unless a module such as pg_stat_statements asks for it",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+json; options=verbose"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=verbose; charset=utf-8`},
			Check: func(t *testing.T, body []byte, _ http.Header) {
				if _, ok := planBody(t, body)["Query Identifier"]; !ok {
					t.Errorf("no Query Identifier:\n%s", body)
				}
			},
		},
		// it "outputs the plan for application/json "
		{
			Description:     "the plan for application/json",
			Query:           "/projects",
			Headers:         accept(`application/vnd.pgrst.plan+json; for="application/json"; options=verbose`),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=verbose; charset=utf-8`},
		},
		{
			Description: "the plan for application/json has the json_agg of PostgREST",
			Skip:        "deliberate: smoothdb serializes the rows in Go, its plan has no json_agg node",
		},
		// it "outputs the plan for application/vnd.pgrst.object "
		{
			Description:     "the plan for application/vnd.pgrst.object",
			Query:           "/projects_view",
			Headers:         accept(`application/vnd.pgrst.plan+json; for="application/vnd.pgrst.object"; options=verbose`),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/vnd.pgrst.object+json"; options=verbose; charset=utf-8`},
		},
		{
			Description: "the plan for application/vnd.pgrst.object has the json_agg -> 0 of PostgREST",
			Skip:        "deliberate: smoothdb serializes the rows in Go, its plan has no json_agg node",
		},

		// describe "writes plans"
		// it "outputs the total cost for an insert"
		{
			Description:     "the total cost for an insert",
			Method:          "POST",
			Query:           "/projects",
			Body:            `{"id":100, "name": "Project 100"}`,
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for an update"
		{
			Description:     "the total cost for an update",
			Method:          "PATCH",
			Query:           "/projects?id=eq.3",
			Body:            `{"name": "Patched Project"}`,
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for a delete"
		{
			Description:     "the total cost for a delete",
			Method:          "DELETE",
			Query:           "/projects?id=in.(1,2,3)",
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for a single upsert"
		{
			Description:     "the total cost for a single upsert",
			Method:          "PUT",
			Query:           "/tiobe_pls?name=eq.Go",
			Body:            `[ { "name": "Go", "rank": 19 } ]`,
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for 2 upserts"
		{
			Description:     "the total cost for 2 upserts",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Body:            `[ { "name": "Python", "rank": 19 }, { "name": "Go", "rank": 20} ]`,
			Headers:         test.Headers{"Prefer": {"resolution=merge-duplicates"}, "Accept": {"application/vnd.pgrst.plan+json"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},
		// it "outputs the total cost for an upsert with 10 rows"
		{
			Description:     "the total cost for an upsert with 10 rows",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Body:            tiobeRows(10),
			Headers:         test.Headers{"Prefer": {"resolution=merge-duplicates"}, "Accept": {"application/vnd.pgrst.plan+json"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           costAbove(0),
		},
		// it "outputs the total cost for an upsert with 100 rows"
		{
			Description:     "the total cost for an upsert with 100 rows",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Body:            tiobeRows(100),
			Headers:         test.Headers{"Prefer": {"resolution=merge-duplicates"}, "Accept": {"application/vnd.pgrst.plan+json"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           costAbove(0),
		},
		// it "outputs the total cost for an upsert with 1000 rows"
		{
			Description:     "the total cost for an upsert with 1000 rows",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Body:            tiobeRows(1000),
			Headers:         test.Headers{"Prefer": {"resolution=merge-duplicates"}, "Accept": {"application/vnd.pgrst.plan+json"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           costAbove(0),
		},
		// it "outputs the plan for application/vnd.pgrst.object"
		{
			Description:     "the plan of a delete for application/vnd.pgrst.object",
			Method:          "DELETE",
			Query:           "/projects?id=eq.6",
			Headers:         test.Headers{"Prefer": {"return=representation"}, "Accept": {`application/vnd.pgrst.plan+json; for="application/vnd.pgrst.object"; options=verbose`}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/vnd.pgrst.object+json"; options=verbose; charset=utf-8`},
			Check:           writtenPlan,
		},
		{
			Description: "the plan of a delete for application/vnd.pgrst.object has the json_agg -> 0 of PostgREST",
			Skip:        "deliberate: smoothdb serializes the rows in Go, its plan has no json_agg node",
		},

		// describe "function plan"
		// it "outputs the total cost for a function call"
		{
			Description:     "the total cost for a function call",
			Query:           "/rpc/getallprojects?id=in.(1,2,3)",
			Headers:         accept("application/vnd.pgrst.plan+json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           writtenPlan,
		},

		// describe "text format"
		// it "outputs the total cost for a function call": its body starts
		// with the Aggregate of PostgREST's statement; smoothdb's starts
		// with the scan.
		{
			Description:     "the text format",
			Query:           "/projects?id=in.(1,2,3)",
			Headers:         accept("application/vnd.pgrst.plan+text"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON},
			Check:           textPlan,
		},
		// it "outputs in text format by default"
		{
			Description:     "the text format by default",
			Query:           "/projects?id=in.(1,2,3)",
			Headers:         accept("application/vnd.pgrst.plan"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON},
			Check:           textPlan,
		},

		// describe "resource embedding costs"
		// it "a one to many doesn't surpass a threshold"
		{
			Description: "the cost of a one to many embed",
			Query:       "/clients?select=*,projects(*)&id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(33.3),
		},
		// it "a many to one doesn't surpass a threshold"
		{
			Description: "the cost of a many to one embed",
			Query:       "/projects?select=*,clients(*)&id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(16.5),
		},
		// it "a many to many doesn't surpass a threshold"
		{
			Description: "the cost of a many to many embed",
			Query:       "/users?select=*,tasks(*)&id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(70.9),
		},
		// context "!inner vs embed not null"
		// it "on an o2m, an !inner has a similar cost to not.null"
		{
			Description: "the cost of an o2m !inner",
			Query:       "/clients?select=*,projects!inner(*)&id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(33.3),
		},
		{
			Description: "the cost of an o2m not.is.null",
			Query:       "/clients?select=*,projects(*)&projects=not.is.null&id=eq.1",
			Skip:        "PostgREST 16.4: a null filter on an embedded resource (rel=is.null); card 32477 (embed filters)",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(33.3),
		},
		// it "on an m2o, an !inner has a similar cost to not.null"
		{
			Description: "the cost of an m2o !inner",
			Query:       "/projects?select=*,clients!inner(*)&id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(16.42),
		},
		{
			Description: "the cost of an m2o not.is.null",
			Query:       "/projects?select=*,clients(*)&clients=not.is.null&id=eq.1",
			Skip:        "PostgREST 16.4: a null filter on an embedded resource (rel=is.null); card 32477 (embed filters)",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(16.42),
		},
		// it "on an m2m, an !inner has a similar cost to not.null"
		{
			Description: "the cost of an m2m !inner",
			Query:       "/users?select=*,tasks!inner(*)&tasks.id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(20888.83),
		},
		{
			Description: "the cost of an m2m not.is.null",
			Query:       "/users?select=*,tasks(*)&tasks.id=eq.1&tasks=not.is.null",
			Skip:        "PostgREST 16.4: a null filter on an embedded resource (rel=is.null); card 32477 (embed filters)",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(20888.83),
		},

		// describe "function call costs"
		// it "should not exceed cost when calling setof composite proc"
		{
			Description: "the cost of a setof composite function",
			Query:       "/rpc/get_projects_below?id=3",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(35.4),
		},
		// it "should not exceed cost when calling setof composite proc with empty params"
		{
			Description: "the cost of a setof composite function without arguments",
			Query:       "/rpc/getallprojects",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(71.0),
		},
		// it "should not exceed cost when calling scalar proc"
		{
			Description: "the cost of a scalar function",
			Query:       "/rpc/add_them?a=3&b=4",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      200,
			Check:       costBelow(0.08),
		},
		// context "function inlining"
		// it "should inline a zero argument function"
		{
			Description: "a function without arguments is inlined",
			Query:       "/rpc/getallusers?id=eq.1",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHasNot("getallusers"),
		},
		// it "should inline a function with arguments"
		{
			Description: "a function with arguments is inlined",
			Query:       "/rpc/getitemrange?min=10&max=15",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHasNot("getitemrange"),
		},
		// context "index usage"
		// it "should use an index for a json arrow operator filter"
		{
			Description: "an index for a json arrow filter",
			Query:       "/bets?data_json->>contractId=eq.1",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHas("Index Cond"),
		},
		// it "should use an index for a jsonb arrow operator filter"
		{
			Description: "an index for a jsonb arrow filter",
			Query:       "/bets?data_jsonb->>contractId=eq.1",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHas("Index"),
		},
		// it "should use an index for ordering on a json arrow operator"
		{
			Description: "an index for an order on a json arrow",
			Query:       "/bets?order=data_json->>contractId",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHas("Index"),
		},
		// it "should use an index for ordering on a jsonb arrow operator"
		{
			Description: "an index for an order on a jsonb arrow",
			Query:       "/bets?order=data_jsonb->>contractId",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      200,
			Check:       bodyHas("Index"),
		},

		// describe "custom media types"
		// it "outputs the plan for a scalar function text/xml"
		{
			Description: "the plan of a scalar function for text/xml",
			Skip:        "PostgREST 16.4: a function that returns xml serves text/xml; smoothdb serves no text/xml",
		},
		// it "outputs the plan for an aggregate application/vnd.twkb"
		{
			Description: "the plan of an aggregate for application/vnd.twkb",
			Skip:        "PostgREST 16.4: media type handlers from aggregates; smoothdb has none",
		},

		// describe "plan of upsert with DEFAULT surrogate primary key"
		// it "test case sensitive sequence is properly quoted in nextval()"
		{
			Description:     "the sequence of a DEFAULT key is quoted in the plan of an upsert",
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
			Method:          "POST",
			Query:           "/Surr_Gen_Default_Upsert?columns=id,name&select=name,extra",
			Body:            `[ { "id": 1, "name": "updated value" }, { "name": "new value" } ]`,
			Headers:         test.Headers{"Prefer": {"return=representation, resolution=merge-duplicates, missing=default"}, "Accept": {"application/vnd.pgrst.plan+json; options=verbose"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="application/json"; options=verbose; charset=utf-8`},
			Check:           bodyHas(`nextval('\"Surr_Gen_Default_Upsert_id_seq\"'::regclass)`),
		},

		// describe "count preference plan costs"
		{
			Description: "the plan of a table read with count=exact has one count aggregate",
			Skip:        "deliberate: smoothdb counts in a statement of its own, the plan is the one of the rows",
		},
		{
			Description: "the plan of a table read with count=exact and a limit has the count aggregates",
			Skip:        "deliberate: smoothdb counts in a statement of its own, the plan is the one of the rows",
		},
		{
			Description: "the plan of a function call with count=exact has one count aggregate",
			Skip:        "deliberate: smoothdb counts in a statement of its own, the plan is the one of the rows",
		},
		{
			Description: "the plan of a function call with count=exact and a limit has the count aggregates",
			Skip:        "deliberate: smoothdb counts in a statement of its own, the plan is the one of the rows",
		},

		// Not in PlanSpec: the answers of PostgREST 16.4 run live
		// (db-plan-enabled = true), on the same requests.
		{
			Description:     "the options are named in a fixed order",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan; options=wal|buffers|analyze|verbose|settings"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+text; for="application/json"; options=analyze|verbose|settings|buffers|wal; charset=utf-8`},
			Check:           bodyHas("Execution Time"),
		},
		{
			Description:     "an unknown option is ignored",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan+text; options=bogus|verbose"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+text; for="application/json"; options=verbose; charset=utf-8`},
			Check:           bodyHas("Output:"),
		},
		{
			Description:     "for=*/* stays as it is",
			Query:           "/projects",
			Headers:         accept(`application/vnd.pgrst.plan; for="*/*"`),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+text; for="*/*"; charset=utf-8`},
			Check:           textPlan,
		},
		{
			Description:     "for=text/csv",
			Query:           "/projects",
			Headers:         accept(`application/vnd.pgrst.plan+json; for="text/csv"`),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": `application/vnd.pgrst.plan+json; for="text/csv"; charset=utf-8`},
			Check:           costAbove(0),
		},
		{
			Description:     "the media type is case insensitive",
			Query:           "/projects",
			Headers:         accept("APPLICATION/VND.PGRST.PLAN+JSON"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planJSONForJSON},
			Check:           costAbove(0),
		},
		{
			Description:     "the first media type that can be served wins",
			Query:           "/projects",
			Headers:         accept("image/png, application/vnd.pgrst.plan"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON},
			Check:           textPlan,
		},
		{
			Description: "a plan for an unknown media type is not acceptable",
			Query:       "/projects",
			Headers:     accept(`application/vnd.pgrst.plan+json; for="image/png"`),
			Status:      406,
			Expected:    `{"code":"PGRST107","details":null,"hint":null,"message":"None of these media types are available: application/vnd.pgrst.plan+json; for=\"image/png\""}`,
		},
		{
			Description: "a plan in an unknown format is not acceptable",
			Query:       "/projects",
			Headers:     accept("application/vnd.pgrst.plan+xml"),
			Status:      406,
			Expected:    `{"code":"PGRST107","details":null,"hint":null,"message":"None of these media types are available: application/vnd.pgrst.plan+xml"}`,
		},
		{
			Description:     "HEAD answers the headers of the plan, without a body",
			Method:          "HEAD",
			Query:           "/projects",
			Headers:         accept("application/vnd.pgrst.plan"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON},
			ExpectedEmpty:   true,
		},
		{
			Description:     "a plan with a count and a range has no Content-Range",
			Query:           "/projects",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.plan"}, "Prefer": {"count=exact"}, "Range": {"0-1"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON, "Content-Range": "", "Content-Location": ""},
			Check:           textPlan,
		},
		{
			Description: "a missing table is 404",
			Query:       "/nonexistent",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      404,
			Expected:    `{"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.nonexistent' in the schema cache"}`,
		},
		{
			Description: "a value of the wrong type is 400",
			Query:       "/projects?id=eq.abc",
			Headers:     accept("application/vnd.pgrst.plan; options=analyze"),
			Status:      400,
			Expected:    `{"code":"22P02","details":null,"hint":null,"message":"invalid input syntax for type integer: \"abc\""}`,
		},
		{
			Description:     "a PUT whose body has not the key of the URL answers its plan",
			Method:          "PUT",
			Query:           "/projects?id=eq.103",
			Body:            `{"id":104,"name":"P"}`,
			Headers:         accept("application/vnd.pgrst.plan"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON},
			Check:           textPlan,
		},
		{
			Description:     "a write with return=headers-only answers 200 and no Location",
			Method:          "POST",
			Query:           "/projects",
			Body:            `{"id":105,"name":"P"}`,
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.plan"}, "Prefer": {"return=headers-only"}},
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": planTextForJSON, "Location": ""},
			Check:           textPlan,
		},
		// analyze runs the write: it commits with tx=commit, and the
		// suite's transaction mode (rollback-allow-override) rolls it back
		// otherwise. Without analyze nothing runs.
		{
			Description: "a plan with analyze and tx=commit writes",
			Method:      "POST",
			Query:       "/projects",
			Body:        `{"id":200,"name":"P200"}`,
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.plan; options=analyze"}, "Prefer": {"tx=commit"}},
			Status:      200,
			Check:       bodyHas("actual time="),
		},
		{
			Description: "a plan with analyze and tx=rollback does not write",
			Method:      "POST",
			Query:       "/projects",
			Body:        `{"id":201,"name":"P201"}`,
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.plan; options=analyze"}, "Prefer": {"tx=rollback"}},
			Status:      200,
			Check:       bodyHas("actual time="),
		},
		{
			Description: "a plan without analyze does not write",
			Method:      "POST",
			Query:       "/projects",
			Body:        `{"id":202,"name":"P202"}`,
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.plan"}, "Prefer": {"tx=commit"}},
			Status:      200,
			Check:       textPlan,
		},
		{
			Description: "only the write of analyze and tx=commit is there",
			Query:       "/projects?id=in.(200,201,202)&select=id",
			Status:      200,
			Expected:    `[{"id":200}]`,
		},
	}
	// a copy of the fixtures: a plan with analyze and tx=commit writes
	withVariant(t, map[string]any{"Database.PlanEnabled": true}, true, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}

// it "doesn't work if db-plan-enabled=false(the default)"
func TestPlanDisabled(t *testing.T) {
	tests := []test.Test{
		{
			Description: "no plan of a read when plans are off",
			Query:       "/projects?id=in.(1,2,3)",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      406,
		},
		{
			Description: "no plan of a function call when plans are off",
			Query:       "/rpc/getallprojects?id=in.(1,2,3)",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      406,
		},
		{
			Description: "no plan of a delete when plans are off",
			Method:      "DELETE",
			Query:       "/projects?id=in.(1,2,3)",
			Headers:     accept("application/vnd.pgrst.plan"),
			Status:      406,
		},
		// PostgREST 16.4 run live (db-plan-enabled = false): the error names
		// the plan media type with its parameters
		{
			Description: "the error names the plan media type with its for",
			Query:       "/projects",
			Headers:     accept("application/vnd.pgrst.plan+json"),
			Status:      406,
			Expected:    `{"code":"PGRST107","details":null,"hint":null,"message":"None of these media types are available: application/vnd.pgrst.plan+json; for=\"application/json\""}`,
		},
		{
			Description:     "with plans off the next media type is served",
			Query:           "/projects?id=eq.1&select=id",
			Headers:         accept("application/vnd.pgrst.plan, application/json"),
			Status:          200,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Expected:        `[{"id":1}]`,
		},
	}
	test.Execute(t, testConfig, tests)
}
