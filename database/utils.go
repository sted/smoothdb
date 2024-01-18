package database

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// compareStruct is used for testing
func compareStruct[T any](dynamicStruct any, realStruct T) bool {
	dynamicVal := reflect.ValueOf(dynamicStruct)
	realVal := reflect.ValueOf(realStruct)

	for i := 0; i < dynamicVal.NumField(); i++ {
		realField := realVal.Field(i)
		if realField.Kind() == reflect.Ptr {
			realField = realField.Elem()
		}
		dynamicField := dynamicVal.Field(i)
		if dynamicField.Kind() == reflect.Ptr {
			dynamicField = dynamicField.Elem()
		}
		if !realField.Equal(dynamicField) {
			return false
		}
	}

	return true
}

// CustomRows is used for testing, to be able to do more iterations on a single query result.
// This is not permitted by pgx.Rows.
type CustomRows struct {
	FieldDescriptions_ []pgconn.FieldDescription
	RawValues_         [][][]byte
	CurrentRow         int
}

func (cr *CustomRows) Next() bool {
	cr.CurrentRow++
	return cr.CurrentRow < len(cr.RawValues_)
}

func (cr *CustomRows) Err() error {
	return nil
}

func (cr *CustomRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (cr *CustomRows) Conn() *pgx.Conn {
	return nil
}

func (cr *CustomRows) Scan(dest ...any) error {
	return nil
}

func (cr *CustomRows) FieldDescriptions() []pgconn.FieldDescription {
	return cr.FieldDescriptions_
}

func (cr *CustomRows) Values() ([]any, error) {
	return nil, nil
}

func (cr *CustomRows) RawValues() [][]byte {
	return cr.RawValues_[cr.CurrentRow]
}

func (cr *CustomRows) Close() {
	cr.CurrentRow = -1
}

func CopyRows(rows pgx.Rows) (*CustomRows, error) {
	// Get the column descriptions
	columns := rows.FieldDescriptions()

	var rawValues [][][]byte

	// Iterate over the rows and read the values
	for rows.Next() {
		rowValues := make([][]byte, len(columns))

		// Copying the values from the current row
		for i, val := range rows.RawValues() {
			rowValues[i] = make([]byte, len(val))
			copy(rowValues[i], val)
		}

		rawValues = append(rawValues, rowValues)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Create a new CustomRows instance with the extracted values and column descriptions
	customRows := &CustomRows{
		FieldDescriptions_: columns,
		RawValues_:         rawValues,
		CurrentRow:         -1,
	}

	return customRows, nil
}
