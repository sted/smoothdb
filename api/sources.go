package api

import (
	"context"
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

	api.Handle("GET", "", prepared(TableListHandler))
	api.Handle("GET", "/$info/:table", prepared(TableGetHandler))

	// RECORDS

	api.Handle("GET", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		json, count, err := database.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			json, err = database.JQTransformResponse(c, json)
		}
		if err == nil {
			return writeRead(c, w, r, json, count)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, body, status, err := readRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.CreateRecordsFromJSON(c, sourcename, records, body, payloadFilters(r))
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				// No representation requested (the default, `return=minimal`):
				// PostgREST answers 201 with no body and no Content-Type, so a
				// client can rely on "2xx write => body only if I asked for it".
				// We used to write the affected-row count here, which is both
				// non-standard and redundant — the count already travels in
				// Content-Range via SetResponseHeaders above.
				return heligo.WriteHeader(w, http.StatusCreated)
			} else {
				return WriteContent(c, w, http.StatusCreated, data)
			}
		} else {
			return WriteError(w, err)
		}
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
		data, count, err := database.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("DELETE", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		data, count, err := database.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	// FUNCTIONS

	api.Handle("GET", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		json, count, err := database.ExecFunction(c, fname, nil, r.URL.Query(), true)
		if err == nil {
			json, err = database.JQTransformResponse(c, json)
		}
		if err == nil {
			return writeRead(c, w, r, json, count)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		records, status, err := ReadRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.ExecFunction(c, fname, records[0], payloadFilters(r), false)
		if err == nil {
			data, err = database.JQTransformResponse(c, data)
		}
		if err == nil {
			if data == nil {
				SetReadResponseHeaders(c, w, r, count)
				return heligo.WriteJSON(w, http.StatusOK, count)
			} else {
				return writeRead(c, w, r, data, count)
			}
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
		return heligo.WriteJSON(w, http.StatusMethodNotAllowed,
			SmoothError{Subsystem: "network", Message: "Cannot use the " + r.Method + " method on RPC"})
	}
	api.Handle("DELETE", "/rpc/:fname", rpcMethodNotAllowed)
	api.Handle("PATCH", "/rpc/:fname", rpcMethodNotAllowed)
	api.Handle("PUT", "/rpc/:fname", rpcMethodNotAllowed)
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
