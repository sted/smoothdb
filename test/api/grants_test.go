package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestGrants(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// drop table table_grants
		{
			Method: "DELETE",
			Query:  "/dbtest/tables/table_grants",
		},
		// create table table_grants
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "table_grants",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "creator", "type": "text", "default": "current_user"}
				]}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tests := []test.Test{
		{
			Description: "grant connect, create to user1",
			Method:      "POST",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Body: `{
				"types": ["connect", "create"],
				"grantee": "user1"
				}`,
			Status: 201,
		},
		{
			Description: "verify grants",
			Method:      "GET",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Expected: `[{"targetname":"dbtest","targettype":"database","types":["TEMPORARY","CONNECT"],"grantee":"","grantor":"admin","acl":"=Tc/admin"},
				{"targetname":"dbtest","targettype":"database","types":["CREATE","TEMPORARY","CONNECT"],"grantee":"admin","grantor":"admin","acl":"admin=CTc/admin"},
				{"targetname":"dbtest","targettype":"database","types":["CREATE","CONNECT"],"grantee":"user1","grantor":"admin","acl":"user1=Cc/admin"}]`,
			Status: 200,
		},
		{
			Description: "grant select, insert to user1",
			Method:      "POST",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Body: `{
				"types": ["select", "insert"],
				"targettype": "table",
				"targetname": "table_grants",
				"grantee": "user1"
				}`,
			Status: 201,
		},
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table_grants",
			Body:        `{"name": "user1"}`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      201,
		},
		{
			Description: "fail to create a record",
			Method:      "POST",
			Query:       "/table_grants",
			Body:        `{"name": "user2"}`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      401,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table_grants?select=name",
			Expected:    `[{"name": "user1"}]`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      200,
		},
		{
			Description: "fail to select",
			Method:      "GET",
			Query:       "/table_grants?select=name",
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      401,
		},
		{
			Description: "verify grants",
			Method:      "GET",
			Query:       "http://localhost:8082/admin/grants/dbtest/table/table_grants",
			Expected: `[
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT","SELECT","UPDATE","DELETE","TRUNCATE","REFERENCES","TRIGGER"],"acl":"admin=arwdDxt/admin","grantee":"admin","grantor":"admin"},
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT","SELECT"],"acl":"user1=ar/admin","grantee":"user1","grantor":"admin"}
				]`,
			Status: 200,
		},
		{
			Description: "grant delete, update to user2",
			Method:      "POST",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Body: `{
				"types": ["delete", "update"],
				"targettype": "table",
				"targetname": "table_grants",
				"grantee": "user2"
				}`,
			Status: 201,
		},
		{
			Description: "fail to update records",
			Method:      "PATCH",
			Query:       "/table_grants",
			Body:        `{"name": "user1-update"}`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      401,
		},
		{
			Description: "update records",
			Method:      "PATCH",
			Query:       "/table_grants",
			Body:        `{"name": "user2-update"}`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      204,
		},
		{
			Description: "fail to delete records",
			Method:      "DELETE",
			Query:       "/table_grants",
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      401,
		},
		{
			Description: "delete records",
			Method:      "DELETE",
			Query:       "/table_grants",
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      204,
		},
		{
			Description: "verify grants",
			Method:      "GET",
			Query:       "http://localhost:8082/admin/grants/dbtest/table/table_grants",
			Expected: `[
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT","SELECT","UPDATE","DELETE","TRUNCATE","REFERENCES","TRIGGER"],"acl":"admin=arwdDxt/admin","grantee":"admin","grantor":"admin"},
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT","SELECT"],"acl":"user1=ar/admin","grantee":"user1","grantor":"admin"},
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["UPDATE","DELETE"],"acl":"user2=wd/admin","grantee":"user2","grantor":"admin"}
				]`,
			Status: 200,
		},
		{
			Description: "revoke select to user1",
			Method:      "DELETE",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Body: `{
				"types": ["select"],
				"targettype": "table",
				"targetname": "table_grants",
				"grantee": "user1"
				}`,
			Status: 200,
		},
		{
			Description: "revoke update to user2",
			Method:      "DELETE",
			Query:       "http://localhost:8082/admin/grants/dbtest",
			Body: `{
				"types": ["update"],
				"targettype": "table",
				"targetname": "table_grants",
				"grantee": "user2"
				}`,
			Status: 200,
		},
		{
			Description: "verify grants",
			Method:      "GET",
			Query:       "http://localhost:8082/admin/grants/dbtest/table/table_grants",
			Expected: `[
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT","SELECT","UPDATE","DELETE","TRUNCATE","REFERENCES","TRIGGER"],"acl":"admin=arwdDxt/admin","grantee":"admin","grantor":"admin"},
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["INSERT"],"acl":"user1=a/admin","grantee":"user1","grantor":"admin"},
				{"targetname":"table_grants","targetschema":"public","targettype":"table",
					"types":["DELETE"],"acl":"user2=d/admin","grantee":"user2","grantor":"admin"}
				]`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
