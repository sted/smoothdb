package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/jqeval"
)

var verboseErrors = true

// SetVerboseErrors controls whether full database error details are returned to clients.
func SetVerboseErrors(v bool) {
	verboseErrors = v
}

// SmoothError is the generic struct for error reporting
type SmoothError struct {
	Subsystem string `json:"subsystem"`
	Message   string `json:"message"`
	Code      string `json:"code"`
	Hint      string `json:"hint"`
	Details   any    `json:"details"`
	Position  int32  `json:"position"`
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
	var dberr *pgconn.PgError
	if errors.As(err, &dberr) {
		status = pgErrorStatus(dberr)
		if status == http.StatusMethodNotAllowed {
			// 25006 read_only_sql_transaction: a write reached by a GET/HEAD, or
			// by a STABLE/IMMUTABLE function. PostgREST answers 405 without an
			// Allow header; RFC 9110 requires one, and POST is the method every
			// resource that can raise this accepts.
			w.Header().Set("Allow", "POST")
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
	} else if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
		heligo.WriteHeader(w, status)
	} else if connectErr := (*pgconn.ConnectError)(nil); errors.As(err, &connectErr) {
		// the text names the database user, the database and the dial address
		status = http.StatusBadRequest
		msg := "cannot connect to the database"
		if verboseErrors {
			msg = err.Error()
		}
		heligo.WriteJSON(w, status, SmoothError{Subsystem: "auth", Message: msg})
	} else {
		status = http.StatusInternalServerError
		msg := "internal server error"
		if verboseErrors {
			msg = err.Error()
		}
		heligo.WriteJSON(w, status, SmoothError{Message: msg})
	}
	return status, err
}

// WriteError is the more general error function, combining the previous two.
func WriteError(w http.ResponseWriter, err error) (int, error) {
	var status int
	switch err.(type) {
	case *database.ParseError, *database.BuildError, *jqeval.Error:
		return WriteBadRequest(w, err)
	case *database.SchemaError:
		status = http.StatusNotAcceptable
		heligo.WriteJSON(w, status, SmoothError{Subsystem: "network", Message: err.Error(), Hint: err.(*database.SchemaError).Hint})
		return status, err
	case *database.AmbiguousFunctionError:
		// PostgREST answers 300 Multiple Choices, with no choice to follow
		status = http.StatusMultipleChoices
		heligo.WriteJSON(w, status, SmoothError{Subsystem: "network", Message: err.Error(), Hint: err.(*database.AmbiguousFunctionError).Hint})
		return status, err
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

// payloadFilters is the request's query for a write that carries a body:
// PostgREST applies ?columns= to a JSON body only (Payload.hs, getPayload),
// while a CSV header, or a form's fields, is the list of columns or of an
// RPC's arguments.
func payloadFilters(r heligo.Request) url.Values {
	filters := r.URL.Query()
	if ct := getContentType(r); ct == "text/csv" || ct == "application/x-www-form-urlencoded" {
		delete(filters, "columns")
	}
	return filters
}

// recordsJSON is the JSON PostgREST turns a CSV or form body into before an
// insert (Payload.hs: csvToJson, the form's params map): every value a JSON
// string, or null. Sent as the insert's body, it gives a json column a JSON
// string whatever the text spells, never the meaning a Go value would have.
func recordsJSON(records []database.Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(records); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// readInputRecords is the low-level function to read and convert the data in the request body.
// For a JSON body it also returns the body itself, which an insert sends as it is,
// and for a CSV or form body the JSON PostgREST would convert it to.
func readInputRecords(r heligo.Request, contentType string) ([]database.Record, []byte, error) {
	var records []database.Record
	var jsonBody []byte

	switch contentType {
	case "application/json":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, err
		}
		jsonBody = body
		isArray := jsonIsArray(body)

		// Create decoder with UseNumber to preserve large integers
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()

		if isArray {
			err = decoder.Decode(&records)
			if err != nil {
				return nil, nil, err
			}
		} else {
			var record database.Record
			err = decoder.Decode(&record)
			if err != nil {
				return nil, nil, err
			}
			records = append(records, record)
		}

	case "text/csv":
		reader := csv.NewReader(r.Body)
		csvData, err := reader.ReadAll()
		if err != nil {
			return nil, nil, err
		}

		if len(csvData) == 0 {
			// no header: PostgREST answers 400 (PGRST102) with the parser's message
			return nil, nil, errors.New(`parse error (not enough input) at ""`)
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
		if jsonBody, err = recordsJSON(records); err != nil {
			return nil, nil, err
		}

	case "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			return nil, nil, err
		}
		// the body's fields only, not the URL's parameters (PostForm), and the
		// last value of a repeated key, as PostgREST's params map keeps it
		record := database.Record{}
		for key, values := range r.PostForm {
			record[key] = values[len(values)-1]
		}
		records = append(records, record)
		var err error
		if jsonBody, err = recordsJSON(records); err != nil {
			return nil, nil, err
		}
	case "application/octet-stream":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, err
		}
		record := database.Record{"": body}
		records = append(records, record)
	}

	return records, jsonBody, nil
}

// ReadRequest reads the input data from a request and manage the preconditions.
// It will emit BadRequest (400), RequestEntityTooLarge (413) or UnsupportedMediaType (415)
// status when appropriate.
// Supports JSON, CSV and x-www-form-urlencoded input data.
func ReadRequest(c context.Context, w http.ResponseWriter, r heligo.Request) (records []database.Record, status int, err error) {
	records, _, status, err = readRequest(c, w, r)
	return
}

// readRequest is ReadRequest that also returns the JSON body the records were
// decoded from, nil for another content type.
func readRequest(c context.Context, w http.ResponseWriter, r heligo.Request) (records []database.Record, jsonBody []byte, status int, err error) {
	ctype := getContentType(r)
	if ctype == "" {
		// "accepted" content-type not supported
		status, err = heligo.WriteHeader(w, http.StatusUnsupportedMediaType)
		return
	} else {
		// read input records
		records, jsonBody, err = readInputRecords(r, ctype)
		if err != nil {
			status, err = WriteBadRequest(w, err)
			return
		}
	}
	// check preconditions
	var zero bool
	sc := database.GetSmoothContext(c)
	options := &sc.QueryOptions
	if r.Method == "POST" {
		// [] as input cause no inserts
		if zero, status, err = writeIfZeroRecordsToInsert(w, records, options); zero {
			return
		}
	} else if r.Method == "PATCH" {
		if len(records) > 1 {
			// A deliberate divergence: PostgREST hands the whole array to
			// json_to_recordset and updates with whichever row PostgreSQL
			// picks; a body that cannot mean one update is refused instead.
			status, err = WriteBadRequest(w, errors.New("a PATCH body must be a single object"))
			return
		}
		// {}, [] and [{}] as input cause no updates
		if zero, status, err = writeIfZeroRecordsToUpdate(w, records, options); zero {
			return
		}
	}
	return records, jsonBody, status, err
}

// SetResponseHeaders sets the response headers (Content-Location and
// Content-Range). A read (GET, HEAD) goes to SetReadResponseHeaders; a write
// has no range, as PostgREST's contentRangeH 1 0: its Content-Range is */*,
// or */count with Prefer: count. It returns a status != 0 if the response
// must take it instead of the usual one (416, 206).
func SetResponseHeaders(ctx context.Context, w http.ResponseWriter, r heligo.Request, count int64) int {
	if r.Method == "GET" || r.Method == "HEAD" {
		return SetReadResponseHeaders(ctx, w, r, count)
	}
	sc := database.GetSmoothContext(ctx)
	options := &sc.QueryOptions
	// @@ must check if the table has a pk
	w.Header().Set("Content-Location", r.RequestURI)
	if options.Count == "" {
		w.Header().Set("Content-Range", "*/*")
	} else {
		w.Header().Set("Content-Range", "*/"+strconv.FormatInt(count, 10))
	}
	return 0
}

// SetReadResponseHeaders sets the headers of a read, from a table or a
// function, whatever the method: Content-Location, and the Content-Range of
// PostgREST's rangeStatusHeader (RangeQuery.hs). rows is the number of rows
// read; the total is known when the request counts them (QueryOptions.Total).
// It returns 206 when the page is shorter than the total, 416 when the offset
// is past it, and 0 otherwise (see WriteRangeNotSatisfiable for the 416).
func SetReadResponseHeaders(ctx context.Context, w http.ResponseWriter, r heligo.Request, rows int64) int {
	sc := database.GetSmoothContext(ctx)
	options := &sc.QueryOptions
	w.Header().Set("Content-Location", r.RequestURI)
	lower := max(options.RangeMin, 0) // the offset
	upper := lower + rows - 1
	counted := options.Count != ""
	rangeString := "*"
	if (!counted || options.Total != 0) && lower <= upper {
		rangeString = strconv.FormatInt(lower, 10) + "-" + strconv.FormatInt(upper, 10)
	}
	if counted {
		rangeString += "/" + strconv.FormatInt(options.Total, 10)
	} else {
		rangeString += "/*"
	}
	w.Header().Set("Content-Range", rangeString)
	switch {
	case !counted:
		return 0
	case lower > options.Total:
		return http.StatusRequestedRangeNotSatisfiable
	case rows < options.Total:
		return http.StatusPartialContent
	}
	return 0
}

// WriteRangeNotSatisfiable writes the 416 of a read whose offset is past its
// total, with PostgREST's PGRST103 error (Response.hs, InvalidRange OutOfBounds).
func WriteRangeNotSatisfiable(ctx context.Context, w http.ResponseWriter) (int, error) {
	options := database.GetQueryOptions(ctx)
	status := http.StatusRequestedRangeNotSatisfiable
	return heligo.WriteJSON(w, status, SmoothError{
		Subsystem: "network",
		Message:   "Requested range not satisfiable",
		Code:      "PGRST103",
		Details: fmt.Sprintf("An offset of %d was requested, but there are only %d rows.",
			max(options.RangeMin, 0), options.Total),
	})
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

// StripPrefix returns a new request with the URL path stripped from a prefix.
// It returns nil if the URL path has nbot been stripped.
// Taken from the handler version in the standard lib: http.StripPrefix.
func StripPrefix(r *http.Request, prefix string) *http.Request {
	p := strings.TrimPrefix(r.URL.Path, prefix)
	rp := strings.TrimPrefix(r.URL.RawPath, prefix)
	if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = p
		r2.URL.RawPath = rp
		return r2
	}
	return nil
}

// authClient calls the external auth server. Its timeout is a backstop: the
// caller bounds each call with its context.
var authClient = &http.Client{Timeout: 30 * time.Second}

// maxAuthReplyBytes caps a reply of the auth server.
const maxAuthReplyBytes = 1 << 20

func postJSON[I any](ctx context.Context, url string, inputData I, output any, errorOutput any) (int, error) {
	jsonData, err := json.Marshal(inputData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal input data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAuthReplyBytes+1))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}
	if len(body) > maxAuthReplyBytes {
		return 0, fmt.Errorf("auth server reply exceeds %d bytes", maxAuthReplyBytes)
	}

	if resp.StatusCode >= 400 {
		if errorOutput != nil {
			if err := json.Unmarshal(body, errorOutput); err != nil {
				return resp.StatusCode, fmt.Errorf("failed to unmarshal error response: %w", err)
			}
		}
		return resp.StatusCode, nil
	}

	if err := json.Unmarshal(body, output); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to unmarshal successful response: %w", err)
	}

	return resp.StatusCode, nil
}
