package test_recursive

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
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
		// Child table with an FK to tree_node — lets us embed a related resource while
		// recursing (single-table recursion composes with embedding).
		{
			Method: "POST",
			Query:  "/recursive_test/tables",
			Body: `{
				"name": "node_tag",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true, "constraints": ["PRIMARY KEY"]},
					{"name": "node_id", "type": "int4", "notnull": true, "constraints": ["REFERENCES tree_node (id)"]},
					{"name": "tag", "type": "text", "notnull": true}
				]
			}`,
		},
		// Relationship table: doc_rel — FKs to doc make doc<->doc_rel an embeddable
		// relationship, which the via()+embed rejection test relies on.
		{
			Method: "POST",
			Query:  "/recursive_test/tables",
			Body: `{
				"name": "doc_rel",
				"columns": [
					{"name": "src_id", "type": "int4", "notnull": true, "constraints": ["REFERENCES doc (id)"]},
					{"name": "dst_id", "type": "int4", "notnull": true, "constraints": ["REFERENCES doc (id)"]},
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
		// Isolated subtree with an ACTIVE node beneath an INACTIVE one. Used to tell
		// result filters (keep 102) apart from walk-prune filters (102 unreachable).
		//   100 Region (active) -> 101 Closed Office (inactive) -> 102 Remote Worker (active)
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 100, "name": "Region", "parent_id": null, "is_active": true}]`,
		},
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 101, "name": "Closed Office", "parent_id": 100, "is_active": false}]`,
		},
		{
			Method: "POST",
			Query:  "/tree_node",
			Body:   `[{"id": 102, "name": "Remote Worker", "parent_id": 101, "is_active": true}]`,
		},
		// Tags hanging off tree nodes (one tag each, for a deterministic embed payload).
		{
			Method: "POST",
			Query:  "/node_tag",
			Body:   `[{"id": 1, "node_id": 1, "tag": "exec"}, {"id": 2, "node_id": 2, "tag": "eng"}]`,
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

	// Computed relationship: a function taking a tree_node ROW and returning its
	// node_tag rows. The admin API can't create functions, so run raw SQL on a
	// connection. This lets us test that COMPUTED-relationship embeds (function of
	// the row) compose with recursion — the case the `__row` fix repairs.
	{
		db, err := s.GetDBE().GetOrCreateActiveDatabase(context.Background(), "recursive_test")
		if err != nil {
			log.Fatal(err)
		}
		dbCtx, dbConn, err := database.ContextWithDb(context.Background(), db, "admin")
		if err != nil {
			log.Fatal(err)
		}
		gi := database.GetSmoothContext(dbCtx)
		_, err = gi.Conn.Exec(dbCtx, `
			CREATE FUNCTION node_tags(tree_node)
			RETURNS SETOF node_tag
			LANGUAGE sql STABLE
			AS $$ SELECT * FROM node_tag WHERE node_id = $1.id $$;
		`)
		if err != nil {
			log.Fatal(err)
		}
		database.ReleaseConn(dbCtx, dbConn)
	}

	// Tables and FKs above are created at runtime through the admin API, which does
	// not refresh the relationship cache. Reload it so embedded resources resolve.
	if err := s.GetDBE().ReloadDatabaseSchema(context.Background(), "recursive_test"); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	os.Exit(code)
}
