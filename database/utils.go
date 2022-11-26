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

func _s(rel, schema string) string {
	return normalize(rel, schema, "", false)
}

func _sq(rel, schema string) string {
	return normalize(rel, schema, "", true)
}

func _st(rel, schema, table string) string {
	return normalize(rel, schema, table, false)
}

func _stq(rel, schema, table string) string {
	return normalize(rel, schema, table, true)
}

func isStar(s string) bool {
	if s == "*" {
		return true
	} else {
		return false
	}
}
