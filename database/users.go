package database

import "context"

type User struct {
	Name     string   `json:"name"`
	MemberOf []string `json:"memberof"`
}

const usersQuery = `SELECT b.rolname
	FROM pg_catalog.pg_auth_members m
		JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
	WHERE m.member = (SELECT r.oid FROM pg_roles r WHERE r.rolname = 'auth')`

func (dbe *DbEngine) GetUsers(ctx context.Context) ([]User, error) {
	conn := GetConn(ctx)
	roles := []User{}
	query := usersQuery + ` ORDER BY 1`
	rows, err := conn.Query(ctx, query)
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

func (dbe *DbEngine) GetUser(ctx context.Context, name string) (*User, error) {
	conn := GetConn(ctx)
	role := &User{}
	err := conn.QueryRow(ctx, usersQuery+" AND b.rolname = $1", name).
		Scan(&role.Name)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (dbe *DbEngine) CreateUser(ctx context.Context, role *User) (*User, error) {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	create := "CREATE ROLE " + role.Name
	create += " NOLOGIN"
	_, err = tx.Exec(ctx, create)
	if err != nil {
		return nil, err
	}

	grant := "GRANT " + role.Name + " TO " + dbe.config.AuthRole
	_, err = tx.Exec(ctx, grant)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return dbe.GetUser(ctx, role.Name)
}

func (dbe *DbEngine) DeleteUser(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP ROLE "+name)
	return err
}
