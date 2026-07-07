package database

import (
	"bytes"
	"context"
	"sort"
	"strconv"

	"github.com/sted/smoothdb/jqeval"
)

// jqParseArgs decodes the raw jq_args query parameter into an args map.
// The same size cap used for programs applies to the raw string.
func jqParseArgs(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, nil
	}
	if len(raw) > jqeval.MaxProgramBytes() {
		return nil, &ParseError{"jq_args exceeds the maximum allowed size (" +
			strconv.Itoa(jqeval.MaxProgramBytes()) + " bytes)"}
	}
	v, err := jqeval.Unmarshal([]byte(raw))
	if err != nil {
		return nil, &ParseError{"invalid jq_args: " + err.Error()}
	}
	args, ok := v.(map[string]any)
	if !ok {
		return nil, &ParseError{"jq_args must be a JSON object"}
	}
	return args, nil
}

// JQTransformResponse applies the jq program carried by the jq= query
// parameter (see QueryOptions) to a fully buffered JSON response body,
// binding jq_args as variables. It returns the body unchanged when no
// program is present. Only JSON content types can be transformed.
func JQTransformResponse(ctx context.Context, data []byte) ([]byte, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if options.JQ == "" || data == nil {
		return data, nil
	}
	if !jqeval.Enabled() {
		return nil, &ParseError{"jq evaluation is disabled (see the JQ configuration section)"}
	}
	if options.ContentType != "application/json" &&
		options.ContentType != "application/vnd.pgrst.object+json" {
		return nil, &ParseError{"jq transforms require a JSON content type"}
	}
	args, err := jqParseArgs(options.JQArgs)
	if err != nil {
		return nil, err
	}
	input, err := jqeval.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	output, err := jqeval.Eval(ctx, options.JQ, input, args)
	if err != nil {
		return nil, err
	}
	return jqeval.Marshal(output)
}

// UpdateRecordsWithJQ performs a jq-update: an atomic read-modify-write of the
// rows matched by the filters, driven by the jq program in the jq= query
// parameter (carried on QueryOptions).
//
// Inside the request transaction (one is started if not already open):
//
//  1. The matched rows are selected FOR UPDATE, capped at MaxUpdateRows+1
//     (exceeding the cap is an error).
//  2. Each row, as a JSON object with the columns visible to the role, is fed
//     to the program. The output must be a JSON object whose keys are columns
//     of the table: it becomes that row's UPDATE ... SET. An empty object
//     leaves the row untouched.
//  3. Any parse/eval error, non-object output or unknown column aborts the
//     whole request: all rows or none.
//
// RLS and triggers apply unchanged. With Prefer: return=representation it
// returns the resulting rows (all visible columns; select= is not applied).
func UpdateRecordsWithJQ(ctx context.Context, table string, filters Filters) ([]byte, int64, error) {
	gi := GetSmoothContext(ctx)
	options := &gi.QueryOptions
	if !jqeval.Enabled() {
		return nil, 0, &ParseError{"jq evaluation is disabled (see the JQ configuration section)"}
	}
	program := options.JQ
	args, err := jqParseArgs(options.JQArgs)
	if err != nil {
		return nil, 0, err
	}
	// compile upfront: fail before touching any row
	if err := jqeval.Parse(program, args); err != nil {
		return nil, 0, err
	}
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, 0, err
	}
	if parts.recursive != nil {
		return nil, 0, &ParseError{"recursive filters are not supported in a jq update"}
	}
	schema := options.Schema
	stack := BuildStack{info: gi.Db.info}
	where, values := whereClause(table, schema, "", parts.whereConditionsTree, 0, stack)

	maxRows := jqeval.MaxUpdateRows()
	query := "SELECT " + quote(table) + ".ctid::text, to_jsonb(" + quote(table) + ") FROM " + _sq(table, schema)
	if where != "" {
		query += " WHERE " + where
	}
	query += " LIMIT " + strconv.Itoa(maxRows+1) + " FOR UPDATE"

	// jq-update is all-or-nothing: make sure it runs in a transaction even
	// when TransactionMode is "none". When a request transaction is already
	// open we join it; an error will roll it back on release (http error).
	ownTx := gi.Conn.PgConn().TxStatus() == 'I'
	if ownTx {
		if _, err = gi.Conn.Exec(ctx, "BEGIN"); err != nil {
			return nil, 0, err
		}
		defer func() {
			if err != nil {
				gi.Conn.Exec(ctx, "ROLLBACK")
			}
		}()
	}

	ctids, inputs, err := jqSelectRows(ctx, query, values, maxRows)
	if err != nil {
		return nil, 0, err
	}

	var representation [][]byte
	for i, ctid := range ctids {
		var rowJSON []byte
		rowJSON, err = jqUpdateRow(ctx, table, schema, program, args, ctid, inputs[i], options.ReturnRepresentation)
		if err != nil {
			return nil, 0, err
		}
		if options.ReturnRepresentation {
			representation = append(representation, rowJSON)
		}
	}

	if ownTx {
		if _, err = gi.Conn.Exec(ctx, "COMMIT"); err != nil {
			return nil, 0, err
		}
	}

	count := int64(len(ctids))
	if !options.ReturnRepresentation {
		return nil, count, nil
	}
	return jqAssembleRows(representation, options)
}

// jqSelectRows runs the FOR UPDATE select and collects ctids and row JSON
// (the result set must be fully read before issuing updates on the same
// connection)
func jqSelectRows(ctx context.Context, query string, values []any, maxRows int) ([]string, [][]byte, error) {
	gi := GetSmoothContext(ctx)
	rows, err := gi.Conn.Query(ctx, query, values...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ctids []string
	var inputs [][]byte
	for rows.Next() {
		var ctid string
		var rowJSON []byte
		if err := rows.Scan(&ctid, &rowJSON); err != nil {
			return nil, nil, err
		}
		ctids = append(ctids, ctid)
		inputs = append(inputs, rowJSON)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(ctids) > maxRows {
		return nil, nil, &ParseError{"jq update would affect more than " +
			strconv.Itoa(maxRows) + " rows (JQ.MaxUpdateRows)"}
	}
	return ctids, inputs, nil
}

// jqUpdateRow evaluates the program on one row and applies the resulting
// UPDATE. It returns the row representation (post-update when the row
// changed) if requested.
func jqUpdateRow(ctx context.Context, table, schema, program string, args map[string]any,
	ctid string, rowJSON []byte, representation bool) ([]byte, error) {

	gi := GetSmoothContext(ctx)
	input, err := jqeval.Unmarshal(rowJSON)
	if err != nil {
		return nil, err
	}
	output, err := jqeval.Eval(ctx, program, input, args)
	if err != nil {
		return nil, err
	}
	obj, ok := output.(map[string]any)
	if !ok {
		return nil, &ParseError{"jq update program must produce a JSON object of columns to update"}
	}
	if len(obj) == 0 {
		// nothing to change for this row
		return rowJSON, nil
	}
	// The row input carries every visible column: unknown output keys are
	// caught here, non-updatable columns are rejected by Postgres below
	inputColumns := input.(map[string]any)
	columns := make([]string, 0, len(obj))
	for k := range obj {
		if _, exists := inputColumns[k]; !exists {
			return nil, &ParseError{"jq update output contains an unknown column: '" + k + "'"}
		}
		columns = append(columns, k)
	}
	sort.Strings(columns)
	var columnList string
	for i, c := range columns {
		if i != 0 {
			columnList += ", "
		}
		columnList += quote(c)
	}
	setJSON, err := jqeval.Marshal(obj)
	if err != nil {
		return nil, err
	}
	// jsonb_populate_record converts the output values to the proper column
	// types (including arrays and composites)
	update := "UPDATE " + _sq(table, schema) +
		" SET (" + columnList + ") = (SELECT " + columnList +
		" FROM jsonb_populate_record(NULL::" + _sq(table, schema) + ", $1)) WHERE ctid = $2::tid"
	if !representation {
		_, err = gi.Conn.Exec(ctx, update, setJSON, ctid)
		return nil, err
	}
	update += " RETURNING to_jsonb(" + quote(table) + ")"
	var updated []byte
	err = gi.Conn.QueryRow(ctx, update, setJSON, ctid).Scan(&updated)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// jqAssembleRows builds the response body from the per-row representations
func jqAssembleRows(rows [][]byte, options *QueryOptions) ([]byte, int64, error) {
	count := int64(len(rows))
	if options.Singular {
		if count != 1 {
			return nil, count, &SerializeError{"JSON object requested, multiple (or no) rows returned"}
		}
		return rows[0], count, nil
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, r := range rows {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.Write(r)
	}
	buf.WriteByte(']')
	return buf.Bytes(), count, nil
}
