package database

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// The CSV output carries an array or a composite as PostgreSQL's own text
// for it, as PostgREST does: its CSV body is the record text of each row
// (SqlFragment.hs, _postgrest_t::text). These functions write that text from
// the binary wire form: array_out's literal ({1,2}, [0:1]={1,2}) and
// record_out's ((1,"a b")), with their quoting rules (arrayfuncs.c,
// rowtypes.c). A value inside them is PostgreSQL's text for its type: t and
// f for a boolean, a space between date and time. A timestamptz is written in
// UTC (+00), as the rest of the output is (card 38673 timestamptz-utc).

// appendPgText writes the PostgreSQL text of one binary value of type typ.
// scratch holds the elements of an array or the fields of a composite while
// they are quoted; it only grows, each value read from its own start. A nil
// scratch is allocated here when the value needs one.
func appendPgText(t, scratch *TextBuilder, buf []byte, typ uint32, info *SchemaInfo) error {
	switch typ {
	case pgtype.Int2OID:
		t.appendInt2(buf)
	case pgtype.Int4OID, pgtype.OIDOID:
		t.appendInt4(buf)
	case pgtype.Int8OID:
		t.appendInt8(buf)
	case pgtype.Float4OID:
		t.appendFloat4(buf, false)
	case pgtype.Float8OID:
		t.appendFloat8(buf, false)
	case pgtype.BoolOID:
		if toBool(buf) {
			t.WriteByte('t')
		} else {
			t.WriteByte('f')
		}
	case pgtype.TextOID, pgtype.VarcharOID, pgtype.BPCharOID, pgtype.NameOID, pgtype.JSONOID:
		t.Write(buf)
	case pgtype.JSONBOID:
		text, err := jsonbText(buf)
		if err != nil {
			return err
		}
		t.Write(text)
	case pgtype.DateOID:
		t.appendDate(buf)
	case pgtype.TimestampOID:
		appendPgTimestamp(t, buf, "")
	case pgtype.TimestamptzOID:
		appendPgTimestamp(t, buf, "+00")
	case pgtype.UUIDOID:
		var uuid [36]byte
		hex.Encode(uuid[0:8], buf[:4])
		uuid[8] = '-'
		hex.Encode(uuid[9:13], buf[4:6])
		uuid[13] = '-'
		hex.Encode(uuid[14:18], buf[6:8])
		uuid[18] = '-'
		hex.Encode(uuid[19:23], buf[8:10])
		uuid[23] = '-'
		hex.Encode(uuid[24:], buf[10:])
		t.Write(uuid[:])
	case pgtype.NumericOID:
		t.appendNumeric(buf, false)
	default:
		ct := info.GetTypeById(typ)
		if ct == nil {
			return noBinaryDecoder(typ, nil)
		}
		switch {
		case ct.IsDomain && ct.DomainBaseId != 0:
			return appendPgText(t, scratch, buf, ct.DomainBaseId, info)
		case ct.IsArray, ct.IsComposite:
			if scratch == nil {
				scratch = new(TextBuilder)
			}
			if ct.IsArray {
				return appendArrayLiteral(t, scratch, buf, info)
			}
			return appendRecordLiteral(t, scratch, buf, ct, info)
		case ct.IsEnum:
			t.Write(buf)
		default:
			return noBinaryDecoder(typ, ct)
		}
	}
	return nil
}

// appendPgTimestamp writes a binary timestamp as PostgreSQL's text: a space
// between date and time, the fraction without trailing zeros, zone after the
// time, and a BC year with the " BC" suffix at the end.
func appendPgTimestamp(t *TextBuilder, buf []byte, zone string) {
	switch toInt64(buf) { // PostgreSQL's sentinels for ±infinity
	case math.MaxInt64:
		t.WriteString("infinity")
		return
	case math.MinInt64:
		t.WriteString("-infinity")
		return
	}
	tim := toTimestamp(buf)
	t.date = t.dateBuf[:0]
	year := tim.Year()
	if year > 0 {
		t.date = tim.AppendFormat(t.date, "2006")
	} else {
		t.date = appendBCYear(t.date, year)
	}
	t.date = tim.AppendFormat(t.date, "-01-02 15:04:05.999999")
	t.date = append(t.date, zone...)
	if year <= 0 {
		t.date = append(t.date, " BC"...)
	}
	t.Write(t.date)
}

// appendArrayLiteral writes a binary array as array_out does: nested braces
// in row-major order, NULL for a null element, and the bounds of every
// dimension in front ([0:1][2:3]=) when one of them does not start at 1.
func appendArrayLiteral(t, elem *TextBuilder, buf []byte, info *SchemaInfo) error {
	numDims := binary.BigEndian.Uint32(buf)
	if numDims == 0 {
		t.WriteString("{}")
		return nil
	}
	if numDims > maxArrayDims {
		return &SerializeError{msg: "array has more dimensions than PostgreSQL allows"}
	}
	elemOID := binary.BigEndian.Uint32(buf[8:])
	var dims [maxArrayDims]uint32
	var lbounds [maxArrayDims]int32
	rp := uint32(12)
	bounds := false
	for d := uint32(0); d < numDims; d++ {
		dims[d] = binary.BigEndian.Uint32(buf[rp:])
		lbounds[d] = int32(binary.BigEndian.Uint32(buf[rp+4:]))
		rp += 8
		if lbounds[d] != 1 {
			bounds = true
		}
	}
	if bounds {
		for d := uint32(0); d < numDims; d++ {
			t.WriteByte('[')
			t.WriteString(strconv.FormatInt(int64(lbounds[d]), 10))
			t.WriteByte(':')
			t.WriteString(strconv.FormatInt(int64(lbounds[d])+int64(dims[d])-1, 10))
			t.WriteByte(']')
		}
		t.WriteByte('=')
	}
	_, err := appendArrayLiteralDim(t, elem, buf, rp, dims[:numDims], elemOID, info)
	return err
}

func appendArrayLiteralDim(t, elem *TextBuilder, buf []byte, rp uint32, dims []uint32, elemOID uint32, info *SchemaInfo) (uint32, error) {
	t.WriteByte('{')
	for i := uint32(0); i < dims[0]; i++ {
		if i > 0 {
			t.WriteByte(',')
		}
		if len(dims) > 1 {
			var err error
			if rp, err = appendArrayLiteralDim(t, elem, buf, rp, dims[1:], elemOID, info); err != nil {
				return rp, err
			}
			continue
		}
		elemLen := binary.BigEndian.Uint32(buf[rp:])
		rp += 4
		if elemLen == 0xFFFFFFFF {
			t.WriteString("NULL")
			continue
		}
		value := buf[rp : rp+elemLen]
		switch elemOID {
		case pgtype.TextOID, pgtype.VarcharOID, pgtype.BPCharOID, pgtype.NameOID:
			// the wire bytes are the text: quoted from them, no copy
			appendArrayElementBytes(t, value)
		case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID, pgtype.OIDOID,
			pgtype.Float4OID, pgtype.Float8OID, pgtype.BoolOID, pgtype.NumericOID:
			// digits, a sign, a dot, an exponent, NaN, Infinity, t or f:
			// never quoted
			if err := appendPgText(t, nil, value, elemOID, info); err != nil {
				return rp, err
			}
		default:
			// the elements share one buffer, read from where each one
			// starts: Reset would drop the buffer and allocate per element.
			// An element that is itself a composite gets a scratch of its own.
			start := elem.Len()
			if err := appendPgText(elem, nil, value, elemOID, info); err != nil {
				return rp, err
			}
			appendArrayElement(t, elem.String()[start:])
		}
		rp += elemLen
	}
	t.WriteByte('}')
	return rp, nil
}

// appendArrayElement writes one element of an array literal, quoted as
// array_out quotes it: an empty value, the word NULL in any case, and a value
// with a quote, a backslash, a brace, the delimiter or whitespace go in
// double quotes, with a backslash before each quote and backslash.
func appendArrayElement(t *TextBuilder, s string) {
	if !arrayElementNeedsQuotes(s) {
		t.WriteString(s)
		return
	}
	writeQuotedArrayElement(t, s)
}

// appendArrayElementBytes is appendArrayElement for wire bytes, without a
// string conversion.
func appendArrayElementBytes(t *TextBuilder, b []byte) {
	if !arrayElementNeedsQuotes(b) {
		t.Write(b)
		return
	}
	writeQuotedArrayElement(t, b)
}

func arrayElementNeedsQuotes[T string | []byte](s T) bool {
	if len(s) == 0 || len(s) == 4 && (s[0]|0x20) == 'n' && (s[1]|0x20) == 'u' &&
		(s[2]|0x20) == 'l' && (s[3]|0x20) == 'l' {
		return true
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\', '{', '}', ',', ' ', '\t', '\n', '\r', '\v', '\f':
			return true
		}
	}
	return false
}

func writeQuotedArrayElement[T string | []byte](t *TextBuilder, s T) {
	t.WriteByte('"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' || s[i] == '\\' {
			t.WriteByte('\\')
		}
		t.WriteByte(s[i])
	}
	t.WriteByte('"')
}

// appendRecordLiteral writes a binary composite as record_out does: the
// fields in parentheses, nothing for a null field.
func appendRecordLiteral(t, field *TextBuilder, buf []byte, typ *Type, info *SchemaInfo) error {
	nfields := binary.BigEndian.Uint32(buf)
	if int(nfields) != len(typ.SubTypeIds) {
		return &SerializeError{msg: "composite field count does not match the cached type"}
	}
	rp := uint32(4)
	t.WriteByte('(')
	for i := uint32(0); i < nfields; i++ {
		if i > 0 {
			t.WriteByte(',')
		}
		oid := binary.BigEndian.Uint32(buf[rp:])
		flen := binary.BigEndian.Uint32(buf[rp+4:])
		rp += 8
		if flen == 0xFFFFFFFF {
			continue
		}
		start := field.Len() // see appendArrayLiteralDim
		if err := appendPgText(field, nil, buf[rp:rp+flen], oid, info); err != nil {
			return err
		}
		appendRecordField(t, field.String()[start:])
		rp += flen
	}
	t.WriteByte(')')
	return nil
}

// appendRecordField writes one field of a record literal, quoted as
// record_out quotes it: an empty value, and a value with a quote, a
// backslash, a parenthesis, a comma or whitespace (isspace, \v included) go
// in double quotes, with each quote and backslash doubled.
func appendRecordField(t *TextBuilder, s string) {
	quote := s == ""
	for i := 0; !quote && i < len(s); i++ {
		switch s[i] {
		case '"', '\\', '(', ')', ',', ' ', '\t', '\n', '\r', '\v', '\f':
			quote = true
		}
	}
	if !quote {
		t.WriteString(s)
		return
	}
	t.WriteByte('"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' || s[i] == '\\' {
			t.WriteByte(s[i])
		}
		t.WriteByte(s[i])
	}
	t.WriteByte('"')
}
