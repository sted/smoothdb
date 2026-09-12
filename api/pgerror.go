package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgErrorStatus maps a PostgreSQL SQLSTATE to the HTTP status PostgREST reports it
// with, following `mapSQLtoHTTP` in src/library/PostgREST/Error.hs.
//
// The shape matters as much as the entries: an explicit list of codes and class
// prefixes, and 400 for everything not on it. An unlisted database error is the
// client's fault — a type mismatch, an unparseable literal, a violated constraint —
// so defaulting to 500 turns ordinary rejections into server errors and leaves the
// caller unable to tell a bad request from an outage.
//
// Two entries deliberately differ from PostgREST, asserted in
// TestPgErrorStatusMatchesPostgREST: insufficient_privilege is 401 whoever the
// caller is, and the "already exists" codes of the DDL API are 409.
func pgErrorStatus(dberr *pgconn.PgError) int {
	code, message := dberr.Code, dberr.Message

	// Exact codes first: every one of them sits in a class mapped differently below.
	switch code {
	case "23503", // foreign_key_violation
		"23505": // unique_violation
		return http.StatusConflict
	case "42P04", // duplicate_database
		"42P06", // duplicate_schema
		"42P07", // duplicate_table
		"42710": // duplicate_object, a role
		return http.StatusConflict
	case "25006": // read_only_sql_transaction
		return http.StatusMethodNotAllowed
	case "21000": // cardinality_violation
		// pg-safeupdate refusing an unqualified write is the client's fault; the
		// rest of the code is a function or view returning more rows than one.
		if strings.HasSuffix(message, "requires a WHERE clause") {
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	case "22023": // invalid_parameter_value
		// A role named in a valid token but absent from the database arrives here.
		if strings.HasPrefix(message, "role") && strings.HasSuffix(message, "does not exist") {
			return http.StatusUnauthorized
		}
		return http.StatusBadRequest
	case "53400": // configuration_limit_exceeded
		return http.StatusInternalServerError
	case "57P01": // admin_shutdown
		return http.StatusServiceUnavailable
	case "P0001": // raise_exception, the code a bare `raise` carries
		return http.StatusBadRequest
	case "42883", // undefined_function
		"42P01": // undefined_table
		return http.StatusNotFound
	case "42P17": // infinite_recursion
		return http.StatusInternalServerError
	case "42501": // insufficient_privilege
		return http.StatusUnauthorized
	}

	// `raise sqlstate 'PT402'` lets a function choose the status. A suffix that is
	// not a status is not one, and stays the server's fault.
	if rest, ok := strings.CutPrefix(code, "PT"); ok {
		if status, err := strconv.Atoi(rest); err == nil && status >= 100 && status <= 599 {
			return status
		}
		return http.StatusInternalServerError
	}

	if len(code) >= 2 {
		switch code[:2] {
		case "08": // connection_exception
			return http.StatusServiceUnavailable
		case "53": // insufficient_resources
			return http.StatusServiceUnavailable
		case "0L", // invalid_grantor
			"0P", // invalid_role_specification
			"28": // invalid_authorization_specification
			return http.StatusForbidden
		case "09", // triggered_action_exception
			"25", // invalid_transaction_state
			"2D", // invalid_transaction_termination
			"38", // external_routine_exception
			"39", // external_routine_invocation_exception
			"3B", // savepoint_exception
			"40", // transaction_rollback
			"54", // program_limit_exceeded
			"55", // object_not_in_prerequisite_state
			"57", // operator_intervention
			"58", // system_error
			"F0", // config_file_error
			"HV", // fdw_error
			"P0": // plpgsql_error
			return http.StatusInternalServerError
		}
	}

	return http.StatusBadRequest
}
