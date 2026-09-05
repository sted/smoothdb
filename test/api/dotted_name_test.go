package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// A dotted source segment (doc.derived.member_choices) used to be quoted part by
// part and sent to Postgres, which answered 42601 "improper qualified name" and
// the client got a 500 — one prepare, one query and one rollback per request.
// The parser must refuse it with a 400 and no statement: the error comes from
// the "network" subsystem, never from "database".
func TestDottedSourceName(t *testing.T) {

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	const body = `{"subsystem":"network","message":"invalid source name \"doc.derived.member_choices\": a table or function name cannot contain a dot; select the schema with the Accept-Profile or Content-Profile header","code":"","hint":"","details":null,"position":0}`

	tests := []test.Test{
		{
			Description: "GET a dotted source name",
			Query:       "/doc.derived.member_choices",
			Expected:    body,
			Status:      400,
		},
		{
			Description: "GET a dotted source name with filters",
			Query:       "/doc.derived.member_choices?select=id&id=eq.1",
			Expected:    body,
			Status:      400,
		},
		{
			Description: "POST to a dotted source name",
			Method:      "POST",
			Query:       "/doc.derived.member_choices",
			Body:        `{"id": 1}`,
			Expected:    body,
			Status:      400,
		},
		{
			Description: "PATCH a dotted source name",
			Method:      "PATCH",
			Query:       "/doc.derived.member_choices?id=eq.1",
			Body:        `{"id": 1}`,
			Expected:    body,
			Status:      400,
		},
		{
			Description: "DELETE a dotted source name",
			Method:      "DELETE",
			Query:       "/doc.derived.member_choices?id=eq.1",
			Expected:    body,
			Status:      400,
		},
		{
			Description: "GET a dotted function name",
			Query:       "/rpc/doc.derived.member_choices",
			Expected:    body,
			Status:      400,
		},
		{
			Description: "POST to a dotted function name",
			Method:      "POST",
			Query:       "/rpc/doc.derived.member_choices",
			Body:        `{}`,
			Expected:    body,
			Status:      400,
		},
		{
			Description: "a missing but well-formed source is still a 404 from the database",
			Query:       "/member_choices_does_not_exist",
			Status:      404,
		},
	}

	test.Execute(t, testConfig, tests)
}
