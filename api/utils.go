package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"unicode"
	"unicode/utf8"

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
	var status int
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		return WriteBadRequest(w, err)
	case *database.SerializeError, *database.ContentTypeError:
		status = http.StatusNotAcceptable
		w.WriteHeader(status)
		return status, err
	case *database.RangeError:
		status = http.StatusRequestedRangeNotSatisfiable
		w.WriteHeader(status)
		return status, err
	default:
		return WriteServerError(w, err)
	}
}

func WriteBadRequest(w http.ResponseWriter, err error) (int, error) {
	var status int
	smootherr := SmoothError{Message: err.Error(), Subsystem: "network"}
	if maxbyteserr, ok := err.(*http.MaxBytesError); ok {
		status = http.StatusRequestEntityTooLarge
		smootherr.Details = fmt.Sprintf("ResponseMaxBytes is configured as %d", maxbyteserr.Limit)
	} else {
		status = http.StatusBadRequest
	}
	heligo.WriteJSON(w, status, smootherr)
	return status, err
}

func WriteServerError(w http.ResponseWriter, err error) (int, error) {
	var status int
	if dberr, ok := err.(*pgconn.PgError); ok {
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

func jsonIsArray(content []byte) bool {
	for len(content) > 0 {
		r, size := utf8.DecodeRune(content)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if !unicode.IsSpace(r) {
			return r == '['
		}
		content = content[size:]
	}
	return false
}

// List of supported input content types
var supportedContentTypes = []string{
	"application/json",
	"text/csv",
	"application/x-www-form-urlencoded",
	"application/octet-stream",
}
var defaultContentType = "application/json"

// getContentType gets the (input) content type and check if it is among
// the supported ones. Return "" otherwise or if we get an invalid header
func getContentType(r heligo.Request) string {
	var contentType string
	var err error
	header := r.Header.Get("Content-Type")
	if header == "" {
		contentType = defaultContentType
	} else {
		contentType, _, err = mime.ParseMediaType(header)
		if err != nil {
			return ""
		}
	}
	for _, ct := range supportedContentTypes {
		if contentType == ct {
			return ct
		}
	}
	return ""
}

func ReadInputRecords(r heligo.Request, contentType string) ([]database.Record, error) {
	var records []database.Record

	switch contentType {
	case "application/json":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		isArray := jsonIsArray(body)

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

	case "text/csv":
		reader := csv.NewReader(r.Body)
		csvData, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}

		// Assuming the first row contains headers
		headers := csvData[0]
		for _, row := range csvData[1:] {
			record := database.Record{}
			for i, value := range row {
				header := headers[i]
				if value != "NULL" {
					record[header] = value
				} else {
					record[header] = nil
				}
			}
			records = append(records, record)
		}

	case "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		record := database.Record{}
		for key, values := range r.Form { // r.Form contains both query params and form values
			record[key] = values[0] // Taking the first value for each key
		}
		records = append(records, record)
	}

	return records, nil
}

func SetResponseHeaders(ctx context.Context, w http.ResponseWriter, r *http.Request, count int64) {
	sc := database.GetSmoothContext(ctx)
	options := sc.QueryOptions
	// @@ must check if the table has a pk
	w.Header().Set("Content-Location", r.RequestURI)
	// Content-Range
	var rangeString string
	if count == 0 {
		rangeString = "*/*"
	} else {
		rangeString = strconv.FormatInt(options.RangeMin, 10) + "-" +
			strconv.FormatInt(options.RangeMin+count-1, 10) + "/*"
	}
	w.Header().Set("Content-Range", rangeString)
}

func WriteContent(ctx context.Context, w http.ResponseWriter, status int, content []byte) (int, error) {
	sc := database.GetSmoothContext(ctx)
	ct := sc.QueryOptions.ContentType
	if ct == "application/json" || ct == "text/csv" {
		ct += "; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(status)
	_, err := w.Write(content)
	return status, err
}
