package database

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TextSerializer interface {
	Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error)
}

type TextBuilder struct {
	strings.Builder
	dateBuf [128]byte
	date    []byte
}

type SerializeError struct {
	msg string // description of error
}

func (e SerializeError) Error() string { return e.msg }

// appendString is adapted from the standard library

var hex = "0123456789abcdef"

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{' ': true, '!': true, '"': false, '#': true, '$': true, '%': true, '&': true, '\'': true, '(': true, ')': true, '*': true, '+': true, ',': true, '-': true, '.': true, '/': true, '0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true, ':': true, ';': true, '<': true, '=': true, '>': true, '?': true, '@': true, 'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true, '[': true, '\\': false, ']': true, '^': true, '_': true, '`': true, 'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true, 'y': true, 'z': true, '{': true, '|': true, '}': true, '~': true, '\u007F': true}

// htmlSafeSet holds the value true if the ASCII character with the given
// array position can be safely represented inside a JSON string, embedded
// inside of HTML <script> tags, without any additional escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), the backslash character ("\"), HTML opening and closing
// tags ("<" and ">"), and the ampersand ("&").
var htmlSafeSet = [utf8.RuneSelf]bool{' ': true, '!': true, '"': false, '#': true, '$': true, '%': true, '&': false, '\'': true, '(': true, ')': true, '*': true, '+': true, ',': true, '-': true, '.': true, '/': true, '0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true, ':': true, ';': true, '<': false, '=': true, '>': false, '?': true, '@': true, 'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true, '[': true, '\\': false, ']': true, '^': true, '_': true, '`': true, 'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true, 'y': true, 'z': true, '{': true, '|': true, '}': true, '~': true, '\u007f': true}

func (t *TextBuilder) appendString(s []byte, escapeHTML bool) {
	start := 0
	for i := 0; i < len(s); {
		if byt := s[i]; byt < utf8.RuneSelf {
			if htmlSafeSet[byt] || (!escapeHTML && safeSet[byt]) {
				i++
				continue
			}
			if start < i {
				t.Write(s[start:i])
			}
			t.WriteByte('\\')
			switch byt {
			case '\\', '"':
				t.WriteByte(byt)
			case '\n':
				t.WriteByte('n')
			case '\r':
				t.WriteByte('r')
			case '\t':
				t.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				t.WriteString(`u00`)
				t.WriteByte(hex[byt>>4])
				t.WriteByte(hex[byt&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRune(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				t.Write(s[start:i])
			}
			t.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				t.Write(s[start:i])
			}
			t.WriteString(`\u202`)
			t.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		t.Write(s[start:])
	}
}

func (t *TextBuilder) appendInt2(buf []byte) {
	t.WriteString(strconv.FormatInt(int64(toInt16(buf)), 10))
}

func (t *TextBuilder) appendInt4(buf []byte) {
	t.WriteString(strconv.FormatInt(int64(toInt32(buf)), 10))
}

func (t *TextBuilder) appendInt8(buf []byte) {
	t.WriteString(strconv.FormatInt(toInt64(buf), 10))
}

func (t *TextBuilder) appendFloat4(buf []byte) {
	t.WriteString(strconv.FormatFloat(float64(toFloat32(buf)), 'g', -1, 32))
}

func (t *TextBuilder) appendFloat8(buf []byte) {
	t.WriteString(strconv.FormatFloat(toFloat64(buf), 'g', -1, 64))
}

func (t *TextBuilder) appendBool(buf []byte) {
	if toBool(buf) {
		t.WriteString("true")
	} else {
		t.WriteString("false")
	}
}

func (t *TextBuilder) appendTimestamp(buf []byte) {
	tim := toTimestamp(buf)
	t.date = t.dateBuf[:0]
	t.date = tim.AppendFormat(t.date, time.RFC3339Nano)
	t.Write(t.date)
}

func (t *TextBuilder) appendDate(buf []byte) {
	pgDate := toDate(buf)
	t.date = t.dateBuf[:0]
	t.date = pgDate.AppendFormat(t.date, "2006-01-02")
	t.Write(t.date)
}

func (t *TextBuilder) appendInterval(buf []byte) {
	t.WriteByte('"')
	microseconds := int64(binary.BigEndian.Uint64(buf))
	days := int32(binary.BigEndian.Uint32(buf[8:]))
	months := int32(binary.BigEndian.Uint32(buf[12:]))

	years := months / 12
	months %= 12
	seconds := microseconds / 1000000
	minutes := seconds / 60
	seconds %= 60
	hours := minutes / 60
	minutes %= 60

	started := false
	if years > 0 {
		t.WriteString(strconv.FormatInt(int64(years), 10))
		t.WriteString(" year")
		if years > 1 {
			t.WriteByte('s')
		}
		started = true
	}
	if months > 0 {
		if started {
			t.WriteByte(' ')
		}
		t.WriteString(strconv.FormatInt(int64(months), 10))
		t.WriteString(" mon")
		if months > 1 {
			t.WriteByte('s')
		}
		started = true
	}
	if days > 0 {
		if started {
			t.WriteByte(' ')
		}
		t.WriteString(strconv.FormatInt(int64(days), 10))
		t.WriteString(" day")
		if days > 1 {
			t.WriteByte('s')
		}
		started = true
	}
	if hours > 0 || minutes > 0 || seconds > 0 {
		if started {
			t.WriteByte(' ')
		}
		timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
		t.WriteString(timeStr)
	}
	t.WriteByte('"')
}

func (t *TextBuilder) appendJSON(buf []byte) {
	t.Write(buf)
}

type appendTyper interface {
	appendType(buf []byte, typ uint32, info *SchemaInfo) error
}

func (t *TextBuilder) appendArray(buf []byte, typ uint32, info *SchemaInfo, at appendTyper) {
	rp := 0
	numDims := binary.BigEndian.Uint32(buf[rp:])
	if numDims != 1 {

	}
	if numDims == 0 { // @@ this needs to be verified
		t.WriteString("[]")
		return
	}
	rp += 4
	// containsNull := binary.BigEndian.Uint32(buf[rp:]) == 1
	// if containsNull {
	// }
	rp += 4
	elemOID := binary.BigEndian.Uint32(buf[rp:])
	if elemOID != typ {

	}
	rp += 4
	elemCount := int(binary.BigEndian.Uint32(buf[rp:]))
	rp += 4
	//elemLowerBound := int32(binary.BigEndian.Uint32(buf[rp:]))
	rp += 4
	t.WriteByte('[')
	for i := 0; i < elemCount; i++ {
		if i > 0 {
			t.WriteByte(',')
		}
		elemLen := int((binary.BigEndian.Uint32(buf[rp:])))
		rp += 4
		at.appendType(buf[rp:], elemOID, info)
		rp += elemLen
	}
	t.WriteByte(']')
}

// 0 = ()      = 00000
// 1 = empty   = 00001
// 2 = [)      = 00010
// 4 = (]      = 00100
// 6 = []      = 00110
// 8 = )       = 01000
// 12 = ]      = 01100
// 16 = (      = 10000
// 18 = [      = 10010
// 24 =        = 11000

const emptyMask = 1
const lowerInclusiveMask = 2
const upperInclusiveMask = 4
const lowerUnboundedMask = 8
const upperUnboundedMask = 16

func (t *TextBuilder) appendRange(buf []byte, typ uint32, info *SchemaInfo, at appendTyper) {
	t.WriteByte('"')
	rp := 0
	rangeType := buf[rp]
	switch {
	case rangeType&emptyMask > 0,
		rangeType&lowerUnboundedMask > 0:
		// nothing to do
	case rangeType&lowerInclusiveMask > 0:
		t.WriteByte('[')
	default:
		t.WriteByte('(')
	}
	rp += 1
	valuLen1 := binary.BigEndian.Uint32(buf[rp:])
	rp += 4
	at.appendType(buf[rp:], typ, info)
	rp += int(valuLen1)
	t.WriteByte(',')
	valuLen2 := binary.BigEndian.Uint32(buf[rp:])
	rp += 4
	at.appendType(buf[rp:], typ, info)
	rp += int(valuLen2)
	switch {
	case rangeType&emptyMask > 0,
		rangeType&upperUnboundedMask > 0:
		// nothing to do
	case rangeType&upperInclusiveMask > 0:
		t.WriteByte(']')
	default:
		t.WriteByte(')')
	}
	t.WriteByte('"')
}

func (t *TextBuilder) appendComposite(buf []byte, typ *Type, info *SchemaInfo, at appendTyper) {
	rp := 0
	// nfields := binary.BigEndian.Uint32(buf[rp:])
	// if nfields := len(t.SubTypeIds) {

	// }
	rp += 4
	t.WriteByte('{')
	for i, tid := range typ.SubTypeIds {
		if i > 0 {
			t.WriteByte(',')
		}
		oid := binary.BigEndian.Uint32(buf[rp:])
		if oid != tid {

		}
		rp += 4
		len := int(binary.BigEndian.Uint32(buf[rp:]))
		rp += 4

		t.WriteByte('"')
		t.WriteString(typ.SubTypeNames[i])
		t.WriteString("\":")
		at.appendType(buf[rp:], tid, info)
		rp += len
	}

	t.WriteByte('}')
}

func (t *TextBuilder) appendEnum(buf []byte) {
	t.Write(buf)
}

type JSONSerializer struct {
	TextBuilder
}

func (j *JSONSerializer) appendType(buf []byte, typ uint32, info *SchemaInfo) error {
	if buf == nil {
		j.WriteString("null")
		return nil
	}
	switch typ {
	case pgtype.Int2OID:
		j.appendInt2(buf)
	case pgtype.Int4OID, pgtype.OIDOID:
		j.appendInt4(buf)
	case pgtype.Int8OID:
		j.appendInt8(buf)
	case pgtype.Float4OID:
		j.appendFloat4(buf)
	case pgtype.Float8OID:
		j.appendFloat8(buf)
	case pgtype.BoolOID:
		j.appendBool(buf)
	case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID, 3614 /*text search*/ :
		j.WriteByte('"')
		j.appendString(buf, true)
		j.WriteByte('"')
	case pgtype.DateOID:
		j.WriteByte('"')
		j.appendDate(buf)
		j.WriteByte('"')
	case pgtype.TimestampOID, pgtype.TimestamptzOID:
		j.WriteByte('"')
		j.appendTimestamp(buf)
		j.WriteByte('"')
	case pgtype.JSONOID, pgtype.JSONBOID:
		j.appendJSON(buf)
	case pgtype.IntervalOID:
		j.appendInterval(buf)
	default:
		ct := info.GetTypeById(typ)
		if ct != nil {
			switch {
			case ct.IsArray:
				j.appendArray(buf, ct.ArraySubType, info, j)
			case ct.IsRange:
				j.appendRange(buf, *ct.RangeSubType, info, j)
			case ct.IsComposite:
				j.appendComposite(buf, ct, info, j)
			case ct.IsEnum:
				j.WriteByte('"')
				j.appendEnum(buf)
				j.WriteByte('"')
			default:
				return fmt.Errorf("cannot serialize, type not supported (%d)", typ)
			}
		} else {
			return fmt.Errorf("cannot serialize, type not supported (%d)", typ)
		}
	}
	return nil
}

func (j *JSONSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	fds := rows.FieldDescriptions()
	var count int64

	if !single {
		j.WriteByte('[')
	}

	for rows.Next() {
		count++
		bufRaw := rows.RawValues()
		if count > 1 {
			j.WriteByte(',')
		}
		if !scalar {
			j.WriteByte('{')
		}
		for i := range fds {
			buf := bufRaw[i]
			fd := fds[i]
			if i > 0 {
				j.WriteByte(',')
			}
			if !scalar {
				j.WriteByte('"')
				j.appendString([]byte(fd.Name), true)
				j.WriteString("\":")
			}
			j.appendType(buf, fd.DataTypeOID, info)
		}
		if !scalar {
			j.WriteByte('}')
		}
	}
	if !single {
		j.WriteByte(']')
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if single {
		// verify that we have at least one and one only row
		if count != 1 {
			return nil, 0, &SerializeError{}
		}
	}
	return []byte(j.String()), count, nil
}

type CSVSerializer struct {
	TextBuilder
}

func (csv *CSVSerializer) appendType(buf []byte, typ uint32, info *SchemaInfo) error {
	if buf == nil {
		csv.WriteString("")
		return nil
	}
	switch typ {
	case pgtype.Int2OID:
		csv.appendInt2(buf)
	case pgtype.Int4OID, pgtype.OIDOID:
		csv.appendInt4(buf)
	case pgtype.Int8OID:
		csv.appendInt8(buf)
	case pgtype.Float4OID:
		csv.appendFloat4(buf)
	case pgtype.Float8OID:
		csv.appendFloat8(buf)
	case pgtype.BoolOID:
		csv.appendBool(buf)
	case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID, 3614 /*text search*/ :
		csv.appendField(buf) //
	case pgtype.DateOID:
		csv.appendDate(buf)
	case pgtype.TimestampOID, pgtype.TimestamptzOID:
		csv.appendTimestamp(buf)
	case pgtype.JSONOID, pgtype.JSONBOID:
		csv.appendField(buf) //
	case pgtype.IntervalOID:
		csv.appendInterval(buf)
	default:
		ct := info.GetTypeById(typ)
		if ct != nil {
			switch {
			case ct.IsArray:
				csv.appendArray(buf, ct.ArraySubType, info, csv)
			case ct.IsRange:
				csv.appendRange(buf, *ct.RangeSubType, info, csv)
			case ct.IsComposite:
				csv.appendComposite(buf, ct, info, csv)
			case ct.IsEnum:
				csv.appendEnum(buf)
			default:
				return fmt.Errorf("cannot serialize, type not supported (%d)", typ)
			}
		} else {
			return fmt.Errorf("cannot serialize, type not supported (%d)", typ)
		}
	}
	return nil
}

func (csv *CSVSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	fds := rows.FieldDescriptions()
	var count int64

	for i := range fds {
		if i > 0 {
			csv.WriteByte(',')
		}
		csv.appendField([]byte(fds[i].Name))
	}
	csv.WriteByte('\n')
	for rows.Next() {
		count++
		bufRaw := rows.RawValues()
		if count > 1 {
			csv.WriteByte('\n')
		}
		for i := range fds {
			buf := bufRaw[i]
			fd := fds[i]
			if i > 0 {
				csv.WriteByte(',')
			}
			csv.appendType(buf, fd.DataTypeOID, info)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if single {
		// verify that we have at least one and one only row
		if count != 1 {
			return nil, 0, &SerializeError{}
		}
	}
	return []byte(csv.String()), count, nil
}

// @@ adapted from encoding/csv:
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/encoding/csv/writer.go
func (csv *CSVSerializer) appendField(buf []byte) {
	field := string(buf)
	if !csv.fieldNeedsQuotes(field) {
		csv.WriteString(field)
		return
	}
	csv.WriteByte('"')
	for len(field) > 0 {
		// Search for special characters.
		i := strings.IndexByte(field, '"')
		if i < 0 {
			i = len(field)
		}
		// Copy verbatim everything before the special character.
		csv.WriteString(field[:i])
		field = field[i:]
		// Encode the special character.
		if len(field) > 0 {
			csv.WriteString(`""`)
			field = field[1:]
		}
	}
	csv.WriteByte('"')
}

// fieldNeedsQuotes reports whether our field must be enclosed in quotes.
// @@ taken directly and adapted from encoding/csv:
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/encoding/csv/writer.go
func (csv CSVSerializer) fieldNeedsQuotes(field string) bool {
	if field == "" {
		return false
	}
	if field == `\.` {
		return true
	}
	for i := 0; i < len(field); i++ {
		c := field[i]
		if c == '\n' || c == '\r' || c == '"' || c == ',' {
			return true
		}
	}
	r1, _ := utf8.DecodeRuneInString(field)
	return unicode.IsSpace(r1)
}

type BinarySerializer struct {
	bytes.Buffer
}

func (b *BinarySerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	fds := rows.FieldDescriptions()
	if len(fds) != 1 {
		return nil, 0, &SerializeError{msg: "application/octet-stream requested but more than one column was selected"}
	}
	for rows.Next() {
		b.Write(rows.RawValues()[0])
	}
	return b.Bytes(), int64(b.Len()), nil
}

type DatabaseJSONSerializer struct{}

func (DatabaseJSONSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	rows.Next()
	values := rows.RawValues()
	if err := rows.Err(); err != nil {
		return []byte{}, 0, err
	}
	return values[0], 0, nil
}
