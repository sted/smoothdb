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

func arrayEquals[T comparable](a1 []T, a2 []T) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i := range a1 {
		if a1[i] != a2[i] {
			return false
		}
	}
	return true
}
