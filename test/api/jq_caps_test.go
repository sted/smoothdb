package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// The test server runs with JQ.MaxOutputBytes: 65536. A program whose output
// serializes above that is answered with a per-item error, not with the
// payload; the other items of the batch are unaffected.
func TestJQOutputCap(t *testing.T) {
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082",
		CommonHeaders: test.Headers{"Authorization": {user1Token}},
	}
	tests := []test.Test{
		{
			Description: "oversized output is refused per item",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": [{"program": "\"a\" * 70000", "input": null}, {"program": ". + 1", "input": 1}]}`,
			Expected:    `[{"error": "jq output of 70002 bytes exceeds the maximum allowed size of 65536 bytes"}, {"output": 2}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig, tests)
}
