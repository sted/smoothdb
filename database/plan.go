package database

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

// The plan of a request: Accept: application/vnd.pgrst.plan[+json|+text],
// with the parameters for (the media type the request would answer) and
// options (analyze|verbose|settings|buffers|wal), as PostgREST's MTVndPlan
// (MediaType.hs). Config.PlanEnabled allows it. The request answers the
// EXPLAIN of its main statement instead of its rows. smoothdb counts in a
// statement of its own: the plan does not show the count.

const planMediaType = "application/vnd.pgrst.plan"

// planOptions are the options of a plan, in the order PostgREST names them
// (toMimePlanOption) and gives them to EXPLAIN (explainF).
var planOptions = []string{"analyze", "verbose", "settings", "buffers", "wal"}

// PlanRequest is a plan that a request asks for.
type PlanRequest struct {
	// Format is "json" or "text".
	Format string
	// For is the media type of the explained statement, as PostgREST
	// names it in the Content-Type of the plan.
	For string
	// Options are the known options, in the order of planOptions.
	Options []string
}

// isPlanMediaType says whether mt (lower case, no parameters) names a plan.
func isPlanMediaType(mt string) bool {
	return mt == planMediaType || mt == planMediaType+"+json" || mt == planMediaType+"+text"
}

// parsePlan reads the plan that the media type mt (see isPlanMediaType)
// with the parameters params asks for. Text is the default format,
// application/json the default for, and an unknown option is ignored, as
// in PostgREST's decodeMediaType.
func parsePlan(mt string, params map[string]string) *PlanRequest {
	p := &PlanRequest{Format: "text", For: "application/json"}
	if mt == planMediaType+"+json" {
		p.Format = "json"
	}
	if f, ok := params["for"]; ok {
		p.For = canonicalPlanFor(f)
	}
	if o, ok := params["options"]; ok {
		given := strings.Split(o, "|")
		for _, opt := range planOptions {
			for _, g := range given {
				if g == opt {
					p.Options = append(p.Options, opt)
					break
				}
			}
		}
	}
	return p
}

// canonicalPlanFor is the name of the media type f in the Content-Type of a
// plan: the short name of the single object takes its +json, as PostgREST's
// toMime.
func canonicalPlanFor(f string) string {
	f = strings.ToLower(strings.TrimSpace(f))
	if f == "application/vnd.pgrst.object" {
		return "application/vnd.pgrst.object+json"
	}
	return f
}

// mediaType is the name of the plan's media type, as PostgREST's toMime:
// the Content-Type of the answer, and the name in PGRST107.
func (p *PlanRequest) mediaType() string {
	s := planMediaType + "+" + p.Format + `; for="` + p.For + `"`
	if len(p.Options) > 0 {
		s += "; options=" + strings.Join(p.Options, "|")
	}
	return s
}

// ContentType is the Content-Type of the answer of the plan.
func (p *PlanRequest) ContentType() string {
	return p.mediaType() + "; charset=utf-8"
}

// planOutputType is the content type a plan explains the statement of, or
// "" when smoothdb does not serve the media type of its for: then the plan
// cannot be served either, as in PostgREST (Negotiate.hs, lookupHandler).
func planOutputType(p *PlanRequest) string {
	switch p.For {
	case "*/*":
		return defaultOutputContentType
	}
	for _, t := range supportedOutputContentTypes {
		if p.For == t {
			return t
		}
	}
	return ""
}

// explain is the EXPLAIN that prefixes the statement of the plan, as
// PostgREST's explainF.
func (p *PlanRequest) explain() string {
	var b strings.Builder
	b.WriteString("EXPLAIN (FORMAT ")
	b.WriteString(strings.ToUpper(p.Format))
	for _, o := range p.Options {
		b.WriteString(", ")
		b.WriteString(strings.ToUpper(o))
	}
	b.WriteString(") ")
	return b.String()
}

// planSerializer writes the rows of an EXPLAIN: each line followed by a
// newline, as PostgREST's planRow (the text format has a row per line, the
// JSON format one row).
type planSerializer struct{}

func (planSerializer) Serialize(rows pgx.Rows, _ bool, _ bool, _ *SchemaInfo) ([]byte, int64, error) {
	defer rows.Close()
	var out []byte
	var n int64
	for rows.Next() {
		raw := rows.RawValues()
		if len(raw) > 0 {
			out = append(out, raw[0]...)
		}
		out = append(out, '\n')
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, n, nil
}

// runPlan answers the plan of the statement query of a request that asks
// for one (see QueryOptions.Plan), instead of running it. With the analyze
// option PostgreSQL runs the statement: a write ends as the transaction of
// the request ends (Prefer: tx=rollback rolls it back where the
// TransactionMode allows it). The statements that come with the query (the
// count, the count of the inserted rows) do not run.
func runPlan(ctx context.Context, query string, values []any) ([]byte, int64, error) {
	options := GetQueryOptions(ctx)
	options.totalPrelude, options.totalQuery, options.totalValues = "", "", nil
	options.insertedQuery = ""
	options.locate = false
	return runReadRows(ctx, options.Plan.explain()+query, values, false, false)
}
