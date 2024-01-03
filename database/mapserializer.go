package database

import (
	"context"
	"encoding/binary"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// This is experimental.
// For now it seems much slower than rowsToStructs

func (db *Database) rowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	fds := rows.FieldDescriptions()
	array := make([]map[string]any, 0, 1000)
	var v any

	for rows.Next() {
		m := make(map[string]any, len(fds))
		bufRaw := rows.RawValues()

		for i := range fds {
			fd := fds[i]
			buf := bufRaw[i]
			if len(buf) == 0 {
				m[fd.Name] = nil
				continue
			}
			switch fd.DataTypeOID {
			case pgtype.Int4OID, pgtype.OIDOID:
				v = int32(binary.BigEndian.Uint32(buf))
			case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
				v = string(buf)
			case pgtype.BoolOID:
				v = buf[0] == 1
			case pgtype.TimestampOID, pgtype.TimestamptzOID:
				microsecSinceY2K := int64(binary.BigEndian.Uint64((buf)))
				v = time.Unix(
					secFromUnixEpochToY2K+microsecSinceY2K/1000000,
					(microsecFromUnixEpochToY2K+microsecSinceY2K)%1000000*1000).UTC()
			}
			m[fd.Name] = v
			array = append(array, m)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return array, nil
}

// /table?filter
func (db *Database) GetMaps(ctx context.Context, query string) ([]map[string]any, error) {
	url, err := url.Parse(query)
	if err != nil {
		return nil, err
	}
	table := url.Path
	filters := url.Query()
	gi := GetSmoothContext(ctx)
	parts, err := gi.RequestParser.parse(table, filters)
	if err != nil {
		return nil, err
	}
	options := gi.QueryOptions
	sel, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, db.info)
	if err != nil {
		return nil, err
	}
	//info := gi.Db.info
	rows, err := gi.Conn.Query(ctx, sel, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.rowsToMaps(rows)
}
