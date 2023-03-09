package database

import (
	"context"
)

type Policy struct {
	Name       string   `json:"name"`
	Table      string   `json:"table"`
	Retrictive bool     `json:"deny"`
	Command    string   `json:"command"`
	Roles      []string `json:"roles"`
	Using      *string  `json:"using"`
	Check      *string  `json:"check"`
}

const policyQuery = `
	SELECT 
		pol.polname name,
		c.relnamespace::regnamespace  || '.' || c.relname tablename,
		NOT pol.polpermissive deny,
		CASE pol.polcmd
			WHEN 'r'::"char" THEN 'SELECT'::text
			WHEN 'a'::"char" THEN 'INSERT'::text
			WHEN 'w'::"char" THEN 'UPDATE'::text
			WHEN 'd'::"char" THEN 'DELETE'::text
			WHEN '*'::"char" THEN 'ALL'::text
			ELSE NULL::text
		END cmd,
		CASE
			WHEN pol.polroles = '{0}'::oid[] THEN string_to_array('public'::text, ''::text)::name[]
			ELSE ARRAY( SELECT pg_authid.rolname
				FROM pg_authid
				WHERE pg_authid.oid = ANY(pol.polroles)
				ORDER BY pg_authid.rolname)
		END roles,
		pg_get_expr(pol.polqual, pol.polrelid) using,
		pg_get_expr(pol.polwithcheck, pol.polrelid) check
	FROM pg_policy pol
		JOIN pg_class c ON c.oid = pol.polrelid`

func (db *Database) GetPolicies(ctx context.Context, ftablename string) ([]Policy, error) {
	conn := GetConn(ctx)
	policies := []Policy{}
	query := policyQuery
	schemaname, tablename := splitTableName(ftablename)
	query += " WHERE c.relname = '" + tablename + "' AND c.relnamespace::regnamespace = '" + schemaname + "'::regnamespace"

	query += " ORDER BY tablename"
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

func (db *Database) CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	conn := GetConn(ctx)
	create := "CREATE POLICY " + policy.Name + " ON " + policy.Table
	if policy.Retrictive {
		create += " AS RESTRICTIVE"
	}
	create += " FOR " + policy.Command
	if len(policy.Roles) != 0 {
		create += " TO "
		for i, role := range policy.Roles {
			if i != 0 {
				create += ", "
			}
			create += role
		}
	}
	if policy.Using != nil {
		create += " USING (" + *policy.Using + ")"
	}
	if policy.Check != nil {
		create += " WITH CHECK (" + *policy.Check + ")"
	}
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (db *Database) DeletePolicy(ctx context.Context, table string, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP POLICY "+name+" ON "+table)
	return err
}
