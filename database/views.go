package database

import "context"

type View struct {
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Definition string `json:"definition"`
	Temporary  bool   `json:"temporary,omitempty"`
}

func (db *Database) GetViews(ctx context.Context) ([]View, error) {
	conn := GetConn(ctx)
	views := []View{}
	rows, err := conn.Query(ctx, `
		SELECT viewname, viewowner, definition 
		FROM pg_views
		WHERE schemaname = 'public'`)
	if err != nil {
		return views, err
	}
	defer rows.Close()

	view := View{}
	for rows.Next() {
		err := rows.Scan(&view.Name, &view.Owner, &view.Definition)
		if err != nil {
			return views, err
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return views, err
	}
	return views, nil
}

func (db *Database) GetView(ctx context.Context, name string) (*View, error) {
	conn := GetConn(ctx)
	view := View{}
	err := conn.QueryRow(ctx, `
		SELECT viewname, viewowner, definition 
		FROM pg_views
		WHERE viewname = $1`, name).
		Scan(&view.Name, &view.Owner)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (db *Database) CreateView(ctx context.Context, view *View) (*View, error) {
	conn := GetConn(ctx)
	create := "CREATE "
	if view.Temporary {
		create += "TEMP "
	}
	create += "VIEW " + view.Name + " AS " + view.Definition
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (db *Database) DeleteView(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP VIEW "+name)
	if err != nil {
		return err
	}
	return nil
}
