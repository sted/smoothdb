package postgrest

import (
	"encoding/json"
	"testing"

	"github.com/sted/smoothdb/test"
)

// liveCheck is a request that no spec case covers. Its answer comes from
// PostgREST 16.4, run live on a copy of pgrest. The checks do not count in
// the compatibility number, which counts ported spec cases only.
type liveCheck struct {
	desc    string
	method  string
	query   string
	body    string
	headers test.Headers
	status  int
	code    string // the error code of the body, when set
	notCode string // an error code the body must not have, when set
	profile string // the Content-Profile header; "" asserts that it is absent
}

func runLiveChecks(t *testing.T, cfg test.Config, checks []liveCheck) {
	t.Helper()
	client := test.InitClient()
	for _, c := range checks {
		body, headers, status, err := test.Exec(client, cfg, &test.Command{
			Description: c.desc, Method: c.method, Query: c.query, Body: c.body, Headers: c.headers})
		if err != nil {
			t.Errorf("%s: %v", c.desc, err)
			continue
		}
		if status != c.status {
			t.Errorf("%s: status %d, want %d (body %s)", c.desc, status, c.status, body)
			continue
		}
		if c.code != "" || c.notCode != "" {
			var e struct{ Code string }
			_ = json.Unmarshal(body, &e)
			if c.code != "" && e.Code != c.code {
				t.Errorf("%s: code %q, want %q (body %s)", c.desc, e.Code, c.code, body)
			}
			if c.notCode != "" && e.Code == c.notCode {
				t.Errorf("%s: code %q, want another (body %s)", c.desc, e.Code, body)
			}
		}
		if got := headers.Get("Content-Profile"); got != c.profile {
			t.Errorf("%s: Content-Profile %q, want %q", c.desc, got, c.profile)
		}
	}
}

// TestLive_AggregatesDisabled: with aggregates off, PostgREST refuses an
// aggregate after it parses the whole query string (Plan.hs, readPlan), and
// before it looks at the payload.
func TestLive_AggregatesDisabled(t *testing.T) {
	json := test.Headers{"Prefer": {"return=representation"}, "Content-Type": {"application/json"}}
	checks := []liveCheck{
		{desc: "an insert of no rows refuses an aggregate", method: "POST",
			query: "/items?select=id.sum()", body: `[]`, headers: json, status: 400, code: "PGRST123"},
		{desc: "an update with an empty body refuses an aggregate", method: "PATCH",
			query: "/items?select=id.sum()&id=eq.-99", body: `{}`, headers: json, status: 400, code: "PGRST123"},
		{desc: "a select parse error comes before the aggregate check",
			query: "/items?select=count(),", status: 400, notCode: "PGRST123"},
		{desc: "an order parse error comes before the aggregate check",
			query: "/items?select=id.sum()&order=foo.bar.baz", status: 400, notCode: "PGRST123"},
	}
	withVariant(t, map[string]any{"Database.AggregatesEnabled": false}, false, func(cfg test.Config) {
		runLiveChecks(t, cfg, checks)
	})
}

// TestLive_ContentProfile: PostgREST names the negotiated schema on every
// response with a content type (Response.hs, contentTypeHeaders), and not on
// a write without a representation.
func TestLive_ContentProfile(t *testing.T) {
	checks := []liveCheck{
		{desc: "an insert of no rows with a representation", method: "POST", query: "/items", body: `[]`,
			headers: test.Headers{"Prefer": {"return=representation"}}, status: 201, profile: "test"},
		{desc: "an update of no rows with a representation", method: "PATCH", query: "/items?id=eq.-99", body: `{}`,
			headers: test.Headers{"Prefer": {"return=representation"}}, status: 200, profile: "test"},
		{desc: "an offset past the counted total", query: "/items?offset=999999",
			headers: test.Headers{"Prefer": {"count=exact"}}, status: 416, code: "PGRST103", profile: "test"},
		{desc: "the root", query: "/", status: 200, profile: "test"},
		{desc: "an insert without a representation", method: "POST", query: "/items", body: `{"id":-5}`,
			status: 201, profile: ""},
	}
	runLiveChecks(t, testConfig, checks)
}
