package database

import "context"

type User struct {
	Name               string   `json:"name"`
	MemberOf           []string `json:"memberof"`
	CanCreateRoles     bool     `json:"cancreateroles"`
	CanCreateDatabases bool     `json:"cancreatedatabases"`
}

const usersQuery = `
	SELECT b.rolname
	FROM pg_catalog.pg_auth_members m
	JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
	WHERE m.member = (SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname = $1)`

func GetUsers(ctx context.Context) ([]User, error) {
	conn := GetConn(ctx)
	roles := []User{}
	query := usersQuery + ` ORDER BY 1`
	rows, err := conn.Query(ctx, query, dbe.authRole)
	if err != nil {
		return roles, err
	}
	defer rows.Close()

	role := User{}
	for rows.Next() {
		err := rows.Scan(&role.Name)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return roles, err
	}
	return roles, nil
}

func GetUser(ctx context.Context, name string) (*User, error) {
	conn := GetConn(ctx)
	role := &User{}
	err := conn.QueryRow(ctx, usersQuery+" AND b.rolname = $2", dbe.authRole, name).
		Scan(&role.Name)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func CreateUser(ctx context.Context, user *User) (*User, error) {
	conn := GetConn(ctx)
	create := "CREATE ROLE " + quote(user.Name)
	create += " NOLOGIN"
	if user.CanCreateRoles {
		create += " CREATEROLE"
	}
	if user.CanCreateDatabases {
		create += " CREATEDB"
	}
	grant := "GRANT " + quote(user.Name) + " TO " + quote(dbe.authRole)
	err := withTx(ctx, conn, func() error {
		if _, err := conn.Exec(ctx, create); err != nil {
			return err
		}
		_, err := conn.Exec(ctx, grant)
		return err
	})
	if err != nil {
		return nil, err
	}
	return GetUser(ctx, user.Name)
}

func DeleteUser(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP ROLE "+quote(name))
	return err
}
