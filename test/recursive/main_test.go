package test_recursive

import (
	"log"
	"os"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/test"
)

var (
	adminToken string
)

func TestMain(m *testing.M) {
	c := map[string]any{
		"Address":             "localhost:8083",
		"AllowAnon":          false,
		"EnableAdminRoute":   true,
		"Database.URL":       "postgresql://postgres:postgres@localhost:5432/postgres",
		"Logging.Level":      "info",
		"Logging.FileLogging": true,
		"Logging.FilePath":   "../../smoothdb.log",
		"Logging.StdOut":     false,
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.jsonc",
			SkipFlags:      true,
		})
	if err != nil {
		log.Fatal(err)
	}

	go s.Start()
	test.WaitForServer("http://localhost:8083")

	postgresToken, _ := authn.GenerateToken("postgres", s.JWTSecret())
	adminToken, _ = authn.GenerateToken("admin", s.JWTSecret())

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8083/admin",
		CommonHeaders: test.Headers{"Authorization": {postgresToken}},
	}

	commands := []test.Command{
		// create role admin (ignore if exists)
		{
			Method: "POST",
			Query:  "/roles",
			Body:   `{"name": "admin", "issuperuser": true, "cancreatedatabases": true}`,
		},
		{
			Method: "POST",
			Query:  "/grants",
			Body:   `{"targetname": "admin", "grantee": "auth"}`,
		},
		// delete and create database
		{
			Method:  "DELETE",
			Query:   "/databases/recursive_test",
			Headers: test.Headers{"Authorization": {adminToken}},
		},
		{
			Method:  "POST",
			Query:   "/databases",
			Body:    `{"name": "recursive_test"}`,
			Headers: test.Headers{"Authorization": {adminToken}},
		},
	}
	test.Prepare(cmdConfig, commands)

	// Create tables
	tableConfig := test.Config{
		BaseUrl:       "http://localhost:8083/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tableCommands := []test.Command{
		// Self-referential table: tree_node
		{
			Method: "POST",
			Query:  "/recursive_test/tables",
			Body: `{
				"name": "tree_node",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true, "constraints": ["PRIMARY KEY"]},
					{"name": "name", "type": "text", "notnull": true},
					{"name": "parent_id", "type": "int4"},
					{"name": "is_active", "type": "boolean", "default": "true"}
				]
			}`,
		},
		// Table for CollHub-like pattern: doc
		{
			Method: "POST",
			Query:  "/recursive_test/tables",
			Body: `{
				"name": "doc",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true, "constraints": ["PRIMARY KEY"]},
					{"name": "name", "type": "text", "notnull": true},
					{"name": "type_name", "type": "text"}
				]
			}`,
		},
		// Relationship table: doc_rel
		{
			Method: "POST",
			Query:  "/recursive_test/tables",
			Body: `{
				"name": "doc_rel",
				"columns": [
					{"name": "src_id", "type": "int4", "notnull": true},
					{"name": "dst_id", "type": "int4", "notnull": true},
					{"name": "rel_type", "type": "text", "notnull": true}
				]
			}`,
		},
	}
	test.Prepare(tableConfig, tableCommands)

	// Create view for CollHub-like graph traversal
	// We need to use RPC or find another way... let's create a view via a function
	// Actually, the admin API doesn't support views directly.
	// We'll use the database API to create a function that simulates a view,
	// or we can test only the self-referential pattern for now.
	// Views can be created via SQL functions if needed.

	// Insert test data
	dataConfig := test.Config{
		BaseUrl:       "http://localhost:8083/api/recursive_test",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	// Tree structure:
	//   1 (CEO)
	//   ├── 2 (VP Eng)
	//   │   ├── 4 (Senior Dev)
	//   │   │   └── 5 (Junior Dev)
	//   │   └── 6 (Inactive Dev, is_active=false)
	//   └── 3 (VP Sales)
	//       └── 7 (Sales Rep)
	dataCommands := []test.Command{
		// Insert root first (no parent)
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 1, "name": "CEO", "parent_id": null, "is_active": true}]`,
		},
		// Insert level 1
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 2, "name": "VP Eng", "parent_id": 1, "is_active": true}, {"id": 3, "name": "VP Sales", "parent_id": 1, "is_active": true}]`,
		},
		// Insert level 2
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 4, "name": "Senior Dev", "parent_id": 2, "is_active": true}, {"id": 6, "name": "Inactive Dev", "parent_id": 2, "is_active": false}]`,
		},
		// Insert level 3
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 5, "name": "Junior Dev", "parent_id": 4, "is_active": true}, {"id": 7, "name": "Sales Rep", "parent_id": 3, "is_active": true}]`,
		},
		// Doc data for CollHub-like pattern
		{
			Method: "POST",
			Query:  "/doc",
			Body:   `[{"id": 1, "name": "Composite", "type_name": "Composite"}, {"id": 2, "name": "Block Hero", "type_name": "Block"}, {"id": 3, "name": "Block CTA", "type_name": "Block"}, {"id": 4, "name": "Hero Title", "type_name": "Fragment"}, {"id": 5, "name": "Hero Text", "type_name": "Fragment"}, {"id": 6, "name": "CTA Text", "type_name": "Fragment"}]`,
		},
		// Relationships: Composite→Blocks→Fragments
		{
			Method: "POST",
			Query:  "/doc_rel",
			Body:   `[{"src_id": 1, "dst_id": 2, "rel_type": "contains"}, {"src_id": 1, "dst_id": 3, "rel_type": "contains"}, {"src_id": 2, "dst_id": 4, "rel_type": "contains"}, {"src_id": 2, "dst_id": 5, "rel_type": "contains"}, {"src_id": 3, "dst_id": 6, "rel_type": "contains"}, {"src_id": 1, "dst_id": 4, "rel_type": "references"}]`,
		},
	}
	test.Prepare(dataConfig, dataCommands)

	code := m.Run()
	os.Exit(code)
}
