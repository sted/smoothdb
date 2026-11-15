package database

import "testing"

// TestSchemaExposed: a Profile header is checked only against
// Database.ExposedSchemas (PostgREST db-schemas); with none configured every
// schema is reachable, whatever the search path (ragtool exposes collhub,
// sources and cron with SchemaSearchPath = [extensions]).
func TestSchemaExposed(t *testing.T) {
	tests := []struct {
		schema  string
		exposed []string
		errMsg  string
		hint    string
	}{
		{"collhub", nil, "", ""},
		{"collhub", []string{}, "", ""},
		{"test", []string{"test", "تست"}, "", ""},
		{"تست", []string{"test", "تست"}, "", ""},
		{"unknown", []string{"test", "تست"}, "Invalid schema: unknown", "Only the following schemas are exposed: test, تست"},
		{"extensions", []string{"collhub"}, "Invalid schema: extensions", "Only the following schemas are exposed: collhub"},
		// with no list, the system schemas stay closed
		{"pg_catalog", nil, "Invalid schema: pg_catalog", systemSchemaHint},
		{"information_schema", nil, "Invalid schema: information_schema", systemSchemaHint},
		{"pg_toast", []string{}, "Invalid schema: pg_toast", systemSchemaHint},
		{"pg_temp", nil, "Invalid schema: pg_temp", systemSchemaHint},
		{"pg_temp_3", nil, "Invalid schema: pg_temp_3", systemSchemaHint},
		// PostgreSQL reserves only the lower-case prefix
		{"PG_data", nil, "", ""},
		{"pgdata", nil, "", ""},
		// a list that names a system schema exposes it
		{"pg_catalog", []string{"api", "pg_catalog"}, "", ""},
		{"information_schema", []string{"api"}, "Invalid schema: information_schema", "Only the following schemas are exposed: api"},
	}
	for i, test := range tests {
		err := schemaExposed(test.schema, test.exposed)
		if test.errMsg == "" {
			if err != nil {
				t.Errorf("%d. unexpected error for %q in %v: %v", i, test.schema, test.exposed, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%d. expected an error for %q in %v", i, test.schema, test.exposed)
			continue
		}
		if err.Error() != test.errMsg || err.Hint != test.hint {
			t.Errorf("%d. expected %q / %q, got %q / %q", i, test.errMsg, test.hint, err.Error(), err.Hint)
		}
	}
}

// TestDefaultSchemaFromConfig: the default schema is the first exposed one
// when ExposedSchemas is set, else the first of the search path, else public.
func TestDefaultSchemaFromConfig(t *testing.T) {
	tests := []struct {
		exposed, searchPath []string
		want                string
	}{
		{nil, nil, "public"},
		{nil, []string{"extensions"}, "extensions"},
		{[]string{"collhub", "sources"}, []string{"extensions"}, "collhub"},
		{[]string{"test", "تست"}, nil, "test"},
	}
	for i, test := range tests {
		got := defaultSchemaFromConfig(&Config{ExposedSchemas: test.exposed, SchemaSearchPath: test.searchPath})
		if got != test.want {
			t.Errorf("%d. expected %q, got %q", i, test.want, got)
		}
	}
}
