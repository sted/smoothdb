package test_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/test"
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
	group_db := router.Group("/benchdb", server.DatabaseMiddlewareStd(srv, false))
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
		// {
		// 	Description: "select",
		// 	Method:      "GET",
		// 	Query:       "/api/dbtest/table_records",
		// },
	}

	for _, t := range tests {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(t.Method, t.Query, nil)
		req.Header.Add("Authorization", user1Token)
		if err != nil {
			panic(err)
		}
		b.Run(t.Description, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				router.ServeHTTP(w, req)
			}

		})
	}
}
