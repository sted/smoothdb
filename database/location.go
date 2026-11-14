package database

import (
	"strings"

	"github.com/jackc/pgx/v5"
)

// LocationSerializer reads the one row of a write with return=headers-only
// (see headersOnly): the number of rows written and the "key=eq.value" items
// of the Location header. Its output is the query string of the header, with
// the keys and the values percent-encoded as PostgREST does
// (renderSimpleQuery). It is empty when the write is not one row of a
// relation with a key, as in PostgREST (Statements.hs, mainWrite).
type LocationSerializer struct{}

func (LocationSerializer) Serialize(rows pgx.Rows, scalar bool, single bool, info *SchemaInfo) ([]byte, int64, error) {
	var n int64
	var items []string
	if rows.Next() {
		if err := rows.Scan(&n, &items); err != nil {
			return nil, 0, err
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if single && n != 1 {
		return nil, 0, notSingleError(n)
	}
	out := []byte{}
	for _, item := range items {
		key, value, _ := strings.Cut(item, "=")
		if len(out) > 0 {
			out = append(out, '&')
		}
		out = appendQueryEscape(out, key)
		out = append(out, '=')
		out = appendQueryEscape(out, value)
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
	return &CardinalityError{Rows: n}
}
