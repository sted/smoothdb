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

// SmoothError is the generic struct for error reporting
type SmoothError struct {
	Subsystem string
	Message   string
	Code      string
	Hint      string
	Details   string
	Position  int32
}

// WriteBadRequest writes a BadRequest or StatusRequestEntityTooLarge status
func WriteBadRequest(w http.ResponseWriter, err error) (int, error) {
	var status int
	smootherr := SmoothError{Message: err.Error(), Subsystem: "network"}
	if maxbyteserr, ok := err.(*http.MaxBytesError); ok {
		status = http.StatusRequestEntityTooLarge
		smootherr.Details = fmt.Sprintf("RequestMaxBytes is configured as %d", maxbyteserr.Limit)
	} else {
		status = http.StatusBadRequest
	}
	heligo.WriteJSON(w, status, smootherr)
	return status, err
}

// WriteServerError write a status related to a database error
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
		heligo.WriteHeader(w, status)
	} else {
		status = http.StatusInternalServerError
		heligo.WriteJSON(w, status, SmoothError{Message: err.Error()})
	}
	return status, err
}

// WriteError is the more general error function, combining the previous two.
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

// List of supported input content types
var supportedInputContentTypes = []string{
	"application/json",
	"text/csv",
	"application/x-www-form-urlencoded",
	"application/octet-stream",
}
var defaultInputContentType = "application/json"

// getContentType gets the (input) content type and check if it is among
// the supported ones. Return "" otherwise or if we get an invalid header
func getContentType(r heligo.Request) string {
	var contentType string
	var err error
	header := r.Header.Get("Content-Type")
	if header == "" {
		contentType = defaultInputContentType
	} else {
		contentType, _, err = mime.ParseMediaType(header)
		if err != nil {
			return ""
		}
	}
	for _, ct := range supportedInputContentTypes {
		if contentType == ct {
			return ct
		}
	}
	return ""
}

// writeString writes a string with the requested content type
func writeString(w http.ResponseWriter, contentType string, content string, status int) (int, error) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, err := w.Write([]byte(content))
	return status, err
}

// writeZeroRecords writes an empty record or sequence of records, respecting the accepted content type
func writeZeroRecords(w http.ResponseWriter, contentType string, status int) (int, error) {
	var content string
	switch contentType {
	case "application/json":
		content = "[]"
	case "text/csv":
		content = ""
	}
	return writeString(w, contentType, content, status)
}

// writeIfZeroRecordsToInsert manages the case when there are 0 records on input for
// an insert operation.
// In that case it will:
// - write a response with content and status
// - return an error if the write operation fails
// - return true to mean "zero records on input"
func writeIfZeroRecordsToInsert(w http.ResponseWriter, records []database.Record,
	options *database.QueryOptions) (bool, int, error) {
	var err error
	var status int
	if len(records) == 0 {
		if options.ReturnRepresentation {
			status = http.StatusCreated
			_, err = writeZeroRecords(w, options.ContentType, status)
		} else {
			status = http.StatusNoContent
			heligo.WriteHeader(w, status)
		}
		return true, status, err
	}
	return false, 0, err
}

// writeIfZeroRecordsToUpdate manages the case when there are 0 records on input for
// an update operation.
// In that case it will:
// - write a response with content and status
// - return an error if the write operation fails
// - return true to mean "zero records on input"
func writeIfZeroRecordsToUpdate(w http.ResponseWriter, records []database.Record,
	options *database.QueryOptions) (bool, int, error) {
	var err error
	var status int
	if len(records) == 0 || len(records[0]) == 0 {
		if options.ReturnRepresentation {
			status = http.StatusOK
			_, err = writeZeroRecords(w, options.ContentType, status)
		} else {
			status = http.StatusNoContent
			heligo.WriteHeader(w, status)
		}
		return false, status, err
	}
	return true, 0, err
}

// jsonIsArray checks if a stringified json is an array
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

// readInputRecords is the low-level function to read and convert the data in the response body
func readInputRecords(r heligo.Request, contentType string) ([]database.Record, error) {
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
		for key, values := range r.Form {
			record[key] = values[0] // taking the first value for each key
		}
		records = append(records, record)
	case "application/octet-stream":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		record := database.Record{"": body}
		records = append(records, record)
	}

	return records, nil
}

// ReadRequest reads the input data from a request and manage the preconditions.
// It will emit BadRequest (400), RequestEntityTooLarge (413) or UnsupportedMediaType (415)
// status when appropriate.
// Supports JSON, CSV and x-www-form-urlencoded input data.
func ReadRequest(c context.Context, w http.ResponseWriter, r heligo.Request) (records []database.Record, status int, err error) {
	ctype := getContentType(r)
	if ctype == "" {
		// "accepted" content-type not supported
		status, err = heligo.WriteHeader(w, http.StatusUnsupportedMediaType)
		return
	} else {
		// read input records
		records, err = readInputRecords(r, ctype)
		if err != nil {
			status, err = WriteBadRequest(w, err)
			return
		}
	}
	// check preconditions
	var zero bool
	sc := database.GetSmoothContext(c)
	options := sc.QueryOptions
	if r.Method == "POST" {
		// [] as input cause no inserts
		if zero, status, err = writeIfZeroRecordsToInsert(w, records, options); zero {
			return
		}
	} else if r.Method == "PATCH" {
		if len(records) > 1 {
			status, err = WriteBadRequest(w, err)
			return
		}
		// {}, [] and [{}] as input cause no updates
		if zero, status, err = writeIfZeroRecordsToUpdate(w, records, options); zero {
			return
		}
	}
	return records, status, err
}

// SetResponseHeaders sets the response headers (for now Content-Range and Content-Location).
// It returns a status != 0 if some contraints are not satisfied and we need to include an error status
// in the response (eg 416 for RequestedRangeNotSatisfiable)
func SetResponseHeaders(ctx context.Context, w http.ResponseWriter, r heligo.Request, count int64) int {
	sc := database.GetSmoothContext(ctx)
	options := sc.QueryOptions
	// @@ must check if the table has a pk
	w.Header().Set("Content-Location", r.RequestURI)
	// Content-Range
	var rangeString string
	if count > 0 &&
		(options.RangeMin <= count || options.Count == "") &&
		((r.Method == "GET" || r.Method == "HEAD") ||
			(options.RangeMin > -1 && options.RangeMax > -1)) {

		if options.RangeMin == -1 {
			options.RangeMin = 0
		}
		if options.RangeMax == -1 || options.Count == "" {
			options.RangeMax = options.RangeMin + count - 1
		}
		rangeString = strconv.FormatInt(options.RangeMin, 10) + "-" +
			strconv.FormatInt(options.RangeMax, 10)
	} else {
		rangeString = "*"
	}
	if options.Count == "" {
		rangeString += "/*"
	} else {
		rangeString += "/" + strconv.FormatInt(count, 10)
	}
	w.Header().Set("Content-Range", rangeString)
	if options.RangeMin > count && options.Count != "" ||
		options.RangeMin > -1 && options.RangeMax > -1 && options.RangeMin > options.RangeMax {
		return http.StatusRequestedRangeNotSatisfiable
	}
	if options.Count != "" && options.HasRange &&
		options.RangeMax-options.RangeMin+1 < count {
		return http.StatusPartialContent
	}
	return 0
}

// WriteContent writes the response and its content type
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
