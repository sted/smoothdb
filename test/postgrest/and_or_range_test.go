package postgrest

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestPostgREST_AndOrParams(t *testing.T) {

	tests := []test.Test{
		// describe "and/or params used for complex boolean logic" $ do
		// context "used with GET" $ do
		//   context "or param" $ do
		// 	it "can do simple logic" $
		// 	  get "/entities?or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can do simple logic",
			Query:       "/entities?or=(id.eq.1,id.eq.2)&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can negate simple logic" $
		// 	  get "/entities?not.or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can negate simple logic",
			Query:       "/entities?not.or=(id.eq.1,id.eq.2)&select=id",
			Expected:    `[{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can be combined with traditional filters" $
		// 	  get "/entities?or=(id.eq.1,id.eq.2)&name=eq.entity 1&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be combined with traditional filters",
			Query:       "/entities?or=(id.eq.1,id.eq.2)&name=eq.entity 1&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		//   context "embedded levels" $ do
		// 	it "can do logic on the second level" $
		// 	  get "/entities?child_entities.or=(id.eq.1,name.eq.child entity 2)&select=id,child_entities(id)" `shouldRespondWith`
		// 		[json|[
		// 		  {"id": 1, "child_entities": [ { "id": 1 }, { "id": 2 } ] }, { "id": 2, "child_entities": []},
		// 		  {"id": 3, "child_entities": []}, {"id": 4, "child_entities": []}
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can do logic on the second level",
			Query:       "/entities?child_entities.or=(id.eq.1,name.eq.child entity 2)&select=id,child_entities(id)",
			Expected: `[
				 		  {"id": 1, "child_entities": [ { "id": 1 }, { "id": 2 } ] }, { "id": 2, "child_entities": []},
				 		  {"id": 3, "child_entities": []}, {"id": 4, "child_entities": []}
				 		]`,
			Headers: nil,
			Status:  200,
		},
		// 	it "can do logic on the third level" $
		// 	  get "/entities?child_entities.grandchild_entities.or=(id.eq.1,id.eq.2)&select=id,child_entities(id,grandchild_entities(id))"
		// 		`shouldRespondWith`
		// 		  [json|[
		// 			{"id": 1, "child_entities": [
		// 			  { "id": 1, "grandchild_entities": [ { "id": 1 }, { "id": 2 } ]},
		// 			  { "id": 2, "grandchild_entities": []},
		// 			  { "id": 4, "grandchild_entities": []},
		// 			  { "id": 5, "grandchild_entities": []}
		// 			]},
		// 			{"id": 2, "child_entities": [
		// 			  { "id": 3, "grandchild_entities": []},
		// 			  { "id": 6, "grandchild_entities": []}
		// 			]},
		// 			{"id": 3, "child_entities": []},
		// 			{"id": 4, "child_entities": []}
		// 		  ]|]

		//   context "and/or params combined" $ do
		// 	it "can be nested inside the same expression" $
		// 	  get "/entities?or=(and(name.eq.entity 2,id.eq.2),and(name.eq.entity 1,id.eq.1))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be nested inside the same expression",
			Query:       "/entities?or=(and(name.eq.entity%202,id.eq.2),and(name.eq.entity%201,id.eq.1))&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can be negated while nested" $
		// 	  get "/entities?or=(not.and(name.eq.entity 2,id.eq.2),not.and(name.eq.entity 1,id.eq.1))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated while nested",
			Query:       "/entities?or=(not.and(name.eq.entity%202,id.eq.2),not.and(name.eq.entity%201,id.eq.1))&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can be combined unnested" $
		// 	  get "/entities?and=(id.eq.1,name.eq.entity 1)&or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be combined unnested",
			Query:       "/entities?and=(id.eq.1,name.eq.entity%201)&or=(id.eq.1,id.eq.2)&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		//   context "operators inside and/or" $ do
		// 	it "can handle eq and neq" $
		// 	  get "/entities?and=(id.eq.1,id.neq.2))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle eq and neq",
			Query:       "/entities?and=(id.eq.1,id.neq.2))&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle lt and gt" $
		// 	  get "/entities?or=(id.lt.2,id.gt.3)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle lt and gt",
			Query:       "/entities?or=(id.lt.2,id.gt.3)&select=id",
			Expected:    `[{"id":1},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle lte and gte" $
		// 	  get "/entities?or=(id.lte.2,id.gte.3)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle lte and gte",
			Query:       "/entities?or=(id.lte.2,id.gte.3)&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle like and ilike" $
		// 	  get "/entities?or=(name.like.*1,name.ilike.*ENTITY 2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle like and ilike",
			Query:       "/entities?or=(name.like.*1,name.ilike.*ENTITY%202)&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle in" $
		// 	  get "/entities?or=(id.in.(1,2),id.in.(3,4))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle in",
			Query:       "/entities?or=(id.in.(1,2),id.in.(3,4))&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle is" $
		// 	  get "/entities?and=(name.is.null,arr.is.null)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle is",
			Query:       "/entities?and=(name.is.null,arr.is.null)&select=id",
			Expected:    `[{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can handle fts" $ do
		// 	  get "/entities?or=(text_search_vector.fts.bar,text_search_vector.fts.baz)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle fts",
			Query:       "/entities?or=(text_search_vector.fts.bar,text_search_vector.fts.baz)&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/tsearch?or=(text_search_vector.plfts(german).Art%20Spass, text_search_vector.plfts(french).amusant%20impossible, text_search_vector.fts(english).impossible)" `shouldRespondWith`
		// 		[json|[
		// 		  {"text_search_vector": "'fun':5 'imposs':9 'kind':3" },
		// 		  {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" },
		// 		  {"text_search_vector": "'art':4 'spass':5 'unmog':7"}
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		// {
		// 	"can handle fts",
		// 	"/entities?and=(name.is.null,arr.is.null)&select=id",
		// 	`[{"id":4}]`,
		// 	nil,
		// 	200,
		// },
		// 	when (actualPgVersion >= pgVersion112) $
		// 	  it "can handle wfts (websearch_to_tsquery)" $
		// 		get "/tsearch?or=(text_search_vector.plfts(german).Art,text_search_vector.plfts(french).amusant,text_search_vector.not.wfts(english).impossible)"
		// 		`shouldRespondWith`
		// 		  [json|[
		// 				 {"text_search_vector": "'also':2 'fun':3 'possibl':8" },
		// 				 {"text_search_vector": "'ate':3 'cat':2 'fat':1 'rat':4" },
		// 				 {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" },
		// 				 {"text_search_vector": "'art':4 'spass':5 'unmog':7" }
		// 		  ]|]
		// 		  { matchHeaders = [matchContentTypeJson] }

		// 	it "can handle cs and cd" $
		// 	  get "/entities?or=(arr.cs.{1,2,3},arr.cd.{1})&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 },{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle cs and cd",
			Query:       "/entities?or=(arr.cs.{1,2,3},arr.cd.{1})&select=id",
			Expected:    `[{"id":1},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},

		// RANGES

		// 	it "can handle range operators" $ do
		// 	  get "/ranges?range=eq.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=eq.[1,3]&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=neq.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=neq.[1,3]&select=id",
			Expected:    `[{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=lt.[1,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=lt.[1,10]&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=gt.[8,11]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=gt.[8,11]&select=id",
			Expected:    `[{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=lte.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=lte.[1,3]&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=gte.[2,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=gte.[2,3]&select=id",
			Expected:    `[{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=cs.[1,2]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=cs.[1,2]&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=cd.[1,6]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=cd.[1,6]&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=ov.[0,4]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=ov.[0,4]&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=sl.[9,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=sl.[9,10]&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=sr.[3,4]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=sr.[3,4]&select=id",
			Expected:    `[{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=nxr.[4,7]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=nxr.[4,7]&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=nxl.[4,7]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=nxl.[4,7]&select=id",
			Expected:    `[{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/ranges?range=adj.(3,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle range operators",
			Query:       "/ranges?range=adj.(3,10]&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},

		// ARRAYS

		// 	it "can handle array operators" $ do
		// 	  get "/entities?arr=eq.{1,2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=eq.{1,2,3}&select=id",
			Expected:    `[{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=neq.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=neq.{1,2}&select=id",
			Expected:    `[{"id":1},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=lt.{2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=lt.{2,3}&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=lt.{2,0}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=lt.{2,0}&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=gt.{1,1}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=gt.{1,1}&select=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=gt.{3}&select=id" `shouldRespondWith`
		// 		[json|[]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=gt.{3}&select=id",
			Expected:    `[]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=lte.{2,1}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=lte.{2,1}&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=lte.{1,2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=lte.{1,2,3}&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=lte.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=lte.{1,2}&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=cs.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=cs.{1,2}&select=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=cd.{1,2,6}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=cd.{1,2,6}&select=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=ov.{3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=ov.{3}&select=id",
			Expected:    `[{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?arr=ov.{2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=ov.{2,3}&select=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	context "operators with not" $ do
		// 	  it "eq, cs, like can be negated" $
		// 		get "/entities?and=(arr.not.cs.{1,2,3},and(id.not.eq.2,name.not.like.*3))&select=id" `shouldRespondWith`
		// 		  [json|[{ "id": 1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "eq, cs, like can be negated",
			Query:       "/entities?and=(arr.not.cs.{1,2,3},and(id.not.eq.2,name.not.like.*3))&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  it "in, is, fts can be negated" $
		// 		get "/entities?and=(id.not.in.(1,3),and(name.not.is.null,text_search_vector.not.fts.foo))&select=id" `shouldRespondWith`
		// 		  [json|[{ "id": 2}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "in, is, fts can be negated",
			Query:       "/entities?and=(id.not.in.(1,3),and(name.not.is.null,text_search_vector.not.fts.foo))&select=id",
			Expected:    `[{"id": 2}]`,
			Headers:     nil,
			Status:      200,
		},
		// {
		// 	"can handle array operators",
		// 	"/entities?arr=ov.{2,3}&select=id",
		// 	`[{"id":2},{"id":3}]`,
		// 	nil,
		// 	200,
		// },
		{
			Description: "can handle array operators",
			Query:       "/entities?arr=ov.{2,3}&select=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  it "lt, gte, cd can be negated" $
		// 		get "/entities?and=(arr.not.cd.{1},or(id.not.lt.1,id.not.gte.3))&select=id" `shouldRespondWith`
		// 		  [json|[{"id": 2}, {"id": 3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "lt, gte, cd can be negated",
			Query:       "/entities?and=(arr.not.cd.{1},or(id.not.lt.1,id.not.gte.3))&select=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  it "gt, lte, ilike can be negated" $
		// 		get "/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id" `shouldRespondWith`
		// 		  [json|[{"id": 1}, {"id": 2}, {"id": 3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "gt, lte, ilike can be negated",
			Query:       "/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},

		//   context "and/or params with quotes" $ do
		// 	it "eq can have quotes" $
		// 	  get "/grandchild_entities?or=(name.eq.\"(grandchild,entity,4)\",name.eq.\"(grandchild,entity,5)\")&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }, { "id": 5 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "eq can have quotes",
			Query:       "/grandchild_entities?or=(name.eq.\"(grandchild,entity,4)\",name.eq.\"(grandchild,entity,5)\")&select=id",
			Expected:    `[{"id":4},{"id":5}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "like and ilike can have quotes" $
		// 	  get "/grandchild_entities?or=(name.like.\"*ity,4*\",name.ilike.\"*ITY,5)\")&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }, { "id": 5 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "like and ilike can have quotes",
			Query:       "/grandchild_entities?or=(name.like.\"*ity,4*\",name.ilike.\"*ITY,5)\")&select=id",
			Expected:    `[{"id":4},{"id":5}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "in can have quotes" $
		// 	  get "/grandchild_entities?or=(id.in.(\"1\",\"2\"),id.in.(\"3\",\"4\"))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "in can have quotes",
			Query:       "/grandchild_entities?or=(id.in.(\"1\",\"2\"),id.in.(\"3\",\"4\"))&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			Headers:     nil,
			Status:      200,
		},

		//   it "allows whitespace" $
		// 	get "/entities?and=( and ( id.in.( 1, 2, 3 ) , id.eq.3 ) , or ( id.eq.2 , id.eq.3 ) )&select=id" `shouldRespondWith`
		// 	  [json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "allows whitespace",
			Query:       "/entities?and=(%20and(%20id.in.(%201,%202,%203%20)%20,%20id.eq.3%20)%20,%20or(%20id.eq.2%20,%20id.eq.3%20)%20)&select=id",
			Expected:    `[{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},

		//   context "multiple and/or conditions" $ do
		// 	it "cannot have zero conditions" $
		// 	  get "/entities?or=()" `shouldRespondWith`
		// 		[json|{
		// 		  "details": "unexpected \")\" expecting field name (* or [a..z0..9_]), negation operator (not) or logic operator (and, or)",
		// 		  "message": "\"failed to parse logic tree (())\" (line 1, column 4)",
		// 		  "code": "PGRST100",
		// 		  "hint": null
		// 		}|] { matchStatus = 400, matchHeaders = [matchContentTypeJson] }
		// {
		// 	"gt, lte, ilike can be negated",
		// 	"/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id",
		// 	`[{"id":1},{"id":2},{"id":3}]`,
		// 	nil,
		// 	200,
		// },
		{
			Description: "gt, lte, ilike can be negated",
			Query:       "/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can have a single condition" $ do
		// 	  get "/entities?or=(id.eq.1)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have a single condition",
			Query:       "/entities?or=(id.eq.1)&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/entities?and=(id.eq.1)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have a single condition",
			Query:       "/entities?and=(id.eq.1)&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can have three conditions" $ do
		// 	  get "/grandchild_entities?or=(id.eq.1, id.eq.2, id.eq.3)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}, {"id":2}, {"id":3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have three conditions",
			Query:       "/grandchild_entities?or=(id.eq.1,%20id.eq.2,%20id.eq.3)&select=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/grandchild_entities?and=(id.in.(1,2), id.in.(3,1), id.in.(1,4))&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have three conditions",
			Query:       "/grandchild_entities?and=(id.in.(1,2),%20id.in.(3,1),%20id.in.(1,4))&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "can have four conditions combining and/or" $ do
		// 	  get "/grandchild_entities?or=( id.eq.1, id.eq.2, and(id.in.(1,3), id.in.(2,3)), id.eq.4 )&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}, {"id":2}, {"id":3}, {"id":4}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have four conditions combining and/or",
			Query:       "/grandchild_entities?or=( id.eq.1, id.eq.2, and(id.in.(1,3), id.in.(2,3)), id.eq.4)&select=id",
			Expected:    `[{"id":1}, {"id":2}, {"id":3}, {"id":4}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	  get "/grandchild_entities?and=( id.eq.1, not.or(id.eq.2, id.eq.3), id.in.(1,4), or(id.eq.1, id.eq.4) )&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can have four conditions combining and/or",
			Query:       "/grandchild_entities?and=( id.eq.1, not.or(id.eq.2, id.eq.3), id.in.(1,4), or(id.eq.1, id.eq.4) )&select=id",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},

		// context "used with POST" $
		//   it "includes related data with filters" $
		// 	request methodPost "/child_entities?select=id,entities(id)&entities.or=(id.eq.2,id.eq.3)&entities.order=id"
		// 		[("Prefer", "return=representation")]
		// 		[json|[
		// 		  {"id":7,"name":"entity 4","parent_id":1},
		// 		  {"id":8,"name":"entity 5","parent_id":2},
		// 		  {"id":9,"name":"entity 6","parent_id":3}
		// 		]|]
		// 	  `shouldRespondWith`
		// 		[json|[{"id": 7, "entities":null}, {"id": 8, "entities": {"id": 2}}, {"id": 9, "entities": {"id": 3}}]|]
		// 		{ matchStatus = 201 }
		{
			Description: "used with POST includes related data with filters",
			Method:      "POST",
			Query:       "/child_entities?select=id,entities(id)&entities.or=(id.eq.2,id.eq.3)&entities.order=id",
			Body: `[
				 		{"id":7,"name":"entity 4","parent_id":1},
				 		{"id":8,"name":"entity 5","parent_id":2},
				 		{"id":9,"name":"entity 6","parent_id":3}
				 	]`,
			Headers:  test.Headers{"Prefer": {"return=representation"}},
			Expected: `[{"id": 7, "entities":null}, {"id": 8, "entities": {"id": 2}}, {"id": 9, "entities": {"id": 3}}]`,
			Status:   201,
		},
		// context "used with PATCH" $
		//   it "succeeds when using and/or params" $
		// 	request methodPatch "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name"
		// 	  [("Prefer", "return=representation")]
		// 	  [json|{ name : "updated grandchild entity"}|] `shouldRespondWith`
		// 	  [json|[{ "id": 1, "name" : "updated grandchild entity"},{ "id": 2, "name" : "updated grandchild entity"}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "used with PATCH succeeds when using and/or params",
			Method:      "PATCH",
			Query:       "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name",
			Body:        `{ "name" : "updated grandchild entity"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 1, "name" : "updated grandchild entity"},{ "id": 2, "name" : "updated grandchild entity"}]`,
			Status:      201,
		},
		// context "used with DELETE" $
		//   it "succeeds when using and/or params" $
		// 	request methodDelete "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name"
		// 		[("Prefer", "return=representation")]
		// 		""
		// 	  `shouldRespondWith`
		// 		[json|[{ "id": 1, "name" : "grandchild entity 1" },{ "id": 2, "name" : "grandchild entity 2" }]|]
		{
			Description: "used with DELETE succeeds when using and/or params",
			Method:      "DELETE",
			Query:       "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name",
			Body:        ``,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 1, "name" : "grandchild entity 1" },{ "id": 2, "name" : "grandchild entity 2" }]`,
			Status:      201,
		},
		// it "can query columns that begin with and/or reserved words" $
		//   get "/grandchild_entities?or=(and_starting_col.eq.smth, or_starting_col.eq.smth)" `shouldRespondWith` 200
		{
			Description: "can query columns that begin with and/or reserved words",
			Query:       "/grandchild_entities?or=(and_starting_col.eq.smth, or_starting_col.eq.smth)",
			Headers:     nil,
			Expected:    ``,
			Status:      200,
		},
		// it "fails when using IN without () and provides meaningful error message" $
		//   get "/entities?or=(id.in.1,2,id.eq.3)" `shouldRespondWith`
		// 	[json|{
		// 	  "details": "unexpected \"1\" expecting \"(\"",
		// 	  "message": "\"failed to parse logic tree ((id.in.1,2,id.eq.3))\" (line 1, column 10)",
		// 	  "code": "PGRST100",
		// 	  "hint": null
		// 	}|] { matchStatus = 400, matchHeaders = [matchContentTypeJson] }

		// it "fails on malformed query params and provides meaningful error message" $ do
		//   get "/entities?or=)(" `shouldRespondWith`
		// 	[json|{
		// 	  "details": "unexpected \")\" expecting \"(\"",
		// 	  "message": "\"failed to parse logic tree ()()\" (line 1, column 3)",
		// 	  "code": "PGRST100",
		// 	  "hint": null
		// 	}|] { matchStatus = 400, matchHeaders = [matchContentTypeJson] }
		//   get "/entities?and=(ord(id.eq.1,id.eq.1),id.eq.2)" `shouldRespondWith`
		// 	[json|{
		// 	  "details": "unexpected \"d\" expecting \"(\"",
		// 	  "message": "\"failed to parse logic tree ((ord(id.eq.1,id.eq.1),id.eq.2))\" (line 1, column 7)",
		// 	  "code": "PGRST100",
		// 	  "hint": null
		// 	}|] { matchStatus = 400, matchHeaders = [matchContentTypeJson] }
		//   get "/entities?or=(id.eq.1,not.xor(id.eq.2,id.eq.3))" `shouldRespondWith`
		// 	[json|{
		// 	  "details": "unexpected \"x\" expecting logic operator (and, or)",
		// 	  "message": "\"failed to parse logic tree ((id.eq.1,not.xor(id.eq.2,id.eq.3)))\" (line 1, column 16)",
		// 	  "code": "PGRST100",
		// 	  "hint": null
		// 	}|] { matchStatus = 400, matchHeaders = [matchContentTypeJson] }
	}

	test.Execute(t, testConfig, tests)
}
