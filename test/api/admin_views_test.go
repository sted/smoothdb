package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// DELETE of a materialized view, POST with replace and PATCH of a view.
func TestAdminViewsRoutes(t *testing.T) {
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS admvw_s CASCADE;
		DROP SCHEMA IF EXISTS admvw_s2 CASCADE;
		CREATE SCHEMA admvw_s;
		CREATE SCHEMA admvw_s2;
		CREATE TABLE admvw_s.t (id int, n int);`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/admin/databases/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Accept-Profile": {"admvw_s"}, "Content-Profile": {"admvw_s"}},
	}
	tests := []test.Test{
		{
			Description: "create a view",
			Method:      "POST",
			Query:       "/views",
			Body:        `{"name": "v", "definition": "SELECT id FROM admvw_s.t"}`,
			Status:      201,
		},
		{
			Description: "replace the view",
			Method:      "POST",
			Query:       "/views",
			Body:        `{"name": "v", "definition": "SELECT id, n FROM admvw_s.t", "replace": true}`,
			Status:      201,
		},
		{
			Description: "create a materialized view",
			Method:      "POST",
			Query:       "/views",
			Body:        `{"name": "mv", "definition": "SELECT id FROM admvw_s.t", "materialized": true}`,
			Status:      201,
		},
		{
			Description: "replace is refused for a materialized view",
			Method:      "POST",
			Query:       "/views",
			Body:        `{"name": "mv", "definition": "SELECT id FROM admvw_s.t", "materialized": true, "replace": true}`,
			Status:      400,
		},
		{
			Description: "rename the materialized view and change its owner",
			Method:      "PATCH",
			Query:       "/views/mv",
			Body:        `{"name": "mv2", "owner": "admin"}`,
			Status:      200,
		},
		{
			Description: "drop the materialized view",
			Method:      "DELETE",
			Query:       "/views/mv2",
			Status:      200,
		},
		{
			Description: "move and rename the view",
			Method:      "PATCH",
			Query:       "/views/v",
			Body:        `{"name": "v2", "schema": "admvw_s2"}`,
			Status:      200,
		},
		{
			Description: "the view is in its new schema",
			Method:      "GET",
			Query:       "/views/v2",
			Headers:     test.Headers{"Accept-Profile": {"admvw_s2"}},
			Status:      200,
		},
		{
			Description: "PATCH of a missing view is a 404",
			Method:      "PATCH",
			Query:       "/views/nosuchview",
			Body:        `{"name": "x"}`,
			Status:      404,
		},
	}
	test.Execute(t, testConfig, tests)
	execSQLAndReload(t, `DROP SCHEMA admvw_s CASCADE; DROP SCHEMA admvw_s2 CASCADE;`)
}
