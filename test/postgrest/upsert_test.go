package postgrest

import (
	"green/green-ds/test"
	"testing"
)

func TestPostgREST_Upsert(t *testing.T) {

	tests := []test.Test{}

	test.Execute(t, testConfig, tests)
}
