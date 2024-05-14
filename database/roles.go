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

type RoleUpdate struct {
	Name                *string `json:"name"`
	IsSuperUser         *bool   `json:"issuperuser"`
	CanLogin            *bool   `json:"canlogin"`
	NoInheritPrivileges *bool   `json:"noinherit"`
	CanCreateRoles      *bool   `json:"cancreateroles"`
	CanCreateDatabases  *bool   `json:"cancreatedatabases"`
	CanBypassRLS        *bool   `json:"canbypassrls"`
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

func UpdateRole(ctx context.Context, name string, role *RoleUpdate) error {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	prefix := "ALTER ROLE " + quote(name)
	alter := prefix

	if role.IsSuperUser != nil {
		if *role.IsSuperUser {
			alter += " SUPERUSER"
		} else {
			alter += " NOSUPERUSER"
		}
	}
	if role.CanLogin != nil {
		if *role.CanLogin {
			alter += " LOGIN"
		} else {
			alter += " NOLOGIN"
		}
	}
	if role.NoInheritPrivileges != nil {
		if *role.NoInheritPrivileges {
			alter += " NOINHERIT"
		} else {
			alter += " INHERIT"
		}
	}
	if role.CanCreateRoles != nil {
		if *role.CanCreateRoles {
			alter += " CREATEROLE"
		} else {
			alter += " NOCREATEROLE"
		}
	}
	if role.CanCreateDatabases != nil {
		if *role.CanCreateDatabases {
			alter += " CREATEDB"
		} else {
			alter += " NOCREATEDB"
		}
	}
	if role.CanBypassRLS != nil {
		if *role.CanBypassRLS {
			alter += " BYPASSRLS"
		} else {
			alter += " NOBYPASSRLS"
		}
	}
	_, err = conn.Exec(ctx, alter)
	if err != nil {
		return err
	}

	// NAME as the last update
	if role.Name != nil && *role.Name != name {
		_, err = conn.Exec(ctx, prefix+" RENAME TO "+*role.Name)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)

}

func DeleteRole(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP ROLE "+quote(name))
	return err
}
