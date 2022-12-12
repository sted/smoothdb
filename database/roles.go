package database

import "context"

type Role struct {
	Name                string
	IsSuperUSer         bool
	CanLogin            bool
	NoInheritPrivileges bool
	CanCreateRoles      bool
	CanCreateDatabase   bool
	CanBypassRLS        bool
}

const rolesQuery = `SELECT 
rolname, rolsuper, rolcanlogin, NOT rolinherit, rolcreaterole, rolcreatedb, rolbypassrls
FROM pg_roles`

func (dbe *DbEngine) GetRoles(ctx context.Context) ([]Role, error) {
	roles := []Role{}
	rows, err := dbe.pool.Query(ctx, rolesQuery)
	if err != nil {
		return roles, err
	}
	defer rows.Close()

	role := Role{}
	for rows.Next() {
		err := rows.Scan(&role.Name, &role.IsSuperUSer, &role.CanLogin,
			&role.NoInheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabase, &role.CanBypassRLS)
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

func (dbe *DbEngine) GetRole(ctx context.Context, name string) (*Role, error) {
	role := &Role{}
	err := dbe.pool.QueryRow(ctx, rolesQuery+" WHERE rolename = $1", name).
		Scan(&role.Name, &role.IsSuperUSer, &role.CanLogin,
			&role.NoInheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabase, &role.CanBypassRLS)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (dbe *DbEngine) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	create := "CREATE ROLE " + role.Name
	if role.IsSuperUSer {
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
	if role.CanCreateDatabase {
		create += " CREATEDB"
	}
	if role.CanBypassRLS {
		create += " BYPASSRLS"
	}
	_, err := dbe.pool.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return dbe.GetRole(ctx, role.Name)
}

func (dbe *DbEngine) DeleteRole(ctx context.Context, name string) error {
	_, err := dbe.pool.Exec(ctx, "DROP ROLE "+name)
	return err
}
