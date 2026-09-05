package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// The test server runs with LoginRateLimit: 5. Five failed logins from this
// client are answered 400 (bad credentials); the sixth is refused with 429 and
// a Retry-After header, before the credentials are even checked.
func TestTokenRateLimit(t *testing.T) {
	testConfig := test.Config{BaseUrl: "http://localhost:8082"}

	attempt := test.Test{
		Description: "failed login attempt",
		Method:      "POST",
		Query:       "/token",
		Body:        `{"email": "nobody", "password": "wrong"}`,
		Status:      400,
	}
	tests := []test.Test{attempt, attempt, attempt, attempt, attempt,
		{
			Description:     "sixth attempt within the minute is rate limited",
			Method:          "POST",
			Query:           "/token",
			Body:            `{"email": "nobody", "password": "wrong"}`,
			Status:          429,
			ExpectedHeaders: map[string]string{"Retry-After": "12"},
		},
	}
	test.Execute(t, testConfig, tests)
}
