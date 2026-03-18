package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Tests for issue #17: Add and read object comments using DDL admin API

func TestComments(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "comments_test",
				"comment": "This is a test table",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "name", "type": "text", "comment": "The name field"}
				],
				"ifnotexists": true
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tests := []test.Test{
		// read table and verify table and column comments are returned
		{
			Description: "table and column comments are returned on GET",
			Query:       "/dbtest/tables/comments_test",
			Expected:    `{"name":"comments_test","schema":"public","owner":"admin","comment":"This is a test table","rowsecurity":false,"columns":[{"name":"id","type":"int4","notnull":true,"default":null,"comment":null,"constraints":null,"table":"comments_test","schema":"public"},{"name":"name","type":"text","notnull":false,"default":null,"comment":"The name field","constraints":null,"table":"comments_test","schema":"public"}],"constraints":null,"hasindexes":false,"hastriggers":false,"ispartition":false}`,
			Status:      200,
		},
		// update table comment via PATCH
		{
			Description: "can update table comment",
			Method:      "PATCH",
			Query:       "/dbtest/tables/comments_test",
			Body:        `{"comment": "Updated table comment"}`,
			Status:      201,
		},
		// update column comment via PATCH
		{
			Description: "can update column comment",
			Method:      "PATCH",
			Query:       "/dbtest/tables/comments_test/columns/id",
			Body:        `{"comment": "The primary key"}`,
			Status:      200,
		},
		// verify updated comments
		{
			Description: "updated comments are returned on GET",
			Query:       "/dbtest/tables/comments_test",
			Expected:    `{"name":"comments_test","schema":"public","owner":"admin","comment":"Updated table comment","rowsecurity":false,"columns":[{"name":"id","type":"int4","notnull":true,"default":null,"comment":"The primary key","constraints":null,"table":"comments_test","schema":"public"},{"name":"name","type":"text","notnull":false,"default":null,"comment":"The name field","constraints":null,"table":"comments_test","schema":"public"}],"constraints":null,"hasindexes":false,"hastriggers":false,"ispartition":false}`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
