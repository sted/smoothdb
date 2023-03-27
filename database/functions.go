package database

import "context"

type Argument struct {
	Name    string
	Type    string
	Out     bool
	Default string
}

type Function struct {
	Name       string
	Arguments  []Argument `json:"arguments"`
	ResultType string     `json:"resulttype"`
	Definition string     `json:"definition"`
	Language   string     `json:"language"`
}

func (db *Database) ExecFunction(ctx context.Context, name string, record Record, filters Filters) ([]byte, int64, error) {
	return db.exec.Execute(ctx, name, record, filters)
}

func (db *Database) GetFunctions(ctx context.Context) ([]Policy, error) {
	conn := GetConn(ctx)
	policies := []Policy{}
	query := policyQuery

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policy := Policy{}
	for rows.Next() {
		err := rows.Scan(&policy.Name, &policy.Table, &policy.Retrictive,
			&policy.Command, &policy.Roles, &policy.Using, &policy.Check)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return policies, nil
}

func composeSignature(name string, args []Argument) string {
	sig := name + "("
	for i, a := range args {
		if i != 0 {
			sig += ", "
		}
		if a.Out {
			sig += "OUT "
		}
		sig += a.Name + " " + a.Type
		if a.Default != "" {
			sig += " DEFAULT " + a.Default
		}
	}
	sig += ")"
	return sig
}

func (db *Database) CreateFunction(ctx context.Context, function *Function) (*Function, error) {
	conn := GetConn(ctx)
	create := "CREATE FUNCTION "
	create += composeSignature(function.Name, function.Arguments)
	create += " RETURNS "
	if function.ResultType != "" {
		create += function.ResultType
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

func (db *Database) DeleteFunction(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP FUNCTION "+name)
	return err
}
