package test_api

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestSessions(t *testing.T) {

	i := 3
	i++

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// drop database db1
		{
			Method: "DELETE",
			Query:  "/db1",
		},
		// create database db1
		{
			Method: "POST",
			Query:  "/",
			Body:   `{"name": "db1"}`,
		},
		// create table records
		{
			Method: "POST",
			Query:  "/db1/tables",
			Body: `{
				"name": "records",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "int4", "type": "int4"}
				]}`,
		},
		// grant select to user1
		{
			Method: "POST",
			Query:  "http://localhost:8082/admin/grants/db1",
			Body: `{
				"types": ["select", "insert"],
				"targettype": "table",
				"targetname": "records",
				"grantee": "user1"
				}`,
		},
		// drop database db2
		{
			Method: "DELETE",
			Query:  "/db2",
		},
		// create database db2
		{
			Method: "POST",
			Query:  "/",
			Body:   `{"name": "db2"}`,
		},
		// create table records
		{
			Method: "POST",
			Query:  "/db2/tables",
			Body: `{
				"name": "records",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "int4", "type": "int4"}
				]}`,
		},
		// grant select to user2
		{
			Method: "POST",
			Query:  "http://localhost:8082/admin/grants/db2",
			Body: `{
				"types": ["select", "insert"],
				"targettype": "table",
				"targetname": "records",
				"grantee": "user2"
				}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	t.Run("flow1", func(t *testing.T) {
		t.Parallel()

		testConfig := test.Config{
			CommonHeaders: test.Headers{"Authorization": {user1Token}},
		}

		tests := []test.Test{
			{
				Method: "GET",
				Query:  "http://localhost:8082/api/db1/records",
				Status: 200,
			},
		}
		for i := 0; i < 10000; i++ {
			test.Execute(t, testConfig, tests)
		}
	})

	t.Run("flow2", func(t *testing.T) {
		t.Parallel()

		testConfig := test.Config{
			CommonHeaders: test.Headers{"Authorization": {user2Token}},
		}

		tests := []test.Test{
			{
				Method: "GET",
				Query:  "http://localhost:8082/api/db2/records",
				Status: 200,
			},
		}
		for i := 0; i < 10000; i++ {
			test.Execute(t, testConfig, tests)
		}
	})

}
