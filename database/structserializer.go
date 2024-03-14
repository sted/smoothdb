package database

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var typeMap = map[uint32]any{
	pgtype.Int2OID:        int16(0),
	pgtype.Int4OID:        int32(0),
	pgtype.OIDOID:         int32(0),
	pgtype.Int8OID:        int64(0),
	pgtype.Float4OID:      float32(0),
	pgtype.Float8OID:      float64(0),
	pgtype.BoolOID:        bool(false),
	pgtype.TextOID:        string(""),
	pgtype.VarcharOID:     string(""),
	pgtype.NameOID:        string(""),
	pgtype.TimestampOID:   time.Now(),
	pgtype.TimestamptzOID: time.Now(),
	pgtype.JSONOID:        string(""),
	pgtype.JSONBOID:       string(""),
}

// rowsToDynStructs converts each row to a struct with this format:
// struct {fieldName1 fieldType1, fieldName1_IsNull bool, fieldName1 fieldType2, fieldName2_IsNull bool, ... }
// If a field is null, the struct will have the correlated fieldName_IsNull as true.
// It returns an array of these structures as []any.
// This function is about twice as fast as rowsToStructsWithPointers, but it has some usage inconveniences.
func rowsToDynStructs(rows pgx.Rows) ([]any, error) {
	fds := rows.FieldDescriptions()
	var structFields []reflect.StructField
	var newField reflect.StructField
	for i := range fds {
		newField = reflect.StructField{
			Name: strings.Title(fds[i].Name),
			Type: reflect.TypeOf(typeMap[fds[i].DataTypeOID]),
		}
		structFields = append(structFields, newField)
		newField = reflect.StructField{
			Name: strings.Title(fds[i].Name + "_IsNull"),
			Type: reflect.TypeOf(false),
		}
		structFields = append(structFields, newField)
	}
	struc := reflect.New(reflect.StructOf(structFields)).Elem()
	array := make([]any, 0, 1000)

	for rows.Next() {
		bufRaw := rows.RawValues()

		for i := range fds {
			buf := bufRaw[i]
			if buf == nil {
				struc.Field(i * 2).SetZero()
				struc.Field(i*2 + 1).SetBool(true)
				continue
			}

			f := struc.Field(i * 2)
			struc.Field(i*2 + 1).SetBool(false)
			t := fds[i].DataTypeOID

			switch t {
			case pgtype.Int2OID:
				f.SetInt(int64(toInt16(buf)))
			case pgtype.Int4OID, pgtype.OIDOID:
				f.SetInt(int64(toInt32(buf)))
			case pgtype.Int8OID:
				f.SetInt(toInt64(buf))
			case pgtype.Float4OID:
				f.SetFloat(float64(toFloat32(buf)))
			case pgtype.Float8OID:
				f.SetFloat(toFloat64(buf))
			case pgtype.BoolOID:
				f.SetBool(toBool(buf))
			case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
				v := string(buf)
				f.SetString(v)
			case pgtype.JSONOID, pgtype.JSONBOID:
				v := string(buf)
				f.SetString(v)
			case pgtype.TimestampOID, pgtype.TimestamptzOID:
				v := toTime(buf)
				fieldPtr := unsafe.Pointer(f.UnsafeAddr())
				*(*time.Time)(fieldPtr) = v
				//f.Set(reflect.ValueOf(v))
			default:
				// Handle unsupported data types
				return nil, fmt.Errorf("unsupported data type OID: %d", t)
			}
		}
		array = append(array, struc.Interface())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return array, nil
}

// rowsToDynStructsWithPointers converts each row to a struct with this format:
// struct {fieldName1 *fieldType1, fieldName2 *fieldType2, ... }
// If a field is null, the struct will have a nil pointer.
// It returns an array of these structures as []any.
// This function is two times slower than rowsToStructs, but it is more straightforward to use.
func rowsToDynStructsWithPointers(rows pgx.Rows) ([]any, error) {
	fds := rows.FieldDescriptions()
	var structFields []reflect.StructField
	var newField reflect.StructField
	for i := range fds {
		newField = reflect.StructField{
			Name: strings.Title(fds[i].Name),
			Type: reflect.PointerTo(reflect.TypeOf(typeMap[fds[i].DataTypeOID])),
		}
		structFields = append(structFields, newField)
	}
	struc := reflect.New(reflect.StructOf(structFields)).Elem()
	array := make([]any, 0, 1000)
	var v reflect.Value

	for rows.Next() {
		bufRaw := rows.RawValues()

		for i := range fds {
			buf := bufRaw[i]
			f := struc.Field(i)
			if buf == nil {
				f.Set(reflect.Zero(f.Type()))
				continue
			}
			t := fds[i].DataTypeOID

			switch t {
			case pgtype.Int2OID:
				fv := toInt16(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.Int4OID, pgtype.OIDOID:
				fv := toInt32(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.Int8OID:
				fv := toInt64(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.Float4OID:
				fv := toFloat32(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.Float8OID:
				fv := toFloat64(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.BoolOID:
				fv := toBool(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
				fv := string(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.JSONOID, pgtype.JSONBOID:
				fv := string(buf)
				v = reflect.ValueOf(&fv)
			case pgtype.TimestampOID, pgtype.TimestamptzOID:
				fv := toTime(buf)
				v = reflect.ValueOf(&fv)
			default:
				// Handle unsupported data types
				return nil, fmt.Errorf("unsupported data type OID: %d", t)
			}
			f.Set(v)
		}
		array = append(array, struc.Interface())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return array, nil
}

func GetDynStructures(ctx context.Context, query string) ([]any, error) {
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
	return rowsToDynStructs(rows)
}
