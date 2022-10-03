package main

import (
	"io"
	"log"
	"net/http"
	"testing"
)

func TestPostgREST_AndOrParams(t *testing.T) {

	tests := []struct {
		description     string
		query           string
		expectedResults string
		headers         []string
		status          int
	}{
		// describe "and/or params used for complex boolean logic" $ do
		// context "used with GET" $ do
		//   context "or param" $ do
		// 	it "can do simple logic" $
		// 	  get "/entities?or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can do simple logic",
			"/entities?or=(id.eq.1,id.eq.2)&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	it "can negate simple logic" $
		// 	  get "/entities?not.or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can negate simple logic",
			"/entities?not.or=(id.eq.1,id.eq.2)&select=id",
			`[{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	it "can be combined with traditional filters" $
		// 	  get "/entities?or=(id.eq.1,id.eq.2)&name=eq.entity 1&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can be combined with traditional filters",
			"/entities?or=(id.eq.1,id.eq.2)&name=eq.entity%201&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		//   context "embedded levels" $ do
		// 	it "can do logic on the second level" $
		// 	  get "/entities?child_entities.or=(id.eq.1,name.eq.child entity 2)&select=id,child_entities(id)" `shouldRespondWith`
		// 		[json|[
		// 		  {"id": 1, "child_entities": [ { "id": 1 }, { "id": 2 } ] }, { "id": 2, "child_entities": []},
		// 		  {"id": 3, "child_entities": []}, {"id": 4, "child_entities": []}
		// 		]|] { matchHeaders = [matchContentTypeJson] }
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
			"can be nested inside the same expression",
			"/entities?or=(and(name.eq.entity%202,id.eq.2),and(name.eq.entity%201,id.eq.1))&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	it "can be negated while nested" $
		// 	  get "/entities?or=(not.and(name.eq.entity 2,id.eq.2),not.and(name.eq.entity 1,id.eq.1))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can be negated while nested",
			"/entities?or=(not.and(name.eq.entity%202,id.eq.2),not.and(name.eq.entity%201,id.eq.1))&select=id",
			`[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	it "can be combined unnested" $
		// 	  get "/entities?and=(id.eq.1,name.eq.entity 1)&or=(id.eq.1,id.eq.2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can be combined unnested",
			"/entities?and=(id.eq.1,name.eq.entity%201)&or=(id.eq.1,id.eq.2)&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		//   context "operators inside and/or" $ do
		// 	it "can handle eq and neq" $
		// 	  get "/entities?and=(id.eq.1,id.neq.2))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle eq and neq",
			"/entities?and=(id.eq.1,id.neq.2))&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	it "can handle lt and gt" $
		// 	  get "/entities?or=(id.lt.2,id.gt.3)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle lt and gt",
			"/entities?or=(id.lt.2,id.gt.3)&select=id",
			`[{"id":1},{"id":4}]`,
			nil,
			200,
		},
		// 	it "can handle lte and gte" $
		// 	  get "/entities?or=(id.lte.2,id.gte.3)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle lte and gte",
			"/entities?or=(id.lte.2,id.gte.3)&select=id",
			`[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	it "can handle like and ilike" $
		// 	  get "/entities?or=(name.like.*1,name.ilike.*ENTITY 2)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle like and ilike",
			"/entities?or=(name.like.*1,name.ilike.*ENTITY%202)&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	it "can handle in" $
		// 	  get "/entities?or=(id.in.(1,2),id.in.(3,4))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle in",
			"/entities?or=(id.in.(1,2),id.in.(3,4))&select=id",
			`[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	it "can handle is" $
		// 	  get "/entities?and=(name.is.null,arr.is.null)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle is",
			"/entities?and=(name.is.null,arr.is.null)&select=id",
			`[{"id":4}]`,
			nil,
			200,
		},
		// 	it "can handle fts" $ do
		// 	  get "/entities?or=(text_search_vector.fts.bar,text_search_vector.fts.baz)&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle fts",
			"/entities?or=(text_search_vector.fts.bar,text_search_vector.fts.baz)&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
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
			"can handle cs and cd",
			"/entities?or=(arr.cs.{1,2,3},arr.cd.{1})&select=id",
			`[{"id":1},{"id":3}]`,
			nil,
			200,
		},

		// RANGES

		// 	it "can handle range operators" $ do
		// 	  get "/ranges?range=eq.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=eq.[1,3]&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=neq.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=neq.[1,3]&select=id",
			`[{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=lt.[1,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=lt.[1,10]&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=gt.[8,11]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=gt.[8,11]&select=id",
			`[{"id":4}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=lte.[1,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=lte.[1,3]&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=gte.[2,3]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=gte.[2,3]&select=id",
			`[{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=cs.[1,2]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=cs.[1,2]&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=cd.[1,6]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=cd.[1,6]&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=ov.[0,4]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=ov.[0,4]&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=sl.[9,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=sl.[9,10]&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=sr.[3,4]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=sr.[3,4]&select=id",
			`[{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=nxr.[4,7]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=nxr.[4,7]&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=nxl.[4,7]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=nxl.[4,7]&select=id",
			`[{"id":3},{"id":4}]`,
			nil,
			200,
		},
		// 	  get "/ranges?range=adj.(3,10]&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle range operators",
			"/ranges?range=adj.(3,10]&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},

		// ARRAYS

		// 	it "can handle array operators" $ do
		// 	  get "/entities?arr=eq.{1,2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=eq.{1,2,3}&select=id",
			`[{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=neq.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=neq.{1,2}&select=id",
			`[{"id":1},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=lt.{2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=lt.{2,3}&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=lt.{2,0}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=lt.{2,0}&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=gt.{1,1}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=gt.{1,1}&select=id",
			`[{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=gt.{3}&select=id" `shouldRespondWith`
		// 		[json|[]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=gt.{3}&select=id",
			`[]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=lte.{2,1}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=lte.{2,1}&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=lte.{1,2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=lte.{1,2,3}&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=lte.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=lte.{1,2}&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=cs.{1,2}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=cs.{1,2}&select=id",
			`[{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=cd.{1,2,6}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=cd.{1,2,6}&select=id",
			`[{"id":1},{"id":2}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=ov.{3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=ov.{3}&select=id",
			`[{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/entities?arr=ov.{2,3}&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 2 }, { "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can handle array operators",
			"/entities?arr=ov.{2,3}&select=id",
			`[{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	context "operators with not" $ do
		// 	  it "eq, cs, like can be negated" $
		// 		get "/entities?and=(arr.not.cs.{1,2,3},and(id.not.eq.2,name.not.like.*3))&select=id" `shouldRespondWith`
		// 		  [json|[{ "id": 1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"eq, cs, like can be negated",
			"/entities?and=(arr.not.cs.{1,2,3},and(id.not.eq.2,name.not.like.*3))&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  it "in, is, fts can be negated" $
		// 		get "/entities?and=(id.not.in.(1,3),and(name.not.is.null,text_search_vector.not.fts.foo))&select=id" `shouldRespondWith`
		// 		  [json|[{ "id": 2}]|] { matchHeaders = [matchContentTypeJson] }
		// {
		// 	"can handle array operators",
		// 	"/entities?arr=ov.{2,3}&select=id",
		// 	`[{"id":2},{"id":3}]`,
		// 	nil,
		// 	200,
		// },
		// 	  it "lt, gte, cd can be negated" $
		// 		get "/entities?and=(arr.not.cd.{1},or(id.not.lt.1,id.not.gte.3))&select=id" `shouldRespondWith`
		// 		  [json|[{"id": 2}, {"id": 3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"lt, gte, cd can be negated",
			"/entities?and=(arr.not.cd.{1},or(id.not.lt.1,id.not.gte.3))&select=id",
			`[{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  it "gt, lte, ilike can be negated" $
		// 		get "/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id" `shouldRespondWith`
		// 		  [json|[{"id": 1}, {"id": 2}, {"id": 3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"gt, lte, ilike can be negated",
			"/entities?and=(name.not.ilike.*ITY2,or(id.not.gt.4,id.not.lte.1))&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},

		//   context "and/or params with quotes" $ do
		// 	it "eq can have quotes" $
		// 	  get "/grandchild_entities?or=(name.eq.\"(grandchild,entity,4)\",name.eq.\"(grandchild,entity,5)\")&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }, { "id": 5 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"eq can have quotes",
			"/grandchild_entities?or=(name.eq.\"(grandchild,entity,4)\",name.eq.\"(grandchild,entity,5)\")&select=id",
			`[{"id":4},{"id":5}]`,
			nil,
			200,
		},
		// 	it "like and ilike can have quotes" $
		// 	  get "/grandchild_entities?or=(name.like.\"*ity,4*\",name.ilike.\"*ITY,5)\")&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 4 }, { "id": 5 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"like and ilike can have quotes",
			"/grandchild_entities?or=(name.like.\"*ity,4*\",name.ilike.\"*ITY,5)\")&select=id",
			`[{"id":4},{"id":5}]`,
			nil,
			200,
		},
		// 	it "in can have quotes" $
		// 	  get "/grandchild_entities?or=(id.in.(\"1\",\"2\"),id.in.(\"3\",\"4\"))&select=id" `shouldRespondWith`
		// 		[json|[{ "id": 1 }, { "id": 2 }, { "id": 3 }, { "id": 4 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"in can have quotes",
			"/grandchild_entities?or=(id.in.(\"1\",\"2\"),id.in.(\"3\",\"4\"))&select=id",
			`[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			nil,
			200,
		},

		//   it "allows whitespace" $
		// 	get "/entities?and=( and ( id.in.( 1, 2, 3 ) , id.eq.3 ) , or ( id.eq.2 , id.eq.3 ) )&select=id" `shouldRespondWith`
		// 	  [json|[{ "id": 3 }]|] { matchHeaders = [matchContentTypeJson] }
		{
			"allows whitespace",
			"/entities?and=(%20and%20(%20id.in.(%201,%202,%203%20)%20,%20id.eq.3%20)%20,%20or%20(%20id.eq.2%20,%20id.eq.3%20)%20)&select=id",
			`[{"id":3}]`,
			nil,
			200,
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
		// 	it "can have a single condition" $ do
		// 	  get "/entities?or=(id.eq.1)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have a single condition",
			"/entities?or=(id.eq.1)&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/entities?and=(id.eq.1)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have a single condition",
			"/entities?and=(id.eq.1)&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	it "can have three conditions" $ do
		// 	  get "/grandchild_entities?or=(id.eq.1, id.eq.2, id.eq.3)&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}, {"id":2}, {"id":3}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have three conditions",
			"/grandchild_entities?or=(id.eq.1,%20id.eq.2,%20id.eq.3)&select=id",
			`[{"id":1},{"id":2},{"id":3}]`,
			nil,
			200,
		},
		// 	  get "/grandchild_entities?and=(id.in.(1,2), id.in.(3,1), id.in.(1,4))&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have three conditions",
			"/grandchild_entities?and=(id.in.(1,2),%20id.in.(3,1),%20id.in.(1,4))&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	it "can have four conditions combining and/or" $ do
		// 	  get "/grandchild_entities?or=( id.eq.1, id.eq.2, and(id.in.(1,3), id.in.(2,3)), id.eq.4 )&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}, {"id":2}, {"id":3}, {"id":4}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have four conditions combining and/or",
			"/grandchild_entities?or=(%20id.eq.1,%20id.eq.2,%20and(id.in.(1,3),%20id.in.(2,3)),%20id.eq.4%20)&select=id",
			`[{"id":1}]`,
			nil,
			200,
		},
		// 	  get "/grandchild_entities?and=( id.eq.1, not.or(id.eq.2, id.eq.3), id.in.(1,4), or(id.eq.1, id.eq.4) )&select=id" `shouldRespondWith`
		// 		[json|[{"id":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			"can have four conditions combining and/or",
			"/grandchild_entities?and=(%20id.eq.1,%20not.or(id.eq.2,%20id.eq.3),%20id.in.(1,4),%20or(id.eq.1,%20id.eq.4)%20)&select=id",
			`[{"id":1}]`,
			nil,
			200,
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

		// context "used with PATCH" $
		//   it "succeeds when using and/or params" $
		// 	request methodPatch "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name"
		// 	  [("Prefer", "return=representation")]
		// 	  [json|{ name : "updated grandchild entity"}|] `shouldRespondWith`
		// 	  [json|[{ "id": 1, "name" : "updated grandchild entity"},{ "id": 2, "name" : "updated grandchild entity"}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// context "used with DELETE" $
		//   it "succeeds when using and/or params" $
		// 	request methodDelete "/grandchild_entities?or=(id.eq.1,id.eq.2)&select=id,name"
		// 		[("Prefer", "return=representation")]
		// 		""
		// 	  `shouldRespondWith`
		// 		[json|[{ "id": 1, "name" : "grandchild entity 1" },{ "id": 2, "name" : "grandchild entity 2" }]|]

		// it "can query columns that begin with and/or reserved words" $
		//   get "/grandchild_entities?or=(and_starting_col.eq.smth, or_starting_col.eq.smth)" `shouldRespondWith` 200

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

	for i, test := range tests {
		query := "http://localhost:8081/databases/pgrest" + test.query
		req, err := http.NewRequest("GET", query, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Accept-Profile", "test")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		json := string(body)
		if test.expectedResults != "" {
			if test.expectedResults != json {
				t.Errorf("\n\n%d. %v\nExpected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.description, test.expectedResults, json, test.query)
			}
		} else if test.status != resp.StatusCode {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.description, test.status, resp.StatusCode, test.query)

		}
		resp.Body.Close()
	}
}
