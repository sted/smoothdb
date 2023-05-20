package database

import "context"

type Type struct {
	Id           uint32   `json:"id"`
	Name         string   `json:"name"`
	IsArray      bool     `json:"isarray"`
	IsComposite  bool     `json:"iscomposite"`
	SubTypeIds   []uint32 `json:"subtypeids"`
	SubTypeNames []string `json:"subtypenames"`
}

const typesQuery = `
	SELECT
		t.oid::int4 oid,
		t.typname name,
		(t.typcategory = 'A') AS isarray,
		(t.typcategory = 'C') AS iscomposite,
		coalesce(array_agg(a.atttypid::int4) filter (where a.atttypid is not null), '{}') subtypeids,
		coalesce(array_agg(a.attname) filter (where a.attname is not null), '{}') subtypenames
	FROM pg_type t
	LEFT JOIN pg_class c ON c.oid = t.typrelid
	LEFT JOIN pg_attribute a ON a.attrelid = t.typrelid
	JOIN pg_namespace n ON n.oid = t.typnamespace
	WHERE t.typrelid = 0 or c.relkind = 'c'
	GROUP by t.oid
`

func (db *Database) GetTypes(ctx context.Context) ([]Type, error) {
	conn := GetConn(ctx)
	types := []Type{}
	rows, err := conn.Query(ctx, typesQuery)
	if err != nil {
		return types, err
	}
	defer rows.Close()

	typ := Type{}
	for rows.Next() {
		err := rows.Scan(&typ.Id, &typ.Name, &typ.IsArray, &typ.IsComposite, &typ.SubTypeIds, &typ.SubTypeNames)
		if err != nil {
			return types, err
		}
		types = append(types, typ)
	}
	if rows.Err() != nil {
		return types, err
	}
	return types, nil
}
