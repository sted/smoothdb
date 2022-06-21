package database

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func Values(conn *pgxpool.Conn, rows pgx.Rows, values []interface{}) error {
	// if rows.closed {
	// 	return nil, errors.New("rows is closed")
	// }

	connInfo := conn.Conn().ConnInfo()

	for i := range rows.FieldDescriptions() {
		buf := rows.RawValues()[i]
		fd := &rows.FieldDescriptions()[i]

		if buf == nil {
			values[i] = nil
			continue
		}

		if dt, ok := connInfo.DataTypeForOID(fd.DataTypeOID); ok {
			value := dt.Value

			switch fd.Format {
			case pgx.TextFormatCode:
				decoder, ok := value.(pgtype.TextDecoder)
				if !ok {
					decoder = &pgtype.GenericText{}
				}
				err := decoder.DecodeText(connInfo, buf)
				if err != nil {
					//rows.fatal(err)
				}
				values[i] = decoder.(pgtype.Value).Get()
			case pgx.BinaryFormatCode:
				decoder, ok := value.(pgtype.BinaryDecoder)
				if !ok {
					decoder = &pgtype.GenericBinary{}
				}
				err := decoder.DecodeBinary(connInfo, buf)
				if err != nil {
					//rows.fatal(err)
				}
				values[i] = value.Get()
			default:
				//rows.fatal(errors.New("Unknown format code"))
			}
		} else {
			switch fd.Format {
			case pgx.TextFormatCode:
				decoder := &pgtype.GenericText{}
				err := decoder.DecodeText(connInfo, buf)
				if err != nil {
					//rows.fatal(err)
				}
				values[i] = decoder.Get()
			case pgx.BinaryFormatCode:
				decoder := &pgtype.GenericBinary{}
				err := decoder.DecodeBinary(connInfo, buf)
				if err != nil {
					//rows.fatal(err)
				}
				values[i] = decoder.Get()
			default:
				//rows.fatal(errors.New("Unknown format code"))
			}
		}

		if rows.Err() != nil {
			return rows.Err()
		}
	}

	return rows.Err()
}

func Values2(conn *pgxpool.Conn, rows pgx.Rows, fds []pgproto3.FieldDescription, values []interface{}) error {
	// if rows.closed {
	// 	return nil, errors.New("rows is closed")
	// }

	connInfo := conn.Conn().ConnInfo()
	buf_raw := rows.RawValues()

	for i := range fds {
		buf := buf_raw[i]
		fd := fds[i]

		if buf == nil {
			values[i] = nil
			continue
		}

		if dt, ok := connInfo.DataTypeForOID(fd.DataTypeOID); ok {
			value := dt.Value
			decoder, ok := value.(pgtype.BinaryDecoder)
			if !ok {
				decoder = &pgtype.GenericBinary{}
			}
			err := decoder.DecodeBinary(connInfo, buf)
			if err != nil {
				//rows.fatal(err)
			}
			values[i] = value.Get()

		} else {
			decoder := &pgtype.GenericBinary{}
			err := decoder.DecodeBinary(connInfo, buf)
			if err != nil {
				//rows.fatal(err)
			}
			values[i] = decoder.Get()
		}

		if rows.Err() != nil {
			return rows.Err()
		}
	}

	return rows.Err()
}

func (db *Database) rowsToJSON(sourcename string, rows pgx.Rows) ([]byte, error) {
	source := db.cachedSources[sourcename]
	array := reflect.New(reflect.SliceOf(source.Struct)).Elem()
	//array := reflect.MakeSlice(reflect.SliceOf(source.Struct), 0, 25000)
	//array := []interface{}{}
	struc := reflect.New(source.Struct).Elem()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return []byte{}, err
		}
		//struc := reflect.New(source.Struct).Elem()
		for i, field := range source.Fields {
			f := struc.Field(i)

			switch values[i].(type) {
			case int32:
				v := values[i].(int32)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.SetInt(int64(v))
				}
			case string:
				v := values[i].(string)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.SetString(v)
				}
			case bool:
				f.SetBool(values[i].(bool))
			case time.Time:
				v := values[i].(time.Time)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.Set(reflect.ValueOf(v))
				}
			case nil:
				if !field.NotNull {
					continue
				}
			}
		}
		array = reflect.Append(array, struc)
		//array = append(array, struc)
	}

	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	var w bytes.Buffer
	encoder := json.NewEncoder(&w)
	encoder.Encode(array.Interface())
	//encoder.Encode(array)
	return w.Bytes(), nil
}

func (db *Database) rowsToJSON2(sourcename string, rows pgx.Rows) ([]byte, error) {
	source := db.cachedSources[sourcename]
	array := []interface{}{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return []byte{}, err
		}
		struc := map[string]interface{}{}
		for i, field := range source.Fields {

			struc[field.Name] = values[i]

		}
		array = append(array, struc)
	}

	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	var w bytes.Buffer
	encoder := json.NewEncoder(&w)
	encoder.Encode(array)
	return w.Bytes(), nil
}

func (db *Database) rowsToJSON3(sourcename string, rows pgx.Rows) ([]byte, error) {
	rows.Next()
	values := rows.RawValues()
	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	return values[0], nil
}

func (db *Database) rowsToJSON4(sourcename string, rows pgx.Rows) ([]byte, error) {
	fieldDescriptions := rows.FieldDescriptions()
	var columns []string
	for _, col := range fieldDescriptions {
		columns = append(columns, string(col.Name))
	}

	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	jsonData, _ := json.Marshal(tableData)

	return jsonData, nil
}

func (db *Database) rowsToJSON5(conn *pgxpool.Conn, sourcename string, rows pgx.Rows) ([]byte, error) {
	source := db.cachedSources[sourcename]
	//array := reflect.New(reflect.SliceOf(source.Struct)).Elem()
	array := reflect.MakeSlice(reflect.SliceOf(source.Struct), 0, 50000)
	//array := []interface{}{}
	struc := reflect.New(source.Struct).Elem()
	fds := rows.FieldDescriptions()
	values := make([]interface{}, len(fds))

	for rows.Next() {
		//err := Values(conn, rows, values)
		err := Values2(conn, rows, fds, values)
		if err != nil {
			return []byte{}, err
		}
		for i, field := range source.Fields {
			f := struc.Field(i)

			switch values[i].(type) {
			case int32:
				v := values[i].(int32)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.SetInt(int64(v))
				}
			case string:
				v := values[i].(string)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.SetString(v)
				}
			case bool:
				v := values[i].(bool)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.SetBool(v)
				}
			case time.Time:
				v := values[i].(time.Time)
				if !field.NotNull {
					f.Set(reflect.ValueOf(&v))
				} else {
					f.Set(reflect.ValueOf(v))
				}
			case nil:
				if !field.NotNull {
					continue
				}
			}
		}
		array = reflect.Append(array, struc)
		//array = append(array, struc)
	}

	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	var w bytes.Buffer
	encoder := json.NewEncoder(&w)
	encoder.Encode(array.Interface())
	//encoder.Encode(array)
	//return json.Marshal(array.Interface())
	return w.Bytes(), nil
}

const microsecFromUnixEpochToY2K int64 = 946684800 * 1000000
const secFromUnixEpochToY2K int64 = 946684800
const iso8601 = "2006-01-02T15:04:05Z"

func (db *Database) rowsToJSON7(conn *pgxpool.Conn, sourcename string, rows pgx.Rows) ([]byte, error) {

	fds := rows.FieldDescriptions()
	first := true
	nfields := len(fds)
	var w []byte

	w = append(w, "["...)

	for rows.Next() {
		bufRaw := rows.RawValues()

		if !first {
			w = append(w, ", "...)
		} else {
			first = false
		}
		w = append(w, "{"...)

		for i := range fds {
			buf := bufRaw[i]
			fd := fds[i]

			w = append(w, "\""...)
			w = append(w, fds[i].Name...)
			w = append(w, "\":"...)

			if buf == nil {
				//tmpValue = nil
				continue
			}

			switch fd.DataTypeOID {
			case pgtype.Int4OID:
				w = strconv.AppendInt(w, int64(binary.BigEndian.Uint32(buf)), 10)
			case pgtype.TextOID:
				w = append(w, "\""...)
				w = append(w, buf...)
				w = append(w, "\""...)
			case pgtype.BoolOID:
				if buf[0] == 1 {
					w = append(w, "true"...)
				} else {
					w = append(w, "false"...)
				}
			case pgtype.TimestampOID:
				w = append(w, "\""...)
				microsecSinceY2K := int64(binary.BigEndian.Uint64(buf))
				tim := time.Unix(
					secFromUnixEpochToY2K+microsecSinceY2K/1000000,
					(microsecFromUnixEpochToY2K+microsecSinceY2K)%1000000*1000).UTC()
				w = tim.AppendFormat(w, time.RFC3339Nano)
				w = append(w, "\""...)
			}
			if i < nfields-1 {
				w = append(w, ", "...)
			}
		}
		w = append(w, "}"...)
	}
	w = append(w, "]"...)

	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	return w, nil
}
