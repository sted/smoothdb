package database

import (
	"strconv"
	"strings"
)

func quote(s string) string {
	return strconv.Quote(s)
}

func quoteParts(s string) string {
	parts := strings.Split(s, ".")
	for i := range parts {
		parts[i] = quote(parts[i])
	}
	return strings.Join(parts, ".")
}

func quoteIf(s string, q bool) string {
	if q {
		return quoteParts(s)
	} else {
		return s
	}
}

func normalize(rel, schema, table string, quote bool) string {
	if table != "" {
		rel = table + "." + rel
	}
	if schema != "" {
		rel = schema + "." + rel
	}
	return quoteIf(rel, quote)
}

// _s adds schema
func _s(rel, schema string) string {
	return normalize(rel, schema, "", false)
}

// _sq adds schema and quotes
func _sq(rel, schema string) string {
	return normalize(rel, schema, "", true)
}

// _st adds schema and table
func _st(rel, schema, table string) string {
	return normalize(rel, schema, table, false)
}

// _stq adds schema, table and quotes
func _stq(rel, schema, table string) string {
	return normalize(rel, schema, table, true)
}

func isStar(s string) bool {
	return s == "*"
}

// from golang.org/x/exp/maps
func mapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
