package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_ComputedRels holds the cases of ComputedRelsSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_ComputedRels(t *testing.T) {
	tests := []test.Test{
		// it "can define a many-to-one relationship with SETOF and ROWS 1 and embed" $
		//   get "/videogames?select=name,designer:computed_designers(name)"
		//     `shouldRespondWith` [json|[
		//       {"name":"Civilization I","designer":{"name":"Sid Meier"}},
		//       {"name":"Civilization II","designer":{"name":"Sid Meier"}},
		//       {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
		//       {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "can define a many-to-one relationship with SETOF and ROWS 1 and embed",
			Method:      "GET",
			Query:       "/videogames?select=name,designer:computed_designers(name)",
			Expected: `[
        {"name":"Civilization I","designer":{"name":"Sid Meier"}},
        {"name":"Civilization II","designer":{"name":"Sid Meier"}},
        {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
        {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "can define a many-to-one relationship without SETOF and embed" $
		//   get "/videogames?select=name,designer:computed_designers_noset(name)"
		//     `shouldRespondWith` [json|[
		//       {"name":"Civilization I","designer":{"name":"Sid Meier"}},
		//       {"name":"Civilization II","designer":{"name":"Sid Meier"}},
		//       {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
		//       {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "can define a many-to-one relationship without SETOF and embed",
			Method:      "GET",
			Query:       "/videogames?select=name,designer:computed_designers_noset(name)",
			Expected: `[
        {"name":"Civilization I","designer":{"name":"Sid Meier"}},
        {"name":"Civilization II","designer":{"name":"Sid Meier"}},
        {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
        {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "can define a one-to-many relationship and embed" $
		//   get "/designers?select=name,videogames:computed_videogames(name)"
		//     `shouldRespondWith` [json|[
		//       {"name":"Sid Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]},
		//       {"name":"Hironobu Sakaguchi","videogames":[{"name":"Final Fantasy I"}, {"name":"Final Fantasy II"}]}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "can define a one-to-many relationship and embed",
			Method:      "GET",
			Query:       "/designers?select=name,videogames:computed_videogames(name)",
			Expected: `[
        {"name":"Sid Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]},
        {"name":"Hironobu Sakaguchi","videogames":[{"name":"Final Fantasy I"}, {"name":"Final Fantasy II"}]}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "works with !inner and count=exact" $ do
		//   request
		//     methodGet
		//     "/designers?select=name,videogames:computed_videogames!inner(name)&videogames.name=eq.Civilization%20I"
		//     [("Prefer", "count=exact")]
		//     ""
		//     `shouldRespondWith` [json|[{"name":"Sid Meier","videogames":[{"name":"Civilization I"}]}]|]
		//       { matchStatus = 200
		//       , matchHeaders = ["Content-Range" <:> "0-0/1"]
		//       }
		{
			Description:     "works with !inner and count=exact",
			Method:          "GET",
			Query:           "/designers?select=name,videogames:computed_videogames!inner(name)&videogames.name=eq.Civilization%20I",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"name":"Sid Meier","videogames":[{"name":"Civilization I"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		//   request
		//     methodGet
		//     "/videogames?select=name,designer:computed_designers!inner(name)&designer.name=like.*Hironobu*"
		//     [("Prefer", "count=exact")]
		//     ""
		//     `shouldRespondWith` [json|[
		//         {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
		//         {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
		//       ]|]
		//       { matchStatus = 200
		//       , matchHeaders = ["Content-Range" <:> "0-1/2"]
		//       }
		{
			Description: "works with !inner and count=exact",
			Method:      "GET",
			Query:       "/videogames?select=name,designer:computed_designers!inner(name)&designer.name=like.*Hironobu*",
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Expected: `[
          {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
          {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
        ]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		//   request
		//     methodGet
		//     "/videogames?select=name,designer:computed_designers_noset!inner(name)&designer.name=like.*Hironobu*"
		//     [("Prefer", "count=exact")]
		//     ""
		//     `shouldRespondWith` [json|[
		//         {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
		//         {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
		//       ]|]
		//       { matchStatus = 200
		//       , matchHeaders = ["Content-Range" <:> "0-1/2"]
		//       }
		{
			Description: "works with !inner and count=exact",
			Method:      "GET",
			Query:       "/videogames?select=name,designer:computed_designers_noset!inner(name)&designer.name=like.*Hironobu*",
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Expected: `[
          {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
          {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
        ]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// it "works with rpc" $ do
		//   get "/rpc/getallvideogames?select=name,designer:computed_designers(name)"
		//     `shouldRespondWith` [json|[
		//       {"name":"Civilization I","designer":{"name":"Sid Meier"}},
		//       {"name":"Civilization II","designer":{"name":"Sid Meier"}},
		//       {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
		//       {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "works with rpc",
			Method:      "GET",
			Query:       "/rpc/getallvideogames?select=name,designer:computed_designers(name)",
			Expected: `[
        {"name":"Civilization I","designer":{"name":"Sid Meier"}},
        {"name":"Civilization II","designer":{"name":"Sid Meier"}},
        {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}},
        {"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//   get "/rpc/getalldesigners?select=name,videogames:computed_videogames(name)"
		//     `shouldRespondWith` [json|[
		//       {"name":"Sid Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]},
		//       {"name":"Hironobu Sakaguchi","videogames":[{"name":"Final Fantasy I"}, {"name":"Final Fantasy II"}]}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "works with rpc",
			Method:      "GET",
			Query:       "/rpc/getalldesigners?select=name,videogames:computed_videogames(name)",
			Expected: `[
        {"name":"Sid Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]},
        {"name":"Hironobu Sakaguchi","videogames":[{"name":"Final Fantasy I"}, {"name":"Final Fantasy II"}]}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "works with mutations" $ do
		//   request
		//     methodPost
		//     "/videogames?select=name,designer:computed_designers(name)"
		//     [("Prefer", "return=representation")]
		//     [json| {"id": 5, "name": "Chrono Trigger", "designer_id": 2} |]
		//     `shouldRespondWith` [json|[ {"name":"Chrono Trigger","designer":{"name":"Hironobu Sakaguchi"}} ]|]
		//       { matchStatus = 201
		//       }
		{
			Description: "works with mutations",
			Method:      "POST",
			Query:       "/videogames?select=name,designer:computed_designers(name)",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `{"id": 5, "name": "Chrono Trigger", "designer_id": 2}`,
			Expected:    `[ {"name":"Chrono Trigger","designer":{"name":"Hironobu Sakaguchi"}} ]`,
			Status:      201,
			Skip:        "PostgREST 16.4: a computed relationship embeds on the RETURNING of a write, smoothdb fails with 42846; card 32484 (embedding remainder)",
		},
		//   request
		//     methodPatch
		//     "/designers?select=name,videogames:computed_videogames(name)&id=eq.1"
		//     [("Prefer", "return=representation")]
		//     [json| {"name": "Sidney K. Meier"} |]
		//     `shouldRespondWith` [json|[ { "name": "Sidney K. Meier", "videogames": [{"name":"Civilization I"}, {"name":"Civilization II"}] } ]|]
		//       { matchStatus = 200
		//       }
		{
			Description: "works with mutations",
			Method:      "PATCH",
			Query:       "/designers?select=name,videogames:computed_videogames(name)&id=eq.1",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `{"name": "Sidney K. Meier"}`,
			Expected:    `[ { "name": "Sidney K. Meier", "videogames": [{"name":"Civilization I"}, {"name":"Civilization II"}] } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: a computed relationship embeds on the RETURNING of a write, smoothdb fails with 42846; card 32484 (embedding remainder)",
		},
		//   request
		//     methodDelete
		//     "/videogames?select=name,designer:computed_designers(name)&id=eq.3"
		//     [("Prefer", "return=representation")]
		//     ""
		//     `shouldRespondWith` [json|[ {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}} ]|]
		//       { matchStatus = 200
		//       }
		{
			Description: "works with mutations",
			Method:      "DELETE",
			Query:       "/videogames?select=name,designer:computed_designers(name)&id=eq.3",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ {"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}} ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: a computed relationship embeds on the RETURNING of a write, smoothdb fails with 42846; card 32484 (embedding remainder)",
		},
		// it "applies data representations to response" $ do
		//   -- A smoke test for data reps in the presence of computed relations.
		//
		//   -- The data rep here title cases the designer name before presentation. So here the lowercase version will be saved,
		//   -- but the title case version returned. Pulling in a computed relation should not confuse this.
		//   request
		//     methodPatch
		//     "/designers?select=name,videogames:computed_videogames(name)&id=eq.1"
		//     [("Prefer", "return=representation"), ("Prefer", "tx=commit")]
		//     [json| {"name": "sidney k. meier"} |]
		//     `shouldRespondWith` [json|[{"name":"Sidney K. Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]}]|]
		//       { matchStatus = 200
		//       }
		{
			Description: "applies data representations to response",
			Method:      "PATCH",
			Query:       "/designers?select=name,videogames:computed_videogames(name)&id=eq.1",
			Headers:     test.Headers{"Prefer": {"return=representation", "tx=commit"}},
			Body:        `{"name": "sidney k. meier"}`,
			Expected:    `[{"name":"Sidney K. Meier","videogames":[{"name":"Civilization I"}, {"name":"Civilization II"}]}]`,
			Status:      200,
			Skip:        "PostgREST 16.4: a computed relationship embeds on the RETURNING of a write, smoothdb fails with 42846; card 32484 (embedding remainder)",
		},
		//
		//   -- Verify it was saved the way we requested (there's no text data rep for this column, so if we select with the wrong casing, it should fail.)
		//   get "/designers?select=id&name=eq.Sidney%20K.%20Meier"
		//     `shouldRespondWith` [json|[]|]
		//       { matchStatus = 200
		//       , matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "applies data representations to response",
			Method:          "GET",
			Query:           "/designers?select=id&name=eq.Sidney%20K.%20Meier",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//   -- But with the right casing it works.
		//   get "/designers?select=id,name&name=eq.sidney%20k.%20meier"
		//     `shouldRespondWith` [json|[{"id": 1, "name":"Sidney K. Meier"}]|]
		//       { matchStatus = 200
		//       , matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "applies data representations to response",
			Method:          "GET",
			Query:           "/designers?select=id,name&name=eq.sidney%20k.%20meier",
			Expected:        `[{"id": 1, "name":"Sidney K. Meier"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: data representations apply to a computed relationship; card 32484 (embedding remainder)",
		},
		//
		//   -- Most importantly, if you read it back even via a computed relation, the data rep should be applied.
		//   get "/videogames?select=name,designer:computed_designers(*)&id=eq.1"
		//     `shouldRespondWith` [json|[
		//       {"name":"Civilization I","designer":{"id": 1, "name":"Sidney K. Meier"}}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "applies data representations to response",
			Method:      "GET",
			Query:       "/videogames?select=name,designer:computed_designers(*)&id=eq.1",
			Expected: `[
        {"name":"Civilization I","designer":{"id": 1, "name":"Sidney K. Meier"}}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: data representations apply to a computed relationship; card 32484 (embedding remainder)",
		},
		//
		//   -- reset the test fixture
		//   request
		//     methodPatch
		//     "/designers?id=eq.1"
		//     [("Prefer", "tx=commit")]
		//     [json| {"name": "Sid Meier"} |]
		//     `shouldRespondWith` 204
		{
			Description: "applies data representations to response",
			Method:      "PATCH",
			Query:       "/designers?id=eq.1",
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
			Body:        `{"name": "Sid Meier"}`,
			Status:      204,
		},
		//   -- need to poke the second one too to prevent inherent ordering from changing
		//   request
		//     methodPatch
		//     "/designers?id=eq.2"
		//     [("Prefer", "tx=commit")]
		//     [json| {"name": "Hironobu Sakaguchi"} |]
		//     `shouldRespondWith` 204
		{
			Description: "applies data representations to response",
			Method:      "PATCH",
			Query:       "/designers?id=eq.2",
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
			Body:        `{"name": "Hironobu Sakaguchi"}`,
			Status:      204,
		},
		// it "works with self joins" $
		//   get "/web_content?select=name,child_web_content(name),parent_web_content(name)&id=in.(0,1)"
		//     `shouldRespondWith` [json|[
		//       {"name":"tardis","child_web_content":[{"name":"fezz"}, {"name":"foo"}, {"name":"bar"}],"parent_web_content":{"name":"wat"}},
		//       {"name":"fezz","child_web_content":[{"name":"wut"}],"parent_web_content":{"name":"tardis"}}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "works with self joins",
			Method:      "GET",
			Query:       "/web_content?select=name,child_web_content(name),parent_web_content(name)&id=in.(0,1)",
			Expected: `[
        {"name":"tardis","child_web_content":[{"name":"fezz"}, {"name":"foo"}, {"name":"bar"}],"parent_web_content":{"name":"wat"}},
        {"name":"fezz","child_web_content":[{"name":"wut"}],"parent_web_content":{"name":"tardis"}}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "can override many-to-one and one-to-many relationships" $ do
		//   get "/videogames?select=*,designers!inner(*)"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "can override many-to-one and one-to-many relationships",
			Method:          "GET",
			Query:           "/videogames?select=*,designers!inner(*)",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a computed relationship overrides the foreign key relationship of the same name; card 32484 (embedding remainder)",
		},
		//   get "/designers?select=*,videogames!inner(*)"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "can override many-to-one and one-to-many relationships",
			Method:          "GET",
			Query:           "/designers?select=*,videogames!inner(*)",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a computed relationship overrides the foreign key relationship of the same name; card 32484 (embedding remainder)",
		},
		// it "can override one-to-one relationships(would give disambiguation errors otherwise)" $ do
		//   get "/first_1?select=*,second_1(*)"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "can override one-to-one relationships(would give disambiguation errors otherwise)",
			Method:          "GET",
			Query:           "/first_1?select=*,second_1(*)",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//   get "/second_1?select=*,first_1(*)"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "can override one-to-one relationships(would give disambiguation errors otherwise)",
			Method:          "GET",
			Query:           "/second_1?select=*,first_1(*)",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "creates queries with the right aliasing" $ do
		//   get "/fee?select=*,jsbaz(*,janedoe(*))"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "creates queries with the right aliasing",
			Method:          "GET",
			Query:           "/fee?select=*,jsbaz(*,janedoe(*))",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a computed relationship whose function has another name than the target; card 32484 (embedding remainder)",
		},
		//   get "/fee?select=*,jsbaz(*,johnsmith(*, fee(*)))"
		//     `shouldRespondWith` [json|[]|]{matchHeaders = [matchContentTypeJson]}
		{
			Description:     "creates queries with the right aliasing",
			Method:          "GET",
			Query:           "/fee?select=*,jsbaz(*,johnsmith(*, fee(*)))",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a computed relationship whose function has another name than the target; card 32484 (embedding remainder)",
		},
		// it "creates queries with the right aliasing when following a normal embed" $ do
		//   get "/projects?select=name,clients(name,computed_projects(name))&limit=1"
		//     `shouldRespondWith` [json|
		//       [{"name":"Windows 7","clients":{"name":"Microsoft","computed_projects":{"name":"Windows 7"}}}]
		//     |]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description:     "creates queries with the right aliasing when following a normal embed",
			Method:          "GET",
			Query:           "/projects?select=name,clients(name,computed_projects(name))&limit=1",
			Expected:        `[{"name":"Windows 7","clients":{"name":"Microsoft","computed_projects":{"name":"Windows 7"}}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//   get "/clients?select=name,projects(name,computed_clients(name))&limit=1"
		//     `shouldRespondWith` [json|[
		//       {"name":"Microsoft","projects":[
		//         {"name":"Windows 7","computed_clients":{"name":"Microsoft"}},
		//         {"name":"Windows 10","computed_clients":{"name":"Microsoft"}}
		//       ]}
		//     ]|]
		//       { matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "creates queries with the right aliasing when following a normal embed",
			Method:      "GET",
			Query:       "/clients?select=name,projects(name,computed_clients(name))&limit=1",
			Expected: `[
        {"name":"Microsoft","projects":[
          {"name":"Windows 7","computed_clients":{"name":"Microsoft"}},
          {"name":"Windows 10","computed_clients":{"name":"Microsoft"}}
        ]}
      ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// context "can be defined using overloaded functions" $ do
		//   it "tables" $ do
		//     get "/items?select=*,computed_rel_overload(*)&limit=1"
		//       `shouldRespondWith` [json|
		//         [{"id":1,"computed_rel_overload":[{"id":1}]}]
		//       |]
		//         { matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description:     "tables",
			Method:          "GET",
			Query:           "/items?select=*,computed_rel_overload(*)&limit=1",
			Expected:        `[{"id":1,"computed_rel_overload":[{"id":1}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//     get "/items2?select=*,computed_rel_overload(*)&limit=1"
		//       `shouldRespondWith` [json|
		//         [{"id":1,"computed_rel_overload":[{"id":1},{"id":2}]}]
		//       |]
		//         { matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description:     "tables",
			Method:          "GET",
			Query:           "/items2?select=*,computed_rel_overload(*)&limit=1",
			Expected:        `[{"id":1,"computed_rel_overload":[{"id":1},{"id":2}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//   it "rpc" $ do
		//     get "/rpc/search?id=1&select=*,computed_rel_overload(*)"
		//       `shouldRespondWith` [json|
		//         [{"id":1,"computed_rel_overload":[{"id":1}]}]
		//       |]
		//         { matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description:     "rpc",
			Method:          "GET",
			Query:           "/rpc/search?id=1&select=*,computed_rel_overload(*)",
			Expected:        `[{"id":1,"computed_rel_overload":[{"id":1}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		//     get "/rpc/search2?id=1&select=*,computed_rel_overload(*)"
		//       `shouldRespondWith` [json|
		//         [{"id":1,"computed_rel_overload":[{"id":1},{"id":2}]}]
		//       |]
		//         { matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description:     "rpc",
			Method:          "GET",
			Query:           "/rpc/search2?id=1&select=*,computed_rel_overload(*)",
			Expected:        `[{"id":1,"computed_rel_overload":[{"id":1},{"id":2}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
	}
	test.Execute(t, testConfig, tests)
}
