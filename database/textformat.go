package database

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// The text-format side of the wire-format rule documented on JSONSerializer:
// a value that arrived as PostgreSQL's text is converted by the shape of its
// type, the way to_json converts the value.

// appendText writes the JSON for a value in text format.
func (j *JSONSerializer) appendText(buf []byte, typ uint32, info *SchemaInfo) error {
	if buf == nil {
		j.WriteString("null")
		return nil
	}
	switch typ {
	case pgtype.TextOID, pgtype.VarcharOID, pgtype.BPCharOID, pgtype.NameOID, pgtype.UnknownOID:
		// the common case, kept ahead of the schema cache lookup
		j.appendQuoted(buf)
	case pgtype.JSONOID, pgtype.JSONBOID:
		j.Write(buf)
	case pgtype.BoolOID:
		if len(buf) > 0 && buf[0] == 't' {
			j.WriteString("true")
		} else {
			j.WriteString("false")
		}
	case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID, pgtype.Float4OID, pgtype.Float8OID, pgtype.NumericOID:
		// NaN, Infinity and -Infinity are strings in JSON, as to_json prints them
		if isNumberToken(buf) {
			j.Write(buf)
		} else {
			j.appendQuoted(buf)
		}
	default:
		ct := info.GetTypeById(typ)
		switch {
		case ct != nil && ct.IsArray:
			return j.appendTextArray(buf, ct.ArraySubType, info)
		case ct != nil && ct.IsComposite:
			return j.appendTextComposite(buf, ct, info)
		default:
			j.appendQuoted(buf)
		}
	}
	return nil
}

func (j *JSONSerializer) appendQuoted(buf []byte) {
	j.WriteByte('"')
	j.appendString(buf, true)
	j.WriteByte('"')
}

// isNumberToken reports whether the text of a numeric type is a JSON number:
// an optional minus sign followed by a digit starts one, NaN and Infinity do
// not.
func isNumberToken(buf []byte) bool {
	if len(buf) > 0 && buf[0] == '-' {
		buf = buf[1:]
	}
	return len(buf) > 0 && buf[0] >= '0' && buf[0] <= '9'
}

// appendTextArray converts an array literal into a JSON array, each element
// converted by appendText with the element type. The literal is array_out's:
// {a,"b c",NULL}, a quoted element escaping quotes and backslashes with a
// backslash, nested braces for the dimensions, and a [lb:ub]= prefix when a
// lower bound is not 1.
func (j *JSONSerializer) appendTextArray(buf []byte, elemTyp uint32, info *SchemaInfo) error {
	p := 0
	if p < len(buf) && buf[p] == '[' {
		for p < len(buf) && buf[p] != '=' {
			p++
		}
		p++
	}
	delim := byte(',')
	if elemTyp == pgtype.BoxOID {
		delim = ';'
	}
	p, err := j.appendTextArrayDim(buf, p, delim, elemTyp, info)
	if err != nil {
		return err
	}
	if p != len(buf) {
		return &SerializeError{msg: "malformed array literal"}
	}
	return nil
}

// appendTextArrayDim parses one {...} starting at buf[p], recursing into the
// nested ones, and returns the position after its closing brace.
func (j *JSONSerializer) appendTextArrayDim(buf []byte, p int, delim byte, elemTyp uint32, info *SchemaInfo) (int, error) {
	malformed := &SerializeError{msg: "malformed array literal"}
	if p >= len(buf) || buf[p] != '{' {
		return p, malformed
	}
	p++
	j.WriteByte('[')
	for n := 0; ; n++ {
		for p < len(buf) && buf[p] == ' ' {
			p++
		}
		if p >= len(buf) {
			return p, malformed
		}
		if buf[p] == '}' {
			p++
			break
		}
		if n > 0 {
			if buf[p] != delim {
				return p, malformed
			}
			p++
			for p < len(buf) && buf[p] == ' ' {
				p++
			}
			j.WriteByte(',')
		}
		var err error
		switch {
		case p < len(buf) && buf[p] == '{':
			p, err = j.appendTextArrayDim(buf, p, delim, elemTyp, info)
		case p < len(buf) && buf[p] == '"':
			elem := []byte{}
			for p++; p < len(buf) && buf[p] != '"'; p++ {
				if buf[p] == '\\' {
					p++
				}
				if p < len(buf) {
					elem = append(elem, buf[p])
				}
			}
			if p >= len(buf) {
				return p, malformed
			}
			p++
			err = j.appendText(elem, elemTyp, info)
		default:
			start := p
			for p < len(buf) && buf[p] != delim && buf[p] != '}' {
				p++
			}
			elem := buf[start:p]
			if len(elem) == 4 && strings.EqualFold(string(elem), "NULL") {
				j.WriteString("null")
			} else {
				err = j.appendText(elem, elemTyp, info)
			}
		}
		if err != nil {
			return p, err
		}
	}
	j.WriteByte(']')
	return p, nil
}

// appendTextComposite converts a record literal into a JSON object with the
// cached field names, each field converted by appendText with its type. The
// literal is record_out's: (1,"a b",,"(2,y)"), an empty field for NULL, a
// quoted field escaping a quote by doubling it and a backslash with a
// backslash.
func (j *JSONSerializer) appendTextComposite(buf []byte, typ *Type, info *SchemaInfo) error {
	malformed := &SerializeError{msg: "malformed record literal for type " + typ.Name}
	p := 0
	if p >= len(buf) || buf[p] != '(' {
		return malformed
	}
	p++
	j.WriteByte('{')
	for i, oid := range typ.SubTypeIds {
		if i > 0 {
			if p >= len(buf) || buf[p] != ',' {
				return &SerializeError{msg: "composite field count does not match the cached type"}
			}
			p++
			j.WriteByte(',')
		}
		j.WriteByte('"')
		j.appendString([]byte(typ.SubTypeNames[i]), true)
		j.WriteString("\":")
		var err error
		if p < len(buf) && buf[p] == '"' {
			elem := []byte{}
			for p++; p < len(buf); p++ {
				if buf[p] == '"' {
					if p+1 < len(buf) && buf[p+1] == '"' {
						p++
					} else {
						break
					}
				} else if buf[p] == '\\' {
					p++
				}
				if p < len(buf) {
					elem = append(elem, buf[p])
				}
			}
			if p >= len(buf) {
				return malformed
			}
			p++
			err = j.appendText(elem, oid, info)
		} else {
			start := p
			for p < len(buf) && buf[p] != ',' && buf[p] != ')' {
				p++
			}
			if p == start {
				j.WriteString("null")
			} else {
				err = j.appendText(buf[start:p], oid, info)
			}
		}
		if err != nil {
			return err
		}
	}
	if p >= len(buf) || buf[p] != ')' {
		return &SerializeError{msg: "composite field count does not match the cached type"}
	}
	if p != len(buf)-1 {
		return malformed
	}
	j.WriteByte('}')
	return nil
}
