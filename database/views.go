package database

import "context"

type View struct {
	Name         string `json:"name"`
	Schema       string `json:"schema"`
	Owner        string `json:"owner"`
	Definition   string `json:"definition"`
	Materialized bool   `json:"materialized"`
}

const viewsQuery = `
	SELECT 
		c.relname,
		n.nspname, 
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
		err := rows.Scan(&view.Name, &view.Schema, &view.Owner, &view.Definition, &kind)
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
	conn, schemaname := GetConnAndSchema(ctx)

	view := View{}
	var kind byte
	err := conn.QueryRow(ctx, viewsQuery+
		" AND c.relname = $1 AND n.nspname = $2", name, schemaname).
		Scan(&view.Name, &view.Schema, &view.Owner, &kind)
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
	fviewname := composeName(ctx, view.Schema, view.Name)
	create += "VIEW " + fviewname + " AS " + view.Definition
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func DeleteView(ctx context.Context, name string) error {
	conn, schemaname := GetConnAndSchema(ctx)
	fviewname := _sq(name, schemaname)

	_, err := conn.Exec(ctx, "DROP VIEW "+fviewname)
	if err != nil {
		return err
	}
	return nil
}

// ViewColDep maps a view column back to its originating base table column,
// extracted from the view's internal query tree (pg_rewrite ev_action).
type ViewColDep struct {
	ViewSchema string
	ViewName   string
	ViewColumn string
	BaseSchema string
	BaseTable  string
	BaseColumn string
}

const viewColDepsQuery = `
WITH RECURSIVE
raw_matches AS (
    SELECT r.ev_class AS view_oid,
           m.arr[1]::int AS resno, m.arr[2] AS resname,
           m.arr[3]::oid AS resorigtbl, m.arr[4]::int AS resorigcol,
           m.ord
    FROM pg_rewrite r,
         LATERAL (
             SELECT arr, ord FROM regexp_matches(r.ev_action::text,
                 ':resno (\d+) :resname (\w+) :ressortgroupref \d+ :resorigtbl (\d+) :resorigcol (\d+)',
                 'g') WITH ORDINALITY AS _(arr, ord)
         ) m
    WHERE r.ev_class IN (
        SELECT c.oid FROM pg_class c
        WHERE c.relkind IN ('v','m')
          AND c.relnamespace NOT IN (
              SELECT oid FROM pg_namespace WHERE nspname ~ '^pg_' OR nspname = 'information_schema'
          )
    )
    AND m.arr[3]::oid != 0
),
view_col_deps AS (
    SELECT DISTINCT ON (rm.view_oid, rm.resno)
           rm.view_oid, rm.resno AS view_colnum, rm.resorigtbl AS base_table_oid, rm.resorigcol AS base_colnum
    FROM raw_matches rm
    JOIN pg_attribute va ON va.attrelid = rm.view_oid AND va.attnum = rm.resno
    WHERE rm.resname = va.attname
    ORDER BY rm.view_oid, rm.resno, rm.ord
),
resolved(view_oid, view_colnum, base_table_oid, base_colnum, depth, path) AS (
    SELECT d.view_oid, d.view_colnum, d.base_table_oid, d.base_colnum,
           1, ARRAY[d.view_oid]
    FROM view_col_deps d
    UNION ALL
    SELECT r.view_oid, r.view_colnum, d.base_table_oid, d.base_colnum,
           r.depth + 1, r.path || d.view_oid
    FROM resolved r
    JOIN view_col_deps d ON d.view_oid = r.base_table_oid
                        AND d.view_colnum = r.base_colnum
    WHERE NOT d.view_oid = ANY(r.path)
      AND r.depth < 10
)
SELECT vn.nspname, vc.relname, va.attname,
       bn.nspname, bt.relname, ba.attname
FROM resolved r
JOIN pg_class vc ON vc.oid = r.view_oid
JOIN pg_namespace vn ON vn.oid = vc.relnamespace
JOIN pg_attribute va ON va.attrelid = r.view_oid AND va.attnum = r.view_colnum
JOIN pg_class bt ON bt.oid = r.base_table_oid
JOIN pg_namespace bn ON bn.oid = bt.relnamespace
JOIN pg_attribute ba ON ba.attrelid = r.base_table_oid AND ba.attnum = r.base_colnum
WHERE bt.relkind NOT IN ('v','m')
ORDER BY r.view_oid, r.view_colnum`

func GetViewColDeps(ctx context.Context) ([]ViewColDep, error) {
	conn := GetConn(ctx)
	rows, err := conn.Query(ctx, viewColDepsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []ViewColDep
	d := ViewColDep{}
	for rows.Next() {
		err := rows.Scan(&d.ViewSchema, &d.ViewName, &d.ViewColumn,
			&d.BaseSchema, &d.BaseTable, &d.BaseColumn)
		if err != nil {
			return nil, err
		}
		deps = append(deps, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deps, nil
}
