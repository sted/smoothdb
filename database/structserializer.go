package database

import (
	"context"
	"encoding/binary"
	"fmt"
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

func rowsToStructs(rows pgx.Rows) ([]any, error) {
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
	array := make([]any, 0, 1000)
	//var v any

	for rows.Next() {
		bufRaw := rows.RawValues()

		for i := range fds {
			f := struc.Field(i)
			buf := bufRaw[i]
			if len(buf) == 0 {
				f.Set(reflect.Zero(f.Type()))
				continue
			}
			if len(buf) != 0 {
				switch fds[i].DataTypeOID {
				case pgtype.Int4OID, pgtype.OIDOID:
					v := int32(binary.BigEndian.Uint32(buf))
					f.Set(reflect.ValueOf(&v))
				case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
					v := string(buf)
					f.Set(reflect.ValueOf(&v))
				case pgtype.BoolOID:
					v := buf[0] == 1
					f.Set(reflect.ValueOf(&v))
				case pgtype.TimestampOID, pgtype.TimestamptzOID:
					microsecSinceY2K := int64(binary.BigEndian.Uint64((buf)))
					v := time.Unix(
						secFromUnixEpochToY2K+microsecSinceY2K/1000000,
						(microsecFromUnixEpochToY2K+microsecSinceY2K)%1000000*1000).UTC()
					f.Set(reflect.ValueOf(&v))
				default:
					// Handle unsupported data types
					return nil, fmt.Errorf("unsupported data type OID: %d", fds[i].DataTypeOID)
				}
			}
		}
		array = append(array, struc.Interface())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return array, nil
}

// /table?filter
func GetStructures(ctx context.Context, query string) ([]any, error) {
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
	sel, values, err := gi.QueryBuilder.BuildSelect(table, parts, options, gi.Db.info)
	if err != nil {
		return nil, err
	}
	rows, err := gi.Conn.Query(ctx, sel, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToStructs(rows)
}
