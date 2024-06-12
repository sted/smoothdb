package database

import "context"

type Argument struct {
	Name string
	Type string
	Mode byte // i IN, o OUT, b INOUT, v VARIADIC, t TABLE
	//Default string
}

type Function struct {
	Name         string     `json:"name"`
	Schema       string     `json:"schema"`
	Arguments    []Argument `json:"arguments"`
	Returns      string     `json:"returns"`
	Language     string     `json:"language"`
	Definition   string     `json:"definition"`
	ReturnTypeId uint32     `json:"rettypeid"`
	ReturnIsSet  bool       `json:"retisset"`
	HasUnnamed   bool       `json:"hasunnamed"` // has unnamed parameters
	HasOut       bool       `json:"hasout"`     // has OUT, INOUT, TABLE parameters
	IsVariadic   bool       `json:"isvariadic"`
}

const functionsQuery = `
	SELECT
		proname name,
		n.nspname schema,
		ARRAY_AGG((COALESCE(_.name, ''), COALESCE(_.type::regtype::text, ''), COALESCE(_.mode, '')) ORDER BY _.idx) args,
		prorettype::regtype rettype,
		l.lanname language,
		COALESCE(pg_catalog.pg_get_function_sqlbody(p.oid), p.prosrc) source,
		prorettype rettypeid,
		proretset retisset,
		BOOL_OR(_.name is null) AND pronargs > 0 hasunnamed,
		COALESCE(proargmodes::text[] && '{t,b,o}', false) hasout,
		p.provariadic != 0 isvariadic
	FROM pg_proc p
	JOIN pg_namespace n ON n.oid = p.pronamespace
	LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
	LEFT JOIN UNNEST(proargnames, proargtypes, proargmodes) WITH ORDINALITY AS _(name, type, mode, idx) ON true
	WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
	GROUP BY p.oid, n.nspname, l.lanname;`

func GetFunctions(ctx context.Context) ([]Function, error) {
	conn := GetConn(ctx)
	functions := []Function{}
	query := functionsQuery

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
			&f.HasUnnamed,
			&f.HasOut,
			&f.IsVariadic)
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

func composeSignature(fname string, args []Argument) string {
	sig := quoteParts(fname) + "("
	for i, a := range args {
		if i != 0 {
			sig += ", "
		}
		switch a.Mode {
		case 'o':
			sig += "OUT "
		case 'b':
			sig += "INOUT "
		}
		sig += a.Name + " " + a.Type
		// if a.Default != "" {
		// 	sig += " DEFAULT " + a.Default
		// }
	}
	sig += ")"
	return sig
}

func CreateFunction(ctx context.Context, function *Function) (*Function, error) {
	conn := GetConn(ctx)
	create := "CREATE FUNCTION "
	fname := _s(function.Name, function.Schema)
	create += composeSignature(fname, function.Arguments)
	create += " RETURNS "
	if function.Returns != "" {
		create += function.Returns
	} else {
		create += "void"
	}
	create += " LANGUAGE "
	if function.Language != "" {
		create += function.Language
	} else {
		create += "sql"
	}
	create += " AS $$ "
	create += function.Definition
	create += " $$;"

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return function, nil
}

func DeleteFunction(ctx context.Context, fname string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP FUNCTION "+quoteParts(fname))
	return err
}

func ExecFunction(ctx context.Context, fname string, record Record, filters Filters, readonly bool) ([]byte, int64, error) {
	return Execute(ctx, fname, record, filters, readonly)
}
