package database

import (
	"github.com/jackc/pgx/v5"
)

// LocationSerializer reads the rows of an insert with return=headers-only
// (see locationReturning). Its output is the query string of the Location
// header, "id=eq.1&name=eq.x", when the insert created exactly one row and
// the table has a primary key; it is empty otherwise, as in PostgREST
// (Statements.hs, mainWrite). A NULL key value is "is.null". The keys and
// the values are percent-encoded as PostgREST does (renderSimpleQuery).
type LocationSerializer struct{}

func (LocationSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	var out []byte
	var n int64
	fields := rows.FieldDescriptions()
	for rows.Next() {
		n++
		if n > 1 {
			out = nil
			continue
		}
		values := rows.RawValues()
		for i, f := range fields {
			if f.Name == "?column?" {
				// RETURNING 1: no primary key
				break
			}
			if len(out) > 0 {
				out = append(out, '&')
			}
			out = appendQueryEscape(out, f.Name)
			out = append(out, '=')
			if values[i] == nil {
				out = append(out, "is.null"...)
			} else {
				out = append(out, "eq."...)
				out = appendQueryEscape(out, string(values[i]))
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if single && n != 1 {
		return nil, 0, notSingleError(n)
	}
	if out == nil {
		out = []byte{}
	}
	return out, n, nil
}

// appendQueryEscape appends s percent-encoded: every byte but the unreserved
// characters of RFC 3986 (letters, digits, "-", "_", ".", "~") becomes %XX,
// a space too.
func appendQueryEscape(out []byte, s string) []byte {
	const hex = "0123456789ABCDEF"
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			c == '-', c == '_', c == '.', c == '~':
			out = append(out, c)
		default:
			out = append(out, '%', hex[c>>4], hex[c&15])
		}
	}
	return out
}

// notSingleError is the error of a request for a single object
// (application/vnd.pgrst.object+json) whose result has n rows, n != 1.
func notSingleError(n int64) error {
	return &SerializeError{}
}
