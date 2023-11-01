package test_api

import (
	"context"
	"heligo"
	"net/http"
	"testing"

	"github.com/smoothdb/smoothdb/server"
	"github.com/smoothdb/smoothdb/test"
)

func BenchmarkBase(b *testing.B) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	router := srv.GetRouter()
	group := router.Group("/bench")
	group.Handle("GET", "/empty", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		return 200, nil
	})
	group_db := router.Group("/benchdb", server.DatabaseMiddleware(srv, false))
	group_db.Handle("GET", "/empty", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		return 200, nil
	})

	commands := []test.Command{ // create table table_records
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "table_records",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "int4", "type": "int4"},
					{"name": "int8", "type": "int8"},
					{"name": "float4", "type": "float4"},
					{"name": "float8", "type": "float8"},
					{"name": "interval", "type": "interval"}
				],
				"ifnotexists": true
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
		NoCookies:     true,
	}

	tests := []test.Test{
		{
			Description: "empty",
			Method:      "GET",
			Query:       "/bench/empty",
		},
		{
			Description: "middleware",
			Method:      "GET",
			Query:       "/benchdb/empty",
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/api/dbtest/table_records",
		},
	}

	client := test.InitClient(false)

	for _, t := range tests {
		command := &test.Command{t.Description, t.Method, t.Query, t.Body, t.Headers}

		b.Run(t.Description, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := test.Exec(client, testConfig, command)
				if err != nil {

				}
			}
		})
	}
}
