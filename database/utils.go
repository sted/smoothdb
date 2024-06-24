package database

import (
	"context"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func escapeIdent(identifier string) string {
	return strings.ReplaceAll(identifier, `"`, `""`)
}

func quote(s string) string {
	return "\"" + escapeIdent(s) + "\""
}

func escapeLiteral(identifier string) string {
	return strings.ReplaceAll(identifier, `'`, `''`)
}

func quoteLit(s string) string {
	return "'" + escapeLiteral(s) + "'"
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
		return quote(s)
	} else {
		return s
	}
}
func quotePartsIf(s string, q bool) string {
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
	return quotePartsIf(rel, quote)
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

// splitTableName splits the full table name (schema.table) to get the schema and the table name.
// If the name has no schema, the default configured schema is returned.
func splitTableName(name string) (schemaname, tablename string) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 1 {
		schemaname = dbe.defaultSchema
		tablename = parts[0]
	} else {
		schemaname = parts[0]
		tablename = parts[1]
	}
	return
}

func composeName(ctx context.Context, schemaname, tablename string) string {
	if schemaname == "" {
		options := GetQueryOptions(ctx)
		schemaname = options.Schema
	}
	return quote(schemaname) + "." + quote(tablename)
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

func IsExist(err error) bool {
	pgerr := err.(*pgconn.PgError)
	if pgerr == nil {
		return false
	}
	code := pgerr.Code
	return code == "42P04" || // duplicate database
		code == "42P06" || // duplicate schema
		code == "42P07" || // duplicate table
		code == "23505" || // unique constraint violation
		code == "42710" // duplicate role
}

func QueryStructures[T any](ctx context.Context, query string, values ...any) ([]T, error) {
	conn := GetConn(ctx)
	rows, _ := conn.Query(ctx, query, values...)
	return pgx.CollectRows(rows, pgx.RowToStructByPos[T])
}

func QueryStructure[T any](ctx context.Context, query string, values ...any) (*T, error) {
	conn := GetConn(ctx)
	rows, _ := conn.Query(ctx, query, values...)
	return pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByPos[T])
}

func rowsToStructs[T any](rows pgx.Rows) ([]T, error) {
	return pgx.CollectRows(rows, pgx.RowToStructByPos[T])
}

func QueryExec(ctx context.Context, query string, values ...any) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, query, values...)
	return err
}

type PostgresConfig struct {
	Host     string // host (e.g. localhost) or absolute path to unix domain socket directory (e.g. /private/tmp)
	Port     uint16
	Database string
	User     string
	Password string
}

func ParsePostgresURL(url string) (*PostgresConfig, error) {
	config, err := pgconn.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	return &PostgresConfig{
		Host:     config.Host,
		Port:     config.Port,
		Database: config.Database,
		User:     config.User,
		Password: config.Password,
	}, nil
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
			if val != nil {
				rowValues[i] = make([]byte, len(val))
			}
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
