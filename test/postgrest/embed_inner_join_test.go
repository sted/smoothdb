package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_EmbedInnerJoin(t *testing.T) {

	tests := []test.Test{
		// 	context "many-to-one relationships" $ do
		// 	it "ignores null embeddings while the default left join doesn't" $ do
		// 	  get "/projects?select=id,clients!inner(id)" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}},
		// 		  {"id":3,"clients":{"id":2}}, {"id":4,"clients":{"id":2}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores null embeddings while the default left join doesn't",
			Query:       "/projects?select=id,clients!inner(id)",
			Expected: `[
						  {"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}},
						  {"id":3,"clients":{"id":2}}, {"id":4,"clients":{"id":2}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/projects?select=id,clients!left(id)" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}},
		// 		  {"id":3,"clients":{"id":2}}, {"id":4,"clients":{"id":2}},
		// 		  {"id":5,"clients":null}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores null embeddings while the default left join doesn't",
			Query:       "/projects?select=id,clients!left(id)",
			Expected: `[
						  {"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}},
						  {"id":3,"clients":{"id":2}}, {"id":4,"clients":{"id":2}},
						  {"id":5,"clients":null}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  request methodHead "/projects?select=id,clients!inner(id)" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-3/4" ]
		// 		}
		{
			Description:     "ignores null embeddings while the default left join doesn't",
			Method:          "HEAD",
			Query:           "/projects?select=id,clients!inner(id)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-3/4"},
			Status:          200,
		},
		// 	it "filters source tables when the embedded table is filtered" $ do
		// 	  get "/projects?select=id,clients!inner(id)&clients.id=eq.1" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"clients":{"id":1}},
		// 		  {"id":2,"clients":{"id":1}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/projects?select=id,clients!inner(id)&clients.id=eq.1",
			Expected: `[
						  {"id":1,"clients":{"id":1}},
						  {"id":2,"clients":{"id":1}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/projects?select=id,clients!inner(id)&clients.id=eq.2" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":3,"clients":{"id":2}},
		// 		  {"id":4,"clients":{"id":2}}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/projects?select=id,clients!inner(id)&clients.id=eq.2",
			Expected: `[
						  {"id":3,"clients":{"id":2}},
						  {"id":4,"clients":{"id":2}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/projects?select=id,clients!inner(id)&clients.id=eq.0" `shouldRespondWith`
		// 		[json|[]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/projects?select=id,clients!inner(id)&clients.id=eq.0",
			Expected:    `[]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  request methodHead "/projects?select=id,clients!inner(id)&clients.id=eq.1" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-1/2" ]
		// 		}
		{
			Description:     "filters source tables when the embedded table is filtered",
			Method:          "HEAD",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// 	it "filters source tables when a two levels below embedded table is filtered" $ do
		// 	  get "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"projects":{"id":1,"clients":{"id":1}}},
		// 		  {"id":2,"projects":{"id":1,"clients":{"id":1}}},
		// 		  {"id":3,"projects":{"id":2,"clients":{"id":1}}},
		// 		  {"id":4,"projects":{"id":2,"clients":{"id":1}}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1",
			Expected: `[
						  {"id":1,"projects":{"id":1,"clients":{"id":1}}},
						  {"id":2,"projects":{"id":1,"clients":{"id":1}}},
						  {"id":3,"projects":{"id":2,"clients":{"id":1}}},
						  {"id":4,"projects":{"id":2,"clients":{"id":1}}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.2" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":5,"projects":{"id":3,"clients":{"id":2}}},
		// 		  {"id":6,"projects":{"id":3,"clients":{"id":2}}},
		// 		  {"id":7,"projects":{"id":4,"clients":{"id":2}}},
		// 		  {"id":8,"projects":{"id":4,"clients":{"id":2}}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.2",
			Expected: `[
						  {"id":5,"projects":{"id":3,"clients":{"id":2}}},
						  {"id":6,"projects":{"id":3,"clients":{"id":2}}},
						  {"id":7,"projects":{"id":4,"clients":{"id":2}}},
						  {"id":8,"projects":{"id":4,"clients":{"id":2}}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  request methodHead "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-3/4" ]
		// 		}
		{
			Description:     "filters source tables when a two levels below embedded table is filtered",
			Method:          "HEAD",
			Query:           "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-3/4"},
			Status:          200,
		},
		// 	it "only affects the source table rows if his direct embedding is an inner join" $ do
		// 	  get "/tasks?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.2" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"projects":null},
		// 		  {"id":2,"projects":null},
		// 		  {"id":3,"projects":null},
		// 		  {"id":4,"projects":null},
		// 		  {"id":5,"projects":{"id":3,"clients":{"id":2}}},
		// 		  {"id":6,"projects":{"id":3,"clients":{"id":2}}},
		// 		  {"id":7,"projects":{"id":4,"clients":{"id":2}}},
		// 		  {"id":8,"projects":{"id":4,"clients":{"id":2}}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "only affects the source table rows if his direct embedding is an inner join",
			Query:       "/tasks?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.2",
			Expected: `[
						  {"id":1,"projects":null},
						  {"id":2,"projects":null},
						  {"id":3,"projects":null},
						  {"id":4,"projects":null},
						  {"id":5,"projects":{"id":3,"clients":{"id":2}}},
						  {"id":6,"projects":{"id":3,"clients":{"id":2}}},
						  {"id":7,"projects":{"id":4,"clients":{"id":2}}},
						  {"id":8,"projects":{"id":4,"clients":{"id":2}}}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  request methodHead "/tasks?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.2" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-7/8" ]
		// 		}
		{
			Description:     "only affects the source table rows if his direct embedding is an inner join",
			Method:          "HEAD",
			Query:           "/tasks?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.2",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-7/8"},
			Status:          200,
		},
		// 	it "works with views" $ do
		// 	  get "/books?select=title,authors!inner(name)&authors.name=eq.George%20Orwell" `shouldRespondWith`
		// 		[json| [{"title":"1984","authors":{"name":"George Orwell"}}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  request methodHead "/books?select=title,authors!inner(name)&authors.name=eq.George%20Orwell" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}

		//   context "one-to-many relationships" $ do
		// 	it "ignores empty array embeddings while the default left join doesn't" $ do
		// 	  get "/entities?select=id,child_entities!inner(id)" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"child_entities":[{"id":1}, {"id":2}, {"id":4}, {"id":5}]},
		// 		  {"id":2,"child_entities":[{"id":3}, {"id":6}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores empty array embeddings while the default left join doesn't",
			Query:       "/entities?select=id,child_entities!inner(id)",
			Expected: `[
						  {"id":1,"child_entities":[{"id":1}, {"id":2}, {"id":4}, {"id":5}]},
						  {"id":2,"child_entities":[{"id":3}, {"id":6}]}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/entities?select=id,child_entities!left(id)" `shouldRespondWith`
		// 		[json| [
		// 		  {"id":1,"child_entities":[{"id":1}, {"id":2}, {"id":4}, {"id":5}]},
		// 		  {"id":2,"child_entities":[{"id":3}, {"id":6}]},
		// 		  {"id":3,"child_entities":[]},
		// 		  {"id":4,"child_entities":[]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores empty array embeddings while the default left join doesn't",
			Query:       "/entities?select=id,child_entities(id)",
			Expected: `[
						  {"id":1,"child_entities":[{"id":1}, {"id":2}, {"id":4}, {"id":5}]},
						  {"id":2,"child_entities":[{"id":3}, {"id":6}]},
						  {"id":3,"child_entities":[]},
						  {"id":4,"child_entities":[]}]`,
			Headers: nil,
			Status:  200,
		},
		// 	  request methodHead "/entities?select=id,child_entities!inner(id)" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-1/2" ]
		// 		}
		{
			Description:     "ignores empty array embeddings while the default left join doesn't",
			Method:          "HEAD",
			Query:           "/entities?select=id,child_entities!inner(id)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// 	it "filters source tables when the embedded table is filtered" $ do
		// 	  get "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.1" `shouldRespondWith`
		// 		[json|[{"id":1,"child_entities":[{"id":1}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.1",
			Expected:    `[{"id":1,"child_entities":[{"id":1}]}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.3" `shouldRespondWith`
		// 		[json|[{"id":2,"child_entities":[{"id":3}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.3",
			Expected:    `[{"id":2,"child_entities":[{"id":3}]}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.0" `shouldRespondWith`
		// 		[json|[]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.0",
			Expected:    `[]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  request methodHead "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.1" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}
		{
			Description:     "filters source tables when the embedded table is filtered",
			Method:          "HEAD",
			Query:           "/entities?select=id,child_entities!inner(id)&child_entities.id=eq.1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// 	it "filters source tables when a two levels below embedded table is filtered" $ do
		// 	  get "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=in.(1,5)"
		// 		`shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"id": 1,
		// 			"child_entities": [
		// 			  { "id": 1, "grandchild_entities": [ { "id": 1 } ] },
		// 			  { "id": 2, "grandchild_entities": [ { "id": 5 } ] }]
		// 		  }
		// 		]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=in.(1,5)",
			Expected: `[
						  {
							"id": 1,
							"child_entities": [
							  { "id": 1, "grandchild_entities": [ { "id": 1 } ] },
							  { "id": 2, "grandchild_entities": [ { "id": 5 } ] }]
						  }
						]`,
			Headers: nil,
			Status:  200,
		},
		// 	  get "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=eq.2" `shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"id": 1,
		// 			"child_entities": [
		// 			  { "id": 1, "grandchild_entities": [ { "id": 2 } ] } ]
		// 		  }
		// 		]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=eq.2",
			Expected: `[
						  {
							"id": 1,
							"child_entities": [
							  { "id": 1, "grandchild_entities": [ { "id": 2 } ] } ]
						  }
						]`,
			Headers: nil,
			Status:  200,
		},
		// 	  request methodHead "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=in.(1,5)" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}
		{
			Description:     "filters source tables when a two levels below embedded table is filtered",
			Method:          "HEAD",
			Query:           "/entities?select=id,child_entities!inner(id,grandchild_entities!inner(id))&child_entities.grandchild_entities.id=in.(1,5)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// 	it "only affects the source table rows if his direct embedding is an inner join" $ do
		// 	  get "/entities?select=id,child_entities!inner(id,grandchild_entities(id))&child_entities.grandchild_entities.id=eq.2" `shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"id": 1,
		// 			"child_entities": [
		// 			  { "id": 1, "grandchild_entities": [ { "id": 2 } ] },
		// 			  { "id": 2, "grandchild_entities": [] },
		// 			  { "id": 4, "grandchild_entities": [] },
		// 			  { "id": 5, "grandchild_entities": [] } ]
		// 		  },
		// 		  {
		// 			"id": 2,
		// 			"child_entities": [
		// 			  { "id": 3, "grandchild_entities": [] },
		// 			  { "id": 6, "grandchild_entities": [] } ]
		// 		  }
		// 		]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "only affects the source table rows if his direct embedding is an inner join",
			Query:       "/entities?select=id,child_entities!inner(id,grandchild_entities(id))&child_entities.grandchild_entities.id=eq.2",
			Expected: `[
						  {
							"id": 1,
							"child_entities": [
							  { "id": 1, "grandchild_entities": [ { "id": 2 } ] },
							  { "id": 2, "grandchild_entities": [] },
							  { "id": 4, "grandchild_entities": [] },
							  { "id": 5, "grandchild_entities": [] } ]
						  },
						  {
							"id": 2,
							"child_entities": [
							  { "id": 3, "grandchild_entities": [] },
							  { "id": 6, "grandchild_entities": [] } ]
						  }
						]`,
			Headers: nil,
		},
		// 	  request methodHead "/entities?select=id,child_entities!inner(id,grandchild_entities(id))&child_entities.grandchild_entities.id=eq.2" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-1/2" ]
		// 		}
		{
			Description:     "only affects the source table rows if his direct embedding is an inner join",
			Method:          "HEAD",
			Query:           "/entities?select=id,child_entities!inner(id,grandchild_entities(id))&child_entities.grandchild_entities.id=eq.2",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// 	it "works with views" $ do
		// 	  get "/authors?select=*,books!inner(*)&books.title=eq.1984" `shouldRespondWith`
		// 		[json| [{"id":1,"name":"George Orwell","books":[{"id":1,"title":"1984","publication_year":1949,"author_id":1,"first_publisher_id":1}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  request methodHead "/authors?select=*,books!inner(*)&books.title=eq.1984" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}

		//   context "many-to-many relationships" $ do
		// 	it "ignores empty array embeddings while the default left join doesn't" $ do
		// 	  get "/products?select=id,suppliers!inner(id)" `shouldRespondWith`
		// 		[json| [
		// 		  {"id":1,"suppliers":[{"id":1}, {"id":2}]},
		// 		  {"id":2,"suppliers":[{"id":1}, {"id":3}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores empty array embeddings while the default left join doesn't",
			Query:       "/products?select=id,suppliers!inner(id)",
			Expected: `[
						  {"id":1,"suppliers":[{"id":1}, {"id":2}]},
						  {"id":2,"suppliers":[{"id":1}, {"id":3}]}]`,
			Headers: nil,
		},
		// 	  get "/products?select=id,suppliers!left(id)" `shouldRespondWith`
		// 		[json| [
		// 		  {"id":1,"suppliers":[{"id":1}, {"id":2}]},
		// 		  {"id":2,"suppliers":[{"id":1}, {"id":3}]},
		// 		  {"id":3,"suppliers":[]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores empty array embeddings while the default left join doesn't",
			Query:       "/products?select=id,suppliers!left(id)",
			Expected: `[
						  {"id":1,"suppliers":[{"id":1}, {"id":2}]},
						  {"id":2,"suppliers":[{"id":1}, {"id":3}]},
						  {"id":3,"suppliers":[]}]`,
			Headers: nil,
		},
		// 	  request methodHead "/products?select=id,suppliers!inner(id)" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-1/2" ]
		// 		}
		{
			Description:     "ignores empty array embeddings while the default left join doesn't",
			Method:          "HEAD",
			Query:           "/products?select=id,suppliers!inner(id)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// 	it "filters source tables when the embedded table is filtered" $ do
		// 	  get "/products?select=id,suppliers!inner(id)&suppliers.id=eq.2" `shouldRespondWith`
		// 		[json| [{"id":1,"suppliers":[{"id":2}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/products?select=id,suppliers!inner(id)&suppliers.id=eq.2",
			Expected:    `[{"id":1,"suppliers":[{"id":2}]}]`,
			Headers:     nil,
		},
		// 	  get "/products?select=id,suppliers!inner(id)&suppliers.id=eq.3" `shouldRespondWith`
		// 		[json| [{"id":2,"suppliers":[{"id":3}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/products?select=id,suppliers!inner(id)&suppliers.id=eq.3",
			Expected:    `[{"id":2,"suppliers":[{"id":3}]}]`,
			Headers:     nil,
		},
		// 	  get "/products?select=id,suppliers!inner(id)&suppliers.id=eq.0" `shouldRespondWith`
		// 		[json| [] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when the embedded table is filtered",
			Query:       "/products?select=id,suppliers!inner(id)&suppliers.id=eq.0",
			Expected:    `[]`,
			Headers:     nil,
		},
		// 	  request methodHead "/products?select=id,suppliers!inner(id)&suppliers.id=eq.2" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}
		{
			Description:     "filters source tables when the embedded table is filtered",
			Method:          "HEAD",
			Query:           "/products?select=id,suppliers!inner(id)&suppliers.id=eq.2",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// 	it "filters source tables when a two levels below embedded table is filtered" $ do
		// 	  get "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.3"
		// 		`shouldRespondWith`
		// 		[json|[{"id":1,"suppliers":[{"id":2,"trade_unions":[{"id":3}]}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.3",
			Expected:    `[{"id":1,"suppliers":[{"id":2,"trade_unions":[{"id":3}]}]}]`,
			Headers:     nil,
		},
		// 	  get "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.4"
		// 		`shouldRespondWith`
		// 		[json|[{"id":1,"suppliers":[{"id":2,"trade_unions":[{"id":4}]}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "filters source tables when a two levels below embedded table is filtered",
			Query:       "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.4",
			Expected:    `[{"id":1,"suppliers":[{"id":2,"trade_unions":[{"id":4}]}]}]`,
			Headers:     nil,
		},
		// 	  request methodHead "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.3" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}
		{
			Description:     "filters source tables when a two levels below embedded table is filtered",
			Method:          "HEAD",
			Query:           "/products?select=id,suppliers!inner(id,trade_unions!inner(id))&suppliers.trade_unions.id=eq.3",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// 	it "only affects the source table rows if his direct embedding is an inner join" $ do
		// 	  get "/products?select=id,suppliers!inner(id,trade_unions(id))&suppliers.trade_unions.id=eq.3" `shouldRespondWith`
		// 		[json|[
		// 		  {"id":1,"suppliers":[{"id":1,"trade_unions":[]}, {"id":2,"trade_unions":[{"id":3}]}]},
		// 		  {"id":2,"suppliers":[{"id":1,"trade_unions":[]}, {"id":3,"trade_unions":[]}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "only affects the source table rows if his direct embedding is an inner join",
			Query:       "/products?select=id,suppliers!inner(id,trade_unions(id))&suppliers.trade_unions.id=eq.3",
			Expected: `[
						  {"id":1,"suppliers":[{"id":1,"trade_unions":[]}, {"id":2,"trade_unions":[{"id":3}]}]},
						  {"id":2,"suppliers":[{"id":1,"trade_unions":[]}, {"id":3,"trade_unions":[]}]}]`,
			Headers: nil,
		},
		// 	  request methodHead "/products?select=id,suppliers!inner(id,trade_unions(id))&suppliers.trade_unions.id=eq.3" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-1/2" ]
		// 		}
		{
			Description:     "only affects the source table rows if his direct embedding is an inner join",
			Method:          "HEAD",
			Query:           "/products?select=id,suppliers!inner(id,trade_unions(id))&suppliers.trade_unions.id=eq.3",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// 	it "works with views" $ do
		// 	  get "/actors?select=*,films!inner(*)&films.title=eq.douze%20commandements" `shouldRespondWith`
		// 		[json| [{"id":1,"name":"john","films":[{"id":12,"title":"douze commandements"}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  get "/films?select=*,actors!inner(*)&actors.name=eq.john" `shouldRespondWith`
		// 		[json| [{"id":12,"title":"douze commandements","actors":[{"id":1,"name":"john"}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  request methodHead "/actors?select=*,films!inner(*)&films.title=eq.douze%20commandements" [("Prefer", "count=exact")] mempty
		// 		`shouldRespondWith` ""
		// 		{ matchStatus  = 200
		// 		, matchHeaders = [ matchContentTypeJson
		// 						 , "Content-Range" <:> "0-0/1" ]
		// 		}

		//   it "works with m2o and m2m relationships combined" $ do
		// 	get "/projects?select=name,clients!inner(name),users!inner(name)" `shouldRespondWith`
		// 	  [json| [
		// 		{"name":"Windows 7","clients":{"name":"Microsoft"},"users":[{"name":"Angela Martin"}, {"name":"Dwight Schrute"}]},
		// 		{"name":"Windows 10","clients":{"name":"Microsoft"},"users":[{"name":"Angela Martin"}]},
		// 		{"name":"IOS","clients":{"name":"Apple"},"users":[{"name":"Michael Scott"}, {"name":"Dwight Schrute"}]},
		// 		{"name":"OSX","clients":{"name":"Apple"},"users":[{"name":"Michael Scott"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with m2o and m2m relationships combined",
			Query:       "/projects?select=name,clients!inner(name),users!inner(name)",
			Expected: `[
						{"name":"Windows 7","clients":{"name":"Microsoft"},"users":[{"name":"Angela Martin"}, {"name":"Dwight Schrute"}]},
						{"name":"Windows 10","clients":{"name":"Microsoft"},"users":[{"name":"Angela Martin"}]},
						{"name":"IOS","clients":{"name":"Apple"},"users":[{"name":"Michael Scott"}, {"name":"Dwight Schrute"}]},
						{"name":"OSX","clients":{"name":"Apple"},"users":[{"name":"Michael Scott"}]}]`,
			Headers: nil,
		},
		// 	request methodHead "/projects?select=name,clients!inner(name),users!inner(name)" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-3/4" ]
		// 	  }
		{
			Description:     "works with m2o and m2m relationships combined",
			Method:          "HEAD",
			Query:           "/projects?select=name,clients!inner(name),users!inner(name)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-3/4"},
			Status:          200,
		},
		//   it "works with rpc" $ do
		// 	get "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1" `shouldRespondWith`
		// 	  [json| [{"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with rpc",
			Query:       "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1",
			Expected:    `[{"id":1,"clients":{"id":1}}, {"id":2,"clients":{"id":1}}]`,
			Headers:     nil,
		},
		// 	request methodHead "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-1/2" ]
		// 	  }
		{
			Description:     "works with rpc",
			Method:          "HEAD",
			Query:           "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		//   it "works when using hints" $ do
		// 	get "/projects?select=id,clients!client!inner(id)&clients.id=eq.2" `shouldRespondWith`
		// 	  [json| [{"id":3,"clients":{"id":2}}, {"id":4,"clients":{"id":2}}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// 	get "/projects?select=id,client!inner(id)&client.id=eq.2" `shouldRespondWith`
		// 	  [json| [{"id":3,"client":{"id":2}}, {"id":4,"client":{"id":2}}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// 	request methodHead "/projects?select=id,clients!client!inner(id)&clients.id=eq.2" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-1/2" ]
		// 	  }

		//   it "works with many one-to-many relationships" $ do
		// 	-- https://github.com/PostgREST/postgrest/issues/1977
		// 	get "/client?select=id,name,contact!inner(name),clientinfo!inner(other)" `shouldRespondWith`
		// 	  [json|[
		// 		{"id":1,"name":"Walmart","contact":[{"name":"Wally Walton"}, {"name":"Wilma Wellers"}],"clientinfo":[{"other":"123 Main St"}]},
		// 		{"id":2,"name":"Target", "contact":[{"name":"Tabby Targo"}],"clientinfo":[{"other":"456 South 3rd St"}]},
		// 		{"id":3,"name":"Big Lots","contact":[{"name":"Bobby Bots"}, {"name":"Bonnie Bits"}, {"name":"Billy Boats"}],"clientinfo":[{"other":"789 Palm Tree Ln"}]}
		// 	  ]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with many one-to-many relationships",
			Query:       "/client?select=id,name,contact!inner(name),clientinfo!inner(other)",
			Expected: `[
						{"id":1,"name":"Walmart","contact":[{"name":"Wally Walton"}, {"name":"Wilma Wellers"}],"clientinfo":[{"other":"123 Main St"}]},
						{"id":2,"name":"Target", "contact":[{"name":"Tabby Targo"}],"clientinfo":[{"other":"456 South 3rd St"}]},
						{"id":3,"name":"Big Lots","contact":[{"name":"Bobby Bots"}, {"name":"Bonnie Bits"}, {"name":"Billy Boats"}],"clientinfo":[{"other":"789 Palm Tree Ln"}]}
					  ]`,
			Headers: nil,
		},
		// 	get "/client?select=id,name,contact!inner(name),clientinfo!inner(other)&contact.name=eq.Wally%20Walton" `shouldRespondWith`
		// 	  [json|[
		// 		{"id":1,"name":"Walmart","contact":[{"name":"Wally Walton"}],"clientinfo":[{"other":"123 Main St"}]}
		// 	  ]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with many one-to-many relationships",
			Query:       "/client?select=id,name,contact!inner(name),clientinfo!inner(other)&contact.name=eq.Wally%20Walton",
			Expected: `[
						{"id":1,"name":"Walmart","contact":[{"name":"Wally Walton"}],"clientinfo":[{"other":"123 Main St"}]}
					  ]`,
			Headers: nil,
		},
		// 	get "/client?select=id,name,contact!inner(name),clientinfo!inner(other)&clientinfo.other=eq.456%20South%203rd%20St" `shouldRespondWith`
		// 	  [json|[
		// 		{"id":2,"name":"Target","clientinfo":[{"other":"456 South 3rd St"}],"contact":[{"name":"Tabby Targo"}]}
		// 	  ]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with many one-to-many relationships",
			Query:       "/client?select=id,name,contact!inner(name),clientinfo!inner(other)&clientinfo.other=eq.456%20South%203rd%20St",
			Expected: `[
						{"id":2,"name":"Target","clientinfo":[{"other":"456 South 3rd St"}],"contact":[{"name":"Tabby Targo"}]}
					  ]`,
			Headers: nil,
		},
		// 	request methodHead "/client?select=id,name,contact!inner(name),clientinfo!inner(other)" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-2/3" ]
		// 	  }
		{
			Description:     "works with many one-to-many relationships",
			Method:          "HEAD",
			Query:           "/client?select=id,name,contact!inner(name),clientinfo!inner(other)",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/3"},
			Status:          200,
		},
		//   it "works alongside another embedding" $ do
		// 	-- https://github.com/PostgREST/postgrest/issues/2342
		// 	get "/books?select=id,authors(name),publishers!inner(name)&id=gte.7"
		// 	  `shouldRespondWith`
		// 	  [json| [
		// 		{"id":7,"authors":{"name":"Harper Lee"},"publishers":{"name":"J. B. Lippincott & Co."}},
		// 		{"id":8,"authors":{"name":"Kurt Vonnegut"},"publishers":{"name":"Delacorte"}},
		// 		{"id":9,"authors":{"name":"Ken Kesey"},"publishers":{"name":"Viking Press & Signet Books"}}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }

		//@@ views

		// {
		// 	Description: "works alongside another embedding",
		// 	Query:       "/books?select=id,authors(name),publishers!inner(name)&id=gte.7",
		// 	Expected: `[
		// 				{"id":7,"authors":{"name":"Harper Lee"},"publishers":{"name":"J. B. Lippincott & Co."}},
		// 				{"id":8,"authors":{"name":"Kurt Vonnegut"},"publishers":{"name":"Delacorte"}},
		// 				{"id":9,"authors":{"name":"Ken Kesey"},"publishers":{"name":"Viking Press & Signet Books"}}]`,
		// 	Headers: nil,
		// },
		// 	request methodHead "/books?select=id,authors(name),publishers!inner(name)&id=gte.7" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-2/3" ]
		// 	  }
		// 	request methodHead "/books?select=id,publishers!inner(name),authors(name)&id=gte.7" [("Prefer", "count=exact")] mempty
		// 	  `shouldRespondWith` ""
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = [ matchContentTypeJson
		// 					   , "Content-Range" <:> "0-2/3" ]
		// 	  }
	}

	test.Execute(t, testConfig, tests)
}
