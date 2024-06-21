package database

import (
	"context"
)

type Schema struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type SchemaUpdate struct {
	Name  *string `json:"name"`
	Owner *string `json:"owner"`
}

const schemaQuery = `
	SELECT n.nspname, pg_catalog.pg_get_userbyid(n.nspowner)
	FROM pg_catalog.pg_namespace n`

func GetSchemas(ctx context.Context) ([]Schema, error) {
	conn := GetConn(ctx)
	schemas := []Schema{}
	rows, err := conn.Query(ctx, schemaQuery+
		" WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' ORDER BY 1;")
	if err != nil {
		return schemas, err
	}
	defer rows.Close()

	schema := Schema{}
	for rows.Next() {
		err := rows.Scan(&schema.Name, &schema.Owner)
		if err != nil {
			return schemas, err
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return schemas, err
	}
	return schemas, nil
}

func GetSchema(ctx context.Context, name string) (*Schema, error) {
	conn := GetConn(ctx)
	schema := &Schema{}
	err := conn.QueryRow(ctx, schemaQuery+" WHERE n.nspname = $1", name).
		Scan(&schema.Name, &schema.Owner)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func CreateSchema(ctx context.Context, schema *Schema) (*Schema, error) {
	conn := GetConn(ctx)
	create := "CREATE SCHEMA " + quote(schema.Name)
	if schema.Owner != "" {
		create += " AUTHORIZATION " + schema.Owner
	}
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return GetSchema(ctx, schema.Name)
}

func UpdateSchema(ctx context.Context, name string, schema *SchemaUpdate) error {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	prefix := "ALTER SCHEMA " + quote(name)

	if schema.Owner != nil {
		_, err = conn.Exec(ctx, prefix+" OWNER TO "+quote(*schema.Owner))
		if err != nil {
			return err
		}
	}
	// NAME as the last update
	if schema.Name != nil && *schema.Name != name {
		_, err = conn.Exec(ctx, prefix+" RENAME TO "+quote(*schema.Name))
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)

}

func DeleteSchema(ctx context.Context, name string, cascade bool) error {
	conn := GetConn(ctx)
	delete := "DROP SCHEMA " + quote(name)
	if cascade {
		delete += " CASCADE"
	}
	_, err := conn.Exec(ctx, delete)
	return err
}
