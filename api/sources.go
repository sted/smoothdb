package api

import (
	"context"
	"io"
	"mime"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
)

func InitSourcesRouter(apiHelper Helper) {

	baseURL := apiHelper.BaseAPIURL()
	if !apiHelper.HasShortAPIURL() {
		baseURL += "/:dbname"
	}
	api := apiHelper.GetRouter().Group(baseURL, authn.DeferPreparation, apiHelper.MiddlewareStd())

	// TABLES

	api.Handle("GET", "", prepared(SchemaTableListHandler))
	api.Handle("GET", "/$info/:table", prepared(SchemaTableGetHandler))

	// RECORDS

	api.Handle("GET", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		json, count, err := database.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if ok, status, err := writePlan(c, w, json); ok {
				return status, err
			}
			json, err = database.JQTransformResponse(c, json)
		}
		if err == nil {
			return writeRead(c, w, r, json, count)
		} else {
			return WriteError(w, database.TableNotFound(c, sourcename, err))
		}
	})

	api.Handle("POST", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, body, status, err := readRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.CreateRecordsFromJSON(c, sourcename, records, body, payloadFilters(r))
		if err != nil {
			return WriteError(w, database.TableNotFound(c, sourcename, err))
		}
		if ok, status, err := writePlan(c, w, data); ok {
			return status, err
		}
		SetResponseHeaders(c, w, r, count)
		options := database.GetQueryOptions(c)
		// A merge that created no row answers 200, as in PostgREST
		// (Response.hs, isInsertIfGTZero).
		status = http.StatusCreated
		if options.MergeDuplicates && options.Inserted == 0 {
			status = http.StatusOK
		}
		switch {
		case options.Return == "headers-only":
			// the created row, when it is one row of a table with a
			// primary key; no body
			if len(data) > 0 {
				w.Header().Set("Location", "/"+sourcename+"?"+string(data))
			}
			return heligo.WriteHeader(w, status)
		case data == nil:
			// No representation requested (the default, `return=minimal`):
			// PostgREST answers with no body and no Content-Type, so a
			// client can rely on "2xx write => body only if I asked for it".
			// The count travels in Content-Range (SetResponseHeaders).
			return heligo.WriteHeader(w, status)
		default:
			return WriteContent(c, w, status, data)
		}
	})

	api.Handle("PUT", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, body, status, err := readRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.UpsertRecordsFromJSON(c, sourcename, records, body, payloadFilters(r))
		if err != nil {
			return WriteError(w, database.TableNotFound(c, sourcename, err))
		}
		if ok, status, err := writePlan(c, w, data); ok {
			return status, err
		}
		SetResponseHeaders(c, w, r, count)
		if data == nil {
			return heligo.WriteHeader(w, http.StatusNoContent)
		}
		// 201 when the row is new, 200 when it replaced one (Response.hs,
		// MutationSingleUpsert)
		status = http.StatusOK
		if database.GetQueryOptions(c).Inserted > 0 {
			status = http.StatusCreated
		}
		return WriteContent(c, w, status, data)
	})

	api.Handle("PATCH", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		if database.GetQueryOptions(c).JQ != "" || hasJQBody(r) {
			return jqUpdateHandler(c, w, r, sourcename)
		}
		records, status, err := ReadRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		record := database.Record{}
		if len(records) > 0 {
			record = records[0]
		}
		data, count, err := database.UpdateRecords(c, sourcename, record, r.URL.Query())
		if err == nil {
			if ok, status, err := writePlan(c, w, data); ok {
				return status, err
			}
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, database.TableNotFound(c, sourcename, err))
		}
	})

	api.Handle("DELETE", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		data, count, err := database.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if ok, status, err := writePlan(c, w, data); ok {
				return status, err
			}
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, database.TableNotFound(c, sourcename, err))
		}
	})

	// FUNCTIONS

	api.Handle("GET", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		json, count, err := database.ExecFunction(c, fname, nil, r.URL.Query(), true)
		if err == nil {
			if ok, status, err := writePlan(c, w, json); ok {
				return status, err
			}
			json, err = database.JQTransformResponse(c, json)
		}
		if err == nil {
			return writeCallResult(c, w, r, json, count)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		record, raw, status, err := readCallRequest(w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.CallFunction(c, fname, record, raw, payloadFilters(r))
		if err == nil {
			if ok, status, err := writePlan(c, w, data); ok {
				return status, err
			}
			data, err = database.JQTransformResponse(c, data)
		}
		if err == nil {
			return writeCallResult(c, w, r, data, count)
		} else {
			return WriteError(w, err)
		}
	})

	// Any other method on a function is refused, as PostgREST does before
	// touching the database (ApiRequest.hs getAction: InvalidRpcMethod,
	// PGRST101); without these routes the request fell through to a 404.
	// The Allow header lists what is routed on /rpc/ (no OPTIONS yet).
	rpcMethodNotAllowed := func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		w.Header().Set("Allow", "GET, HEAD, POST")
		return writeErrorBody(w, http.StatusMethodNotAllowed,
			SmoothError{Code: "PGRST101", Message: "Cannot use the " + r.Method + " method on RPC"})
	}
	api.Handle("DELETE", "/rpc/:fname", rpcMethodNotAllowed)
	api.Handle("PATCH", "/rpc/:fname", rpcMethodNotAllowed)
	api.Handle("PUT", "/rpc/:fname", rpcMethodNotAllowed)
}

// writeCallResult answers a function call. A function that returns void
// (nil data) answers 204 with no body and no Content-Type, as PostgREST
// (Response.hs: funcReturnsVoid).
func writeCallResult(c context.Context, w http.ResponseWriter, r heligo.Request, data []byte, count int64) (int, error) {
	if data == nil {
		SetReadResponseHeaders(c, w, r, count)
		return heligo.WriteHeader(w, http.StatusNoContent)
	}
	return writeRead(c, w, r, data, count)
}

// rawBodyTypes maps the content types whose body can go whole to a single
// unnamed parameter to the type of that parameter (PostgREST's RawPay).
var rawBodyTypes = map[string]string{
	"text/plain":               "text",
	"text/xml":                 "xml",
	"application/octet-stream": "bytea",
}

// readCallRequest reads the body of a POST /rpc call, as PostgREST's
// getPayload for a routine. It returns the arguments decoded from the body
// and the body itself, for a function with a single unnamed parameter (see
// database.RawBody). No body skips the call, as an empty insert does: the
// function runs whatever the body.
//   - A text/plain, text/xml or octet-stream body is only raw.
//   - An empty JSON body, or an empty array, passes no argument. An unnamed
//     json parameter takes the body itself: the empty text fails (22P02),
//     as in PostgREST.
//   - A form passes all the values of a repeated field (database.QueryValues).
func readCallRequest(w http.ResponseWriter, r heligo.Request) (database.Record, *database.RawBody, int, error) {
	ctype := "application/json"
	if header := r.Header.Get("Content-Type"); header != "" {
		var err error
		if ctype, _, err = mime.ParseMediaType(header); err != nil {
			status, err := writeUnacceptableBody(w, r)
			return nil, nil, status, err
		}
	}
	if typ, ok := rawBodyTypes[ctype]; ok {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			status, err := WriteBadRequest(w, err)
			return nil, nil, status, err
		}
		return nil, &database.RawBody{Type: typ, Data: body}, 0, nil
	}
	if ctype == "application/json" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			status, err := WriteBadRequest(w, err)
			return nil, nil, status, err
		}
		raw := &database.RawBody{Type: "json", Data: body}
		if len(body) == 0 {
			return database.Record{}, raw, 0, nil
		}
		records, err := decodeJSONRecords(body)
		if err != nil {
			status, err := writeBodyError(w, err)
			return nil, nil, status, err
		}
		if len(records) == 0 {
			return database.Record{}, raw, 0, nil
		}
		return records[0], raw, 0, nil
	}
	if getContentType(r) == "" {
		status, err := writeUnacceptableBody(w, r)
		return nil, nil, status, err
	}
	records, _, err := readInputRecords(r, ctype)
	if err != nil {
		status, err := writeBodyError(w, err)
		return nil, nil, status, err
	}
	if len(records) == 0 {
		// a CSV body with a header and no row
		return database.Record{}, nil, 0, nil
	}
	record := records[0]
	if ctype == "application/x-www-form-urlencoded" {
		for key, values := range r.PostForm {
			record[key] = database.QueryValues(values)
		}
	}
	return record, nil, 0, nil
}

// writeRead answers a read of rows: with 206 when the page is shorter than
// its total, and with the 416 error instead of the rows when the offset is
// past it (see SetReadResponseHeaders).
func writeRead(c context.Context, w http.ResponseWriter, r heligo.Request, content []byte, rows int64) (int, error) {
	switch status := SetReadResponseHeaders(c, w, r, rows); status {
	case http.StatusRequestedRangeNotSatisfiable:
		return WriteRangeNotSatisfiable(c, w)
	case 0:
		return WriteContent(c, w, http.StatusOK, content)
	default:
		return WriteContent(c, w, status, content)
	}
}

// prepared applies the request's role and claims before h, which reaches the
// database outside the query executor (see authn.DeferPreparation).
func prepared(h heligo.Handler) heligo.Handler {
	return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		if err := database.PrepareNow(c); err != nil {
			return WriteError(w, err)
		}
		return h(c, w, r)
	}
}
