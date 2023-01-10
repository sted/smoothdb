package database

import "context"

type Privilege struct {
	PrivilegeTypes []string `json:"privileges"`
	TargetType     string   `json:"targettype"`
	TargetName     string   `json:"targetname"`
	Roles          []string `json:"roles"`
}

func (db *Database) GetPrivileges(ctx context.Context, ftablename string) ([]Privilege, error) {
	conn := GetConn(ctx)
	privileges := []Privilege{}
	query := policyQuery
	schemaname, tablename := splitTableName(ftablename)
	query += " WHERE c.relname = '" + tablename + "' AND c.relnamespace::regnamespace = '" + schemaname + "'::regnamespace"

	query += " ORDER BY tablename"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	privilege := Privilege{}
	for rows.Next() {
		// err := rows.Scan(&privilege.Name, &privilege.Table, &privilege.Retrictive,
		// 	&privilege.Command, &privilege.Roles, &privilege.Using, &privilege.Check)
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

	_, err := conn.Exec(ctx, "")
	if err != nil {
		return nil, err
	}
	return privilege, nil
}

func (db *Database) DeletePrivilege(ctx context.Context, table string, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "REVOKE "+name+" ON "+table)
	return err
}
