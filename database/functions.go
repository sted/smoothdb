package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Argument struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   byte   `json:"mode"`   // i IN, o OUT, b INOUT, v VARIADIC, t TABLE; JSON uses argumentModes
	TypeId uint32 `json:"typeid"` // OID of the argument type
}

// argumentModes maps the proargmodes letters to the words of the JSON API.
// Mode 0 (proargmodes NULL: every argument is IN) is "in".
var argumentModes = map[byte]string{
	0: "in", 'i': "in", 'o': "out", 'b': "inout", 'v': "variadic", 't': "table",
}

// argumentJSON is Argument with the mode as a word.
type argumentJSON struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	TypeId uint32 `json:"typeid"`
}

func (a Argument) MarshalJSON() ([]byte, error) {
	return json.Marshal(argumentJSON{a.Name, a.Type, argumentModes[a.Mode], a.TypeId})
}

func (a *Argument) UnmarshalJSON(data []byte) error {
	var aj argumentJSON
	if err := json.Unmarshal(data, &aj); err != nil {
		return err
	}
	mode := strings.ToLower(aj.Mode)
	a.Name, a.Type, a.TypeId, a.Mode = aj.Name, aj.Type, aj.TypeId, 0
	if mode == "" || mode == "in" {
		return nil
	}
	for letter, word := range argumentModes {
		if letter != 0 && word == mode {
			a.Mode = letter
			return nil
		}
	}
	return fmt.Errorf("invalid argument mode %q: use in, out, inout, variadic or table", aj.Mode)
}

type Function struct {
	Name         string     `json:"name"`
	Schema       string     `json:"schema"`
	Arguments    []Argument `json:"arguments"`
	ArgDefaults  int        `json:"argdefaults"` // pronargdefaults: how many of the last input arguments have a default
	Returns      string     `json:"returns"`
	Language     string     `json:"language"`
	Definition   string     `json:"definition"`
	ReturnTypeId uint32     `json:"rettypeid"`
	ReturnIsSet  bool       `json:"retisset"`
	ReturnRows   float32    `json:"retrows"`    // prorows: estimated number of result rows
	HasUnnamed   bool       `json:"hasunnamed"` // has unnamed parameters
	HasOut       bool       `json:"hasout"`     // has OUT, INOUT, TABLE parameters
	IsVariadic   bool       `json:"isvariadic"`
	Volatility   string     `json:"volatility"`        // provolatile: i IMMUTABLE, s STABLE, v VOLATILE; input also takes the words
	Strict       bool       `json:"strict"`            // proisstrict: RETURNS NULL ON NULL INPUT
	Security     string     `json:"security"`          // prosecdef: "invoker" or "definer"
	Replace      bool       `json:"replace,omitempty"` // input only: CREATE OR REPLACE
	Kind         string     `json:"-"`                 // prokind: f function, p procedure, a aggregate, w window
}

const functionsQuery = `
	SELECT
		proname name,
		n.nspname schema,
		ARRAY_AGG((COALESCE(_.name, ''), COALESCE(_.type::regtype::text, ''), COALESCE(_.mode, ''), COALESCE(_.type::int4, 0)) ORDER BY _.idx) args,
		prorettype::regtype rettype,
		l.lanname language,
		COALESCE(pg_catalog.pg_get_function_sqlbody(p.oid), p.prosrc) source,
		prorettype rettypeid,
		proretset retisset,
		prorows retrows,
		BOOL_OR(_.name is null) AND pronargs > 0 hasunnamed,
		COALESCE(proargmodes::text[] && '{t,b,o}', false) hasout,
		p.provariadic != 0 isvariadic,
		p.provolatile::text volatility,
		p.pronargdefaults argdefaults,
		p.proisstrict strict,
		CASE WHEN p.prosecdef THEN 'definer' ELSE 'invoker' END security,
		p.prokind::text kind
	FROM pg_proc p
	JOIN pg_namespace n ON n.oid = p.pronamespace
	LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
	LEFT JOIN UNNEST(proargnames, COALESCE(proallargtypes, proargtypes::oid[]), proargmodes) WITH ORDINALITY AS _(name, type, mode, idx) ON true
	WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'`

const functionsGroupBy = `
	GROUP BY p.oid, n.nspname, l.lanname`

// GetFunctions reads every routine for the schema cache. A function with no
// argument comes with one empty argument of type 0 (see inputArguments).
func GetFunctions(ctx context.Context) ([]Function, error) {
	return readFunctions(ctx, functionsQuery+functionsGroupBy)
}

// ListFunctions reads the functions for the /admin API: functions only
// (prokind 'f'), so procedures, aggregates and window functions are not
// listed. A function with no argument has an empty argument list.
func ListFunctions(ctx context.Context) ([]Function, error) {
	functions, err := readFunctions(ctx, functionsQuery+" AND p.prokind = 'f'"+functionsGroupBy+" ORDER BY 1, 2")
	if err != nil {
		return nil, err
	}
	for i := range functions {
		args := []Argument{}
		for _, a := range functions[i].Arguments {
			if a.TypeId != 0 {
				args = append(args, a)
			}
		}
		functions[i].Arguments = args
	}
	return functions, nil
}

func readFunctions(ctx context.Context, query string) ([]Function, error) {
	conn := GetConn(ctx)
	functions := []Function{}

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	f := Function{}
	for rows.Next() {
		err := rows.Scan(
			&f.Name,
			&f.Schema,
			&f.Arguments,
			&f.Returns,
			&f.Language,
			&f.Definition,
			&f.ReturnTypeId,
			&f.ReturnIsSet,
			&f.ReturnRows,
			&f.HasUnnamed,
			&f.HasOut,
			&f.IsVariadic,
			&f.Volatility,
			&f.ArgDefaults,
			&f.Strict,
			&f.Security,
			&f.Kind)
		if err != nil {
			return nil, err
		}
		functions = append(functions, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return functions, nil
}

// composeSignature writes the argument list of CREATE FUNCTION. The names
// are quoted identifiers; the types are type expressions and stay as given.
func composeSignature(fname string, args []Argument) (string, error) {
	sig := fname + "("
	for i, a := range args {
		if i != 0 {
			sig += ", "
		}
		switch a.Mode {
		case 0, 'i':
		case 'o':
			sig += "OUT "
		case 'b':
			sig += "INOUT "
		case 'v':
			sig += "VARIADIC "
		default:
			return "", &BuildError{"invalid argument mode for " + a.Name + ": use returns TABLE (...) for table columns"}
		}
		if a.Name != "" {
			sig += quote(a.Name) + " "
		}
		sig += a.Type
	}
	sig += ")"
	return sig, nil
}

// dollarTag returns a dollar-quote tag that does not occur in body: $fn$,
// then $fn1$, $fn2$, ...
func dollarTag(body string) string {
	tag := "$fn$"
	for i := 1; strings.Contains(body, tag); i++ {
		tag = "$fn" + strconv.Itoa(i) + "$"
	}
	return tag
}

var functionVolatilities = map[string]string{
	"immutable": "IMMUTABLE", "i": "IMMUTABLE",
	"stable": "STABLE", "s": "STABLE",
	"volatile": "VOLATILE", "v": "VOLATILE",
}

var functionSecurities = map[string]string{
	"invoker": "SECURITY INVOKER", "definer": "SECURITY DEFINER",
}

func buildCreateFunction(fname string, function *Function) (string, error) {
	create := "CREATE "
	if function.Replace {
		create += "OR REPLACE "
	}
	create += "FUNCTION "
	sig, err := composeSignature(fname, function.Arguments)
	if err != nil {
		return "", err
	}
	create += sig
	create += " RETURNS "
	if function.Returns != "" {
		create += function.Returns
	} else {
		create += "void"
	}
	create += " LANGUAGE "
	if function.Language != "" {
		create += quote(strings.ToLower(function.Language))
	} else {
		create += "sql"
	}
	if function.Volatility != "" {
		v, ok := functionVolatilities[strings.ToLower(function.Volatility)]
		if !ok {
			return "", &BuildError{"invalid volatility: " + function.Volatility + " (use immutable, stable or volatile)"}
		}
		create += " " + v
	}
	if function.Strict {
		create += " STRICT"
	}
	if function.Security != "" {
		s, ok := functionSecurities[strings.ToLower(function.Security)]
		if !ok {
			return "", &BuildError{"invalid security: " + function.Security + " (use invoker or definer)"}
		}
		create += " " + s
	}
	tag := dollarTag(function.Definition)
	create += " AS " + tag + " " + function.Definition + " " + tag
	return create, nil
}

func CreateFunction(ctx context.Context, function *Function) (*Function, error) {
	conn := GetConn(ctx)
	create, err := buildCreateFunction(composeName(ctx, function.Schema, function.Name), function)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, create); err != nil {
		return nil, err
	}
	return function, nil
}

// DeleteFunction drops the function name of the request's schema. The name
// must address one function: an overloaded name fails (42725).
func DeleteFunction(ctx context.Context, name string) error {
	conn, schemaname := GetConnAndSchema(ctx)
	_, err := conn.Exec(ctx, "DROP FUNCTION "+qualify(schemaname, name))
	return err
}

// DeleteFunctionSignature drops the function name with the input argument
// types types (empty: no argument). Each type is resolved by PostgreSQL and
// written back in its canonical form, so no client text reaches the DDL.
func DeleteFunctionSignature(ctx context.Context, name string, types []string) error {
	conn, schemaname := GetConnAndSchema(ctx)
	canonical := make([]string, len(types))
	for i, t := range types {
		var c *string
		if err := conn.QueryRow(ctx, "SELECT to_regtype($1)::text", strings.TrimSpace(t)).Scan(&c); err != nil {
			return err
		}
		if c == nil {
			return &BuildError{"unknown argument type: " + t}
		}
		canonical[i] = *c
	}
	_, err := conn.Exec(ctx, "DROP FUNCTION "+qualify(schemaname, name)+"("+strings.Join(canonical, ", ")+")")
	return err
}

func ExecFunction(ctx context.Context, fname string, record Record, filters Filters, readonly bool) ([]byte, int64, error) {
	return Execute(ctx, fname, record, nil, filters, readonly)
}

// RawBody is the body of a POST /rpc call that can go whole to a function
// with a single unnamed parameter (PostgREST's RawPay and its JSON
// fallback). Type is the parameter type the body can fill: "json" (also
// jsonb), "text", "xml" or "bytea".
type RawBody struct {
	Type string
	Data []byte
}

// QueryValues are the values of one argument that come from the query
// string or from a form: text, and repeated when the key is repeated. A
// VARIADIC parameter takes all of them as an array; any other parameter
// takes the last one, as in PostgREST (CallPlan.hs toRpcParams).
type QueryValues []string

// CallFunction calls the function fname with the arguments of a POST body:
// record, decoded from it, and raw, the body itself (nil when the content
// type cannot fill a single unnamed parameter).
func CallFunction(ctx context.Context, fname string, record Record, raw *RawBody, filters Filters) ([]byte, int64, error) {
	return Execute(ctx, fname, record, raw, filters, false)
}
