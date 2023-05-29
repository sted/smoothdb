package database

import (
	"context"
	"fmt"
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
		n.nspname NOT IN ('pg_catalog', 'information_schema')`

//AND  pg_table_is_visible(c.oid)

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
		default:
			return fmt.Errorf("invalid privilege string")
		}
		priv.Types = append(priv.Types, privilegeType)
	}
	return nil
}

func (dbe *DbEngine) GetDatabasePrivileges(ctx context.Context, dbname string) ([]Privilege, error) {
	conn := GetConn(ctx)
	privileges := []Privilege{}

	query := privilegesDatabaseQuery
	if dbname != "" {
		query += " WHERE datname = '" + dbname + "'"
	}

	rows, err := conn.Query(ctx, query)
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

func (db *Database) GetPrivileges(ctx context.Context, targetType string, targetName string) ([]Privilege, error) {
	conn := GetConn(ctx)
	privileges := []Privilege{}
	var query string

	if targetType == "table" {
		query = privilegesRelationQuery
		if targetName != "" {
			query += " AND c.relname = '" + targetName + "'"
		}
	}

	rows, err := conn.Query(ctx, query)
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

	return privileges, nil
}

func (db *Database) CreatePrivilege(ctx context.Context, privilege *Privilege) (*Privilege, error) {
	conn := GetConn(ctx)
	create := "GRANT "
	if len(privilege.Types) != 0 {
		for i, t := range privilege.Types {
			if i != 0 {
				create += ", "
			}
			create += t
		}
		create += " ON " + privilege.TargetType + " " + privilege.TargetName
	} else {
		// grant role to role
		create += privilege.TargetName
	}
	create += " TO " + privilege.Grantee
	// to implement: with grant option

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return privilege, nil
}

func (db *Database) DeletePrivilege(ctx context.Context, privilege *Privilege) error {
	conn := GetConn(ctx)
	delete := "REVOKE "
	if len(privilege.Types) != 0 {
		for i, t := range privilege.Types {
			if i != 0 {
				delete += ", "
			}
			delete += t
		}
		delete += " ON " + privilege.TargetType + " " + privilege.TargetName
	} else {
		// revoke role from role
		delete += privilege.TargetName
	}
	delete += " FROM " + privilege.Grantee
	// to implement: with grant option

	_, err := conn.Exec(ctx, delete)
	return err
}
