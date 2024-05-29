package database

import (
	"context"
)

type Schema struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
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

func CreateSchema(ctx context.Context, name string) (*Schema, error) {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "CREATE SCHEMA "+quote(name))
	if err != nil {
		return nil, err
	}
	schema := &Schema{}
	err = conn.QueryRow(ctx, schemaQuery+" WHERE n.nspname = $1", name).Scan(&schema.Name, &schema.Owner)
	if err != nil {
		return nil, err
	}
	return schema, nil
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
