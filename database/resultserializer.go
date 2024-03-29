package database

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const microsecFromUnixEpochToY2K int64 = 946684800 * 1000000
const secFromUnixEpochToY2K int64 = 946684800

type ResultSerializer interface {
	Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error)
}

type DirectJSONSerializer struct {
	strings.Builder
	dateBuf [128]byte
	date    []byte
}

type SerializeError struct {
	msg string // description of error
}

func (e *SerializeError) Error() string { return e.msg }

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

func (d *DirectJSONSerializer) appendString(s []byte, escapeHTML bool) {
	d.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] || (!escapeHTML && safeSet[b]) {
				i++
				continue
			}
			if start < i {
				d.Write(s[start:i])
			}
			d.WriteByte('\\')
			switch b {
			case '\\', '"':
				d.WriteByte(b)
			case '\n':
				d.WriteByte('n')
			case '\r':
				d.WriteByte('r')
			case '\t':
				d.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				d.WriteString(`u00`)
				d.WriteByte(hex[b>>4])
				d.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRune(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				d.Write(s[start:i])
			}
			d.WriteString(`\ufffd`)
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
				d.Write(s[start:i])
			}
			d.WriteString(`\u202`)
			d.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		d.Write(s[start:])
	}
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendInt2(buf []byte) {
	d.WriteString(strconv.FormatInt(int64(toInt16(buf)), 10))
}

func (d *DirectJSONSerializer) appendInt4(buf []byte) {
	d.WriteString(strconv.FormatInt(int64(toInt32(buf)), 10))
}

func (d *DirectJSONSerializer) appendInt8(buf []byte) {
	d.WriteString(strconv.FormatInt(toInt64(buf), 10))
}

func (d *DirectJSONSerializer) appendFloat4(buf []byte) {
	d.WriteString(strconv.FormatFloat(float64(toFloat32(buf)), 'g', -1, 32))
}

func (d *DirectJSONSerializer) appendFloat8(buf []byte) {
	d.WriteString(strconv.FormatFloat(toFloat64(buf), 'g', -1, 64))
}

func (d *DirectJSONSerializer) appendBool(buf []byte) {
	if toBool(buf) {
		d.WriteString("true")
	} else {
		d.WriteString("false")
	}
}

func (d *DirectJSONSerializer) appendTime(buf []byte) {
	d.WriteByte('"')
	tim := toTime(buf)
	d.date = d.dateBuf[:0]
	d.date = tim.AppendFormat(d.date, time.RFC3339Nano)
	d.Write(d.date)
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendInterval(buf []byte) {
	d.WriteByte('"')
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
		d.WriteString(strconv.FormatInt(int64(years), 10))
		d.WriteString(" year")
		if years > 1 {
			d.WriteByte('s')
		}
		started = true
	}
	if months > 0 {
		if started {
			d.WriteByte(' ')
		}
		d.WriteString(strconv.FormatInt(int64(months), 10))
		d.WriteString(" mon")
		if months > 1 {
			d.WriteByte('s')
		}
		started = true
	}
	if days > 0 {
		if started {
			d.WriteByte(' ')
		}
		d.WriteString(strconv.FormatInt(int64(days), 10))
		d.WriteString(" day")
		if days > 1 {
			d.WriteByte('s')
		}
		started = true
	}
	if hours > 0 || minutes > 0 || seconds > 0 {
		if started {
			d.WriteByte(' ')
		}
		timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
		d.WriteString(timeStr)
	}
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendJSON(buf []byte) {
	d.Write(buf)
}

func (d *DirectJSONSerializer) appendArray(buf []byte, t uint32, info *SchemaInfo) {
	rp := 0
	numDims := binary.BigEndian.Uint32(buf[rp:])
	if numDims != 1 {

	}
	rp += 4
	// containsNull := binary.BigEndian.Uint32(buf[rp:]) == 1
	// if containsNull {
	// }
	rp += 4
	elemOID := binary.BigEndian.Uint32(buf[rp:])
	if elemOID != t {

	}
	rp += 4
	elemCount := int(binary.BigEndian.Uint32(buf[rp:]))
	rp += 4
	//elemLowerBound := int32(binary.BigEndian.Uint32(buf[rp:]))
	rp += 4
	d.WriteByte('[')
	for i := 0; i < elemCount; i++ {
		if i > 0 {
			d.WriteByte(',')
		}
		elemLen := int((binary.BigEndian.Uint32(buf[rp:])))
		rp += 4
		d.appendType(buf[rp:], elemOID, info)
		rp += elemLen
	}
	d.WriteByte(']')
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

func (d *DirectJSONSerializer) appendRange(buf []byte, t uint32, info *SchemaInfo) {
	d.WriteByte('"')
	rp := 0
	rangeType := buf[rp]
	switch {
	case rangeType&emptyMask > 0,
		rangeType&lowerUnboundedMask > 0:
		// nothing to do
	case rangeType&lowerInclusiveMask > 0:
		d.WriteByte('[')
	default:
		d.WriteByte('(')
	}
	rp += 1
	valuLen1 := binary.BigEndian.Uint32(buf[rp:])
	rp += 4
	d.appendType(buf[rp:], t, info)
	rp += int(valuLen1)
	d.WriteByte(',')
	valuLen2 := binary.BigEndian.Uint32(buf[rp:])
	rp += 4
	d.appendType(buf[rp:], t, info)
	rp += int(valuLen2)
	switch {
	case rangeType&emptyMask > 0,
		rangeType&upperUnboundedMask > 0:
		// nothing to do
	case rangeType&upperInclusiveMask > 0:
		d.WriteByte(']')
	default:
		d.WriteByte(')')
	}
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendComposite(buf []byte, t *Type, info *SchemaInfo) {
	rp := 0
	// nfields := binary.BigEndian.Uint32(buf[rp:])
	// if nfields := len(t.SubTypeIds) {

	// }
	rp += 4
	d.WriteByte('{')
	for i, tid := range t.SubTypeIds {
		if i > 0 {
			d.WriteByte(',')
		}
		oid := binary.BigEndian.Uint32(buf[rp:])
		if oid != tid {

		}
		rp += 4
		len := int(binary.BigEndian.Uint32(buf[rp:]))
		rp += 4

		d.WriteByte('"')
		d.WriteString(t.SubTypeNames[i])
		d.WriteString("\":")
		d.appendType(buf[rp:], tid, info)
		rp += len
	}

	d.WriteByte('}')
}

func (d *DirectJSONSerializer) appendEnum(buf []byte) {
	d.WriteByte('"')
	d.Write(buf)
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendTextSearch(buf []byte) {
	d.WriteByte('"')
	d.Write(buf)
	d.WriteByte('"')
}

func (d *DirectJSONSerializer) appendType(buf []byte, t uint32, info *SchemaInfo) error {
	if buf == nil {
		d.WriteString("null")
		return nil
	}
	switch t {
	case pgtype.Int2OID:
		d.appendInt2(buf)
	case pgtype.Int4OID, pgtype.OIDOID:
		d.appendInt4(buf)
	case pgtype.Int8OID:
		d.appendInt8(buf)
	case pgtype.Float4OID:
		d.appendFloat4(buf)
	case pgtype.Float8OID:
		d.appendFloat8(buf)
	case pgtype.BoolOID:
		d.appendBool(buf)
	case pgtype.TextOID, pgtype.VarcharOID, pgtype.NameOID:
		d.appendString(buf, true)
	case pgtype.TimestampOID, pgtype.TimestamptzOID:
		d.appendTime(buf)
	case pgtype.JSONOID, pgtype.JSONBOID:
		d.appendJSON(buf)
	case pgtype.IntervalOID:
		d.appendInterval(buf)
	case 3614: // text search
		d.appendTextSearch(buf)
	default:
		ct := info.GetTypeById(t)
		if ct != nil {
			switch {
			case ct.IsArray:
				d.appendArray(buf, ct.ArraySubType, info)
			case ct.IsRange:
				d.appendRange(buf, *ct.RangeSubType, info)
			case ct.IsComposite:
				d.appendComposite(buf, ct, info)
			case ct.IsEnum:
				d.appendEnum(buf)
			default:
				return fmt.Errorf("cannot serialize, type not supported (%d)", t)
			}
		} else {
			return fmt.Errorf("cannot serialize, type not supported (%d)", t)
		}
	}
	return nil
}

func (d *DirectJSONSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	fds := rows.FieldDescriptions()
	var count int64

	if !single {
		d.WriteByte('[')
	}

	for rows.Next() {
		count++
		bufRaw := rows.RawValues()
		if count > 1 {
			d.WriteByte(',')
		}
		if !scalar {
			d.WriteByte('{')
		}
		for i := range fds {
			buf := bufRaw[i]
			fd := fds[i]
			if i > 0 {
				d.WriteByte(',')
			}
			if !scalar {
				d.appendString([]byte(fds[i].Name), true)
				d.WriteByte(':')
			}
			d.appendType(buf, fd.DataTypeOID, info)
		}
		if !scalar {
			d.WriteByte('}')
		}
	}
	if !single {
		d.WriteByte(']')
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
	return []byte(d.String()), count, nil
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
