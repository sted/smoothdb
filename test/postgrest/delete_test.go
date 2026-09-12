package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Delete(t *testing.T) {

	tests := []test.Test{
		// 	context "existing record" $ do
		// 	it "succeeds with 204 and deletion count" $
		// 	  request methodDelete "/items?id=eq.1"
		// 		  []
		// 		  ""
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus  = 204
		// 		  , matchHeaders = [ matchHeaderAbsent hContentType
		//  , "Content-Range" <:> "*/*" ]
		//	  }
		{
			Description:     "succeeds with 204 and deletion count",
			Method:          "DELETE",
			Query:           "/items?id=eq.1",
			Body:            ``,
			Headers:         nil,
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          204,
		},
		// 	it "returns the deleted item and count if requested" $
		// 	  request methodDelete "/items?id=eq.2" [("Prefer", "return=representation"), ("Prefer", "count=exact")] ""
		// 		`shouldRespondWith` [json|[{"id":2}]|]
		// 		{ matchStatus  = 200
		// 		, matchHeaders = ["Content-Range" <:> "*/1"]
		// 		}
		{
			Description:     "returns the deleted item and count if requested",
			Method:          "DELETE",
			Query:           "/items?id=eq.2",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"return=representation", "count=exact"}},
			Expected:        `[{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/1"},
			Status:          200,
		},
		// 	it "ignores ?select= when return not set or return=minimal" $ do
		// 	  request methodDelete "/items?id=eq.3&select=id"
		// 		  []
		// 		  ""
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus  = 204
		// 		  , matchHeaders = [ matchHeaderAbsent hContentType
		// 						   , "Content-Range" <:> "*/*" ]
		// 		  }
		{
			Description: "ignores ?select= when return not set or return=minimal",
			Method:      "DELETE",
			Query:       "/items?id=eq.3&select=id",
			Body:        ``,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		// 	  request methodDelete "/items?id=eq.3&select=id"
		// 		  [("Prefer", "return=minimal")]
		// 		  ""
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus  = 204
		// 		  , matchHeaders = [ matchHeaderAbsent hContentType
		// 						   , "Content-Range" <:> "*/*" ]
		// 		  }
		{
			Description: "ignores ?select= when return not set or return=minimal",
			Method:      "DELETE",
			Query:       "/items?id=eq.3&select=id",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=minimal"}},
			Expected:    ``,
			Status:      204,
		},
		// 	it "returns the deleted item and shapes the response" $
		// 	  request methodDelete "/complex_items?id=eq.2&select=id,name" [("Prefer", "return=representation")] ""
		// 		`shouldRespondWith` [json|[{"id":2,"name":"Two"}]|]
		// 		{ matchStatus  = 200
		// 		, matchHeaders = ["Content-Range" <:> "*/*"]
		// 		}
		{
			Description: "returns the deleted item and shapes the response",
			Method:      "DELETE",
			Query:       "/complex_items?id=eq.2&select=id,name",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":2,"name":"Two"}]`,
			Status:      200,
		},
		// 	it "can rename and cast the selected columns" $
		// 	  request methodDelete "/complex_items?id=eq.3&select=ciId:id::text,ciName:name" [("Prefer", "return=representation")] ""
		// 		`shouldRespondWith` [json|[{"ciId":"3","ciName":"Three"}]|]
		{
			Description: "can rename and cast the selected columns",
			Method:      "DELETE",
			Query:       "/complex_items?id=eq.3&select=ciId:id::text,ciName:name",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"ciId":"3","ciName":"Three"}]`,
			Status:      200,
		},
		// 	it "can embed (parent) entities" $
		// 	  request methodDelete "/tasks?id=eq.8&select=id,name,project:projects(id)" [("Prefer", "return=representation")] ""
		// 		`shouldRespondWith` [json|[{"id":8,"name":"Code OSX","project":{"id":4}}]|]
		// 		{ matchStatus  = 200
		// 		, matchHeaders = ["Content-Range" <:> "*/*"]
		// 		}
		{
			Description: "can embed (parent) entities",
			Method:      "DELETE",
			Query:       "/tasks?id=eq.8&select=id,name,project:projects(id)",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":8,"name":"Code OSX","project":{"id":4}}]`,
			Status:      200,
		},
		// 	it "embeds an O2O relationship after delete" $ do
		// 	  request methodDelete "/students?id=eq.1&select=name,students_info(address)"
		// 			  [("Prefer", "return=representation")] ""
		// 		`shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"name": "John Doe",
		// 			"students_info":{"address":"Street 1"}
		// 		  }
		// 		]|]
		// 		{ matchStatus  = 200,
		// 		  matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "embeds an O2O relationship after delete",
			Method:      "DELETE",
			Query:       "/students?id=eq.1&select=name,students_info(address)",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"name": "John Doe", "students_info":{"address":"Street 1"}}]`,
			Status:      200,
		},
		// 	  request methodDelete "/students_info?id=eq.1&select=address,students(name)"
		// 			  [("Prefer", "return=representation")] ""
		// 		`shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"address": "Street 1",
		// 			"students":{"name": "John Doe"}
		// 		  }
		// 		]|]
		// 		{ matchStatus  = 200,
		// 		  matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "embeds an O2O relationship after delete",
			Method:      "DELETE",
			Query:       "/students_info?id=eq.1&select=address,students(name)",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"address": "Street 1","students":{"name": "John Doe"}}]`,
			Status:      200,
		},
		//   context "known route, no records matched" $
		// 	it "includes [] body if return=rep" $
		// 	  request methodDelete "/items?id=eq.101"
		// 		[("Prefer", "return=representation")] ""
		// 		`shouldRespondWith` "[]"
		// 		{ matchStatus  = 200
		// 		, matchHeaders = ["Content-Range" <:> "*/*"]
		// 		}
		{
			Description: "known route, no records matched includes [] body if return=rep",
			Method:      "DELETE",
			Query:       "/items?id=eq.101",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},
		//   context "totally unknown route" $
		// 	it "fails with 404" $
		// 	  request methodDelete "/foozle?id=eq.101" [] "" `shouldRespondWith` 404
		{
			Description: "totally unknown route fails with 404",
			Method:      "DELETE",
			Query:       "/foozle?id=eq.101",
			Body:        ``,
			Headers:     nil,
			Expected:    ``,
			Status:      404,
		},
		//   context "table with limited privileges" $ do
		// 	it "fails deleting the row when return=representation and selecting all the columns" $
		// 	  request methodDelete "/app_users?id=eq.1" [("Prefer", "return=representation")] mempty
		// 		  `shouldRespondWith` 401
		{
			Description: "fails deleting the row when return=representation and selecting all the columns",
			Method:      "DELETE",
			Query:       "/app_users?id=eq.1",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    ``,
			Status:      401,
		},
		// 	it "succeeds deleting the row when return=representation and selecting only the privileged columns" $
		// 	  request methodDelete "/app_users?id=eq.1&select=id,email" [("Prefer", "return=representation")]
		// 		[json| { "password": "passxyz" } |]
		// 		  `shouldRespondWith` [json|[ { "id": 1, "email": "test@123.com" } ]|]
		// 		  { matchStatus  = 200
		// 		  , matchHeaders = ["Content-Range" <:> "*/*"]
		// 		  }
		{
			Description: "succeeds deleting the row when return=representation and selecting only the privileged columns",
			Method:      "DELETE",
			Query:       "/app_users?id=eq.1&select=id,email",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "id": 1, "email": "test@123.com" } ]`,
			Status:      200,
		},
		// 	it "suceeds deleting the row with no explicit select when using return=minimal" $
		// 	  request methodDelete "/app_users?id=eq.2"
		// 		  [("Prefer", "return=minimal")]
		// 		  mempty
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus = 204
		// 		  , matchHeaders = [matchHeaderAbsent hContentType]
		// 		  }
		{
			Description: "succeeds deleting the row with no explicit select when using return=minimal",
			Method:      "DELETE",
			Query:       "/app_users?id=eq.2",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=minimal"}},
			Expected:    ``,
			Status:      204,
		},
		// 	it "suceeds deleting the row with no explicit select by default" $
		// 	  request methodDelete "/app_users?id=eq.3"
		// 		  []
		// 		  mempty
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus = 204
		// 		  , matchHeaders = [matchHeaderAbsent hContentType]
		// 		  }
		{
			Description: "succeeds deleting the row with no explicit select by default",
			Method:      "DELETE",
			Query:       "/app_users?id=eq.3",
			Body:        ``,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		// The "limited delete" context (DELETE narrowed by limit/offset, PGRST109,
		// PGRST110) was dropped upstream in PostgREST 13.0.0 (#3013 limited-updates,
		// "Drop support for Limited updates/deletes"): limit and offset are ignored
		// on PATCH and DELETE and every matching row is written. The replacement
		// guard is `Prefer: max-affected` (card 32482 max-affected).
		// @@ added: the parity is deliberate — limit/offset do not narrow the delete
		// (select=b keeps the check independent of the row order)
		{
			Description: "limit and offset are ignored on DELETE: every matching row is deleted (PostgREST 13+, #3013)",
			Method:      "DELETE",
			Query:       "/no_pk?b=eq.0&select=b&limit=1&offset=1",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "b": "0" }, { "b": "0" }]`,
			Status:      200,
		},
		//   context "with ordering" $
		// 	it "works with request method DELETE and embedded resource" $ do
		// 	  request methodDelete "/artists?id=lt.3&select=id,name,albums(title)&order=id.desc"
		// 		[("Prefer", "return=representation")]
		// 		""
		// 		`shouldRespondWith`
		// 		[json| [ {"id":2,"name":"black country, new road","albums":[{"title": "ants from up above"}]}, {"id":1,"name":"duster","albums":[{"title": "stratosphere"},{"title": "contemporary movement"}]}]
		// 		|]
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [matchContentTypeJson, "Preference-Applied" <:> "return=representation"]
		// 		}
		{
			Description: "with ordering works with request method DELETE and embedded resource",
			Method:      "DELETE",
			Query:       "/artists?id=lt.3&select=id,name,albums(title)&order=id.desc",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ {"id":2,"name":"black country, new road","albums":[{"title": "ants from up above"}]}, {"id":1,"name":"duster","albums":[{"title": "stratosphere"},{"title": "contemporary movement"}]}]`,
			Status:      200,
		},
		// @@ added: an embedded order applies to the embed of the representation
		// (the lateral join of the _source CTE); no top-level order here, so this
		// is the control that the embed path of a mutation orders on its own
		{
			Description: "embedded order applies to the representation of a DELETE",
			Method:      "DELETE",
			Query:       "/artists?id=eq.1&select=id,albums(title)&albums.order=title.asc",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":1,"albums":[{"title": "contemporary movement"},{"title": "stratosphere"}]}]`,
			Status:      200,
		},
		// @@ added: regression for 42702 "column reference is ambiguous" — with select=*
		// plus an embed, the fk column must not be added twice to the RETURNING clause
		{
			Description: "star select with embed does not duplicate the fk in returning",
			Method:      "DELETE",
			Query:       "/tasks?id=eq.8&select=*,projects(id,name)",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":8,"name":"Code OSX","project_id":4,"projects":{"id":4,"name":"OSX"}}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
