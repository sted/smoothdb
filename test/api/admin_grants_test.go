package test_api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sted/smoothdb/test"
)

// POST and DELETE under /grants/:db/tables/:name, a column grant, and GET of
// the grants of a schema.
func TestAdminGrantsRoutes(t *testing.T) {
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS admgr_s CASCADE;
		CREATE SCHEMA admgr_s;
		CREATE TABLE admgr_s.t (a int, b int);`)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/grants/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	tests := []test.Test{
		{
			Description: "grant select on a table under tables/",
			Method:      "POST",
			Query:       "/tables/admgr_s.t",
			Body:        `{"types": ["select"], "grantee": "user1"}`,
			Status:      201,
		},
		{
			Description: "grant update on one column",
			Method:      "POST",
			Query:       "/tables/admgr_s.t",
			Body:        `{"types": ["update"], "columns": ["b"], "grantee": "user1"}`,
			Status:      201,
		},
		{
			Description: "delete with columns is a 400",
			Method:      "POST",
			Query:       "/tables/admgr_s.t",
			Body:        `{"types": ["delete"], "columns": ["b"], "grantee": "user1"}`,
			Status:      400,
		},
		{
			Description: "revoke under tables/",
			Method:      "DELETE",
			Query:       "/tables/admgr_s.t",
			Body:        `{"types": ["select"], "grantee": "user1"}`,
			Status:      200,
		},
		{
			Description: "grant usage on the schema",
			Method:      "POST",
			Query:       "/schema/admgr_s",
			Body:        `{"types": ["usage"], "grantee": "user1"}`,
			Status:      201,
		},
	}
	test.Execute(t, testConfig, tests[:3])
	// the column grant is not a table grant: user1 has SELECT on the table only
	if got := user1Grants(t, testConfig, "/tables/admgr_s.t"); got != "SELECT" {
		t.Errorf("table grants of user1: %q, want SELECT", got)
	}
	// the column grant is listed too, with its column
	if got := user1ColumnGrants(t, testConfig, "/tables/admgr_s.t"); got != "UPDATE(b)" {
		t.Errorf("column grants of user1: %q, want UPDATE(b)", got)
	}
	test.Execute(t, testConfig, tests[3:])
	if got := user1Grants(t, testConfig, "/schema/admgr_s"); got != "USAGE" {
		t.Errorf("schema grants of user1: %q, want USAGE", got)
	}
	execSQLAndReload(t, `DROP SCHEMA admgr_s CASCADE`)
}

// user1Grants returns the privilege types of user1 that GET query lists for
// the whole object (not for some columns).
func user1Grants(t *testing.T, config test.Config, query string) string {
	t.Helper()
	var types []string
	for _, p := range user1Privileges(t, config, query) {
		if len(p.Columns) == 0 {
			types = append(types, p.Types...)
		}
	}
	return strings.Join(types, ",")
}

// user1ColumnGrants returns the column privileges of user1, as TYPE(col,...).
func user1ColumnGrants(t *testing.T, config test.Config, query string) string {
	t.Helper()
	var out []string
	for _, p := range user1Privileges(t, config, query) {
		if len(p.Columns) > 0 {
			out = append(out, strings.Join(p.Types, ",")+"("+strings.Join(p.Columns, ",")+")")
		}
	}
	return strings.Join(out, " ")
}

type listedPrivilege struct {
	Grantee string   `json:"grantee"`
	Types   []string `json:"types"`
	Columns []string `json:"columns"`
}

// user1Privileges returns the privileges of user1 that GET query lists.
func user1Privileges(t *testing.T, config test.Config, query string) []listedPrivilege {
	t.Helper()
	body, _, status, err := test.Exec(test.InitClient(), config, &test.Command{Method: "GET", Query: query})
	if err != nil || status != 200 {
		t.Fatalf("GET %s: %d %v", query, status, err)
	}
	var privileges []listedPrivilege
	if err := json.Unmarshal(body, &privileges); err != nil {
		t.Fatalf("GET %s: %v: %s", query, err, body)
	}
	var out []listedPrivilege
	for _, p := range privileges {
		if p.Grantee == "user1" {
			out = append(out, p)
		}
	}
	return out
}
