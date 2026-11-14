package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_ExtraSearchPath holds the cases of ExtraSearchPathSpec.hs (PostgREST 16.4), converted by hspec2go.
// The spec runs with db-extra-search-path = public, extensions and
// EXTRA "@/\#~_-": the cases run on a variant server with public and
// extensions after test on Database.SchemaSearchPath. The schema
// EXTRA "@/\#~_- is not on it: smoothdb does not quote the entries of
// search_path, and PostgreSQL rejects the name.
func TestPostgREST_ExtraSearchPath(t *testing.T) {
	tests := []test.Test{
		// it "finds the ltree <@ operator on the public schema" $
		//   request methodGet "/ltree_sample?path=cd.Top.Science.Astronomy" [] ""
		//     `shouldRespondWith` [json|[
		//       {"path":"Top.Science.Astronomy"},
		//       {"path":"Top.Science.Astronomy.Astrophysics"},
		//       {"path":"Top.Science.Astronomy.Cosmology"}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "finds the ltree <@ operator on the public schema",
			Method:      "GET",
			Query:       "/ltree_sample?path=cd.Top.Science.Astronomy",
			Expected: `[
        {"path":"Top.Science.Astronomy"},
        {"path":"Top.Science.Astronomy.Astrophysics"},
        {"path":"Top.Science.Astronomy.Cosmology"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "finds the ltree ~ operator on the public schema" $ do
		//   request methodGet "/ltree_sample?path=match.*.Science" [] ""
		//     `shouldRespondWith` [json|[
		//       {"path": "Top.Science"}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "finds the ltree ~ operator on the public schema",
			Method:      "GET",
			Query:       "/ltree_sample?path=match.*.Science",
			Expected: `[
        {"path": "Top.Science"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//
		//   request methodGet "/ltree_sample?path=match.*.Astronomy.*" [] ""
		//     `shouldRespondWith` [json|[
		//       {"path": "Top.Science.Astronomy"},
		//       {"path": "Top.Science.Astronomy.Astrophysics"},
		//       {"path": "Top.Science.Astronomy.Cosmology"},
		//       {"path": "Top.Collections.Pictures.Astronomy"},
		//       {"path": "Top.Collections.Pictures.Astronomy.Stars"},
		//       {"path": "Top.Collections.Pictures.Astronomy.Galaxies"},
		//       {"path": "Top.Collections.Pictures.Astronomy.Astronauts"}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "finds the ltree ~ operator on the public schema",
			Method:      "GET",
			Query:       "/ltree_sample?path=match.*.Astronomy.*",
			Expected: `[
        {"path": "Top.Science.Astronomy"},
        {"path": "Top.Science.Astronomy.Astrophysics"},
        {"path": "Top.Science.Astronomy.Cosmology"},
        {"path": "Top.Collections.Pictures.Astronomy"},
        {"path": "Top.Collections.Pictures.Astronomy.Stars"},
        {"path": "Top.Collections.Pictures.Astronomy.Galaxies"},
        {"path": "Top.Collections.Pictures.Astronomy.Astronauts"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//
		//   request methodGet "/ltree_sample?path=match.*.Collections.*{1,2}" [] ""
		//     `shouldRespondWith` [json|[
		//       {"path": "Top.Collections.Pictures"},
		//       {"path": "Top.Collections.Pictures.Astronomy"}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "finds the ltree ~ operator on the public schema",
			Method:      "GET",
			Query:       "/ltree_sample?path=match.*.Collections.*{1,2}",
			Expected: `[
        {"path": "Top.Collections.Pictures"},
        {"path": "Top.Collections.Pictures.Astronomy"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "finds the ltree nlevel function on the public schema, used through a computed column" $
		//   request methodGet "/ltree_sample?select=number_of_labels&path=eq.Top.Science" [] ""
		//     `shouldRespondWith` [json|[{"number_of_labels":2}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "finds the ltree nlevel function on the public schema, used through a computed column",
			Method:          "GET",
			Query:           "/ltree_sample?select=number_of_labels&path=eq.Top.Science",
			Expected:        `[{"number_of_labels":2}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "finds the isn = operator on the extensions schema" $
		//   request methodGet "/isn_sample?id=eq.978-0-393-04002-9&select=name" [] ""
		//     `shouldRespondWith` [json|[{"name":"Mathematics: From the Birth of Numbers"}]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "finds the isn = operator on the extensions schema",
			Method:          "GET",
			Query:           "/isn_sample?id=eq.978-0-393-04002-9&select=name",
			Expected:        `[{"name":"Mathematics: From the Birth of Numbers"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "finds the isn is_valid function on the extensions schema" $
		//   request methodGet "/rpc/is_valid_isbn?input=978-0-393-04002-9" [] ""
		//     `shouldRespondWith` [json|true|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "finds the isn is_valid function on the extensions schema",
			Method:          "GET",
			Query:           "/rpc/is_valid_isbn?input=978-0-393-04002-9",
			Expected:        `true`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "finds a function on a schema with uppercase and special characters in its name" $
		//   request methodGet "/rpc/special_extended_schema?val=value" [] ""
		//     `shouldRespondWith` [json|"value"|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "finds a function on a schema with uppercase and special characters in its name",
			Method:          "GET",
			Query:           "/rpc/special_extended_schema?val=value",
			Expected:        `"value"`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a schema with quotes and special characters on db-extra-search-path; smoothdb does not quote the entries of search_path, and PostgreSQL rejects the name; no card",
		},
		// it "can detect fk relations through multiple views recursively when middle views are in extra search path" $
		//   get "/consumers_extra_view?select=*,orders_view(*)" `shouldRespondWith` 200
		{
			Description: "can detect fk relations through multiple views recursively when middle views are in extra search path",
			Method:      "GET",
			Query:       "/consumers_extra_view?select=*,orders_view(*)",
			Status:      200,
		},
	}
	withVariant(t, map[string]any{"Database.SchemaSearchPath": []string{"test", "public", "extensions"}}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}
