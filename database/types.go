package database

import "context"

type Type struct {
	Id           uint32   `json:"id"`
	Name         string   `json:"name"`
	Schema       string   `json:"schema"`
	IsArray      bool     `json:"isarray"`
	IsRange      bool     `json:"isrange"`
	IsComposite  bool     `json:"iscomposite"`
	IsTable      bool     `json:"istable"`
	IsEnum       bool     `json:"isenum"`
	ArraySubType uint32   `json:"arraysubtype"`
	RangeSubType *uint32  `json:"rangesubtype"`
	SubTypeIds   []uint32 `json:"subtypeids"`
	SubTypeNames []string `json:"subtypenames"`
}

const typesQuery = `
	SELECT
		t.oid::int4 oid,
		t.typname name,
		n.nspname schema,
		(t.typcategory = 'A') AS isarray,
		(t.typcategory = 'R') AS isrange,
		(t.typcategory = 'C' AND COALESCE(c.relkind = 'c', false)) AS iscomposite,
		(t.typcategory = 'C' AND COALESCE(c.relkind IN ('r','v'), false)) AS istable,
		(t.typcategory = 'E') AS isenum,
		t.typelem arraysubtype,
		r.rngsubtype rangesubtype,
		COALESCE(array_agg(a.atttypid::int4) filter (where a.atttypid is not null), '{}') subtypeids,
		COALESCE(array_agg(a.attname) filter (where a.attname is not null), '{}') subtypenames
	FROM pg_type t
	LEFT JOIN pg_class c ON c.oid = t.typrelid
	LEFT JOIN pg_attribute a ON a.attrelid = t.typrelid
	LEFT JOIN pg_range r ON r.rngtypid = t.oid
	JOIN pg_namespace n ON n.oid = t.typnamespace
	GROUP by t.oid, n.nspname, c.relkind, r.rngsubtype;
`

func GetTypes(ctx context.Context) ([]Type, error) {
	conn := GetConn(ctx)
	types := []Type{}
	rows, err := conn.Query(ctx, typesQuery)
	if err != nil {
		return types, err
	}
	defer rows.Close()

	typ := Type{}
	for rows.Next() {
		err := rows.Scan(&typ.Id, &typ.Name, &typ.Schema,
			&typ.IsArray, &typ.IsRange, &typ.IsComposite, &typ.IsTable, &typ.IsEnum,
			&typ.ArraySubType, &typ.RangeSubType,
			&typ.SubTypeIds, &typ.SubTypeNames)
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
