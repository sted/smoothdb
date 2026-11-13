package database

import (
	"context"
	"fmt"
	"strings"
)

type Privilege struct {
	TargetName   string   `json:"targetname"`
	TargetSchema string   `json:"targetschema,omitempty"`
	TargetType   string   `json:"targettype"`        // database, schema, table, column, function
	Columns      []string `json:"columns,omitempty"` // to insert column privileges
	Types        []string `json:"types"`
	Grantee      string   `json:"grantee"`
	Grantor      string   `json:"grantor"`
	ACL          string   `json:"acl"`
}

const privilegesDatabaseQuery = `
	SELECT datname name, unnest(datacl) acl 
	FROM pg_catalog.pg_database`

const privilegesRelationQuery = `
	SELECT c.relname name,
		n.nspname schema, 
		CASE c.relkind 
			WHEN 'r' THEN 'table' 
			WHEN 'v' THEN 'view' 
			WHEN 'm' THEN 'materialized view' 
			WHEN 'S' THEN 'sequence' 
			WHEN 'f' THEN 'foreign table'
			WHEN 'p' THEN 'partitioned table' 
		END, 
		unnest(c.relacl) acl 
	FROM pg_catalog.pg_class c 
	JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace 
	WHERE c.relkind IN ('r','v','m','S','f','p') AND 
		n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'`

//AND  pg_table_is_visible(c.oid)

// privilegesColumnQuery lists the column ACL entries of one table ($1 name,
// $2 schema), in column order. A column grant lives in pg_attribute.attacl,
// not in the table's relacl.
const privilegesColumnQuery = `
	SELECT a.attname, unnest(a.attacl) acl
	FROM pg_catalog.pg_attribute a
	JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
	JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relname = $1 AND n.nspname = $2
		AND a.attnum > 0 AND NOT a.attisdropped AND a.attacl IS NOT NULL
	ORDER BY a.attnum`

const privilegesSchemaQuery = `
	SELECT n.nspname name, '' schema, 'schema', unnest(n.nspacl) acl
	FROM pg_catalog.pg_namespace n
	WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'`

const privilegesFunctionQuery = `
	SELECT p.proname name, n.nspname schema, 'function', unnest(p.proacl) acl
	FROM pg_catalog.pg_proc p
	JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
	WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'`

func parsePrivilege(s string, priv *Privilege) error {
	var privilegeLetters string
	var privilegeType string

	var p = 0
	for i, r := range s {
		if r == '=' {
			priv.Grantee = s[p:i]
			p = i + 1
		} else if r == '/' {
			privilegeLetters = s[p:i]
			priv.Grantor = s[i+1:]
		}
	}
	priv.Types = nil
	for _, l := range privilegeLetters {
		switch l {
		case 'r':
			privilegeType = "SELECT"
		case 'a':
			privilegeType = "INSERT"
		case 'w':
			privilegeType = "UPDATE"
		case 'd':
			privilegeType = "DELETE"
		case 'D':
			privilegeType = "TRUNCATE"
		case 'x':
			privilegeType = "REFERENCES"
		case 't':
			privilegeType = "TRIGGER"
		case 'C':
			privilegeType = "CREATE"
		case 'c':
			privilegeType = "CONNECT"
		case 'T':
			privilegeType = "TEMPORARY"
		case 'X':
			privilegeType = "EXECUTE"
		case 'U':
			privilegeType = "USAGE"
		case 's':
			privilegeType = "SET"
		case 'A':
			privilegeType = "ALTER SYSTEM"
		case 'm': // PostgreSQL 17+
			privilegeType = "MAINTAIN"
		default:
			return fmt.Errorf("invalid privilege string %q: unknown privilege %q", s, l)
		}
		priv.Types = append(priv.Types, privilegeType)
	}
	return nil
}

func GetDatabasePrivileges(ctx context.Context, dbname string) ([]Privilege, error) {
	conn := GetConn(ctx)
	privileges := []Privilege{}

	query := privilegesDatabaseQuery
	var args []any
	if dbname != "" {
		query += " WHERE datname = $1"
		args = append(args, dbname)
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		privilege := Privilege{TargetType: "database"}
		err := rows.Scan(&privilege.TargetName, &privilege.ACL)
		if err != nil {
			return nil, err
		}
		err = parsePrivilege(privilege.ACL, &privilege)
		if err != nil {
			return nil, err
		}
		privileges = append(privileges, privilege)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return privileges, nil
}

func GetPrivileges(ctx context.Context, targetType string, targetName string) ([]Privilege, error) {
	conn := GetConn(ctx)
	privileges := []Privilege{}
	var query string

	var args []any
	switch targetType {
	case "table": // @@ table includes views etc for now
		query = privilegesRelationQuery
		if targetName != "" {
			schemaname, tablename := splitTableName(targetName)
			query += " AND c.relname = $1 AND n.nspname = $2"
			args = append(args, tablename, schemaname)
		}
	case "schema":
		query = privilegesSchemaQuery
		if targetName != "" {
			query += " AND n.nspname = $1"
			args = append(args, targetName)
		}
	case "function":
		// every overload of the name: the ACL is per signature
		query = privilegesFunctionQuery
		if targetName != "" {
			schemaname, fname := splitTableName(targetName)
			query += " AND p.proname = $1 AND n.nspname = $2"
			args = append(args, fname, schemaname)
		}
	default:
		return privileges, nil
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		privilege := Privilege{TargetType: targetType}
		err := rows.Scan(&privilege.TargetName, &privilege.TargetSchema, &privilege.TargetType, &privilege.ACL)
		if err != nil {
			return nil, err
		}
		err = parsePrivilege(privilege.ACL, &privilege)
		if err != nil {
			return nil, err
		}
		privileges = append(privileges, privilege)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if targetType == "table" && targetName != "" {
		schemaname, tablename := splitTableName(targetName)
		columnPrivileges, err := getColumnPrivileges(ctx, conn, tablename, schemaname)
		if err != nil {
			return nil, err
		}
		privileges = append(privileges, columnPrivileges...)
	}
	return privileges, nil
}

// getColumnPrivileges returns the column grants of a table. The ACL entries
// that are equal on several columns (same grantee, privileges and grantor)
// become one privilege with the list of those columns.
func getColumnPrivileges(ctx context.Context, conn *DbConn, table, schema string) ([]Privilege, error) {
	rows, err := conn.Query(ctx, privilegesColumnQuery, table, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	privileges := []Privilege{}
	byACL := map[string]int{}
	for rows.Next() {
		var column, acl string
		if err := rows.Scan(&column, &acl); err != nil {
			return nil, err
		}
		if i, ok := byACL[acl]; ok {
			privileges[i].Columns = append(privileges[i].Columns, column)
			continue
		}
		privilege := Privilege{TargetName: table, TargetSchema: schema, TargetType: "table", ACL: acl, Columns: []string{column}}
		if err := parsePrivilege(acl, &privilege); err != nil {
			return nil, err
		}
		byACL[acl] = len(privileges)
		privileges = append(privileges, privilege)
	}
	return privileges, rows.Err()
}

// validPrivilegeTypes is the closed set of GRANT/REVOKE privilege keywords.
// Privilege verbs and object types cannot be bind parameters and are
// interpolated verbatim, so they must be whitelisted rather than escaped.
var validPrivilegeTypes = map[string]struct{}{
	"SELECT": {}, "INSERT": {}, "UPDATE": {}, "DELETE": {}, "TRUNCATE": {},
	"REFERENCES": {}, "TRIGGER": {}, "CREATE": {}, "CONNECT": {}, "TEMPORARY": {},
	"TEMP": {}, "EXECUTE": {}, "USAGE": {}, "SET": {}, "ALTER SYSTEM": {},
	"MAINTAIN": {}, "ALL": {}, "ALL PRIVILEGES": {},
}

// validColumnPrivilegeTypes are the privileges that a GRANT can give on
// single columns.
var validColumnPrivilegeTypes = map[string]struct{}{
	"SELECT": {}, "INSERT": {}, "UPDATE": {}, "REFERENCES": {},
}

// validTargetTypes is the closed set of object types that can follow ON in a
// GRANT/REVOKE ... ON <type> statement.
var validTargetTypes = map[string]struct{}{
	"DATABASE": {}, "SCHEMA": {}, "TABLE": {}, "SEQUENCE": {}, "COLUMN": {},
	"FUNCTION": {}, "PROCEDURE": {}, "ROUTINE": {}, "DOMAIN": {}, "TYPE": {},
	"TABLESPACE": {}, "LANGUAGE": {}, "LARGE OBJECT": {},
	"FOREIGN DATA WRAPPER": {}, "FOREIGN SERVER": {},
}

// buildGrantRevoke assembles a GRANT (grant=true) or REVOKE statement, validating
// the privilege verbs and target type against fixed whitelists and quoting the
// target name and grantee. See injection_test.go.
func buildGrantRevoke(privilege *Privilege, grant bool) (string, error) {
	var b strings.Builder
	if grant {
		b.WriteString("GRANT ")
	} else {
		b.WriteString("REVOKE ")
	}
	var columns string
	for i, c := range privilege.Columns {
		if i != 0 {
			columns += ", "
		}
		columns += quote(c)
	}
	if len(privilege.Types) != 0 {
		for i, t := range privilege.Types {
			upper := strings.ToUpper(strings.TrimSpace(t))
			if _, ok := validPrivilegeTypes[upper]; !ok {
				return "", &BuildError{"invalid privilege type: " + t}
			}
			if columns != "" {
				if _, ok := validColumnPrivilegeTypes[upper]; !ok {
					return "", &BuildError{"invalid privilege type with columns: " + t + " (use SELECT, INSERT, UPDATE or REFERENCES)"}
				}
			}
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(t)
			if columns != "" {
				b.WriteString(" (" + columns + ")")
			}
		}
		if _, ok := validTargetTypes[strings.ToUpper(strings.TrimSpace(privilege.TargetType))]; !ok {
			return "", &BuildError{"invalid target type: " + privilege.TargetType}
		}
		b.WriteString(" ON " + privilege.TargetType + " " + quoteParts(privilege.TargetName))
	} else {
		// grant/revoke role to/from role
		b.WriteString(quoteParts(privilege.TargetName))
	}
	if grant {
		b.WriteString(" TO " + quote(privilege.Grantee))
	} else {
		b.WriteString(" FROM " + quote(privilege.Grantee))
	}
	// to implement: with grant option
	return b.String(), nil
}

func CreatePrivilege(ctx context.Context, privilege *Privilege) (*Privilege, error) {
	create, err := buildGrantRevoke(privilege, true)
	if err != nil {
		return nil, err
	}
	conn := GetConn(ctx)
	if _, err := conn.Exec(ctx, create); err != nil {
		return nil, err
	}
	return privilege, nil
}

func DeletePrivilege(ctx context.Context, privilege *Privilege) error {
	revoke, err := buildGrantRevoke(privilege, false)
	if err != nil {
		return err
	}
	conn := GetConn(ctx)
	_, err = conn.Exec(ctx, revoke)
	return err
}
