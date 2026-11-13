package database

import (
	"context"
	"strings"
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

type PolicyUpdate struct {
	Name  *string  `json:"name"`
	Roles []string `json:"roles"`
	Using *string  `json:"using"`
	Check *string  `json:"check"`
}

// policyCommands is the closed set of the FOR clause; empty means ALL.
var policyCommands = map[string]struct{}{
	"ALL": {}, "SELECT": {}, "INSERT": {}, "UPDATE": {}, "DELETE": {},
}

const policyQuery = `
	SELECT 
		pol.polname name,
		n.nspname || '.' || c.relname tablename,
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
	JOIN pg_class c ON c.oid = pol.polrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace`

// GetPolicies lists the policies of the table tablename of the request's
// schema (the profile header). The name is literal: a dot is part of it.
func GetPolicies(ctx context.Context, tablename string) ([]Policy, error) {
	conn, schemaname := GetConnAndSchema(ctx)
	policies := []Policy{}
	query := policyQuery
	query += " WHERE c.relname = $1 AND n.nspname = $2"
	query += " ORDER BY tablename, name"
	rows, err := conn.Query(ctx, query, tablename, schemaname)
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

func composeRoles(roles []string) string {
	var list string
	for i, role := range roles {
		if i != 0 {
			list += ", "
		}
		list += quote(role)
	}
	return list
}

// buildCreatePolicy writes CREATE POLICY on ftablename, the quoted and
// qualified table name. USING and WITH CHECK are SQL expressions by design
// (like PostgREST RLS policies) and cannot be parameterized: CreatePolicy is
// only reachable via the admin API, which requires EnableAdminRoute and
// authentication.
func buildCreatePolicy(ftablename string, policy *Policy) (string, error) {
	command := strings.ToUpper(strings.TrimSpace(policy.Command))
	if command == "" {
		command = "ALL"
	}
	if _, ok := policyCommands[command]; !ok {
		return "", &BuildError{"invalid policy command: " + policy.Command + " (use ALL, SELECT, INSERT, UPDATE or DELETE)"}
	}
	create := "CREATE POLICY " + quote(policy.Name) + " ON " + ftablename
	if policy.Retrictive {
		create += " AS RESTRICTIVE"
	}
	create += " FOR " + command
	if len(policy.Roles) != 0 {
		create += " TO " + composeRoles(policy.Roles)
	}
	if policy.Using != nil {
		create += " USING (" + *policy.Using + ")"
	}
	if policy.Check != nil {
		create += " WITH CHECK (" + *policy.Check + ")"
	}
	return create, nil
}

// CreatePolicy creates a policy on the table policy.Table of the request's
// schema. The table name is literal.
func CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	conn, schemaname := GetConnAndSchema(ctx)
	create, err := buildCreatePolicy(qualify(schemaname, policy.Table), policy)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, create); err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdatePolicy alters the roles, USING and WITH CHECK of the policy name on
// the table tablename of the request's schema, then renames it.
func UpdatePolicy(ctx context.Context, tablename string, name string, policy *PolicyUpdate) error {
	conn, schemaname := GetConnAndSchema(ctx)
	ftablename := qualify(schemaname, tablename)
	return withTx(ctx, conn, func() error {
		alter := "ALTER POLICY " + quote(name) + " ON " + ftablename
		var clauses string
		if len(policy.Roles) != 0 {
			clauses += " TO " + composeRoles(policy.Roles)
		}
		if policy.Using != nil {
			clauses += " USING (" + *policy.Using + ")"
		}
		if policy.Check != nil {
			clauses += " WITH CHECK (" + *policy.Check + ")"
		}
		if clauses != "" {
			if _, err := conn.Exec(ctx, alter+clauses); err != nil {
				return err
			}
		}
		// NAME as the last update
		if policy.Name != nil && *policy.Name != name {
			if _, err := conn.Exec(ctx, alter+" RENAME TO "+quote(*policy.Name)); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeletePolicy drops the policy name on the table tablename of the request's
// schema.
func DeletePolicy(ctx context.Context, tablename string, name string) error {
	conn, schemaname := GetConnAndSchema(ctx)
	_, err := conn.Exec(ctx, "DROP POLICY "+quote(name)+" ON "+qualify(schemaname, tablename))
	return err
}
