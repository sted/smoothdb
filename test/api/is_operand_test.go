package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// The operand of `is` is one of five keywords, as in PostgREST; a quoted or
// unknown operand is refused by the parser with 400 before any SQL runs, so the
// table need not even exist.
func TestIsOperandRejected(t *testing.T) {
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	// a filter that fails to parse is PostgREST's PGRST100 (QPError)
	body := func(filter string) string {
		return `{"code":"PGRST100","details":"IS operator requires null, not_null, true, false or unknown","hint":null,"message":"failed to parse filter (` + filter + `)"}`
	}
	tests := []test.Test{
		{Description: "quoted operand", Query: `/whatever?a=is."foo"`, Expected: body(`is.\"foo\"`), Status: 400},
		{Description: "quoted keyword", Query: `/whatever?a=is."null"`, Expected: body(`is.\"null\"`), Status: 400},
		{Description: "quoted operand on a JSON path", Query: `/whatever?data->>k=is."null"`, Expected: body(`is.\"null\"`), Status: 400},
		{Description: "bare non-keyword", Query: `/whatever?a=is.foo`, Expected: body(`is.foo`), Status: 400},
	}
	test.Execute(t, testConfig, tests)
}
