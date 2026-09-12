package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// bytea, inet, cidr, macaddr, time, point, bit and their arrays are requested
// by pgx in binary format; the serializer used to treat every type outside
// its binary switch as text and copy the raw bytes through. Each value must
// come out as PostgreSQL's to_json prints it, which is what PostgREST returns
// (its body is json_agg of the row): bytea as the \x hex form. The id and
// name columns are the positive controls. Expected bodies were verified
// against `select to_json(col)` on PostgreSQL 16.
func TestWireFormatTypes(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "wire_formats",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "name", "type": "text"},
					{"name": "by", "type": "bytea"},
					{"name": "ip", "type": "inet"},
					{"name": "net", "type": "cidr"},
					{"name": "mac", "type": "macaddr"},
					{"name": "mac8", "type": "macaddr8"},
					{"name": "tm", "type": "time"},
					{"name": "tmz", "type": "timetz"},
					{"name": "pt", "type": "point"},
					{"name": "bx", "type": "box"},
					{"name": "bt", "type": "bit(4)"},
					{"name": "vb", "type": "bit varying"},
					{"name": "by_arr", "type": "bytea[]"},
					{"name": "ip_arr", "type": "inet[]"},
					{"name": "tm_arr", "type": "time[]"}
				],
				"ifnotexists": true
			}`,
		},
		// a function returning a bytea, for the rpc path (as a table, whose
		// shape does not depend on the function being in the schema cache)
		{
			Method: "POST",
			Query:  "/dbtest/functions",
			Body: `{
				"name": "wire_format_blob",
				"returns": "table(blob bytea)",
				"definition": "select '\\x0102'::bytea"
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "insert one full row and one row of nulls",
			Method:      "POST",
			Query:       "/wire_formats",
			Body: `[
				{"id": 1, "name": "full", "by": "\\x0102", "ip": "192.168.1.1", "net": "192.168.1.0/24",
				 "mac": "08:00:2b:01:02:03", "mac8": "08:00:2b:01:02:03:04:05", "tm": "12:34:56", "tmz": "12:34:56+02",
				 "pt": "(1,2)", "bx": "((1,2),(3,4))", "bt": "1010", "vb": "10",
				 "by_arr": "{\"\\\\x01\",\"\\\\x02\"}", "ip_arr": "{10.0.0.1,::1}", "tm_arr": "{01:02:03,04:05:06}"},
				{"id": 2, "name": "nulls", "by": null, "ip": null, "net": null,
				 "mac": null, "mac8": null, "tm": null, "tmz": null,
				 "pt": null, "bx": null, "bt": null, "vb": null,
				 "by_arr": null, "ip_arr": null, "tm_arr": null}
			]`,
			Status: 201,
		},
	})

	// One Execute per case: Execute stops at the first body mismatch.
	tests := []test.Test{
		{
			Description: "bytea is the \\x hex text",
			Query:       "/wire_formats?id=eq.1&select=id,name,by",
			Expected:    `[{"id":1,"name":"full","by":"\\x0102"}]`,
			Status:      200,
		},
		{
			Description: "network types",
			Query:       "/wire_formats?id=eq.1&select=ip,net,mac,mac8",
			Expected:    `[{"ip":"192.168.1.1","net":"192.168.1.0/24","mac":"08:00:2b:01:02:03","mac8":"08:00:2b:01:02:03:04:05"}]`,
			Status:      200,
		},
		{
			Description: "time and timetz",
			Query:       "/wire_formats?id=eq.1&select=tm,tmz",
			Expected:    `[{"tm":"12:34:56","tmz":"12:34:56+02"}]`,
			Status:      200,
		},
		{
			Description: "geometric and bit types",
			Query:       "/wire_formats?id=eq.1&select=pt,bx,bt,vb",
			Expected:    `[{"pt":"(1,2)","bx":"(3,4),(1,2)","bt":"1010","vb":"10"}]`,
			Status:      200,
		},
		{
			Description: "arrays of those types",
			Query:       "/wire_formats?id=eq.1&select=by_arr,ip_arr,tm_arr",
			Expected:    `[{"by_arr":["\\x01","\\x02"],"ip_arr":["10.0.0.1","::1"],"tm_arr":["01:02:03","04:05:06"]}]`,
			Status:      200,
		},
		{
			Description: "nulls stay null",
			Query:       "/wire_formats?id=eq.2&select=by,ip,tm,pt,bt,by_arr",
			Expected:    `[{"by":null,"ip":null,"tm":null,"pt":null,"bt":null,"by_arr":null}]`,
			Status:      200,
		},
		{
			Description: "mixed binary and text columns in one row",
			Query:       "/wire_formats?name=eq.full&select=id,by,ip,tm,bt",
			Expected:    `[{"id":1,"by":"\\x0102","ip":"192.168.1.1","tm":"12:34:56","bt":"1010"}]`,
			Status:      200,
		},
		{
			Description: "return=representation after an update",
			Method:      "PATCH",
			Query:       "/wire_formats?id=eq.2&select=id,by,tm",
			Body:        `{"by": "\\xff", "tm": "23:59:59"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":2,"by":"\\xff","tm":"23:59:59"}]`,
			Status:      200,
		},
		{
			Description: "a function returning a bytea",
			Method:      "POST",
			Query:       "/rpc/wire_format_blob",
			Body:        `{}`,
			Expected:    `[{"blob":"\\x0102"}]`,
			Status:      200,
		},
		// CSV shares the serializer's dispatch: PostgreSQL's text for the value.
		{
			Description: "csv",
			Query:       "/wire_formats?id=eq.1&select=by,ip,mac,tm,pt,bt",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected: "by,ip,mac,tm,pt,bt\n" +
				`\x0102,192.168.1.1,08:00:2b:01:02:03,12:34:56,"(1,2)",1010`,
			Status: 200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			test.Execute(t, testConfig, []test.Test{tc})
		})
	}
}
