package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// The status a PostgreSQL error is reported with must match PostgREST's, whose
// mapping lives in `mapSQLtoHTTP` (src/library/PostgREST/Error.hs): an explicit
// list of codes and class prefixes, and **400 for everything else** — a database
// error is a client error unless it is known to be the server's fault.
//
// Deliberate divergences, asserted here so they stay deliberate:
//   - 42501 insufficient_privilege is 401 regardless of authentication, where
//     PostgREST answers 403 to an authenticated caller. smoothdb's middleware
//     already answers 401 before Postgres is reached, and the ported PostgREST
//     suites run anonymously, where PostgREST answers 401 too.
//   - the "already exists" codes of the DDL API (duplicate database, schema,
//     table, role) are 409. PostgREST has no DDL surface, so these reach its
//     catch-all 400; a Conflict is the accurate answer for /admin.
func TestPgErrorStatusMatchesPostgREST(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		want    int
	}{
		// The regression this table was written for: a type mismatch is the
		// client's fault (`name=is.true` on a text column), not the server's.
		{"datatype_mismatch", "42804", "argument of IS TRUE must be type boolean, not type text", http.StatusBadRequest},
		{"undefined_object (text search config)", "42704", `text search configuration "bogus" does not exist`, http.StatusBadRequest},
		{"syntax_error", "42601", "syntax error at or near", http.StatusBadRequest},
		{"invalid_datetime_format", "22007", `invalid input syntax for type timestamp: "x"`, http.StatusBadRequest},
		{"numeric_value_out_of_range", "22003", "value out of range", http.StatusBadRequest},
		{"not_null_violation", "23502", `null value in column "a" violates not-null constraint`, http.StatusBadRequest},
		{"check_violation", "23514", "violates check constraint", http.StatusBadRequest},

		// Already correct, and must not move.
		{"invalid_text_representation", "22P02", `invalid input syntax for type bigint: "x"`, http.StatusBadRequest},
		{"undefined_column", "42703", "column t.x does not exist", http.StatusBadRequest},
		{"undefined_table", "42P01", `relation "t" does not exist`, http.StatusNotFound},
		{"undefined_function", "42883", "function f() does not exist", http.StatusNotFound},
		{"unique_violation", "23505", "duplicate key value violates unique constraint", http.StatusConflict},
		{"infinite_recursion", "42P17", "infinite recursion detected in rules for relation", http.StatusInternalServerError},
		{"insufficient_privilege", "42501", "permission denied for table t", http.StatusUnauthorized},

		// The DDL API's own conflicts.
		{"duplicate_database", "42P04", `database "d" already exists`, http.StatusConflict},
		{"duplicate_schema", "42P06", `schema "s" already exists`, http.StatusConflict},
		{"duplicate_table", "42P07", `relation "t" already exists`, http.StatusConflict},
		{"duplicate_object (role)", "42710", `role "r" already exists`, http.StatusConflict},

		// Class prefixes PostgREST maps away from the 400 default.
		{"08 connection_exception", "08006", "connection failure", http.StatusServiceUnavailable},
		{"09 triggered_action_exception", "09000", "triggered action exception", http.StatusInternalServerError},
		{"0L invalid_grantor", "0L000", "invalid grantor", http.StatusForbidden},
		{"0P invalid_role_specification", "0P000", "invalid role specification", http.StatusForbidden},
		{"foreign_key_violation", "23503", "violates foreign key constraint", http.StatusConflict},
		{"read_only_sql_transaction", "25006", "cannot execute in a read-only transaction", http.StatusMethodNotAllowed},
		{"25 invalid_transaction_state", "25000", "invalid transaction state", http.StatusInternalServerError},
		{"28 invalid_authorization", "28000", "invalid authorization specification", http.StatusForbidden},
		{"2D invalid_transaction_termination", "2D000", "invalid transaction termination", http.StatusInternalServerError},
		{"38 external_routine_exception", "38000", "external routine exception", http.StatusInternalServerError},
		{"39 external_routine_invocation", "39000", "external routine invocation exception", http.StatusInternalServerError},
		{"3B savepoint_exception", "3B000", "savepoint exception", http.StatusInternalServerError},
		{"40 transaction_rollback", "40001", "could not serialize access", http.StatusInternalServerError},
		{"configuration_limit_exceeded", "53400", "configuration limit exceeded", http.StatusInternalServerError},
		{"53 insufficient_resources", "53300", "too many clients already", http.StatusServiceUnavailable},
		{"54 program_limit_exceeded", "54001", "statement too complex", http.StatusInternalServerError},
		{"55 object_not_in_prerequisite_state", "55000", "object not in prerequisite state", http.StatusInternalServerError},
		{"admin_shutdown", "57P01", "terminating connection due to administrator command", http.StatusServiceUnavailable},
		{"57 operator_intervention", "57014", "canceling statement due to user request", http.StatusInternalServerError},
		{"58 system_error", "58000", "system error", http.StatusInternalServerError},
		{"F0 config_file_error", "F0000", "config file error", http.StatusInternalServerError},
		{"HV fdw_error", "HV000", "fdw error", http.StatusInternalServerError},
		{"raise_exception", "P0001", "bad thing", http.StatusBadRequest},
		{"P0 plpgsql_error", "P0002", "no data found", http.StatusInternalServerError},

		// Message-dependent, exactly as PostgREST reads them.
		{"cardinality_violation from pg-safeupdate", "21000", "DELETE requires a WHERE clause", http.StatusBadRequest},
		{"cardinality_violation from a subquery", "21000", "more than one row returned by a subquery used as an expression", http.StatusInternalServerError},
		{"invalid_parameter_value, missing role", "22023", `role "nobody" does not exist`, http.StatusUnauthorized},
		{"invalid_parameter_value, anything else", "22023", "invalid parameter value", http.StatusBadRequest},

		// PTxxx lets a function pick the status: `raise sqlstate 'PT402'`.
		{"PT402", "PT402", "Payment Required", http.StatusPaymentRequired},
		{"PT301", "PT301", "Moved Permanently", http.StatusMovedPermanently},
		{"PT with a non-numeric status", "PT40A", "Wrong", http.StatusInternalServerError},

		// Unknown to both: the client's fault by default.
		{"unknown code", "ZZ999", "who knows", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			status, _ := WriteServerError(w, &pgconn.PgError{Code: tt.code, Message: tt.message})
			if status != tt.want {
				t.Errorf("%s (%s): got %d, want %d", tt.name, tt.code, status, tt.want)
			}
			if w.Code != status {
				t.Errorf("%s (%s): wrote %d to the response but returned %d", tt.name, tt.code, w.Code, status)
			}
		})
	}
}
