package database

import "context"

type View struct {
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	Definition   string `json:"definition"`
	Materialized bool   `json:"materialized"`
}

const viewsQuery = `
	SELECT n.nspname || '.' || c.relname,                                  
		pg_get_userbyid(c.relowner) AS viewowner,               
		pg_get_viewdef(c.oid) AS definition,
		c.relkind                  
	FROM (pg_class c                                         
	LEFT JOIN pg_namespace n ON ((n.oid = c.relnamespace)))
	WHERE (c.relkind = ANY (ARRAY['v'::"char", 'm'::"char"]))`

func GetViews(ctx context.Context) ([]View, error) {
	conn := GetConn(ctx)
	views := []View{}
	rows, err := conn.Query(ctx, viewsQuery+
		" AND n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' ORDER BY 1")
	if err != nil {
		return views, err
	}
	defer rows.Close()

	view := View{}
	var kind byte
	for rows.Next() {
		err := rows.Scan(&view.Name, &view.Owner, &view.Definition, &kind)
		if err != nil {
			return views, err
		}
		if kind == 'v' {
			view.Materialized = false
		} else {
			view.Materialized = true
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return views, err
	}
	return views, nil
}

func GetView(ctx context.Context, name string) (*View, error) {
	conn := GetConn(ctx)

	schemaname, viewname := splitTableName(name)
	view := View{}
	var kind byte
	err := conn.QueryRow(ctx, viewsQuery+
		" AND c.relname = $1 AND n.nspname = $2", viewname, schemaname).
		Scan(&view.Name, &view.Owner, &kind)
	if err != nil {
		return nil, err
	}
	if kind == 'v' {
		view.Materialized = false
	} else {
		view.Materialized = true
	}
	return &view, nil
}

func CreateView(ctx context.Context, view *View) (*View, error) {
	conn := GetConn(ctx)
	create := "CREATE "
	if view.Materialized {
		create += "MATERIALIZED "
	}
	create += "VIEW " + quoteParts(view.Name) + " AS " + view.Definition
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func DeleteView(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP VIEW "+quoteParts(name))
	if err != nil {
		return err
	}
	return nil
}
