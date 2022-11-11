package database

import "context"

type Role struct {
	Name              string
	IsSuperUSer       bool
	CanLogin          bool
	InheritPrivileges bool
	CanCreateRoles    bool
	CanCreateDatabase bool
	CanBypassRLS      bool
}

const rolesQuery = `SELECT rolname FROM pg_roles`

func (dbe *DBEngine) GetRoles(ctx context.Context) ([]Role, error) {
	roles := []Role{}
	rows, err := dbe.pool.Query(ctx, rolesQuery)
	if err != nil {
		return roles, err
	}
	defer rows.Close()

	role := Role{}
	for rows.Next() {
		err := rows.Scan(&role.Name, &role.IsSuperUSer, &role.CanLogin,
			&role.InheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabase, &role.CanBypassRLS)
		if err != nil {
			return roles, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return roles, err
	}
	return roles, nil
}

func (dbe *DBEngine) getRole(ctx context.Context, name string) (*Role, error) {
	role := &Role{}
	err := dbe.pool.QueryRow(ctx, rolesQuery+" WHERE rolename = $1", name).
		Scan(&role.Name, &role.IsSuperUSer, &role.CanLogin,
			&role.InheritPrivileges, &role.CanCreateRoles, &role.CanCreateDatabase, &role.CanBypassRLS)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (dbe *DBEngine) CreateRole(ctx context.Context, name string) (*Role, error) {
	_, err := dbe.pool.Exec(ctx, "CREATE ROLE "+name)
	if err != nil {
		return nil, err
	}
	return dbe.getRole(ctx, name)
}

func (dbe *DBEngine) DeleteRole(ctx context.Context, name string) error {
	_, err := dbe.pool.Exec(ctx, "DROP ROLE "+name)
	return err
}
