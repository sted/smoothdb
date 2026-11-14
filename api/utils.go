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
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/jqeval"
)

var verboseErrors = true

// SetVerboseErrors controls whether full database error details are returned to clients.
func SetVerboseErrors(v bool) {
	verboseErrors = v
}

// SmoothError is an error body, in PostgREST's shape (Error.hs,
// errorPayload): code, details, hint and message. Code is a PGRST code for
// an error that smoothdb decides, the SQLSTATE for a database error, and ""
// for an error that PostgREST does not have. Details is a string or a JSON
// value. An empty Details or Hint is written as null.
type SmoothError struct {
	Code    string
	Message string
	Details any
	Hint    string
}

// errorBody is the JSON of a SmoothError. The fields are in the order that
// PostgREST writes them: aeson sorts the keys of an object.
type errorBody struct {
	Code    string `json:"code"`
	Details any    `json:"details"`
	Hint    any    `json:"hint"`
	Message string `json:"message"`
}

// writeErrorBody writes an error with its status. The JSON escapes no HTML
// character, as aeson does, so the Content-Length is PostgREST's. The JSON
// encoder is acceptable here: an error body is small and not on the path of
// the rows.
func writeErrorBody(w http.ResponseWriter, status int, e SmoothError) (int, error) {
	body := errorBody{Code: e.Code, Message: e.Message}
	if s, ok := e.Details.(string); !ok || s != "" {
		body.Details = e.Details
	}
	if e.Hint != "" {
		body.Hint = e.Hint
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}
	data := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if e.Code != "" {
		// PostgREST names the code of every error in this header (RFC 9209)
		w.Header().Set("Proxy-Status", "PostgREST; error="+e.Code)
	}
	w.WriteHeader(status)
	_, err := w.Write(data)
	return status, err
}

// WriteBadRequest writes a BadRequest or StatusRequestEntityTooLarge status
func WriteBadRequest(w http.ResponseWriter, err error) (int, error) {
	var status int
	smootherr := SmoothError{Message: err.Error()}
	if maxbyteserr, ok := err.(*http.MaxBytesError); ok {
		status = http.StatusRequestEntityTooLarge
		smootherr.Details = fmt.Sprintf("RequestMaxBytes is configured as %d", maxbyteserr.Limit)
	} else {
		status = http.StatusBadRequest
	}
	writeErrorBody(w, status, smootherr)
	return status, err
}

// writeRequestError writes an error that smoothdb decides before the
// database runs the request, with its PGRST code.
func writeRequestError(w http.ResponseWriter, err *database.RequestError) (int, error) {
	return writeErrorBody(w, err.Status, SmoothError{
		Code: err.Code, Message: err.Message, Details: err.Details, Hint: err.Hint,
	})
}

// WriteServerError write a status related to a database error
func WriteServerError(w http.ResponseWriter, err error) (int, error) {
	var status int
	var dberr *pgconn.PgError
	if errors.As(err, &dberr) {
		status, _ = writeDatabaseError(w, dberr, authn.IsAuthenticated(w))
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
		writeErrorBody(w, status, SmoothError{Message: msg})
	} else {
		status = http.StatusInternalServerError
		msg := "internal server error"
		if verboseErrors {
			msg = err.Error()
		}
		writeErrorBody(w, status, SmoothError{Message: msg})
	}
	return status, err
}

// WriteError is the more general error function, combining the previous two.
func WriteError(w http.ResponseWriter, err error) (int, error) {
	var status int
	switch e := err.(type) {
	case *database.RequestError:
		writeRequestError(w, e)
		return e.Status, err
	case *database.ParseError:
		code, message, details := e.Body()
		writeErrorBody(w, http.StatusBadRequest, SmoothError{Code: code, Message: message, Details: details})
		return http.StatusBadRequest, err
	case *database.BuildError, *jqeval.Error:
		return WriteBadRequest(w, err)
	case *database.CodedParseError:
		// a parse error with a code of its own, such as PGRST123
		status = http.StatusBadRequest
		writeErrorBody(w, status, SmoothError{Code: e.Code, Message: e.Msg})
		return status, err
	case *database.SchemaError:
		// PostgREST's UnacceptableSchema (PGRST106)
		status = http.StatusNotAcceptable
		writeErrorBody(w, status, SmoothError{Code: "PGRST106", Message: err.Error(), Hint: e.Hint})
		return status, err
	case *database.AmbiguousFunctionError:
		// PostgREST's AmbiguousRpc (PGRST203): 300 Multiple Choices, with no
		// choice to follow
		status = http.StatusMultipleChoices
		writeErrorBody(w, status, SmoothError{Code: "PGRST203", Message: err.Error(), Hint: e.Hint})
		return status, err
	case *database.FunctionNotFoundError:
		// PostgREST's NoRpc (PGRST202)
		status = http.StatusNotFound
		writeErrorBody(w, status, SmoothError{Code: "PGRST202", Message: err.Error()})
		return status, err
	case *database.CardinalityError:
		// PostgREST's SingularityError (PGRST116)
		status = http.StatusNotAcceptable
		writeErrorBody(w, status, SmoothError{
			Code:    "PGRST116",
			Message: "Cannot coerce the result to a single JSON object",
			Details: "The result contains " + strconv.FormatInt(e.Rows, 10) + " rows",
		})
		return status, err
	case *database.ContentTypeError:
		// PostgREST's MediaTypeError (PGRST107)
		status = http.StatusNotAcceptable
		writeErrorBody(w, status, SmoothError{Code: "PGRST107", Message: err.Error()})
		return status, err
	case *database.SerializeError:
		// A value that the serializer cannot decode is the server's fault,
		// not a failed content negotiation. PostgREST builds the body in SQL
		// and has no such error.
		status = http.StatusInternalServerError
		msg := "internal server error"
		if verboseErrors {
			msg = err.Error()
		}
		writeErrorBody(w, status, SmoothError{Message: msg})
		return status, err
	case *database.PutError:
		// PostgREST's PGRST105, PGRST114 and PGRST115 of a PUT
		status = e.Status
		writeErrorBody(w, status, SmoothError{Code: e.Code, Message: e.Message})
		return status, err
	case *database.RangeError:
		status = http.StatusRequestedRangeNotSatisfiable
		writeErrorBody(w, status, SmoothError{Code: "PGRST103", Message: err.Error()})
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
			setContentProfile(w, options)
			_, err = writeZeroRecords(w, options.ContentType, status)
		} else {
			status = http.StatusNoContent
			heligo.WriteHeader(w, status)
		}
		return false, status, err
	}
	return true, 0, err
}

// onlyJSONSpace reports whether content holds only JSON whitespace.
func onlyJSONSpace(content []byte) bool {
	for _, b := range content {
		switch b {
		case ' ', '\t', '\r', '\n':
		default:
			return false
		}
	}
	return true
}

// firstNonSpace returns the first byte of a JSON text after its whitespace,
// and 0 for a text that is only whitespace.
func firstNonSpace(content []byte) byte {
	for _, b := range content {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		}
		return b
	}
	return 0
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

// writeBodyError writes an error of the reading of a body: PGRST102 for a
// body that is not valid, 400 or 413 for the others (see WriteBadRequest).
func writeBodyError(w http.ResponseWriter, err error) (int, error) {
	if rerr, ok := err.(*database.RequestError); ok {
		return writeRequestError(w, rerr)
	}
	return WriteBadRequest(w, err)
}

// writeUnacceptableBody writes PostgREST's answer to a body that it has no
// reader for: an invalid body (PGRST102) that names the media type in full
// (Payload.hs, getPayload), not 415.
func writeUnacceptableBody(w http.ResponseWriter, r heligo.Request) (int, error) {
	return writeRequestError(w, database.InvalidBody("Content-Type not acceptable: "+r.Header.Get("Content-Type")))
}

// jsonBodyError is PostgREST's InvalidBody (PGRST102) for a JSON body that
// does not decode to records: a body that is not JSON, or JSON that is not
// an object or an array of objects (Payload.hs, getPayload).
func jsonBodyError(body []byte) error {
	if json.Valid(body) {
		return database.InvalidBody("All object keys must match")
	}
	return database.InvalidBody("Empty or invalid json")
}

// decodeJSONRecords decodes a JSON body as PostgREST's getPayload reads it:
// an object or an array of objects. A scalar, null included, is no record
// (Payload.hs, payloadAttributes). A body that is not one JSON value is
// PGRST102. Numbers stay json.Number, so that large integers keep their
// digits.
func decodeJSONRecords(body []byte) ([]database.Record, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var records []database.Record
	switch firstNonSpace(body) {
	case '[':
		if err := decoder.Decode(&records); err != nil {
			return nil, jsonBodyError(body)
		}
	case '{':
		var record database.Record
		if err := decoder.Decode(&record); err != nil {
			return nil, jsonBodyError(body)
		}
		records = append(records, record)
	default:
		if !json.Valid(body) {
			return nil, database.InvalidBody("Empty or invalid json")
		}
		return nil, nil
	}
	// The body is one JSON value: text after it makes the body invalid,
	// as PostgREST decodes the whole body.
	if !onlyJSONSpace(body[decoder.InputOffset():]) {
		return nil, database.InvalidBody("Empty or invalid json")
	}
	return records, nil
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
		if records, err = decodeJSONRecords(body); err != nil {
			return nil, nil, err
		}
		if c := firstNonSpace(body); c != '[' && c != '{' {
			if r.URL.Query().Has("columns") {
				// with ?columns PostgREST sends the body as it is, and
				// PostgreSQL refuses a scalar: one empty record keeps the
				// request going to the database with the body
				return []database.Record{nil}, body, nil
			}
			// PostgREST reads a scalar body, null included, as an empty
			// array (Payload.hs, payloadAttributes)
			return nil, []byte("[]"), nil
		}

	case "text/csv":
		reader := csv.NewReader(r.Body)
		csvData, err := reader.ReadAll()
		if errors.Is(err, csv.ErrFieldCount) {
			return nil, nil, database.InvalidBody("All lines must have same number of fields")
		}
		var parseErr *csv.ParseError
		if errors.As(err, &parseErr) {
			return nil, nil, database.InvalidBody(err.Error())
		}
		if err != nil {
			return nil, nil, err
		}

		if len(csvData) == 0 {
			// no header: PostgREST answers 400 (PGRST102) with the parser's message
			return nil, nil, database.InvalidBody(`parse error (not enough input) at ""`)
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
		// PostgREST answers a body it has no reader for as an invalid body,
		// with the media type in full (Payload.hs, getPayload), not with 415
		status, err = writeUnacceptableBody(w, r)
		return
	} else {
		// read input records
		records, jsonBody, err = readInputRecords(r, ctype)
		if err != nil {
			status, err = writeBodyError(w, err)
			return
		}
	}
	// check preconditions
	// A POST of no rows ([]) runs its insert all the same, as in PostgREST:
	// the table must exist, a single object is refused (406), a merge
	// answers 200 and an insert 201.
	if r.Method == "PATCH" {
		if len(records) > 1 {
			// A deliberate divergence: PostgREST hands the whole array to
			// json_to_recordset and updates with whichever row PostgreSQL
			// picks; a body that cannot mean one update is refused instead.
			status, err = WriteBadRequest(w, errors.New("a PATCH body must be a single object"))
			return
		}
		// With ?columns PostgREST sends a scalar body to json_to_recordset,
		// and PostgreSQL refuses it. smoothdb builds the update from the
		// record, so it answers the error of PostgreSQL itself.
		if jsonBody != nil && r.URL.Query().Has("columns") {
			if c := firstNonSpace(jsonBody); c != '[' && c != '{' {
				status, err = WriteServerError(w, &pgconn.PgError{Severity: "ERROR", Code: "22023",
					Message: "cannot call json_to_recordset on a scalar"})
				return
			}
		}
		// {}, [] and [{}] update nothing, but run all the same: the
		// relation must exist, and a single object is refused (406)
	}
	return records, jsonBody, status, err
}

// SetResponseHeaders sets the response headers (Content-Location,
// Content-Range and Preference-Applied). A read (GET, HEAD) goes to
// SetReadResponseHeaders. The Content-Range of a write follows PostgREST
// (Response.hs, actionResponse): an update reports the rows it wrote,
// 0-(count-1); an insert and a delete report no range. The total is * or,
// with Prefer: count, the count. It returns a status != 0 if the response
// must take it instead of the usual one (416, 206).
func SetResponseHeaders(ctx context.Context, w http.ResponseWriter, r heligo.Request, count int64) int {
	if r.Method == "GET" || r.Method == "HEAD" {
		return SetReadResponseHeaders(ctx, w, r, count)
	}
	sc := database.GetSmoothContext(ctx)
	options := &sc.QueryOptions
	// no Content-Location: PostgREST sends it on reads only (Response.hs)
	if r.Method == "PUT" {
		// a PUT has no Content-Range (Response.hs, MutationSingleUpsert)
		setPreferenceApplied(w, r, options)
		return 0
	}
	rangeString := "*"
	if r.Method == "PATCH" && count > 0 {
		rangeString = "0-" + strconv.FormatInt(count-1, 10)
	}
	if options.Count == "" {
		rangeString += "/*"
	} else {
		rangeString += "/" + strconv.FormatInt(count, 10)
	}
	w.Header().Set("Content-Range", rangeString)
	setPreferenceApplied(w, r, options)
	return 0
}

// setPreferenceApplied sets the Preference-Applied header: the preferences
// of the request that smoothdb honours, in PostgREST's order (Preferences.hs,
// prefAppliedHeader). The resolution applies to an insert with a conflict
// target, the return preference to a table write, the transaction
// preference when Database.TransactionMode allows an override.
func setPreferenceApplied(w http.ResponseWriter, r heligo.Request, options *database.QueryOptions) {
	var applied []string
	if r.Method == "POST" && options.ResolutionApplied {
		if options.IgnoreDuplicates {
			applied = append(applied, "resolution=ignore-duplicates")
		} else {
			applied = append(applied, "resolution=merge-duplicates")
		}
	}
	if options.Return != "" && isTableWrite(r) {
		applied = append(applied, "return="+options.Return)
	}
	// a write lists the count preference it was sent, as PostgREST, also
	// count=planned that leaves the total unknown; a read lists the count it
	// honours
	if isTableWrite(r) && options.CountPreference != "" {
		applied = append(applied, "count="+options.CountPreference)
	} else if options.Count != "" {
		applied = append(applied, "count="+options.Count)
	}
	if database.TxOverride() {
		if options.TxCommit {
			applied = append(applied, "tx=commit")
		} else if options.TxRollback {
			applied = append(applied, "tx=rollback")
		}
	}
	if len(applied) > 0 {
		w.Header().Set("Preference-Applied", strings.Join(applied, ", "))
	}
}

// isTableWrite says whether the request writes a table: a write method on a
// path other than /rpc/.
func isTableWrite(r heligo.Request) bool {
	switch r.Method {
	case "POST", "PATCH", "DELETE", "PUT":
		return r.Param("fname") == ""
	}
	return false
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
	setPreferenceApplied(w, r, options)
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
	setContentProfile(w, options)
	return writeErrorBody(w, status, SmoothError{
		Message: "Requested range not satisfiable",
		Code:    "PGRST103",
		Details: fmt.Sprintf("An offset of %d was requested, but there are only %d rows.",
			max(options.RangeMin, 0), options.Total),
	})
}

// setContentProfile names the negotiated schema of a response with a body,
// as PostgREST does (Response.hs, contentTypeHeaders).
func setContentProfile(w http.ResponseWriter, options *database.QueryOptions) {
	if options.SchemaNegotiated {
		w.Header().Set("Content-Profile", options.Schema)
	}
}

// WriteContent writes the response and its content type
func WriteContent(ctx context.Context, w http.ResponseWriter, status int, content []byte) (int, error) {
	sc := database.GetSmoothContext(ctx)
	ct := sc.QueryOptions.ContentType
	if ct == "application/json" || ct == "text/csv" {
		ct += "; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	setContentProfile(w, &sc.QueryOptions)
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
