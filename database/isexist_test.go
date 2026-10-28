package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsExist asserted err.(*pgconn.PgError) unchecked: any other error type,
// nil included, panicked, and a wrapped PgError was not recognized.
func TestIsExist(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"duplicate role", &pgconn.PgError{Code: "42710"}, true},
		{"other PgError", &pgconn.PgError{Code: "42501"}, false},
		{"wrapped duplicate database", fmt.Errorf("create: %w", &pgconn.PgError{Code: "42P04"}), true},
		{"not a PgError", errors.New("connection reset"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("IsExist panicked: %v", r)
				}
			}()
			if got := IsExist(c.err); got != c.want {
				t.Errorf("IsExist = %v, want %v", got, c.want)
			}
		})
	}
}
