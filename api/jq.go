package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/jqeval"
)

// maxJQBatchEvals caps the number of evaluations in a single POST /jq call
const maxJQBatchEvals = 200

type jqEvalItem struct {
	Program string          `json:"program"`
	Input   json.RawMessage `json:"input"`
	Args    map[string]any  `json:"args"`
}

type jqBatchRequest struct {
	ParseOnly bool         `json:"parse_only"`
	Evals     []jqEvalItem `json:"evals"`
}

type jqOutputItem struct {
	Output json.RawMessage `json:"output"`
}

type jqErrorItem struct {
	Error string `json:"error"`
}

// InitJQRoute registers the standalone jq evaluation endpoint.
// It is called only when jq evaluation is enabled, so the route 404s otherwise.
func InitJQRoute(apiHelper Helper) {
	router := apiHelper.GetRouter()
	jq := router.Group("/jq", apiHelper.MiddlewareDBE())

	// POST /jq: batch evaluation of jq programs, or compile-only validation
	// with parse_only. Errors are reported per item; 400 is reserved for a
	// malformed envelope.
	jq.Handle("POST", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var req jqBatchRequest
		err := r.ReadJSON(&req)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		if len(req.Evals) > maxJQBatchEvals {
			return WriteBadRequest(w, fmt.Errorf("too many evals in a single call (max %d)", maxJQBatchEvals))
		}
		results := make([]any, 0, len(req.Evals))
		for _, item := range req.Evals {
			results = append(results, jqEvalOne(c, &item, req.ParseOnly))
		}
		return heligo.WriteJSON(w, http.StatusOK, results)
	})
}

// jqContentType is the media type for a raw jq program in a request body.
// There is no registered media type for jq, so we use the vendor tree,
// following the application/vnd.pgrst.* precedent.
const jqContentType = "application/vnd.smoothdb.jq"

// hasJQBody reports whether the request carries a raw jq program in its body
func hasJQBody(r heligo.Request) bool {
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && ct == jqContentType
}

// jqUpdateHandler handles a jq-update PATCH /{table}?{filters}: an atomic
// read-modify-write of the matched rows, driven by a jq program given either
// in the jq= query parameter (with an empty body) or as the raw request body
// with Content-Type: application/jq.
func jqUpdateHandler(c context.Context, w http.ResponseWriter, r heligo.Request, sourcename string) (int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return WriteBadRequest(w, err)
	}
	options := database.GetQueryOptions(c)
	if hasJQBody(r) {
		if options.JQ != "" {
			return WriteBadRequest(w, fmt.Errorf("the jq program must be given either in the jq= parameter or in the request body, not both"))
		}
		options.JQ = string(body)
	} else if len(bytes.TrimSpace(body)) != 0 {
		return WriteBadRequest(w, fmt.Errorf("a request body and the jq= parameter cannot be used together"))
	}
	data, count, err := database.UpdateRecordsWithJQ(c, sourcename, r.URL.Query())
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
}

func jqEvalOne(ctx context.Context, item *jqEvalItem, parseOnly bool) any {
	if parseOnly {
		if err := jqeval.Parse(item.Program, item.Args); err != nil {
			return jqErrorItem{err.Error()}
		}
		return struct{}{}
	}
	var input any
	if len(item.Input) != 0 {
		var err error
		input, err = jqeval.Unmarshal(item.Input)
		if err != nil {
			return jqErrorItem{err.Error()}
		}
	}
	output, err := jqeval.Eval(ctx, item.Program, input, item.Args)
	if err != nil {
		return jqErrorItem{err.Error()}
	}
	encoded, err := jqeval.Marshal(output)
	if err != nil {
		return jqErrorItem{err.Error()}
	}
	return jqOutputItem{encoded}
}
