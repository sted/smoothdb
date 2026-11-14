package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestInvalidResourcePath: a path with more segments than a table or
// /rpc/name answers PostgREST's PGRST125 for every method, before the method
// is checked (ApiRequest.hs getResource). The controls show that the routes
// for a table and for /rpc/name still win over the fallback route.
func TestInvalidResourcePath(t *testing.T) {
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	invalid := `{"code":"PGRST125","details":null,"hint":null,"message":"Invalid path specified in request URL"}`
	proxy := func(code string) map[string]string {
		return map[string]string{"Proxy-Status": "PostgREST; error=" + code}
	}
	tests := []test.Test{
		{Description: "GET with two segments", Query: "/a/b", Expected: invalid, ExpectedHeaders: proxy("PGRST125"), Status: 404},
		{Description: "POST with three segments", Method: "POST", Query: "/a/b/c", Body: `{}`, Expected: invalid, Status: 404},
		{Description: "PATCH with two segments", Method: "PATCH", Query: "/a/b", Body: `{}`, Expected: invalid, Status: 404},
		{Description: "PUT with two segments", Method: "PUT", Query: "/a/b", Body: `{}`, Expected: invalid, Status: 404},
		{Description: "DELETE with two segments", Method: "DELETE", Query: "/a/b", Expected: invalid, Status: 404},
		{Description: "HEAD with two segments", Method: "HEAD", Query: "/a/b", Status: 404},
		{Description: "a segment after /rpc/name", Query: "/rpc/f/x", Expected: invalid, Status: 404},
		{Description: "DELETE with a segment after /rpc/name", Method: "DELETE", Query: "/rpc/f/x", Expected: invalid, Status: 404},
		// CleanPaths shortens this path after the match: the route reads no
		// path parameter, so it cannot panic on the shorter path
		{Description: "dot segments", Query: "/a/../../x", Expected: invalid, Status: 404},
		// controls: the routes of a table and of a function are taken first
		{Description: "control: a missing table", Query: "/no_such_table", ExpectedHeaders: proxy("PGRST205"), Status: 404},
		{Description: "control: a missing function", Query: "/rpc/no_such_function", ExpectedHeaders: proxy("PGRST202"), Status: 404},
		{Description: "control: DELETE on /rpc/name", Method: "DELETE", Query: "/rpc/f", ExpectedHeaders: proxy("PGRST101"), Status: 405},
	}
	test.Execute(t, testConfig, tests)
}
