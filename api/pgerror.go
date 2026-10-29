package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
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
// insufficient_privilege depends on the caller, as in PostgREST: 401 for an
// anonymous request, which may succeed with a token, 403 for an authenticated
// one, which will not. One group of entries deliberately differs from
// PostgREST, asserted in TestPgErrorStatusMatchesPostgREST: the "already
// exists" codes of the DDL API are 409.
func pgErrorStatus(dberr *pgconn.PgError, authenticated bool) int {
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
		if authenticated {
			return http.StatusForbidden
		}
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

// writeDatabaseError writes a database error with the status PostgREST reports
// it with. A 401 carries the bare challenge, as PostgREST's PgError headers do.
func writeDatabaseError(w http.ResponseWriter, dberr *pgconn.PgError, authenticated bool) (int, error) {
	status := pgErrorStatus(dberr, authenticated)
	switch status {
	case http.StatusMethodNotAllowed:
		// 25006 read_only_sql_transaction: a write reached by a GET/HEAD, or
		// by a STABLE/IMMUTABLE function. PostgREST answers 405 without an
		// Allow header; RFC 9110 requires one, and POST is the method every
		// resource that can raise this accepts.
		w.Header().Set("Allow", "POST")
	case http.StatusUnauthorized:
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	smoothErr := SmoothError{
		Subsystem: "database",
		Message:   dberr.Message,
		Code:      dberr.Code,
	}
	if verboseErrors {
		smoothErr.Hint = dberr.Hint
		smoothErr.Details = dberr.Detail
	}
	heligo.WriteJSON(w, status, smoothErr)
	return status, dberr
}

// WriteAuthError is the error writer of the auth middleware: a refusal of its
// own with its status, code and challenge, anything else — the database errors
// of the role switch — as WriteServerError writes it.
func WriteAuthError(w http.ResponseWriter, err error, authenticated bool) (int, error) {
	var aerr *authn.Error
	if !errors.As(err, &aerr) {
		return WriteServerError(authn.WithAuthenticated(w, authenticated), err)
	}
	if aerr.Challenge != "" {
		w.Header().Set("WWW-Authenticate", aerr.Challenge)
	}
	subsystem := "database" // an unknown database, a pool with no connection
	if aerr.Status == http.StatusUnauthorized || aerr.Status == http.StatusForbidden {
		subsystem = "auth"
	}
	smoothErr := SmoothError{Subsystem: subsystem, Message: aerr.Message, Code: aerr.Code}
	if verboseErrors && aerr.Err != nil && aerr.Err.Error() != aerr.Message {
		smoothErr.Details = aerr.Err.Error()
	}
	heligo.WriteJSON(w, aerr.Status, smoothErr)
	return aerr.Status, err
}
