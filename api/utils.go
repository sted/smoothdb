package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

type SmoothError struct {
	Subsystem string
	Message   string
	Code      string
	Hint      string
	Details   string
	Position  int32
}

func noRecordsForInsert(ctx context.Context, w http.ResponseWriter, records []database.Record) (bool, int) {
	if len(records) == 0 {
		gi := database.GetSmoothContext(ctx)
		var status int
		if gi.QueryOptions.ReturnRepresentation {
			status = http.StatusCreated
			heligo.WriteJSONString(w, status, []byte("[]"))
		} else {
			status = http.StatusNoContent
			heligo.WriteEmpty(w, status)
		}
		return true, status
	}
	return false, 0
}

func noRecordsForUpdate(ctx context.Context, w http.ResponseWriter, records []database.Record) (bool, int) {
	if len(records) == 0 || len(records[0]) == 0 {
		gi := database.GetSmoothContext(ctx)
		var status int
		if gi.QueryOptions.ReturnRepresentation {
			status = http.StatusOK
			heligo.WriteJSONString(w, status, []byte("[]"))
		} else {
			status = http.StatusNoContent
			heligo.WriteEmpty(w, status)
		}
		return true, status
	}
	return false, 0
}

func WriteError(w http.ResponseWriter, err error) (int, error) {
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		return WriteBadRequest(w, err)
	case *database.SerializeError:
		w.WriteHeader(http.StatusNotAcceptable)
		return http.StatusNotAcceptable, err
	default:
		return WriteServerError(w, err)
	}
}

func WriteBadRequest(w http.ResponseWriter, err error) (int, error) {
	heligo.WriteJSON(w, http.StatusBadRequest, SmoothError{Message: err.Error()})
	return http.StatusBadRequest, err
}

func WriteServerError(w http.ResponseWriter, err error) (int, error) {
	var status int
	if _, ok := err.(*pgconn.PgError); ok {
		dberr := err.(*pgconn.PgError)
		var status int
		switch dberr.Code {
		case "42501":
			status = http.StatusUnauthorized
		case "42P01", // undefined_table
			"42883": // undefined_function
			status = http.StatusNotFound
		case "42P04", // duplicate database
			"42P06", // duplicate schema
			"42P07", // duplicate table
			"23505", // unique constraint violation
			"42710": // duplicated role
			status = http.StatusConflict
		case "22P02", // invalid_text_representation
			"42703": // undefined_column
			status = http.StatusBadRequest
		default:
			status = http.StatusInternalServerError
		}
		heligo.WriteJSON(w, status, SmoothError{
			Subsystem: "database",
			Message:   dberr.Message,
			Code:      dberr.Code,
			Hint:      dberr.Hint,
			Details:   dberr.Detail,
		})
	} else if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
		heligo.WriteEmpty(w, status)
	} else {
		status = http.StatusInternalServerError
		heligo.WriteJSON(w, status, SmoothError{Message: err.Error()})
	}
	return status, err
}

func ReadInputRecords(r heligo.Request) ([]database.Record, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	isArray := false
	for _, c := range body {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		isArray = c == '['
		break
	}
	var records []database.Record
	if isArray {
		err = json.Unmarshal(body, &records)
		if err != nil {
			return nil, err
		}
	} else {
		var record database.Record
		err = json.Unmarshal(body, &record)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
