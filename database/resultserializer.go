package database

import (
	"context"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

type ResultSerializer interface {
	Serialize(ctx context.Context, rows pgx.Rows) ([]byte, error)
}

type DirectJSONSerializer struct{}

func (DirectJSONSerializer) Serialize(ctx context.Context, rows pgx.Rows) ([]byte, error) {

	fds := rows.FieldDescriptions()
	first := true
	//nfields := len(fds)
	var w []byte

	w = append(w, "["...)

	for rows.Next() {
		bufRaw := rows.RawValues()

		if !first {
			w = append(w, ',')
		} else {
			first = false
		}
		w = append(w, '{')

		for i := range fds {
			buf := bufRaw[i]
			fd := fds[i]

			if i > 0 {
				w = append(w, ',')
			}

			w = append(w, '"')
			w = append(w, fds[i].Name...)
			w = append(w, "\":"...)

			if buf == nil {
				w = append(w, "null"...)
				continue
			}

			switch fd.DataTypeOID {
			case pgtype.Int4OID:
				w = strconv.AppendInt(w, int64(binary.BigEndian.Uint32(buf)), 10)
			case pgtype.Int8OID:
				w = strconv.AppendInt(w, int64(binary.BigEndian.Uint64(buf)), 10)
			case pgtype.TextOID, pgtype.VarcharOID:
				w = append(w, '"')
				w = append(w, buf...)
				w = append(w, '"')
			case pgtype.BoolOID:
				if buf[0] == 1 {
					w = append(w, "true"...)
				} else {
					w = append(w, "false"...)
				}
			case pgtype.TimestampOID:
			case pgtype.TimestamptzOID:
				w = append(w, '"')
				microsecSinceY2K := int64(binary.BigEndian.Uint64(buf))
				tim := time.Unix(
					secFromUnixEpochToY2K+microsecSinceY2K/1000000,
					(microsecFromUnixEpochToY2K+microsecSinceY2K)%1000000*1000).UTC()
				w = tim.AppendFormat(w, time.RFC3339Nano)
				w = append(w, '"')
			default:
				w = append(w, "\"n\\a\""...)
			}
		}
		w = append(w, '}')
	}
	w = append(w, ']')

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return w, nil
}

type DatabaseJSONSerializer struct{}

func (DatabaseJSONSerializer) Serialize(ctx context.Context, rows pgx.Rows) ([]byte, error) {
	rows.Next()
	values := rows.RawValues()
	if err := rows.Err(); err != nil {
		return []byte{}, err
	}
	return values[0], nil
}

type ReflectSerializer struct{}

func (ReflectSerializer) Serialize(ctx context.Context, rows pgx.Rows) ([]byte, error) {
	return []byte{}, nil
}
