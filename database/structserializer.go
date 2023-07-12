package database

import (
	"context"
	"encoding/binary"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var typeMap2 = map[uint32]any{
	pgtype.Int4OID:      int32(0),
	pgtype.OIDOID:       int32(0),
	pgtype.TextOID:      string(""),
	pgtype.BoolOID:      bool(false),
	pgtype.TimestampOID: time.Now(),
}

func (db *Database) rowsToStructs(conn *pgx.Conn, sourcename string, rows pgx.Rows) ([]any, error) {

	fds := rows.FieldDescriptions()

	var structFields []reflect.StructField
	for i := range fds {
		newField := reflect.StructField{
			Name: strings.Title(fds[i].Name),
			Type: reflect.PtrTo(reflect.TypeOf(typeMap2[fds[i].DataTypeOID])),
			//Tag:  reflect.StructTag("`json:\"" + strings.ToLower(field.Name+"\"`")),
		}
		structFields = append(structFields, newField)
	}
	struc := reflect.New(reflect.StructOf(structFields)).Elem()
	//array := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(struc.Interface())), 0, 5000)
	array := make([]any, 0, 1000)

	for rows.Next() {

		bufs := rows.RawValues()

		for i := range fds {
			f := struc.Field(i)

			switch fds[i].DataTypeOID {
			case pgtype.Int4OID, pgtype.OIDOID:
				v := int32(binary.BigEndian.Uint32(bufs[i]))
				f.Set(reflect.ValueOf(&v))
			case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
				v := string(bufs[i])
				f.Set(reflect.ValueOf(&v))
			case pgtype.BoolOID:
				v := bufs[i][0] == 1
				f.Set(reflect.ValueOf(&v))
			case pgtype.TimestampOID, pgtype.TimestamptzOID:
				microsecSinceY2K := int64(binary.BigEndian.Uint64((bufs[i])))
				v := time.Unix(
					secFromUnixEpochToY2K+microsecSinceY2K/1000000,
					(microsecFromUnixEpochToY2K+microsecSinceY2K)%1000000*1000).UTC()
				f.Set(reflect.ValueOf(&v))
			}
		}
		//array = reflect.Append(array, struc)
		array = append(array, struc.Interface())
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return array, nil
}

// /table?filter
func (db *Database) GetStructures(ctx context.Context, query string) ([]any, error) {
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
	table = _s(table, options.Schema)
	return db.rowsToStructs(gi.Conn, table, rows)
}
