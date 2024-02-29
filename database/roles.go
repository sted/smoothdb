package database

import "context"

type Role struct {
	Name                string   `json:"name"`
	IsSuperUser         bool     `json:"issuperuser"`
	CanLogin            bool     `json:"canlogin"`
	NoInheritPrivileges bool     `json:"noinherit"`
	CanCreateRoles      bool     `json:"cancreateroles"`
	CanCreateDatabases  bool     `json:"cancreatedatabases"`
	CanBypassRLS        bool     `json:"canbypassrls"`
	MemberOf            []string `json:"memberof"` // readonly for now
}

const rolesQuery = `
	SELECT r.rolname, r.rolsuper, r.rolcanlogin, NOT r.rolinherit, r.rolcreaterole, r.rolcreatedb, r.rolbypassrls,
		ARRAY(SELECT b.rolname
			FROM pg_catalog.pg_auth_members m
			JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
	WHERE m.member = r.oid) as memberof
	FROM pg_catalog.pg_roles r`

func GetRoles(ctx context.Context) ([]Role, error) {
	conn := GetConn(ctx)
	roles := []Role{}
	query := rolesQuery + ` WHERE r.rolname !~ '^pg_' ORDER BY 1`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return roles, err
	}
	defer rows.Close()

	role := Role{}
	for rows.Next() {
		err := rows.Scan(&role.Name, &role.IsSuperUser, &role.CanLogin,
			&role.NoInheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabases, &role.CanBypassRLS, &role.MemberOf)
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

func GetRole(ctx context.Context, name string) (*Role, error) {
	conn := GetConn(ctx)
	role := &Role{}
	err := conn.QueryRow(ctx, rolesQuery+" WHERE r.rolname = $1", name).
		Scan(&role.Name, &role.IsSuperUser, &role.CanLogin,
			&role.NoInheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabases, &role.CanBypassRLS, &role.MemberOf)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func CreateRole(ctx context.Context, role *Role) (*Role, error) {
	conn := GetConn(ctx)
	create := "CREATE ROLE " + quote(role.Name)
	if role.IsSuperUser {
		create += " SUPERUSER"
	}
	if role.CanLogin {
		create += " LOGIN"
	}
	if role.NoInheritPrivileges {
		create += " NOINHERIT"
	}
	if role.CanCreateRoles {
		create += " CREATEROLE"
	}
	if role.CanCreateDatabases {
		create += " CREATEDB"
	}
	if role.CanBypassRLS {
		create += " BYPASSRLS"
	}
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return GetRole(ctx, role.Name)
}

func DeleteRole(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP ROLE "+quote(name))
	return err
}
