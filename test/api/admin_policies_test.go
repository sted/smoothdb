package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// The policy routes take the schema from the profile header, whitelist the
// command and accept PATCH.
func TestAdminPoliciesRoutes(t *testing.T) {
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS admpol_s CASCADE;
		CREATE SCHEMA admpol_s;
		CREATE TABLE admpol_s.t (id int);`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/admin/databases/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Accept-Profile": {"admpol_s"}, "Content-Profile": {"admpol_s"}},
	}
	tests := []test.Test{
		{
			Description: "create a policy with no command",
			Method:      "POST",
			Query:       "/tables/t/policies",
			Body:        `{"name": "p1", "using": "id > 0"}`,
			Status:      201,
		},
		{
			Description: "a bad command is a 400",
			Method:      "POST",
			Query:       "/tables/t/policies",
			Body:        `{"name": "p2", "command": "SELECT USING (true) --", "using": "id > 0"}`,
			Status:      400,
		},
		{
			Description: "alter and rename the policy",
			Method:      "PATCH",
			Query:       "/tables/t/policies/p1",
			Body:        `{"name": "p1b", "roles": ["public"], "using": "id > 1", "check": "id < 9"}`,
			Status:      200,
		},
		{
			Description: "list the policies of the header schema",
			Method:      "GET",
			Query:       "/tables/t/policies",
			Expected:    `[{"name":"p1b","table":"admpol_s.t","deny":false,"command":"ALL","roles":["public"],"using":"(id > 1)","check":"(id < 9)"}]`,
			Status:      200,
		},
		{
			Description: "drop the policy",
			Method:      "DELETE",
			Query:       "/tables/t/policies/p1b",
			Status:      200,
		},
	}
	test.Execute(t, testConfig, tests)
	execSQLAndReload(t, `DROP SCHEMA admpol_s CASCADE`)
}
