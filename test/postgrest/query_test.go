package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Query(t *testing.T) {

	tests := []test.Test{
		//	describe "Filtering response" $ do
		// it "matches with equality" $
		//   get "/items?id=eq.5"
		//     `shouldRespondWith` [json| [{"id":5}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-0/*"] }
		{
			Description:     "matches with equality",
			Query:           "/items?id=eq.5",
			Headers:         nil,
			Expected:        `[{"id":5}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},
		// it "matches with equality using not operator" $
		//   get "/items?id=not.eq.5&order=id"
		//     `shouldRespondWith` [json| [{"id":1},{"id":2},{"id":3},{"id":4},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-13/*"] }
		{
			Description:     "matches with equality using not operator",
			Query:           "/items?id=not.eq.5&order=id",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-13/*"},
			Status:          200,
		},
		// it "matches with more than one condition using not operator" $
		//   get "/simple_pk?k=like.*yx&extra=not.eq.u" `shouldRespondWith` "[]"
		{
			Description: "matches with more than one condition using not operator",
			Query:       "/simple_pk?k=like.*yx&extra=not.eq.u",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},
		// it "matches with inequality using not operator" $ do
		//   get "/items?id=not.lt.14&order=id.asc"
		//     `shouldRespondWith` [json| [{"id":14},{"id":15}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-1/*"] }
		{
			Description:     "matches with inequality using not operator",
			Query:           "/items?id=not.lt.14&order=id.asc",
			Headers:         nil,
			Expected:        `[{"id":14},{"id":15}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   get "/items?id=not.gt.2&order=id.asc"
		//     `shouldRespondWith` [json| [{"id":1},{"id":2}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-1/*"] }
		{
			Description:     "matches with inequality using not operator",
			Query:           "/items?id=not.gt.2&order=id.asc",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		// it "matches items IN" $
		//   get "/items?id=in.(1,3,5)"
		//     `shouldRespondWith` [json| [{"id":1},{"id":3},{"id":5}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-2/*"] }
		{
			Description:     "matches items IN",
			Query:           "/items?id=in.(1,3,5)",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":3},{"id":5}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		// it "matches items NOT IN using not operator" $
		//   get "/items?id=not.in.(2,4,6,7,8,9,10,11,12,13,14,15)"
		//     `shouldRespondWith` [json| [{"id":1},{"id":3},{"id":5}] |]
		//     { matchHeaders = ["Content-Range" <:> "0-2/*"] }
		{
			Description:     "matches items NOT IN using not operator",
			Query:           "/items?id=not.in.(2,4,6,7,8,9,10,11,12,13,14,15)",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":3},{"id":5}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		// it "matches nulls using not operator" $
		//   get "/no_pk?a=not.is.null" `shouldRespondWith`
		//     [json| [{"a":"1","b":"0"},{"a":"2","b":"0"}] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches nulls using not operator",
			Query:       "/no_pk?a=not.is.null",
			Headers:     nil,
			Expected:    `[{"a":"1","b":"0"},{"a":"2","b":"0"}]`,
			Status:      200,
		},
		// it "matches nulls in varchar and numeric fields alike" $ do
		//   get "/no_pk?a=is.null" `shouldRespondWith`
		//     [json| [{"a": null, "b": null}] |]
		{
			Description: "matches nulls in varchar and numeric fields alike",
			Query:       "/nullable_integer?a=is.null",
			Headers:     nil,
			Expected:    `[{"a":null}]`,
			Status:      200,
		},
		//     { matchHeaders = [matchContentTypeJson] }
		//   get "/nullable_integer?a=is.null" `shouldRespondWith` [json|[{"a":null}]|]
		{
			Description: "matches nulls in varchar and numeric fields alike",
			Query:       "/no_pk?a=is.null",
			Headers:     nil,
			Expected:    `[{"a":null,"b":null}]`,
			Status:      200,
		},
		// it "matches with trilean values" $ do
		//   get "/chores?done=is.true" `shouldRespondWith`
		//     [json| [{"id": 1, "name": "take out the garbage", "done": true }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values",
			Query:       "/chores?done=is.true",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"take out the garbage","done":true}]`,
			Status:      200,
		},
		//   get "/chores?done=is.false" `shouldRespondWith`
		//     [json| [{"id": 2, "name": "do the laundry", "done": false }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values",
			Query:       "/chores?done=is.false",
			Headers:     nil,
			Expected:    `[{"id":2,"name":"do the laundry","done":false}]`,
			Status:      200,
		},
		//   get "/chores?done=is.unknown" `shouldRespondWith`
		//     [json| [{"id": 3, "name": "wash the dishes", "done": null }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values",
			Query:       "/chores?done=is.unknown",
			Headers:     nil,
			Expected:    `[{"id":3,"name":"wash the dishes","done":null}]`,
			Status:      200,
		},
		// it "matches with trilean values in upper or mixed case" $ do
		//   get "/chores?done=is.NULL" `shouldRespondWith`
		//     [json| [{"id": 3, "name": "wash the dishes", "done": null }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values in upper or mixed case",
			Query:       "/chores?done=is.NULL",
			Headers:     nil,
			Expected:    `[{"id":3,"name":"wash the dishes","done":null}]`,
			Status:      200,
		},
		//   get "/chores?done=is.TRUE" `shouldRespondWith`
		//     [json| [{"id": 1, "name": "take out the garbage", "done": true }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values in upper or mixed case",
			Query:       "/chores?done=is.TRUE",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"take out the garbage","done":true}]`,
			Status:      200,
		},
		//   get "/chores?done=is.FAlSe" `shouldRespondWith`
		//     [json| [{"id": 2, "name": "do the laundry", "done": false }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values in upper or mixed case",
			Query:       "/chores?done=is.FAlSe",
			Headers:     nil,
			Expected:    `[{"id":2,"name":"do the laundry","done":false}]`,
			Status:      200,
		},
		//   get "/chores?done=is.UnKnOwN" `shouldRespondWith`
		//     [json| [{"id": 3, "name": "wash the dishes", "done": null }] |]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with trilean values in upper or mixed case",
			Query:       "/chores?done=is.UnKnOwN",
			Headers:     nil,
			Expected:    `[{"id":3,"name":"wash the dishes","done":null}]`,
			Status:      200,
		},
		// it "fails if 'is' used and there's no null or trilean value" $ do
		//   get "/chores?done=is.nil" `shouldRespondWith` 400
		{
			Description: "fails if 'is' used and there's no null or trilean value",
			Query:       "/chores?done=is.nil",
			Headers:     nil,
			Expected:    "",
			Status:      400,
		},
		//   get "/chores?done=is.ok"  `shouldRespondWith` 400
		{
			Description: "fails if 'is' used and there's no null or trilean value",
			Query:       "/chores?done=is.ok",
			Headers:     nil,
			Expected:    "",
			Status:      400,
		},
		// it "matches with like" $ do
		//   get "/simple_pk?k=like.*yx" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"}]|]
		{
			Description: "matches with like",
			Query:       "/simple_pk?k=like.*yx",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=like.xy*" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"}]|]
		{
			Description: "matches with like",
			Query:       "/simple_pk?k=like.xy*",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=like.*YY*" `shouldRespondWith`
		//     [json|[{"k":"xYYx","extra":"v"}]|]
		{
			Description: "matches with like",
			Query:       "/simple_pk?k=like.*YY*",
			Headers:     nil,
			Expected:    `[{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with like using not operator" $
		//   get "/simple_pk?k=not.like.*yx" `shouldRespondWith`
		//     [json|[{"k":"xYYx","extra":"v"}]|]
		{
			Description: "matches with like using not operator",
			Query:       "/simple_pk?k=not.like.*yx",
			Headers:     nil,
			Expected:    `[{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with ilike" $ do
		//   get "/simple_pk?k=ilike.xy*&order=extra.asc" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]|]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with ilike",
			Query:       "/simple_pk?k=ilike.xy*&order=extra.asc",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=ilike.*YY*&order=extra.asc" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]|]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with ilike",
			Query:       "/simple_pk?k=ilike.*YY*&order=extra.asc",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with ilike using not operator" $
		//   get "/simple_pk?k=not.ilike.xy*&order=extra.asc" `shouldRespondWith` "[]"
		{
			Description: "matches with ilike using not operator",
			Query:       "/simple_pk?k=not.ilike.xy*&order=extra.asc",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},
		// it "matches with ~" $ do
		//   get "/simple_pk?k=match.yx$" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"}]|]
		{
			Description: "matches with ~",
			Query:       "/simple_pk?k=match.yx$",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=match.^xy" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"}]|]
		{
			Description: "matches with ~",
			Query:       "/simple_pk?k=match.^xy",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=match.YY" `shouldRespondWith`
		//     [json|[{"k":"xYYx","extra":"v"}]|]
		{
			Description: "matches with ~",
			Query:       "/simple_pk?k=match.YY",
			Headers:     nil,
			Expected:    `[{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with ~ using not operator" $
		//   get "/simple_pk?k=not.match.yx$" `shouldRespondWith`
		//     [json|[{"k":"xYYx","extra":"v"}]|]
		{
			Description: "matches with ~",
			Query:       "/simple_pk?k=match.YY",
			Headers:     nil,
			Expected:    `[{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with ~*" $ do
		//   get "/simple_pk?k=imatch.^xy&order=extra.asc" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]|]
		//     { matchHeaders = [matchContentTypeJson] }

		{
			Description: "matches with ~*",
			Query:       "/simple_pk?k=imatch.^xy&order=extra.asc",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		//   get "/simple_pk?k=imatch..*YY.*&order=extra.asc" `shouldRespondWith`
		//     [json|[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]|]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with ~*",
			Query:       "/simple_pk?k=imatch..*YY.*&order=extra.asc",
			Headers:     nil,
			Expected:    `[{"k":"xyyx","extra":"u"},{"k":"xYYx","extra":"v"}]`,
			Status:      200,
		},
		// it "matches with ~* using not operator" $
		//   get "/simple_pk?k=not.imatch.^xy&order=extra.asc" `shouldRespondWith` "[]"
		{
			Description: "matches with ~* using not operator",
			Query:       "/simple_pk?k=not.imatch.^xy&order=extra.asc",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},

		// describe "Full text search operator" $ do
		// it "finds matches with to_tsquery" $
		//   get "/tsearch?text_search_vector=fts.impossible" `shouldRespondWith`
		// 	[json| [{"text_search_vector": "'fun':5 'imposs':9 'kind':3" }] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with to_tsquery",
			Query:       "/tsearch?text_search_vector=fts.impossible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]`,
			Status:      200,
		},
		// it "can use lexeme boolean operators(&=%26, |=%7C, !) in to_tsquery" $ do
		//   get "/tsearch?text_search_vector=fts.fun%26possible" `shouldRespondWith`
		// 	[json| [ {"text_search_vector": "'also':2 'fun':3 'possibl':8"}] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use lexeme boolean operators(&=%26, |=%7C, !) in to_tsquery",
			Query:       "/tsearch?text_search_vector=fts.fun%26possible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'also':2 'fun':3 'possibl':8"}]`,
			Status:      200,
		},
		//   get "/tsearch?text_search_vector=fts.impossible%7Cpossible" `shouldRespondWith`
		// 	[json| [
		// 	{"text_search_vector": "'fun':5 'imposs':9 'kind':3"},
		// 	{"text_search_vector": "'also':2 'fun':3 'possibl':8"}] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use lexeme boolean operators(&=%26, |=%7C, !) in to_tsquery",
			Query:       "/tsearch?text_search_vector=fts.impossible%7Cpossible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"},{"text_search_vector":"'also':2 'fun':3 'possibl':8"}]`,
			Status:      200,
		},
		//   get "/tsearch?text_search_vector=fts.fun%26!possible" `shouldRespondWith`
		// 	[json| [ {"text_search_vector": "'fun':5 'imposs':9 'kind':3"}] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use lexeme boolean operators(&=%26, |=%7C, !) in to_tsquery",
			Query:       "/tsearch?text_search_vector=fts.fun%26!possible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]`,
			Status:      200,
		},
		// it "finds matches with plainto_tsquery" $
		//   get "/tsearch?text_search_vector=plfts.The%20Fat%20Rats" `shouldRespondWith`
		// 	[json| [ {"text_search_vector": "'ate':3 'cat':2 'fat':1 'rat':4" }] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with plainto_tsquery",
			Query:       "/tsearch?text_search_vector=plfts.The%20Fat%20Rats",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'ate':3 'cat':2 'fat':1 'rat':4"}]`,
			Status:      200,
		},
		// when (actualPgVersion >= pgVersion112) $ do
		//   it "finds matches with websearch_to_tsquery" $
		// 	  get "/tsearch?text_search_vector=wfts.The%20Fat%20Rats" `shouldRespondWith`
		// 		  [json| [ {"text_search_vector": "'ate':3 'cat':2 'fat':1 'rat':4" }] |]
		// 		  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with websearch_to_tsquery",
			Query:       "/tsearch?text_search_vector=wfts.The%20Fat%20Rats",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'ate':3 'cat':2 'fat':1 'rat':4"}]`,
			Status:      200,
		},
		//   it "can use boolean operators(and, or, -) in websearch_to_tsquery" $ do
		// 	get "/tsearch?text_search_vector=wfts.fun%20and%20possible"
		// 	  `shouldRespondWith`
		// 		[json| [ {"text_search_vector": "'also':2 'fun':3 'possibl':8"}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use boolean operators(and, or, -) in websearch_to_tsquery",
			Query:       "/tsearch?text_search_vector=wfts.fun%20and%20possible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'also':2 'fun':3 'possibl':8"}]`,
			Status:      200,
		},
		// 	get "/tsearch?text_search_vector=wfts.impossible%20or%20possible"
		// 	  `shouldRespondWith`
		// 		[json| [
		// 		  {"text_search_vector": "'fun':5 'imposs':9 'kind':3"},
		// 		  {"text_search_vector": "'also':2 'fun':3 'possibl':8"}]
		// 			|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use boolean operators(and, or, -) in websearch_to_tsquery",
			Query:       "/tsearch?text_search_vector=wfts.impossible%20or%20possible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"},{"text_search_vector":"'also':2 'fun':3 'possibl':8"}]`,
			Status:      200,
		},
		// 	get "/tsearch?text_search_vector=wfts.fun%20and%20-possible"
		// 	  `shouldRespondWith`
		// 		[json| [ {"text_search_vector": "'fun':5 'imposs':9 'kind':3"}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can use boolean operators(and, or, -) in websearch_to_tsquery",
			Query:       "/tsearch?text_search_vector=wfts.fun%20and%20-possible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]`,
			Status:      200,
		},
		// it "finds matches with different dictionaries" $ do
		//   get "/tsearch?text_search_vector=fts(french).amusant" `shouldRespondWith`
		// 	[json| [{"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" }] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with different dictionaries",
			Query:       "/tsearch?text_search_vector=fts(french).amusant",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"}]`,
			Status:      200,
		},
		//   get "/tsearch?text_search_vector=plfts(french).amusant%20impossible" `shouldRespondWith`
		// 	[json| [{"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" }] |]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with different dictionaries",
			Query:       "/tsearch?text_search_vector=plfts(french).amusant%20impossible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"}]`,
			Status:      200,
		},
		//   when (actualPgVersion >= pgVersion112) $
		// 	  get "/tsearch?text_search_vector=wfts(french).amusant%20impossible"
		// 		  `shouldRespondWith`
		// 			[json| [{"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" }] |]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with different dictionaries",
			Query:       "/tsearch?text_search_vector=wfts(french).amusant%20impossible",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"}]`,
			Status:      200,
		},
		// it "can be negated with not operator" $ do
		//   get "/tsearch?text_search_vector=not.fts.impossible%7Cfat%7Cfun" `shouldRespondWith`
		// 	[json| [
		// 	  {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4"},
		// 	  {"text_search_vector": "'art':4 'spass':5 'unmog':7"}]|]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated with not operator",
			Query:       "/tsearch?text_search_vector=not.fts.impossible%7Cfat%7Cfun",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status:      200,
		},
		//   get "/tsearch?text_search_vector=not.fts(english).impossible%7Cfat%7Cfun" `shouldRespondWith`
		// 	[json| [
		// 	  {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4"},
		// 	  {"text_search_vector": "'art':4 'spass':5 'unmog':7"}]|]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated with not operator",
			Query:       "/tsearch?text_search_vector=not.fts(english).impossible%7Cfat%7Cfun",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status:      200,
		},
		//   get "/tsearch?text_search_vector=not.plfts.The%20Fat%20Rats" `shouldRespondWith`
		// 	[json| [
		// 	  {"text_search_vector": "'fun':5 'imposs':9 'kind':3"},
		// 	  {"text_search_vector": "'also':2 'fun':3 'possibl':8"},
		// 	  {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4"},
		// 	  {"text_search_vector": "'art':4 'spass':5 'unmog':7"}]|]
		// 	{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated with not operator",
			Query:       "/tsearch?text_search_vector=not.plfts.The%20Fat%20Rats",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"},{"text_search_vector":"'also':2 'fun':3 'possibl':8"},{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status:      200,
		},
		//   when (actualPgVersion >= pgVersion112) $
		// 	  get "/tsearch?text_search_vector=not.wfts(english).impossible%20or%20fat%20or%20fun"
		// 		  `shouldRespondWith`
		// 			[json| [
		// 			  {"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4"},
		// 			  {"text_search_vector": "'art':4 'spass':5 'unmog':7"}]|]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated with not operator",
			Query:       "/tsearch?text_search_vector=not.wfts(english).impossible%20or%20fat%20or%20fun",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status:      200,
		},
		// context "Use of the phraseto_tsquery function" $ do
		//   it "finds matches" $
		// 	get "/tsearch?text_search_vector=phfts.The%20Fat%20Cats" `shouldRespondWith`
		// 	  [json| [{"text_search_vector": "'ate':3 'cat':2 'fat':1 'rat':4" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches",
			Query:       "/tsearch?text_search_vector=phfts.The%20Fat%20Cats",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'ate':3 'cat':2 'fat':1 'rat':4"}]`,
			Status:      200,
		},
		//   it "finds matches with different dictionaries" $
		// 	get "/tsearch?text_search_vector=phfts(german).Art%20Spass" `shouldRespondWith`
		// 	  [json| [{"text_search_vector": "'art':4 'spass':5 'unmog':7" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finds matches with different dictionaries",
			Query:       "/tsearch?text_search_vector=phfts(german).Art%20Spass",
			Headers:     nil,
			Expected:    `[{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status:      200,
		},
		//   it "can be negated with not operator" $
		// 	get "/tsearch?text_search_vector=not.phfts(english).The%20Fat%20Cats" `shouldRespondWith`
		// 	  [json| [
		// 		{"text_search_vector": "'fun':5 'imposs':9 'kind':3"},
		// 		{"text_search_vector": "'also':2 'fun':3 'possibl':8"},
		// 		{"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4"},
		// 		{"text_search_vector": "'art':4 'spass':5 'unmog':7"}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be negated with not operator",
			Query:       "/tsearch?text_search_vector=not.phfts(english).The%20Fat%20Cats",
			Headers:     nil,
			Expected: `[
				{"text_search_vector":"'fun':5 'imposs':9 'kind':3"},
				{"text_search_vector":"'also':2 'fun':3 'possibl':8"},
				{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},
				{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status: 200,
		},
		//   it "can be used with or query param" $
		// 	get "/tsearch?or=(text_search_vector.phfts(german).Art%20Spass, text_search_vector.phfts(french).amusant, text_search_vector.fts(english).impossible)" `shouldRespondWith`
		// 	  [json|[
		// 		{"text_search_vector": "'fun':5 'imposs':9 'kind':3" },
		// 		{"text_search_vector": "'amus':5 'fair':7 'impossibl':9 'peu':4" },
		// 		{"text_search_vector": "'art':4 'spass':5 'unmog':7"}
		// 	  ]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be used with or query param",
			Query:       "/tsearch?or=(text_search_vector.phfts(german).Art%20Spass,text_search_vector.phfts(french).amusant,text_search_vector.fts(english).impossible)",
			Headers:     nil,
			Expected: `[
				{"text_search_vector":"'fun':5 'imposs':9 'kind':3"},
				{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},
				{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]`,
			Status: 200,
		},
		// 	it "matches with computed column" $
		// 	get "/items?always_true=eq.true&order=id.asc" `shouldRespondWith`
		// 	  [json| [{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with computed column",
			Query:       "/items?always_true=eq.true&order=id.asc",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}]`,
			Status:      200,
		},
		//   it "order by computed column" $
		// 	get "/items?order=anti_id.desc" `shouldRespondWith`
		// 	  [json| [{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "order by computed column",
			Query:       "/items?order=anti_id.desc",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},{"id":14},{"id":15}]`,
			Status:      200,
		},
		//   it "cannot access a computed column that is outside of the config schema" $
		// 	get "/items?always_false=is.false" `shouldRespondWith` 400
		{
			Description: "cannot access a computed column that is outside of the config schema",
			Query:       "/items?always_false=is.false",
			Status:      400,
		},
		//   it "matches filtering nested items" $
		// 	get "/clients?select=id,projects(id,tasks(id,name))&projects.tasks.name=like.Design*" `shouldRespondWith`
		// 	  [json|[{"id":1,"projects":[{"id":1,"tasks":[{"id":1,"name":"Design w7"}]},{"id":2,"tasks":[{"id":3,"name":"Design w10"}]}]},{"id":2,"projects":[{"id":3,"tasks":[{"id":5,"name":"Design IOS"}]},{"id":4,"tasks":[{"id":7,"name":"Design OSX"}]}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches filtering nested items",
			Query:       "/clients?select=id,projects(id,tasks(id,name))&projects.tasks.name=like.Design*",
			Headers:     nil,
			Expected:    `[{"id":1,"projects":[{"id":1,"tasks":[{"id":1,"name":"Design w7"}]},{"id":2,"tasks":[{"id":3,"name":"Design w10"}]}]},{"id":2,"projects":[{"id":3,"tasks":[{"id":5,"name":"Design IOS"}]},{"id":4,"tasks":[{"id":7,"name":"Design OSX"}]}]}]`,
			Status:      200,
		},
		//   it "errs when the embedded resource doesn't exist and an embedded filter is applied to it" $ do
		// 	get "/clients?select=*&non_existent_projects.name=like.*NonExistent*" `shouldRespondWith`
		// 	  [json|
		// 		{"hint":"Verify that 'non_existent_projects' is included in the 'select' query parameter.",
		// 		 "details":null,
		// 		 "code":"PGRST108",
		// 		 "message":"Cannot apply filter because 'non_existent_projects' is not an embedded resource in this request"}|]
		// 	  { matchStatus  = 400
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }
		// @@ returns 200 failing to recognize that non_existent_projects was not selected
		// we need to check not inserted where nodes
		// {
		// 	Description: "errs when the embedded resource doesn't exist and an embedded filter is applied to it",
		// 	Query:       "/clients?select=*&non_existent_projects.name=like.*NonExistent*",
		// 	Headers:     nil,
		// 	Status:      400,
		// },
		// 	get "/clients?select=*,amiga_projects:projects(*)&amiga_projectsss.name=ilike.*Amiga*" `shouldRespondWith`
		// 	  [json|
		// 		{"hint":"Verify that 'amiga_projectsss' is included in the 'select' query parameter.",
		// 		 "details":null,
		// 		 "code":"PGRST108",
		// 		 "message":"Cannot apply filter because 'amiga_projectsss' is not an embedded resource in this request"}|]
		// 	  { matchStatus  = 400
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }
		// 	get "/clients?select=id,projects(id,tasks(id,name))&projects.tasks2.name=like.Design*" `shouldRespondWith`
		// 	  [json|
		// 		{"hint":"Verify that 'tasks2' is included in the 'select' query parameter.",
		// 		 "details":null,
		// 		 "code":"PGRST108",
		// 		 "message":"Cannot apply filter because 'tasks2' is not an embedded resource in this request"}|]
		// 	  { matchStatus  = 400
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }

		//   it "matches with cs operator" $
		// 	get "/complex_items?select=id&arr_data=cs.{2}" `shouldRespondWith`
		// 	  [json|[{"id":2},{"id":3}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with cs operator",
			Query:       "/complex_items?select=id&arr_data=cs.{2}",
			Headers:     nil,
			Expected:    `[{"id":2},{"id":3}]`,
			Status:      200,
		},
		//   it "matches with cd operator" $
		// 	get "/complex_items?select=id&arr_data=cd.{1,2,4}" `shouldRespondWith`
		// 	  [json|[{"id":1},{"id":2}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "matches with cd operator",
			Query:       "/complex_items?select=id&arr_data=cd.{1,2,4}",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		// describe "Shaping response with select parameter" $ do
		//   it "selectStar works in absense of parameter" $
		// 	get "/complex_items?id=eq.3" `shouldRespondWith`
		// 	  [json|[{"id":3,"name":"Three","settings":{"foo":{"int":1,"bar":"baz"}},"arr_data":[1,2,3],"field-with_sep":1}]|]
		{
			Description: "selectStar works in absense of parameter",
			Query:       "/complex_items?id=eq.3",
			Headers:     nil,
			Expected:    `[{"id":3,"name":"Three","settings":{"foo":{"int":1,"bar":"baz"}},"arr_data":[1,2,3],"field-with_sep":3}]`,
			Status:      200,
		},
		//   it "dash `-` in column names is accepted" $
		// 	get "/complex_items?id=eq.3&select=id,field-with_sep" `shouldRespondWith`
		// 	  [json|[{"id":3,"field-with_sep":1}]|]
		{
			Description: "dash `-` in column names is accepted",
			Query:       "/complex_items?id=eq.3&select=id,field-with_sep",
			Headers:     nil,
			Expected:    `[{"id":3,"field-with_sep":3}]`,
			Status:      200,
		},
		//   it "one simple column" $
		// 	get "/complex_items?select=id" `shouldRespondWith`
		// 	  [json| [{"id":1},{"id":2},{"id":3}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "one simple column",
			Query:       "/complex_items?select=id",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Status:      200,
		},
		//   it "rename simple column" $
		// 	get "/complex_items?id=eq.1&select=myId:id" `shouldRespondWith`
		// 	  [json| [{"myId":1}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "rename simple column",
			Query:       "/complex_items?id=eq.1&select=myId:id",
			Headers:     nil,
			Expected:    `[{"myId":1}]`,
			Status:      200,
		},
		//   it "one simple column with casting (text)" $
		// 	get "/complex_items?select=id::text" `shouldRespondWith`
		// 	  [json| [{"id":"1"},{"id":"2"},{"id":"3"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "one simple column with casting (text)",
			Query:       "/complex_items?select=id::text",
			Headers:     nil,
			Expected:    `[{"id":"1"},{"id":"2"},{"id":"3"}]`,
			Status:      200,
		},
		//   it "rename simple column with casting" $
		// 	get "/complex_items?id=eq.1&select=myId:id::text" `shouldRespondWith`
		// 	  [json| [{"myId":"1"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "rename simple column with casting",
			Query:       "/complex_items?id=eq.1&select=myId:id::text",
			Headers:     nil,
			Expected:    `[{"myId":"1"}]`,
			Status:      200,
		},
		//   it "json column" $
		// 	get "/complex_items?id=eq.1&select=settings" `shouldRespondWith`
		// 	  [json| [{"settings":{"foo":{"int":1,"bar":"baz"}}}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "json column",
			Query:       "/complex_items?id=eq.1&select=settings",
			Headers:     nil,
			Expected:    `[{"settings":{"foo":{"int":1,"bar":"baz"}}}]`,
			Status:      200,
		},
		//   it "fails on bad casting (wrong cast type)" $
		// 	get "/complex_items?select=id::fakecolumntype"
		// 	  `shouldRespondWith` [json| {"hint":null,"details":null,"code":"42704","message":"type \"fakecolumntype\" does not exist"} |]
		// 	  { matchStatus  = 400
		// 	  , matchHeaders = []
		// 	  }

		//   it "can cast types with underscore and numbers" $
		// 	get "/oid_test?select=id,oid_col::int,oid_array_col::_int4"
		// 	  `shouldRespondWith` [json|
		// 		[{"id":1,"oid_col":12345,"oid_array_col":[1,2,3,4,5]}]
		// 	  |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can cast types with underscore and number",
			Query:       "/oid_test?select=id,oid_col::int,oid_array_col::_int4",
			Headers:     nil,
			Expected:    `[{"id":1,"oid_col":12345,"oid_array_col":[1,2,3,4,5]}]`,
			Status:      200,
		},
		//   it "requesting parents and children" $
		// 	get "/projects?id=eq.1&select=id, name, clients(*), tasks(id, name)" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting parents and children",
			Query:       "/projects?id=eq.1&select=id, name, clients(*), tasks(id, name)",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]`,
			Status:      200,
		},
		// @@ added by me
		{
			Description: "requesting parents and children with filters",
			Query:       "/projects?id=eq.1&select=id, name, clients(*), tasks(id, name)&tasks.name=eq.Code%20w7",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":2,"name":"Code w7"}]}]`,
			Status:      200,
		},
		// @@ added by me
		{
			Description: "requesting parents and children with two embedded filters",
			Query:       "/projects?select=id,name,clients(*),tasks(*)&id=not.eq.4&clients.id=eq.1&tasks.id=in.(1,3,5)",
			Headers:     nil,
			Expected: `[
				{"id": 1,"name": "Windows 7", "clients": {"id": 1,"name": "Microsoft"}, "tasks": [{"id": 1,"name": "Design w7","project_id": 1}]},
				{"id": 2,"name": "Windows 10", "clients": {"id": 1,"name": "Microsoft"}, "tasks": [{"id": 3,"name": "Design w10","project_id": 2}]},
				{"id": 3,"name": "IOS", "clients": null, "tasks": [{"id": 5,"name": "Design IOS","project_id": 3 }]},
				{"id": 5,"name": "Orphan", "clients": null, "tasks": []}
			]`,
			Status: 200,
		},
		//   it "requesting parent and renaming primary key" $
		// 	get "/projects?select=name,client:clients(clientId:id,name)" `shouldRespondWith`
		// 	  [json|[
		// 		{"name":"Windows 7","client":{"name": "Microsoft", "clientId": 1}},
		// 		{"name":"Windows 10","client":{"name": "Microsoft", "clientId": 1}},
		// 		{"name":"IOS","client":{"name": "Apple", "clientId": 2}},
		// 		{"name":"OSX","client":{"name": "Apple", "clientId": 2}},
		// 		{"name":"Orphan","client":null}
		// 	  ]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting parent and renaming primary key",
			Query:       "/projects?select=name,client:clients(clientId:id,name)",
			Headers:     nil,
			Expected: `[
				{"name":"Windows 7","client":{"name": "Microsoft", "clientId": 1}},
				{"name":"Windows 10","client":{"name": "Microsoft", "clientId": 1}},
				{"name":"IOS","client":{"name": "Apple", "clientId": 2}},
				{"name":"OSX","client":{"name": "Apple", "clientId": 2}},
				{"name":"Orphan","client":null}
			]`,
			Status: 200,
		},
		//   it "requesting parent and specifying/renaming one key of the composite primary key" $ do
		// 	get "/comments?select=*,users_tasks(userId:user_id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"commenter_id":1,"user_id":2,"task_id":6,"content":"Needs to be delivered ASAP","users_tasks":{"userId": 2}}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting parent and specifying/renaming one key of the composite primary key",
			Query:       "/comments?select=*,users_tasks(userId:user_id)",
			Headers:     nil,
			Expected:    `[{"id":1,"commenter_id":1,"user_id":2,"task_id":6,"content":"Needs to be delivered ASAP","users_tasks":{"userId": 2}}]`,
			Status:      200,
		},
		// 	get "/comments?select=*,users_tasks(taskId:task_id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"commenter_id":1,"user_id":2,"task_id":6,"content":"Needs to be delivered ASAP","users_tasks":{"taskId": 6}}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting parent and specifying/renaming one key of the composite primary key",
			Query:       "/comments?select=*,users_tasks(taskId:task_id)",
			Headers:     nil,
			Expected:    `[{"id":1,"commenter_id":1,"user_id":2,"task_id":6,"content":"Needs to be delivered ASAP","users_tasks":{"taskId": 6}}]`,
			Status:      200,
		},
		//   it "requesting parents and children while renaming them" $
		// 	get "/projects?id=eq.1&select=myId:id, name, project_client:clients(*), project_tasks:tasks(id, name)" `shouldRespondWith`
		// 	  [json|[{"myId":1,"name":"Windows 7","project_client":{"id":1,"name":"Microsoft"},"project_tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting parents and children while renaming them",
			Query:       "/projects?id=eq.1&select=myId:id, name, project_client:clients(*), project_tasks:tasks(id, name)",
			Headers:     nil,
			Expected:    `[{"myId":1,"name":"Windows 7","project_client":{"id":1,"name":"Microsoft"},"project_tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]`,
			Status:      200,
		},
		//   it "requesting parents and filtering parent columns" $
		// 	get "/projects?id=eq.1&select=id, name, clients(id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","clients":{"id":1}}]|]
		{
			Description: "requesting parents and filtering parent columns",
			Query:       "/projects?id=eq.1&select=id, name, clients(id)",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","clients":{"id":1}}]`,
			Status:      200,
		},
		//   it "rows with missing parents are included" $
		// 	get "/projects?id=in.(1,5)&select=id,clients(id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"clients":{"id":1}},{"id":5,"clients":null}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "rows with missing parents are included",
			Query:       "/projects?id=in.(1,5)&select=id,clients(id)",
			Headers:     nil,
			Expected:    `[{"id":1,"clients":{"id":1}},{"id":5,"clients":null}]`,
			Status:      200,
		},
		//   it "rows with no children return [] instead of null" $
		// 	get "/projects?id=in.(5)&select=id,tasks(id)" `shouldRespondWith`
		// 	  [json|[{"id":5,"tasks":[]}]|]
		{
			Description: "rows with no children return [] instead of null",
			Query:       "/projects?id=in.(5)&select=id,tasks(id)",
			Headers:     nil,
			Expected:    `[{"id":5,"tasks":[]}]`,
			Status:      200,
		},
		//   it "requesting children 2 levels" $
		// 	get "/clients?id=eq.1&select=id,projects(id,tasks(id))" `shouldRespondWith`
		// 	  [json|[{"id":1,"projects":[{"id":1,"tasks":[{"id":1},{"id":2}]},{"id":2,"tasks":[{"id":3},{"id":4}]}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting children 2 levels",
			Query:       "/clients?id=eq.1&select=id,projects(id,tasks(id))",
			Expected:    `[{"id":1,"projects":[{"id":1,"tasks":[{"id":1},{"id":2}]},{"id":2,"tasks":[{"id":3},{"id":4}]}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting many<->many relation" $
		// 	get "/tasks?select=id,users(id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"users":[{"id":1},{"id":3}]},{"id":2,"users":[{"id":1}]},{"id":3,"users":[{"id":1}]},{"id":4,"users":[{"id":1}]},{"id":5,"users":[{"id":2},{"id":3}]},{"id":6,"users":[{"id":2}]},{"id":7,"users":[{"id":2}]},{"id":8,"users":[]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting many<->many relation",
			Query:       "/tasks?select=id,users(id)",
			Expected:    `[{"id":1,"users":[{"id":1},{"id":3}]},{"id":2,"users":[{"id":1}]},{"id":3,"users":[{"id":1}]},{"id":4,"users":[{"id":1}]},{"id":5,"users":[{"id":2},{"id":3}]},{"id":6,"users":[{"id":2}]},{"id":7,"users":[{"id":2}]},{"id":8,"users":[]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting many<->many relation with rename" $
		// 	get "/tasks?id=eq.1&select=id,theUsers:users(id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"theUsers":[{"id":1},{"id":3}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting many<->many relation with rename",
			Query:       "/tasks?id=eq.1&select=id,theUsers:users(id)",
			Expected:    `[{"id":1,"theUsers":[{"id":1},{"id":3}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting many<->many relation reverse" $
		// 	get "/users?select=id,tasks(id)" `shouldRespondWith`
		// 	  [json|[{"id":1,"tasks":[{"id":1},{"id":2},{"id":3},{"id":4}]},{"id":2,"tasks":[{"id":5},{"id":6},{"id":7}]},{"id":3,"tasks":[{"id":1},{"id":5}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting many<->many relation reverse",
			Query:       "/users?select=id,tasks(id)",
			Expected:    `[{"id":1,"tasks":[{"id":1},{"id":2},{"id":3},{"id":4}]},{"id":2,"tasks":[{"id":5},{"id":6},{"id":7}]},{"id":3,"tasks":[{"id":1},{"id":5}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting many<->many relation using composite key" $
		// 	get "/files?filename=eq.autoexec.bat&project_id=eq.1&select=filename,users_tasks(user_id,task_id)" `shouldRespondWith`
		// 	  [json|[{"filename":"autoexec.bat","users_tasks":[{"user_id":1,"task_id":1},{"user_id":3,"task_id":1}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// @@ modified to surround autoexec.bat with %22 - this should not be necessary here
		{
			Description: "requesting many<->many relation using composite key",
			Query:       "/files?filename=eq.%22autoexec.bat%22&project_id=eq.1&select=filename,users_tasks(user_id,task_id)",
			Expected:    `[{"filename":"autoexec.bat","users_tasks":[{"user_id":1,"task_id":1},{"user_id":3,"task_id":1}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting data using many<->many relation defined by composite keys" $
		// 	get "/users_tasks?user_id=eq.1&task_id=eq.1&select=user_id,files(filename,content)" `shouldRespondWith`
		// 	  [json|[{"user_id":1,"files":[{"filename":"autoexec.bat","content":"@ECHO OFF"},{"filename":"command.com","content":"#include <unix.h>"},{"filename":"README.md","content":"# make $$$!"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting data using many<->many relation defined by composite keys",
			Query:       "/users_tasks?user_id=eq.1&task_id=eq.1&select=user_id,files(filename,content)",
			Expected:    `[{"user_id":1,"files":[{"filename":"autoexec.bat","content":"@ECHO OFF"},{"filename":"command.com","content":"#include <unix.h>"},{"filename":"README.md","content":"# make $$$!"}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "requesting data using many<->many (composite keys) relation using hint" $
		// 	get "/users_tasks?user_id=eq.1&task_id=eq.1&select=user_id,files!touched_files(filename,content)" `shouldRespondWith`
		// 	  [json|[{"user_id":1,"files":[{"filename":"autoexec.bat","content":"@ECHO OFF"},{"filename":"command.com","content":"#include <unix.h>"},{"filename":"README.md","content":"# make $$$!"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// {
		// 	Description: "requesting data using many<->many (composite keys) relation using hint",
		// 	Query:       "/users_tasks?user_id=eq.1&task_id=eq.1&select=user_id,files!touched_files(filename,content)",
		// 	Expected:    `[{"user_id":1,"files":[{"filename":"autoexec.bat","content":"@ECHO OFF"},{"filename":"command.com","content":"#include <unix.h>"},{"filename":"README.md","content":"# make $$$!"}]}]`,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		//   it "requesting children with composite key" $
		// 	get "/users_tasks?user_id=eq.2&task_id=eq.6&select=*, comments(content)" `shouldRespondWith`
		// 	  [json|[{"user_id":2,"task_id":6,"comments":[{"content":"Needs to be delivered ASAP"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "requesting children with composite key",
			Query:       "/users_tasks?user_id=eq.2&task_id=eq.6&select=*, comments(content)",
			Headers:     nil,
			Expected:    `[{"user_id":2,"task_id":6,"comments":[{"content":"Needs to be delivered ASAP"}]}]`,
			Status:      200,
		},
		//   -- https://github.com/PostgREST/postgrest/issues/2070
		//   it "one-to-many embeds without a disambiguation error due to wrongly generated many-to-many relationships" $
		// 	get "/plate?select=*,well(*)" `shouldRespondWith`
		// 	  [json|[]|]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// 	context "one to one relationships" $ do
		//   it "works when having a pk as fk" $ do
		//     get "/students_info?select=address,students(name)" `shouldRespondWith`
		//       [json|[{"address":"Street 1","students":{"name":"John Doe"}}, {"address":"Street 2","students":{"name":"Jane Doe"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having a pk as fk",
			Query:       "/students_info?select=address,students(name)",
			Headers:     nil,
			Expected:    `[{"address":"Street 1","students":{"name":"John Doe"}}, {"address":"Street 2","students":{"name":"Jane Doe"}}]`,
			Status:      200,
		},
		//     get "/students?select=name,students_info(address)" `shouldRespondWith`
		//       [json|[{"name":"John Doe","students_info":{"address":"Street 1"}},{"name":"Jane Doe","students_info":{"address":"Street 2"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having a pk as fk",
			Query:       "/students?select=name,students_info(address)",
			Headers:     nil,
			Expected:    `[{"name":"John Doe","students_info":{"address":"Street 1"}},{"name":"Jane Doe","students_info":{"address":"Street 2"}}]`,
			Status:      200,
		},
		//   it "works when having a fk with a unique constraint" $ do
		//     get "/country?select=name,capital(name)" `shouldRespondWith`
		//       [json|[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having a fk with a unique constraint",
			Query:       "/country?select=name,capital(name)",
			Headers:     nil,
			Expected:    `[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]`,
			Status:      200,
		},
		//     get "/capital?select=name,country(name)" `shouldRespondWith`
		//       [json|[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having a fk with a unique constraint",
			Query:       "/capital?select=name,country(name)",
			Headers:     nil,
			Expected:    `[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]`,
			Status:      200,
		},
		//   it "works when using column as target" $ do
		//     get "/capital?select=name,country_id(name)" `shouldRespondWith`
		//       [json|[{"name":"Kabul","country_id":{"name":"Afghanistan"}}, {"name":"Algiers","country_id":{"name":"Algeria"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as target",
			Query:       "/capital?select=name,country_id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Kabul","country_id":{"name":"Afghanistan"}}, {"name":"Algiers","country_id":{"name":"Algeria"}}]`,
			Status:      200,
		},
		//     get "/capital?select=name,capital_country_id_fkey(name)" `shouldRespondWith`
		//       [json|[{"name":"Kabul","capital_country_id_fkey":{"name":"Afghanistan"}}, {"name":"Algiers","capital_country_id_fkey":{"name":"Algeria"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		// @@ column as fk constraint
		{
			Description: "works when using column as target",
			Query:       "/capital?select=name,capital_country_id_fkey(name)",
			Headers:     nil,
			Expected:    `[{"name":"Kabul","capital_country_id_fkey":{"name":"Afghanistan"}}, {"name":"Algiers","capital_country_id_fkey":{"name":"Algeria"}}]`,
			Status:      200,
		},
		//     get "/country?select=name,capital_country_id_fkey(name)" `shouldRespondWith`
		//       [json|[{"name":"Afghanistan","capital_country_id_fkey":{"name":"Kabul"}}, {"name":"Algeria","capital_country_id_fkey":{"name":"Algiers"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		// @@ column as fk constraint
		{
			Description: "works when using column as target",
			Query:       "/country?select=name,capital_country_id_fkey(name)",
			Headers:     nil,
			Expected:    `[{"name":"Afghanistan","capital_country_id_fkey":{"name":"Kabul"}}, {"name":"Algeria","capital_country_id_fkey":{"name":"Algiers"}}]`,
			Status:      200,
		},
		//     get "/country?select=name,id(name)" `shouldRespondWith`
		//       [json|[{"name":"Afghanistan","id":{"name":"Kabul"}}, {"name":"Algeria","id":{"name":"Algiers"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as target",
			Query:       "/country?select=name,id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Afghanistan","id":{"name":"Kabul"}}, {"name":"Algeria","id":{"name":"Algiers"}}]`,
			Status:      200,
		},
		//   it "works when using column as hint" $ do
		//     get "/country?select=name,capital!id(name)" `shouldRespondWith`
		//       [json|[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as hint",
			Query:       "/country?select=name,capital!id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]`,
			Status:      200,
		},
		//     get "/country?select=name,capital!country_id(name)" `shouldRespondWith`
		//       [json|[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as hint",
			Query:       "/country?select=name,capital!country_id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Afghanistan","capital":{"name":"Kabul"}}, {"name":"Algeria","capital":{"name":"Algiers"}}]`,
			Status:      200,
		},
		//     get "/capital?select=name,country!id(name)" `shouldRespondWith`
		//       [json|[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as hint",
			Query:       "/capital?select=name,country!id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]`,
			Status:      200,
		}, //     get "/capital?select=name,country!country_id(name)" `shouldRespondWith`
		//       [json|[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when using column as hint",
			Query:       "/capital?select=name,country!country_id(name)",
			Headers:     nil,
			Expected:    `[{"name":"Kabul","country":{"name":"Afghanistan"}}, {"name":"Algiers","country":{"name":"Algeria"}}]`,
			Status:      200,
		},
		//   describe "computed columns" $ do
		// 	it "computed column on table" $
		// 	  get "/items?id=eq.1&select=id,always_true" `shouldRespondWith`
		// 		[json|[{"id":1,"always_true":true}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "computed column on table",
			Query:       "/items?id=eq.1&select=id,always_true",
			Headers:     nil,
			Expected:    `[{"id":1,"always_true":true}]`,
			Status:      200,
		},
		// 	it "computed column on rpc" $
		// 	  get "/rpc/search?id=1&select=id,always_true" `shouldRespondWith`
		// 		[json|[{"id":1,"always_true":true}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "computed column on rpc",
			Query:       "/rpc/search?id=1&select=id,always_true",
			Headers:     nil,
			Expected:    `[{"id":1,"always_true":true}]`,
			Status:      200,
		},
		// 	it "overloaded computed columns on both tables" $ do
		// 	  get "/items?id=eq.1&select=id,computed_overload" `shouldRespondWith`
		// 		[json|[{"id":1,"computed_overload":true}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "overloaded computed columns on both tables",
			Query:       "/items?id=eq.1&select=id,computed_overload",
			Headers:     nil,
			Expected:    `[{"id":1,"computed_overload":true}]`,
			Status:      200,
		},
		// 	  get "/items2?id=eq.1&select=id,computed_overload" `shouldRespondWith`
		// 		[json|[{"id":1,"computed_overload":true}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "overloaded computed columns on both tables",
			Query:       "/items2?id=eq.1&select=id,computed_overload",
			Headers:     nil,
			Expected:    `[{"id":1,"computed_overload":true}]`,
			Status:      200,
		},
		// 	it "overloaded computed column on rpc" $
		// 	  get "/rpc/search?id=1&select=id,computed_overload" `shouldRespondWith`
		// 		[json|[{"id":1,"computed_overload":true}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "overloaded computed column on rpc",
			Query:       "/rpc/search?id=1&select=id,computed_overload",
			Headers:     nil,
			Expected:    `[{"id":1,"computed_overload":true}]`,
			Status:      200,
		},
		//   when (actualPgVersion >= pgVersion110) $ do
		// 	describe "partitioned tables embedding" $ do
		// 	  it "can request a table as parent from a partitioned table" $
		// 		get "/car_models?name=in.(DeLorean,Murcielago)&select=name,year,car_brands(name)&order=name.asc" `shouldRespondWith`
		// 		  [json|
		// 			[{"name":"DeLorean","year":1981,"car_brands":{"name":"DMC"}},
		// 			 {"name":"Murcielago","year":2001,"car_brands":{"name":"Lamborghini"}}] |]
		// 		  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request a table as parent from a partitioned table",
			Query:       "/car_models?name=in.(DeLorean,Murcielago)&select=name,year,car_brands(name)&order=name.asc",
			Headers:     nil,
			Expected: `[
				{"name":"DeLorean","year":1981,"car_brands":{"name":"DMC"}},
				{"name":"Murcielago","year":2001,"car_brands":{"name":"Lamborghini"}}
			]`,
			Status: 200,
		},
		// 	  it "can request partitioned tables as children from a table" $
		// 		get "/car_brands?select=name,car_models(name,year)&order=name.asc&car_models.order=name.asc" `shouldRespondWith`
		// 		  [json|
		// 			[{"name":"DMC","car_models":[{"name":"DeLorean","year":1981}]},
		// 			 {"name":"Ferrari","car_models":[{"name":"F310-B","year":1997}]},
		// 			 {"name":"Lamborghini","car_models":[{"name":"Murcielago","year":2001},{"name":"Veneno","year":2013}]}] |]
		// 		  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request partitioned tables as children from a table",
			Query:       "/car_brands?select=name,car_models(name,year)&order=name.asc&car_models.order=name.asc",
			Headers:     nil,
			Expected: `[
				{"name":"DMC","car_models":[{"name":"DeLorean","year":1981}]},
				{"name":"Ferrari","car_models":[{"name":"F310-B","year":1997}]},
				{"name":"Lamborghini","car_models":[{"name":"Murcielago","year":2001},{"name":"Veneno","year":2013}]}
			]`,
			Status: 200,
		},
		// 	  when (actualPgVersion >= pgVersion121) $ do
		// 		it "can request tables as children from a partitioned table" $
		// 		  get "/car_models?name=in.(DeLorean,F310-B)&select=name,year,car_racers(name)&order=name.asc" `shouldRespondWith`
		// 			[json|
		// 			  [{"name":"DeLorean","year":1981,"car_racers":[]},
		// 			   {"name":"F310-B","year":1997,"car_racers":[{"name":"Michael Schumacher"}]}] |]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request tables as children from a partitioned table",
			Query:       "/car_models?name=in.(DeLorean,F310-B)&select=name,year,car_racers(name)&order=name.asc",
			Headers:     nil,
			Expected: `[
				{"name":"DeLorean","year":1981,"car_racers":[]},
				{"name":"F310-B","year":1997,"car_racers":[{"name":"Michael Schumacher"}]}
			]`,
			Status: 200,
		},
		// 		it "can request a partitioned table as parent from a table" $
		// 		  get "/car_racers?select=name,car_models(name,year)&order=name.asc" `shouldRespondWith`
		// 			[json|
		// 			  [{"name":"Alain Prost","car_models":null},
		// 			   {"name":"Michael Schumacher","car_models":{"name":"F310-B","year":1997}}] |]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request a partitioned table as parent from a table",
			Query:       "/car_racers?select=name,car_models(name,year)&order=name.asc",
			Headers:     nil,
			Expected: `[
				{"name":"Alain Prost","car_models":null},
				{"name":"Michael Schumacher","car_models":{"name":"F310-B","year":1997}}
			]`,
			Status: 200,
		},
		// 		it "can request partitioned tables as children from a partitioned table" $
		// 		  get "/car_models?name=in.(DeLorean,Murcielago,Veneno)&select=name,year,car_model_sales(date,quantity)&order=name.asc" `shouldRespondWith`
		// 			[json|
		// 			  [{"name":"DeLorean","year":1981,"car_model_sales":[{"date":"2021-01-14","quantity":7},{"date":"2021-01-15","quantity":9}]},
		// 			   {"name":"Murcielago","year":2001,"car_model_sales":[{"date":"2021-02-11","quantity":1},{"date":"2021-02-12","quantity":3}]},
		// 			   {"name":"Veneno","year":2013,"car_model_sales":[]}] |]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request partitioned tables as children from a partitioned table",
			Query:       "/car_models?name=in.(DeLorean,Murcielago,Veneno)&select=name,year,car_model_sales(date,quantity)&order=name.asc",
			Headers:     nil,
			Expected: `[
				{"name":"DeLorean","year":1981,"car_model_sales":[{"date":"2021-01-14","quantity":7},{"date":"2021-01-15","quantity":9}]},
				{"name":"Murcielago","year":2001,"car_model_sales":[{"date":"2021-02-11","quantity":1},{"date":"2021-02-12","quantity":3}]},
				{"name":"Veneno","year":2013,"car_model_sales":[]}
			]`,
			Status: 200,
		},
		// 		it "can request a partitioned table as parent from a partitioned table" $ do
		// 		  get "/car_model_sales?date=in.(2021-01-15,2021-02-11)&select=date,quantity,car_models(name,year)&order=date.asc" `shouldRespondWith`
		// 			[json|
		// 			  [{"date":"2021-01-15","quantity":9,"car_models":{"name":"DeLorean","year":1981}},
		// 			   {"date":"2021-02-11","quantity":1,"car_models":{"name":"Murcielago","year":2001}}] |]
		// 			{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "can request a partitioned table as parent from a partitioned table",
			Query:       "/car_model_sales?date=in.(2021-01-15,2021-02-11)&select=date,quantity,car_models(name,year)&order=date.asc",
			Headers:     nil,
			Expected: `[
				{"date":"2021-01-15","quantity":9,"car_models":{"name":"DeLorean","year":1981}},
				{"date":"2021-02-11","quantity":1,"car_models":{"name":"Murcielago","year":2001}}
			]`,
			Status: 200,
		},
		// 		it "can request many to many relationships between partitioned tables ignoring the intermediate table partitions" $
		// 		  get "/car_models?select=name,year,car_dealers(name,city)&order=name.asc&limit=4" `shouldRespondWith`
		// 			[json|
		// 			  [{"name":"DeLorean","year":1981,"car_dealers":[{"name":"Springfield Cars S.A.","city":"Springfield"}]},
		// 			   {"name":"F310-B","year":1997,"car_dealers":[]},
		// 			   {"name":"Murcielago","year":2001,"car_dealers":[{"name":"The Best Deals S.A.","city":"Franklin"}]},
		// 			   {"name":"Veneno","year":2013,"car_dealers":[]}] |]
		// 			{ matchStatus  = 200
		// 			, matchHeaders = [matchContentTypeJson]
		// 			}
		{
			Description: "can request many to many relationships between partitioned tables ignoring the intermediate table partitions",
			Query:       "/car_models?select=name,year,car_dealers(name,city)&order=name.asc&limit=4",
			Headers:     nil,
			Expected: `[{"name":"DeLorean","year":1981,"car_dealers":[{"name":"Springfield Cars S.A.","city":"Springfield"}]},
			 			   {"name":"F310-B","year":1997,"car_dealers":[]},
			 			   {"name":"Murcielago","year":2001,"car_dealers":[{"name":"The Best Deals S.A.","city":"Franklin"}]},
			 			   {"name":"Veneno","year":2013,"car_dealers":[]}]`,
			Status: 200,
		},
		// 		it "cannot request partitions as children from a partitioned table" $
		// 		  get "/car_models?id=in.(1,2,4)&select=id,name,car_model_sales_202101(id)&order=id.asc" `shouldRespondWith`
		// 			[json|
		// 			  {"hint":"Verify that 'car_models' and 'car_model_sales_202101' exist in the schema 'test' and that there is a foreign key relationship between them. If a new relationship was created, try reloading the schema cache.",
		// 			   "details":null,
		// 			   "code":"PGRST200",
		// 			   "message":"Could not find a relationship between 'car_models' and 'car_model_sales_202101' in the schema cache"} |]
		// 			{ matchStatus  = 400
		// 			, matchHeaders = [matchContentTypeJson]
		// 			}
		{
			Description: "cannot request partitions as children from a partitioned table",
			Query:       "/car_models?id=in.(1,2,4)&select=id,name,car_model_sales_202101(id)&order=id.asc",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		// 		it "cannot request a partitioned table as parent from a partition" $
		// 		  get "/car_model_sales_202101?select=id,name,car_models(id,name)&order=id.asc" `shouldRespondWith`
		// 			[json|
		// 			  {"hint":"Verify that 'car_model_sales_202101' and 'car_models' exist in the schema 'test' and that there is a foreign key relationship between them. If a new relationship was created, try reloading the schema cache.",
		// 			   "details":null,
		// 			   "code":"PGRST200",
		// 			   "message":"Could not find a relationship between 'car_model_sales_202101' and 'car_models' in the schema cache"} |]
		// 			{ matchStatus  = 400
		// 			, matchHeaders = [matchContentTypeJson]
		// 			}
		{
			Description: "cannot request a partitioned table as parent from a partition",
			Query:       "/car_model_sales_202101?select=id,name,car_models(id,name)&order=id.asc",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		// 		it "cannot request a partition as parent from a partitioned table" $
		// 		  get "/car_model_sales?id=in.(1,3,4)&select=id,name,car_models_default(id,name)&order=id.asc" `shouldRespondWith`
		// 			[json|
		// 			  {"hint":"Verify that 'car_model_sales' and 'car_models_default' exist in the schema 'test' and that there is a foreign key relationship between them. If a new relationship was created, try reloading the schema cache.",
		// 			   "details":null,
		// 			   "code":"PGRST200",
		// 			   "message":"Could not find a relationship between 'car_model_sales' and 'car_models_default' in the schema cache"} |]
		// 			{ matchStatus  = 400
		// 			, matchHeaders = [matchContentTypeJson]
		// 			}
		{
			Description: "cannot request a partition as parent from a partitioned table",
			Query:       "/car_model_sales?id=in.(1,3,4)&select=id,name,car_models_default(id,name)&order=id.asc",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		// 		it "cannot request partitioned tables as children from a partition" $
		// 		  get "/car_models_default?select=id,name,car_model_sales(id,name)&order=id.asc" `shouldRespondWith`
		// 			[json|
		// 			  {"hint":"Verify that 'car_models_default' and 'car_model_sales' exist in the schema 'test' and that there is a foreign key relationship between them. If a new relationship was created, try reloading the schema cache.",
		// 			   "details":null,
		// 			   "code":"PGRST200",
		// 			   "message":"Could not find a relationship between 'car_models_default' and 'car_model_sales' in the schema cache"} |]
		// 			{ matchStatus  = 400
		// 			, matchHeaders = [matchContentTypeJson]
		// 			}
		{
			Description: "cannot request partitioned tables as children from a partition",
			Query:       "/car_models_default?select=id,name,car_model_sales(id,name)&order=id.asc",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		//   describe "view embedding" $ do
		// 	it "can detect fk relations through views to tables in the public schema" $
		// 	  get "/consumers_view?select=*,orders_view(*)" `shouldRespondWith` 200

		// @@ not implemented
		// {
		// 	Description: "can detect fk relations through views to tables in the public schema",
		// 	Query:       "/consumers_view?select=*,orders_view(*)",
		// 	Expected:    ``,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		// 	it "can detect fk relations through materialized views to tables in the public schema" $
		// 	  get "/materialized_projects?select=*,users(*)" `shouldRespondWith` 200

		// 	it "can request two parents" $
		// 	  get "/articleStars?select=createdAt,article:articles(id),user:users(name)&limit=1"
		// 		`shouldRespondWith`
		// 		  [json|[{"createdAt":"2015-12-08T04:22:57.472738","article":{"id": 1},"user":{"name": "Angela Martin"}}]|]

		// @@ not implemented: articleStars is a view
		// {
		// 	Description: "can request two parents",
		// 	Query:       "/articleStars?select=createdAt,article:articles(id),user:users(name)&limit=1",
		// 	Expected:    `[{"createdAt":"2015-12-08T04:22:57.472738","article":{"id": 1},"user":{"name": "Angela Martin"}}]`,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		// 	it "can detect relations in views from exposed schema that are based on tables in private schema and have columns renames" $
		// 	  get "/articles?id=eq.1&select=id,articleStars(users(*))" `shouldRespondWith`
		// 		[json|[{"id":1,"articleStars":[{"users":{"id":1,"name":"Angela Martin"}},{"users":{"id":2,"name":"Michael Scott"}},{"users":{"id":3,"name":"Dwight Schrute"}}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works when requesting parents and children on views" $
		// 	  get "/projects_view?id=eq.1&select=id, name, clients(*), tasks(id, name)" `shouldRespondWith`
		// 		[json|[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works when requesting parents and children on views with renamed keys" $
		// 	  get "/projects_view_alt?t_id=eq.1&select=t_id, name, clients(*), tasks(id, name)" `shouldRespondWith`
		// 		[json|[{"t_id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "detects parent relations when having many views of a private table" $ do
		// 	  get "/books?select=title,author:authors(name)&id=eq.5" `shouldRespondWith`
		// 		[json|[ { "title": "Farenheit 451", "author": { "name": "Ray Bradbury" } } ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  get "/forties_books?select=title,author:authors(name)&limit=1" `shouldRespondWith`
		// 		[json|[ { "title": "1984", "author": { "name": "George Orwell" } } ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  get "/fifties_books?select=title,author:authors(name)&limit=1" `shouldRespondWith`
		// 		[json|[ { "title": "The Catcher in the Rye", "author": { "name": "J.D. Salinger" } } ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  get "/sixties_books?select=title,author:authors(name)&limit=1" `shouldRespondWith`
		// 		[json|[ { "title": "To Kill a Mockingbird", "author": { "name": "Harper Lee" } } ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "can detect fk relations through multiple views recursively when all views are in api schema" $ do
		// 	  get "/consumers_view_view?select=*,orders_view(*)" `shouldRespondWith` 200

		// 	it "works with views that have subselects" $
		// 	  get "/authors_books_number?select=*,books(title)&id=eq.1" `shouldRespondWith`
		// 		[json|[ {"id":1, "name":"George Orwell","num_in_forties":1,"num_in_fifties":0,"num_in_sixties":0,"num_in_all_decades":1,
		// 				 "books":[{"title":"1984"}]} ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works with views that have case subselects" $
		// 	  get "/authors_have_book_in_decade?select=*,books(title)&id=eq.3" `shouldRespondWith`
		// 		[json|[ {"id":3,"name":"Antoine de Saint-Exupéry","has_book_in_forties":true,"has_book_in_fifties":false,"has_book_in_sixties":false,
		// 				 "books":[{"title":"The Little Prince"}]} ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works with views that have subselect in the FROM clause" $
		// 	  get "/forties_and_fifties_books?select=title,first_publisher,author:authors(name)&id=eq.1" `shouldRespondWith`
		// 		[json|[{"title":"1984","first_publisher":"Secker & Warburg","author":{"name":"George Orwell"}}]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works with views that have subselects in a function call" $
		// 	  get "/authors_have_book_in_decade2?select=*,books(title)&id=eq.3"
		// 		`shouldRespondWith`
		// 		  [json|[ {"id":3,"name":"Antoine de Saint-Exupéry","has_book_in_forties":true,"has_book_in_fifties":false,
		// 				   "has_book_in_sixties":false,"books":[{"title":"The Little Prince"}]} ]|]

		// 	it "works with views that have CTE" $
		// 	  get "/odd_years_publications?select=title,publication_year,first_publisher,author:authors(name)&id=in.(1,2,3)" `shouldRespondWith`
		// 		[json|[
		// 		  {"title":"1984","publication_year":1949,"first_publisher":"Secker & Warburg","author":{"name":"George Orwell"}},
		// 		  {"title":"The Diary of a Young Girl","publication_year":1947,"first_publisher":"Contact Publishing","author":{"name":"Anne Frank"}},
		// 		  {"title":"The Little Prince","publication_year":1947,"first_publisher":"Reynal & Hitchcock","author":{"name":"Antoine de Saint-Exupéry"}} ]|]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works when having a capitalized table name and camelCase fk column" $
		// 	  get "/foos?select=*,bars(*)" `shouldRespondWith` 200

		// @@ not implemented (view)
		// {
		// 	Description: "works when having a capitalized table name and camelCase fk column",
		// 	Query:       "/foos?select=*,bars(*)",
		// 	Expected:    ``,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		// 	it "works when embedding a view with a table that has a long compound pk" $ do
		// 	  get "/player_view?select=id,contract(purchase_price)&id=in.(1,3,5,7)" `shouldRespondWith`
		// 		[json|
		// 		  [{"id":1,"contract":[{"purchase_price":10}]},
		// 		   {"id":3,"contract":[{"purchase_price":30}]},
		// 		   {"id":5,"contract":[{"purchase_price":50}]},
		// 		   {"id":7,"contract":[]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	  get "/contract?select=tournament,player_view(first_name)&limit=3" `shouldRespondWith`
		// 		[json|
		// 		  [{"tournament":"tournament_1","player_view":{"first_name":"first_name_1"}},
		// 		   {"tournament":"tournament_2","player_view":{"first_name":"first_name_2"}},
		// 		   {"tournament":"tournament_3","player_view":{"first_name":"first_name_3"}}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "works when embedding a view with a view that referes to a table that has a long compound pk" $ do
		// 	  get "/player_view?select=id,contract_view(purchase_price)&id=in.(1,3,5,7)" `shouldRespondWith`
		// 		[json|
		// 		  [{"id":1,"contract_view":[{"purchase_price":10}]},
		// 		   {"id":3,"contract_view":[{"purchase_price":30}]},
		// 		   {"id":5,"contract_view":[{"purchase_price":50}]},
		// 		   {"id":7,"contract_view":[]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		// 	  get "/contract_view?select=tournament,player_view(first_name)&limit=3" `shouldRespondWith`
		// 		[json|
		// 		  [{"tournament":"tournament_1","player_view":{"first_name":"first_name_1"}},
		// 		   {"tournament":"tournament_2","player_view":{"first_name":"first_name_2"}},
		// 		   {"tournament":"tournament_3","player_view":{"first_name":"first_name_3"}}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "can embed a view that has group by" $
		// 	  get "/projects_count_grouped_by?select=number_of_projects,client:clients(name)&order=number_of_projects" `shouldRespondWith`
		// 		[json|
		// 		  [{"number_of_projects":1,"client":null},
		// 		   {"number_of_projects":2,"client":{"name":"Microsoft"}},
		// 		   {"number_of_projects":2,"client":{"name":"Apple"}}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }

		// 	it "can embed a view that has a subselect containing a select in a where" $
		// 	  get "/authors_w_entities?select=name,entities,books(title)&id=eq.1" `shouldRespondWith`
		// 		[json| [{"name":"George Orwell","entities":[3, 4],"books":[{"title":"1984"}]}] |]
		// 		{ matchHeaders = [matchContentTypeJson] }

		//   describe "aliased embeds" $ do
		// 	it "works with child relation" $
		// 	  get "/space?select=id,zones:zone(id,name),stores:zone(id,name)&zones.zone_type_id=eq.2&stores.zone_type_id=eq.3" `shouldRespondWith`
		// 		[json|[
		// 		  { "id":1,
		// 			"zones": [ {"id":1,"name":"zone 1"}, {"id":2,"name":"zone 2"}],
		// 			"stores": [ {"id":3,"name":"store 3"}, {"id":4,"name":"store 4"}]}
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "aliased embeds works with child relation",
			Query:       "/space?select=id,zones:zone(id,name),stores:zone(id,name)&zones.zone_type_id=eq.2&stores.zone_type_id=eq.3",
			Headers:     nil,
			Expected: `[
				 		  { "id":1,
				 			"zones": [ {"id":1,"name":"zone 1"}, {"id":2,"name":"zone 2"}],
				 			"stores": [ {"id":3,"name":"store 3"}, {"id":4,"name":"store 4"}]}
				 		]`,
			Status: 200,
		},
		// 	it "works with many to many relation" $
		// 	  get "/users?select=id,designTasks:tasks(id,name),codeTasks:tasks(id,name)&designTasks.name=like.*Design*&codeTasks.name=like.*Code*" `shouldRespondWith`
		// 		[json|[
		// 		   { "id":1,
		// 			 "designTasks":[ { "id":1, "name":"Design w7" }, { "id":3, "name":"Design w10" } ],
		// 			 "codeTasks":[ { "id":2, "name":"Code w7" }, { "id":4, "name":"Code w10" } ] },
		// 		   { "id":2,
		// 			 "designTasks":[ { "id":5, "name":"Design IOS" }, { "id":7, "name":"Design OSX" } ],
		// 			 "codeTasks":[ { "id":6, "name":"Code IOS" } ] },
		// 		   { "id":3,
		// 			 "designTasks":[ { "id":1, "name":"Design w7" }, { "id":5, "name":"Design IOS" } ],
		// 			 "codeTasks":[ ] }
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with many to many relation",
			Query:       "/users?select=id,designTasks:tasks(id,name),codeTasks:tasks(id,name)&designTasks.name=like.*Design*&codeTasks.name=like.*Code*",
			Headers:     nil,
			Expected: `[
						   { "id":1,
							 "designTasks":[ { "id":1, "name":"Design w7" }, { "id":3, "name":"Design w10" } ],
							 "codeTasks":[ { "id":2, "name":"Code w7" }, { "id":4, "name":"Code w10" } ] },
						   { "id":2,
							 "designTasks":[ { "id":5, "name":"Design IOS" }, { "id":7, "name":"Design OSX" } ],
							 "codeTasks":[ { "id":6, "name":"Code IOS" } ] },
						   { "id":3,
							 "designTasks":[ { "id":1, "name":"Design w7" }, { "id":5, "name":"Design IOS" } ],
							 "codeTasks":[ ] }
						]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,tasks(name),other:tasks(name)&id=in.(3,4,5)&tasks.name=like.*Design*",
			Expected: `[{
				"id": 3,
				"tasks": [{"name": "Design IOS"}],
				"other": [{"name": "Design IOS"},{"name": "Code IOS"}]
			  },
			  {
				"id": 4,
				"tasks": [{"name": "Design OSX"}],
				"other": [{"name": "Design OSX"},{"name": "Code OSX"}]
			  },
			  {
				"id": 5,
				"tasks": [],
				"other": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,other:tasks(name),tasks:tasks(name)&id=in.(3,4,5)&tasks.name=like.*Design*",
			Expected: `[{
				"id": 3,
				"other": [{"name": "Design IOS"}],
				"tasks": [{"name": "Design IOS"},{"name": "Code IOS"}]
			  },
			  {
				"id": 4,
				"other": [{"name": "Design OSX"}],
				"tasks": [{"name": "Design OSX"},{"name": "Code OSX"}]
			  },
			  {
				"id": 5,
				"other": [],
				"tasks": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,designTasks:tasks(name),tasks(name)&id=in.(3,4,5)&designTasks.name=like.*Design*",
			Expected: `[{
				"id": 3,
				"designTasks": [{"name": "Design IOS"}],
				"tasks": [{"name": "Design IOS"},{"name": "Code IOS"}
				]
			  },{
				"id": 4,
				"designTasks": [{"name": "Design OSX"}],
				"tasks": [{"name": "Design OSX"},{"name": "Code OSX"}]
			  },{
				"id": 5,
				"designTasks": [],
				"tasks": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,tasks(name),codeTasks:tasks(name)&id=in.(3,4,5)&codeTasks.name=like.*Code*",
			Expected: `[{
				"id": 3,
				"codeTasks": [{"name": "Code IOS"}],
				"tasks": [{"name": "Design IOS"},{"name": "Code IOS"}
				]
			  },{
				"id": 4,
				"codeTasks": [{"name": "Code OSX"}],
				"tasks": [{"name": "Design OSX"},{"name": "Code OSX"}]
			  },{
				"id": 5,
				"codeTasks": [],
				"tasks": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,designTasks:tasks(name),codeTasks:tasks(name)&id=in.(3,4,5)&codeTasks.name=like.*Code*",
			Expected: `[{
				"id": 3,
				"codeTasks": [{"name": "Code IOS"}],
				"designTasks": [{"name": "Design IOS"},{"name": "Code IOS"}
				]
			  },{
				"id": 4,
				"codeTasks": [{"name": "Code OSX"}],
				"designTasks": [{"name": "Design OSX"},{"name": "Code OSX"}]
			  },{
				"id": 5,
				"codeTasks": [],
				"designTasks": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds",
			Query:       "/projects?select=id,designTasks:tasks(name),codeTasks:tasks(name)&id=in.(3,4,5)&designTasks.name=like.*Design*&codeTasks.name=like.*Code*",
			Expected: `[{
				"id": 3,
				"codeTasks": [{"name": "Code IOS"}],
				"designTasks": [{"name": "Design IOS"}]
			  },{
				"id": 4,
				"codeTasks": [{"name": "Code OSX"}],
				"designTasks": [{"name": "Design OSX"}]
			  },{
				"id": 5,
				"codeTasks": [],
				"designTasks": []
			  }]`,
			Status: 200,
		},
		// @@ added
		{
			Description: "works with aliased embeds with two levels",
			Query:       "/projects?select=id,designTasks:tasks(name,users(id,name)),codeTasks:tasks(name,users(id,name))&designTasks.name=like.*Design*&codeTasks.name=like.*Code*",
			Expected: `[
				{
					"id": 1,
					"designTasks": [{"name": "Design w7","users": [{"id": 1,"name": "Angela Martin"},{"id": 3,"name": "Dwight Schrute"}]}],
					"codeTasks": [{"name": "Code w7","users": [	{"id": 1,"name": "Angela Martin"}]}]
				},
				{
					"id": 2,
					"designTasks": [{"name": "Design w10","users": [{"id": 1,"name": "Angela Martin"}]}],
					"codeTasks": [{"name": "Code w10","users": [{"id": 1,"name": "Angela Martin"}]}]
				},
				{
					"id": 3,
					"designTasks": [{"name": "Design IOS","users": [{"id": 2,"name": "Michael Scott"},	{"id": 3,"name": "Dwight Schrute"}]}],
					"codeTasks": [{"name": "Code IOS","users": [{"id": 2,"name": "Michael Scott"}]}]
				},
				{
					"id": 4,
					"designTasks": [{"name": "Design OSX","users": [{"id": 2,"name": "Michael Scott"}]}],
					"codeTasks": [{"name": "Code OSX","users": []}]
				},
				{
					"id": 5,
					"designTasks": [],
					"codeTasks": []
				}
			]`,
			Status: 200,
		},

		// @@ added
		{
			Description: "works with aliased embeds with two levels",
			Query:       "/projects?select=id,designTasks:tasks(name,users(id,name)),codeTasks:tasks(name,users(id,name))&designTasks.name=like.*Design*&designTasks.users.id=eq.3&codeTasks.name=like.*Code*&codeTasks.users.id=eq.2",
			Expected: `[
			{
				"id": 1,
				"designTasks": [{"name": "Design w7","users": [{"id": 3,"name": "Dwight Schrute"}]}],
				"codeTasks": [{"name": "Code w7","users": []}]
			},
			{
				"id": 2,
				"designTasks": [{"name": "Design w10","users": []}],
				"codeTasks": [{"name": "Code w10","users": []}]
			},
			{
				"id": 3,
				"designTasks": [{"name": "Design IOS","users": [{"id": 3,"name": "Dwight Schrute"}]}],
				"codeTasks": [{"name": "Code IOS","users": [{"id": 2,"name": "Michael Scott"}]}]
			},
			{
				"id": 4,
				"designTasks": [{"name": "Design OSX","users": []}],
				"codeTasks": [{"name": "Code OSX","users": []}]
			},
			{
				"id": 5,
				"designTasks": [],
				"codeTasks": []
			}
		]`,
			Status: 200,
		},

		// 	it "works with an aliased child plus non aliased child" $
		// 	  get "/projects?select=id,name,designTasks:tasks(name,users(id,name))&designTasks.name=like.*Design*&designTasks.users.id=in.(1,2)" `shouldRespondWith`
		// 		[json|[
		// 		  {
		// 			"id":1, "name":"Windows 7",
		// 			"designTasks":[ { "name":"Design w7", "users":[ { "id":1, "name":"Angela Martin" } ] } ] },
		// 		  {
		// 			"id":2, "name":"Windows 10",
		// 			"designTasks":[ { "name":"Design w10", "users":[ { "id":1, "name":"Angela Martin" } ] } ] },
		// 		  {
		// 			"id":3, "name":"IOS",
		// 			"designTasks":[ { "name":"Design IOS", "users":[ { "id":2, "name":"Michael Scott" } ] } ] },
		// 		  {
		// 			"id":4, "name":"OSX",
		// 			"designTasks":[ { "name":"Design OSX", "users":[ { "id":2, "name":"Michael Scott" } ] } ] },
		// 		  {
		// 			"id":5, "name":"Orphan",
		// 			"designTasks":[ ] }
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with an aliased child plus non aliased child",
			Query:       "/projects?select=id,name,designTasks:tasks(name,users(id,name))&designTasks.name=like.*Design*&designTasks.users.id=in.(1,2)",
			Headers:     nil,
			Expected: `[	  {
							"id":1, "name":"Windows 7",
							"designTasks":[ { "name":"Design w7", "users":[ { "id":1, "name":"Angela Martin" } ] } ] },
						  {
							"id":2, "name":"Windows 10",
							"designTasks":[ { "name":"Design w10", "users":[ { "id":1, "name":"Angela Martin" } ] } ] },
						  {
							"id":3, "name":"IOS",
							"designTasks":[ { "name":"Design IOS", "users":[ { "id":2, "name":"Michael Scott" } ] } ] },
						  {
							"id":4, "name":"OSX",
							"designTasks":[ { "name":"Design OSX", "users":[ { "id":2, "name":"Michael Scott" } ] } ] },
						  {
							"id":5, "name":"Orphan",
							"designTasks":[ ] }
						]`,
			Status: 200,
		},
		// 	it "works with two aliased children embeds plus and/or" $
		// 	  get "/entities?select=id,children:child_entities(id,gChildren:grandchild_entities(id))&children.and=(id.in.(1,2,3))&children.gChildren.or=(id.eq.1,id.eq.2)" `shouldRespondWith`
		// 		[json|[
		// 		  { "id":1,
		// 			"children":[
		// 			  {"id":1,"gChildren":[{"id":1}, {"id":2}]},
		// 			  {"id":2,"gChildren":[]}]},
		// 		  { "id":2,
		// 			"children":[
		// 			  {"id":3,"gChildren":[]}]},
		// 		  { "id":3,"children":[]},
		// 		  { "id":4,"children":[]}
		// 		]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with two aliased children embeds plus and/or",
			Query:       "/entities?select=id,children:child_entities(id,gChildren:grandchild_entities(id))&children.and=(id.in.(1,2,3))&children.gChildren.or=(id.eq.1,id.eq.2)",
			Headers:     nil,
			Expected: `[
						  { "id":1,
							"children":[
							  {"id":1,"gChildren":[{"id":1}, {"id":2}]},
							  {"id":2,"gChildren":[]}]},
						  { "id":2,
							"children":[
							  {"id":3,"gChildren":[]}]},
						  { "id":3,"children":[]},
						  { "id":4,"children":[]}
						]`,
			Status: 200,
		},
		// describe "ordering response" $ do
		//   it "by a column asc" $
		// 	get "/items?id=lte.2&order=id.asc"
		// 	  `shouldRespondWith` [json| [{"id":1},{"id":2}] |]
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-1/*"]
		// 	  }
		{
			Description:     "by a column asc",
			Query:           "/items?id=lte.2&order=id.asc",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "by a column desc" $
		// 	get "/items?id=lte.2&order=id.desc"
		// 	  `shouldRespondWith` [json| [{"id":2},{"id":1}] |]
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-1/*"]
		// 	  }
		{
			Description:     "by a column desc",
			Query:           "/items?id=lte.2&order=id.desc",
			Headers:         nil,
			Expected:        `[{"id":2},{"id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "by a column with nulls first" $
		// 	get "/no_pk?order=a.nullsfirst"
		// 	  `shouldRespondWith` [json| [{"a":null,"b":null},
		// 							{"a":"1","b":"0"},
		// 							{"a":"2","b":"0"}
		// 							] |]
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-2/*"]
		// 	  }
		{
			Description:     "by a column with nulls first",
			Query:           "/no_pk?order=a.nullsfirst",
			Headers:         nil,
			Expected:        `[{"a":null,"b":null},{"a":"1","b":"0"},{"a":"2","b":"0"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		//   it "by a column asc with nulls last" $
		// 	get "/no_pk?order=a.asc.nullslast"
		// 	  `shouldRespondWith` [json| [{"a":"1","b":"0"},
		// 							{"a":"2","b":"0"},
		// 							{"a":null,"b":null}] |]
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-2/*"]
		// 	  }
		{
			Description:     "by a column asc with nulls last",
			Query:           "/no_pk?order=a.asc.nullslast",
			Headers:         nil,
			Expected:        `[{"a":"1","b":"0"},{"a":"2","b":"0"},{"a":null,"b":null}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		//   it "by a column desc with nulls first" $
		// 	get "/no_pk?order=a.desc.nullsfirst"
		// 	  `shouldRespondWith` [json| [{"a":null,"b":null},
		// 							{"a":"2","b":"0"},
		// 							{"a":"1","b":"0"}] |]
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-2/*"]
		// 	  }
		{
			Description:     "by a column desc with nulls first",
			Query:           "/no_pk?order=a.desc.nullsfirst",
			Headers:         nil,
			Expected:        `[{"a":null,"b":null},{"a":"2","b":"0"},{"a":"1","b":"0"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		//   it "by a column desc with nulls last" $
		// 	get "/no_pk?order=a.desc.nullslast"
		// 	  `shouldRespondWith` [json| [{"a":"2","b":"0"},
		// 							{"a":"1","b":"0"},
		// 							{"a":null,"b":null}] |]
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-2/*"]
		// 	  }
		{
			Description:     "by a column desc with nulls last",
			Query:           "/no_pk?order=a.desc.nullslast",
			Headers:         nil,
			Expected:        `[{"a":"2","b":"0"},{"a":"1","b":"0"},{"a":null,"b":null}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		//   it "by two columns with nulls and direction specified" $
		// 	get "/projects?select=client_id,id,name&order=client_id.desc.nullslast,id.desc"
		// 	  `shouldRespondWith` [json|
		// 		[{"client_id":2,"id":4,"name":"OSX"},
		// 		 {"client_id":2,"id":3,"name":"IOS"},
		// 		 {"client_id":1,"id":2,"name":"Windows 10"},
		// 		 {"client_id":1,"id":1,"name":"Windows 7"},
		// 		 {"client_id":null,"id":5,"name":"Orphan"}]
		// 	  |]
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-4/*"]
		// 	  }
		{
			Description: "by two columns with nulls and direction specified",
			Query:       "/projects?select=client_id,id,name&order=client_id.desc.nullslast,id.desc",
			Headers:     nil,
			Expected: `[
				{"client_id":2,"id":4,"name":"OSX"},
				{"client_id":2,"id":3,"name":"IOS"},
				{"client_id":1,"id":2,"name":"Windows 10"},
				{"client_id":1,"id":1,"name":"Windows 7"},
				{"client_id":null,"id":5,"name":"Orphan"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-4/*"},
			Status:          200,
		},
		//   it "by a column with no direction or nulls specified" $
		// 	get "/items?id=lte.2&order=id"
		// 	  `shouldRespondWith` [json| [{"id":1},{"id":2}] |]
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Range" <:> "0-1/*"]
		// 	  }
		{
			Description:     "by a column with no direction or nulls specified",
			Query:           "/items?id=lte.2&order=id",
			Headers:         nil,
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "without other constraints" $
		// 	get "/items?order=id.asc" `shouldRespondWith` 200
		{
			Description: "without other constraints",
			Query:       "/items?order=id.asc",
			Headers:     nil,
			Expected:    ``,
			Status:      200,
		},
		//   it "ordering embeded entities" $
		// 	get "/projects?id=eq.1&select=id, name, tasks(id, name)&tasks.order=name.asc" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","tasks":[{"id":2,"name":"Code w7"},{"id":1,"name":"Design w7"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "ordering embeded entities",
			Query:       "/projects?id=eq.1&select=id, name, tasks(id, name)&tasks.order=name.asc",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","tasks":[{"id":2,"name":"Code w7"},{"id":1,"name":"Design w7"}]}]`,
			Status:      200,
		},
		//   it "ordering embeded entities with alias" $
		// 	get "/projects?id=eq.1&select=id, name, the_tasks:tasks(id, name)&tasks.order=name.asc" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","the_tasks":[{"id":2,"name":"Code w7"},{"id":1,"name":"Design w7"}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "ordering embeded entities with alias",
			Query:       "/projects?id=eq.1&select=id, name, the_tasks:tasks(id, name)&tasks.order=name.asc",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","the_tasks":[{"id":2,"name":"Code w7"},{"id":1,"name":"Design w7"}]}]`,
			Status:      200,
		},
		//   it "ordering embeded entities, two levels" $
		// 	get "/projects?id=eq.1&select=id, name, tasks(id, name, users(id, name))&tasks.order=name.asc&tasks.users.order=name.desc" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","tasks":[{"id":2,"name":"Code w7","users":[{"id":1,"name":"Angela Martin"}]},{"id":1,"name":"Design w7","users":[{"id":3,"name":"Dwight Schrute"},{"id":1,"name":"Angela Martin"}]}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }

		//   it "ordering embeded parents does not break things" $
		// 	get "/projects?id=eq.1&select=id, name, clients(id, name)&clients.order=name.asc" `shouldRespondWith`
		// 	  [json|[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"}}]|]
		{
			Description: "ordering embeded parents does not break things",
			Query:       "/projects?id=eq.1&select=id, name, clients(id, name)&clients.order=name.asc",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"}}]`,
			Status:      200,
		},
		//   context "order syntax errors" $ do
		// 	it "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled" $ do
		// 	  get "/items?order=id.ac" `shouldRespondWith`
		// 		[json|{"details":"unexpected \"c\" expecting \"asc\", \"desc\", \"nullsfirst\" or \"nullslast\"","message":"\"failed to parse order (id.ac)\" (line 1, column 4)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled",
			Query:       "/items?order=id.ac",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.descc" `shouldRespondWith`
		// 		[json|{"details":"unexpected 'c' expecting delimiter (.), \",\" or end of input","message":"\"failed to parse order (id.descc)\" (line 1, column 8)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled",
			Query:       "/items?order=id.descc",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.nulsfist" `shouldRespondWith`
		// 		[json|{"details":"unexpected \"n\" expecting \"asc\", \"desc\", \"nullsfirst\" or \"nullslast\"","message":"\"failed to parse order (id.nulsfist)\" (line 1, column 4)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled",
			Query:       "/items?order=id.nulsfist",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.nullslasttt" `shouldRespondWith`
		// 		[json|{"details":"unexpected 't' expecting \",\" or end of input","message":"\"failed to parse order (id.nullslasttt)\" (line 1, column 13)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled",
			Query:       "/items?order=id.nullslasttt",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.smth34" `shouldRespondWith`
		// 		[json|{"details":"unexpected \"s\" expecting \"asc\", \"desc\", \"nullsfirst\" or \"nullslast\"","message":"\"failed to parse order (id.smth34)\" (line 1, column 4)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when asc/desc/nulls{first,last} are misspelled",
			Query:       "/items?order=id.smth34",
			Headers:     nil,
			Status:      400,
		},
		// 	it "gives meaningful error messages when nulls{first,last} are misspelled after asc/desc" $ do
		// 	  get "/items?order=id.asc.nlsfst" `shouldRespondWith`
		// 		[json|{"details":"unexpected \"l\" expecting \"nullsfirst\" or \"nullslast\"","message":"\"failed to parse order (id.asc.nlsfst)\" (line 1, column 8)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when nulls{first,last} are misspelled after asc/desc",
			Query:       "/items?order=id.asc.nlsfst",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.asc.nullslasttt" `shouldRespondWith`
		// 		[json|{"details":"unexpected 't' expecting \",\" or end of input","message":"\"failed to parse order (id.asc.nullslasttt)\" (line 1, column 17)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when nulls{first,last} are misspelled after asc/desc",
			Query:       "/items?order=id.asc.nullslasttt",
			Headers:     nil,
			Status:      400,
		},
		// 	  get "/items?order=id.asc.smth34" `shouldRespondWith`
		// 		[json|{"details":"unexpected \"s\" expecting \"nullsfirst\" or \"nullslast\"","message":"\"failed to parse order (id.asc.smth34)\" (line 1, column 8)","code":"PGRST100","hint":null}|]
		// 		{ matchStatus  = 400
		// 		, matchHeaders = [matchContentTypeJson]
		// 		}
		{
			Description: "gives meaningful error messages when nulls{first,last} are misspelled after asc/desc",
			Query:       "/items?order=id.asc.smth34",
			Headers:     nil,
			Status:      400,
		},
		// describe "Accept headers" $ do
		//   it "should respond an unknown accept type with 406" $
		// 	request methodGet "/simple_pk"
		// 			(acceptHdrs "text/unknowntype") ""
		// 	  `shouldRespondWith`
		// 	  [json|{"message":"None of these media types are available: text/unknowntype","code":"PGRST107","details":null,"hint":null}|]
		// 	  { matchStatus  = 406
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }
		{
			Description: "should respond an unknown accept type with 406",
			Query:       "/simple_pk",
			Headers:     test.Headers{"Accept": []string{"text/unknowntype"}},
			Status:      406,
		},
		//   it "should respond correctly to */* in accept header" $
		// 	request methodGet "/simple_pk"
		// 			(acceptHdrs "*/*") ""
		// 	  `shouldRespondWith` 200
		{
			Description: "should respond correctly to */* in accept header",
			Query:       "/simple_pk",
			Headers:     test.Headers{"Accept": []string{"*/*"}},
			Status:      200,
		},
		//   it "*/* should rescue an unknown type" $
		// 	request methodGet "/simple_pk"
		// 			(acceptHdrs "text/unknowntype, */*") ""
		// 	  `shouldRespondWith` 200
		{
			Description: "*/* should rescue an unknown type",
			Query:       "/simple_pk",
			Headers:     test.Headers{"Accept": []string{"text/unknowntype", "*/*"}},
			Status:      200,
		},
		//   it "specific available preference should override */*" $ do
		// 	r <- request methodGet "/simple_pk"
		// 			(acceptHdrs "text/csv, */*") ""
		// 	liftIO $ do
		// 	  let respHeaders = simpleHeaders r
		// 	  respHeaders `shouldSatisfy` matchHeader
		// 		"Content-Type" "text/csv; charset=utf-8"
		{
			Description:     "specific available preference should override */*",
			Query:           "/simple_pk",
			Headers:         test.Headers{"Accept": []string{"text/csv", "*/*"}},
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          200,
		},
		//   it "honors client preference even when opposite of server preference" $ do
		// 	r <- request methodGet "/simple_pk"
		// 			(acceptHdrs "text/csv, application/json") ""
		// 	liftIO $ do
		// 	  let respHeaders = simpleHeaders r
		// 	  respHeaders `shouldSatisfy` matchHeader
		// 		"Content-Type" "text/csv; charset=utf-8"
		{
			Description:     "honors client preference even when opposite of server preference",
			Query:           "/simple_pk",
			Headers:         test.Headers{"Accept": []string{"text/csv, application/json"}},
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          200,
		},
		//   it "should respond correctly to multiple types in accept header" $
		// 	request methodGet "/simple_pk"
		// 			(acceptHdrs "text/unknowntype, text/csv") ""
		// 	  `shouldRespondWith` 200
		{
			Description: "should respond correctly to multiple types in accept header",
			Query:       "/simple_pk",
			Headers:     test.Headers{"Accept": []string{"text/unknowntype", "text/csv"}},
			Status:      200,
		},
		//   it "should respond with CSV to 'text/csv' request" $
		// 	request methodGet "/simple_pk"
		// 			(acceptHdrs "text/csv; version=1") ""
		// 	  `shouldRespondWith` "k,extra\nxyyx,u\nxYYx,v"
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		// 	  }
		{
			Description:     "should respond with CSV to 'text/csv' request",
			Query:           "/simple_pk",
			Headers:         test.Headers{"Accept": []string{"text/csv; version=1"}},
			Expected:        "k,extra\nxyyx,u\nxYYx,v",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          200,
		},
		// describe "Canonical location" $ do
		//   it "Sets Content-Location with alphabetized params" $
		// 	get "/no_pk?b=eq.1&a=eq.1"
		// 	  `shouldRespondWith` "[]"
		// 	  { matchStatus  = 200
		// 	  , matchHeaders = ["Content-Location" <:> "/no_pk?a=eq.1&b=eq.1"]
		// 	  }

		//   it "Omits question mark when there are no params" $ do
		// 	r <- get "/simple_pk"
		// 	liftIO $ do
		// 	  let respHeaders = simpleHeaders r
		// 	  respHeaders `shouldSatisfy` matchHeader
		// 		"Content-Location" "/simple_pk"

		// describe "weird requests" $ do
		//   it "can query as normal" $ do
		// 	get "/Escap3e;" `shouldRespondWith`
		// 	  [json| [{"so6meIdColumn":1},{"so6meIdColumn":2},{"so6meIdColumn":3},{"so6meIdColumn":4},{"so6meIdColumn":5}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can query as normal",
			Query:       "/Escap3e;",
			Headers:     nil,
			Expected:    `[{"so6meIdColumn":1},{"so6meIdColumn":2},{"so6meIdColumn":3},{"so6meIdColumn":4},{"so6meIdColumn":5}]`,
			Status:      200,
		},
		// 	get "/ghostBusters" `shouldRespondWith`
		// 	  [json| [{"escapeId":1},{"escapeId":3},{"escapeId":5}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can query as normal",
			Query:       "/ghostBusters",
			Headers:     nil,
			Expected:    `[{"escapeId":1},{"escapeId":3},{"escapeId":5}]`,
			Status:      200,
		},
		//   it "fails if an operator is not given" $
		// 	get "/ghostBusters?id=0" `shouldRespondWith` [json| {"details":"Failed to parse [(\"id\",\"0\")]","message":"Unexpected param or filter missing operator","code":"PGRST104","hint":null} |]
		// 	  { matchStatus  = 400
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }
		{
			Description: "fails if an operator is not given",
			Query:       "/ghostBusters?id=0",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		//   it "will embed a collection" $
		// 	get "/Escap3e;?select=ghostBusters(*)" `shouldRespondWith`
		// 	  [json| [{"ghostBusters":[{"escapeId":1}]},{"ghostBusters":[]},{"ghostBusters":[{"escapeId":3}]},{"ghostBusters":[]},{"ghostBusters":[{"escapeId":5}]}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "will embed a collection",
			Query:       "/Escap3e;?select=ghostBusters(*)",
			Expected:    `[{"ghostBusters":[{"escapeId":1}]},{"ghostBusters":[]},{"ghostBusters":[{"escapeId":3}]},{"ghostBusters":[]},{"ghostBusters":[{"escapeId":5}]}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "will select and filter a column that has spaces" $
		// 	get "/Server%20Today?select=Just%20A%20Server%20Model&Just%20A%20Server%20Model=like.*91*" `shouldRespondWith`
		// 	  [json|[
		// 		{"Just A Server Model":" IBM,9113-550 (P5-550)"},
		// 		{"Just A Server Model":" IBM,9113-550 (P5-550)"},
		// 		{"Just A Server Model":" IBM,9131-52A (P5-52A)"},
		// 		{"Just A Server Model":" IBM,9133-55A (P5-55A)"}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "will select and filter a column that has spaces",
			Query:       "/Server%20Today?select=Just%20A%20Server%20Model&Just%20A%20Server%20Model=like.*91*",
			Headers:     nil,
			Expected: `
				[{"Just A Server Model":" IBM,9113-550 (P5-550)"},
				{"Just A Server Model":" IBM,9113-550 (P5-550)"},
				{"Just A Server Model":" IBM,9131-52A (P5-52A)"},
				{"Just A Server Model":" IBM,9133-55A (P5-55A)"}]`,
			Status: 200,
		},
		//   it "will select and filter a quoted column that has PostgREST reserved characters" $
		// 	get "/pgrst_reserved_chars?select=%22:arr-%3Eow::cast%22,%22(inside,parens)%22,%22a.dotted.column%22,%22%20%20col%20%20w%20%20space%20%20%22&%22*id*%22=eq.1" `shouldRespondWith`
		// 	  [json|[{":arr->ow::cast":" arrow-1 ","(inside,parens)":" parens-1 ","a.dotted.column":" dotted-1 ","  col  w  space  ":" space-1"}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "will select and filter a quoted column that has PostgREST reserved characters",
			Query:       "/pgrst_reserved_chars?select=%22:arr-%3Eow::cast%22,%22(inside,parens)%22,%22a.dotted.column%22,%22%20%20col%20%20w%20%20space%20%20%22&%22*id*%22=eq.1",
			Expected:    `[{":arr->ow::cast":" arrow-1 ","(inside,parens)":" parens-1 ","a.dotted.column":" dotted-1 ","  col  w  space  ":" space-1"}]`,
			Headers:     nil,
			Status:      200,
		},
		//   it "will select and filter a column that has dollars in(without double quoting)" $
		// 	get "/do$llar$s?select=a$num$&a$num$=eq.100" `shouldRespondWith`
		// 	  [json|[{"a$num$":100}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// @@ type not supported
		// {
		// 	Description: "will select and filter a column that has dollars in(without double quoting)",
		// 	Query:       "/do$llar$s?select=a$num$&a$num$=eq.100",
		// 	Expected:    `[{"a$num$":100}]`,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		// context "binary output" $ do
		//   it "can query if a single column is selected" $
		// 	request methodGet "/images_base64?select=img&name=eq.A.png" (acceptHdrs "application/octet-stream") ""
		// 	  `shouldRespondWith` "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC"
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Type" <:> "application/octet-stream"]
		// 	  }
		{
			Description:     "binary output: can query if a single column is selected",
			Query:           "/images_base64?select=img&name=eq.A.png",
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}},
			Expected:        `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/octet-stream"},
			Status:          200,
		},
		//   it "can get raw output with Accept: text/plain" $
		// 	request methodGet "/projects?select=name&id=eq.1" (acceptHdrs "text/plain") ""
		// 	  `shouldRespondWith` "Windows 7"
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Type" <:> "text/plain; charset=utf-8"]
		// 	  }

		//   it "can get raw xml output with Accept: text/xml" $
		// 	request methodGet "/xmltest?select=xml" (acceptHdrs "text/xml") ""
		// 	  `shouldRespondWith`
		// 	  "<myxml>foo</myxml>bar<foobar><baz/></foobar>"
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Type" <:> "text/xml; charset=utf-8"]
		// 	  }

		//   it "fails if a single column is not selected" $ do
		// 	request methodGet "/images?select=img,name&name=eq.A.png" (acceptHdrs "application/octet-stream") ""
		// 	  `shouldRespondWith`
		// 		[json| {"message":"application/octet-stream requested but more than one column was selected","code":"PGRST113","details":null,"hint":null} |]
		// 		{ matchStatus = 406 }
		{
			Description: "binary output: fails if a single column is not selected",
			Query:       "/images?select=img,name&name=eq.A.png",
			Headers:     test.Headers{"Accept": []string{"application/octet-stream"}},
			Status:      406,
		},
		// 	request methodGet "/images?select=*&name=eq.A.png"
		// 		(acceptHdrs "application/octet-stream")
		// 		""
		// 	  `shouldRespondWith`
		// 		[json| {"message":"application/octet-stream requested but more than one column was selected","code":"PGRST113","details":null,"hint":null} |]
		// 		{ matchStatus = 406 }
		{
			Description: "binary output: fails if a single column is not selected",
			Query:       "/images?select=*&name=eq.A.png",
			Headers:     test.Headers{"Accept": []string{"application/octet-stream"}},
			Status:      406,
		},
		// 	request methodGet "/images?name=eq.A.png"
		// 		(acceptHdrs "application/octet-stream")
		// 		""
		// 	  `shouldRespondWith`
		// 		[json| {"message":"application/octet-stream requested but more than one column was selected","code":"PGRST113","details":null,"hint":null} |]
		// 		{ matchStatus = 406 }
		{
			Description: "binary output: fails if a single column is not selected",
			Query:       "/images?name=eq.A.png",
			Headers:     test.Headers{"Accept": []string{"application/octet-stream"}},
			Status:      406,
		},
		//   it "concatenates results if more than one row is returned" $
		// 	request methodGet "/images_base64?select=img&name=in.(A.png,B.png)" (acceptHdrs "application/octet-stream") ""
		// 	  `shouldRespondWith` "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCCiVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEX///8AAP94wDzzAAAAL0lEQVQIW2NgwAb+HwARH0DEDyDxwAZEyGAhLODqHmBRzAcn5GAS///A1IF14AAA5/Adbiiz/0gAAAAASUVORK5CYII="
		// 	  { matchStatus = 200
		// 	  , matchHeaders = ["Content-Type" <:> "application/octet-stream"]
		// 	  }
		{
			Description:     "binary output: concatenates results if more than one row is returned",
			Query:           "/images_base64?select=img&name=in.(A.png,B.png)",
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}},
			Expected:        `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCCiVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEX///8AAP94wDzzAAAAL0lEQVQIW2NgwAb+HwARH0DEDyDxwAZEyGAhLODqHmBRzAcn5GAS///A1IF14AAA5/Adbiiz/0gAAAAASUVORK5CYII=`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/octet-stream"},
			Status:          200,
		},
		///

		// describe "values with quotes in IN and NOT IN" $ do
		//   it "succeeds when only quoted values are present" $ do
		// 	get "/w_or_wo_comma_names?name=in.(\"Hebdon, John\")" `shouldRespondWith`
		// 	  [json| [{"name":"Hebdon, John"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds when only quoted values are present",
			Query:       "/w_or_wo_comma_names?name=in.(\"Hebdon, John\")",
			Headers:     nil,
			Expected:    `[{"name":"Hebdon, John"}]`,
			Status:      200,
		},
		// 	get "/w_or_wo_comma_names?name=in.(\"Hebdon, John\",\"Williams, Mary\",\"Smith, Joseph\")" `shouldRespondWith`
		// 	  [json| [{"name":"Hebdon, John"},{"name":"Williams, Mary"},{"name":"Smith, Joseph"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds when only quoted values are present",
			Query:       "/w_or_wo_comma_names?name=in.(\"Hebdon, John\",\"Williams, Mary\",\"Smith, Joseph\")",
			Headers:     nil,
			Expected:    `[{"name":"Hebdon, John"},{"name":"Williams, Mary"},{"name":"Smith, Joseph"}]`,
			Status:      200,
		},
		// 	get "/w_or_wo_comma_names?name=not.in.(\"Hebdon, John\",\"Williams, Mary\",\"Smith, Joseph\")&limit=3" `shouldRespondWith`
		// 	  [json| [ { "name": "David White" }, { "name": "Larry Thompson" }, { "name": "Double O Seven(007)" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds when only quoted values are present",
			Query:       "/w_or_wo_comma_names?name=not.in.(\"Hebdon, John\",\"Williams, Mary\",\"Smith, Joseph\")&limit=3",
			Headers:     nil,
			Expected:    `[{"name":"David White"},{"name":"Larry Thompson"},{"name":"Double O Seven(007)"}]`,
			Status:      200,
		},

		//   it "succeeds w/ and w/o quoted values" $ do
		// 	get "/w_or_wo_comma_names?name=in.(David White,\"Hebdon, John\")" `shouldRespondWith`
		// 	  [json| [{"name":"Hebdon, John"},{"name":"David White"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds w/ and w/o quoted values",
			Query:       "/w_or_wo_comma_names?name=in.(David White,\"Hebdon, John\")",
			Headers:     nil,
			Expected:    `[{"name":"Hebdon, John"},{"name":"David White"}]`,
			Status:      200,
		},
		// 	get "/w_or_wo_comma_names?name=not.in.(\"Hebdon, John\",Larry Thompson,\"Smith, Joseph\")&limit=3" `shouldRespondWith`
		// 	  [json| [ { "name": "Williams, Mary" }, { "name": "David White" }, { "name": "Double O Seven(007)" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds w/ and w/o quoted values",
			Query:       "/w_or_wo_comma_names?name=not.in.(\"Hebdon,%20John\",Larry%20Thompson,\"Smith,%20Joseph\")&limit=3",
			Headers:     nil,
			Expected:    `[{"name":"Williams, Mary"},{"name":"David White"},{"name":"Double O Seven(007)"}]`,
			Status:      200,
		},
		// 	get "/w_or_wo_comma_names?name=in.(\"Double O Seven(007)\")" `shouldRespondWith`
		// 	  [json| [{"name":"Double O Seven(007)"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds w/ and w/o quoted values",
			Query:       "/w_or_wo_comma_names?name=in.(\"Double O Seven(007)\")",
			Headers:     nil,
			Expected:    `[{"name":"Double O Seven(007)"}]`,
			Status:      200,
		},
		//   context "escaped chars" $ do
		// 	it "accepts escaped double quotes" $
		// 	  get "/w_or_wo_comma_names?name=in.(\"Double\\\"Quote\\\"McGraw\\\"\")" `shouldRespondWith`
		// 		[json| [ { "name": "Double\"Quote\"McGraw\"" } ] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts escaped double quotes",
			Query:       "/w_or_wo_comma_names?name=in.(\"Double\\\"Quote\\\"McGraw\\\"\")",
			Headers:     nil,
			Expected:    `[{"name":"Double\"Quote\"McGraw\""}]`,
			Status:      200,
		},
		// 	it "accepts escaped backslashes" $ do
		// 	  get "/w_or_wo_comma_names?name=in.(\"\\\\\")" `shouldRespondWith`
		// 		[json| [{ "name": "\\" }] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts escaped backslashes",
			Query:       "/w_or_wo_comma_names?name=in.(\"\\\\\")",
			Headers:     nil,
			Expected:    `[{"name":"\\"}]`,
			Status:      200,
		},
		// 	  get "/w_or_wo_comma_names?name=in.(\"/\\\\Slash/\\\\Beast/\\\\\")" `shouldRespondWith`
		// 		[json| [ { "name": "/\\Slash/\\Beast/\\" } ] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts escaped backslashes",
			Query:       "/w_or_wo_comma_names?name=in.(\"/\\\\Slash/\\\\Beast/\\\\\")",
			Headers:     nil,
			Expected:    `[{"name":"/\\Slash/\\Beast/\\"}]`,
			Status:      200,
		},
		// 	it "passes any escaped char as the same char" $
		// 	  get "/w_or_wo_comma_names?name=in.(\"D\\a\\vid W\\h\\ite\")" `shouldRespondWith`
		// 		[json| [{ "name": "David White" }] |]
		// 		{ matchHeaders = [matchContentTypeJson] }
		{
			Description: "passes any escaped char as the same char",
			Query:       "/w_or_wo_comma_names?name=in.(\"D\\a\\vid%20W\\h\\ite\")",
			Headers:     nil,
			Expected:    `[{"name":"David White"}]`,
			Status:      200,
		},
		// describe "IN values without quotes" $ do
		//   it "accepts single double quotes as values" $ do
		// 	get "/w_or_wo_comma_names?name=in.(\")" `shouldRespondWith`
		// 	  [json| [{ "name": "\"" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// @@ do not work as intended, pathological case
		// works with \\\" instead of \"
		{
			Description: "accepts single double quotes as values",
			Query:       "/w_or_wo_comma_names?name=in.(\\\")",
			Expected:    `[{"name":"\""}]`,
			Headers:     nil,
			Status:      200,
		},
		// @@ do not work as intended, pathological case

		// 	get "/w_or_wo_comma_names?name=in.(Double\"Quote\"McGraw\")" `shouldRespondWith`
		// 	  [json| [ { "name": "Double\"Quote\"McGraw\"" } ] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		// @@ do not work as intended, pathological case
		// {
		// 	Description: "accepts single double quotes as values",
		// 	Query:       "/w_or_wo_comma_names?name=in.(Double\"Quote\"McGraw\")",
		// 	Expected:    `[{"name":"Double\"Quote\"McGraw\""}]`,
		// 	Headers:     nil,
		// 	Status:      200,
		// },
		// @@ do not work as intended, pathological case
		//   it "accepts backslashes as values" $ do
		// 	get "/w_or_wo_comma_names?name=in.(\\)" `shouldRespondWith`
		// 	  [json| [{ "name": "\\" }] |]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts backslashes as values",
			Query:       "/w_or_wo_comma_names?name=in.(\\\\)",
			Expected:    `[{"name":"\\"}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/w_or_wo_comma_names?name=in.(/\\Slash/\\Beast/\\)" `shouldRespondWith`
		// 	  [json| [ { "name": "/\\Slash/\\Beast/\\" } ] |]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// describe "IN and NOT IN empty set" $ do
		//   context "returns an empty result for IN when no value is present" $ do
		// 	it "works for integer" $
		// 	  get "/items_with_different_col_types?int_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?int_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for text" $
		// 	  get "/items_with_different_col_types?text_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?text_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for bool" $
		// 	  get "/items_with_different_col_types?bool_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?bool_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for bytea" $
		// 	  get "/items_with_different_col_types?bin_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?bin_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for char" $
		// 	  get "/items_with_different_col_types?char_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?char_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for date" $
		// 	  get "/items_with_different_col_types?date_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?date_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for real" $
		// 	  get "/items_with_different_col_types?real_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?real_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		// 	it "works for time" $
		// 	  get "/items_with_different_col_types?time_data=in.()" `shouldRespondWith`
		// 		[json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result for IN when no value is present",
			Query:       "/items_with_different_col_types?time_data=in.()",
			Expected:    `[]`,
			Status:      200,
		},
		//   it "returns all results for not.in when no value is present" $
		// 	get "/items_with_different_col_types?int_data=not.in.()&select=int_data" `shouldRespondWith`
		// 	  [json| [{int_data: 1}] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns all results for not.in when no value is present",
			Query:       "/items_with_different_col_types?int_data=not.in.()&select=int_data",
			Headers:     nil,
			Expected:    `[{"int_data": 1}]`,
			Status:      200,
		},
		//   it "returns an empty result ignoring spaces" $
		// 	get "/items_with_different_col_types?int_data=in.(    )" `shouldRespondWith`
		// 	  [json| [] |] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns an empty result ignoring spaces",
			Query:       "/items_with_different_col_types?int_data=in.(    )",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},
		//   it "only returns an empty result set if the in value is empty" $
		// 	get "/items_with_different_col_types?int_data=in.( ,3,4)"
		// 	  `shouldRespondWith` (
		// 	  if actualPgVersion >= pgVersion121 then
		// 	  [json| {"hint":null,"details":null,"code":"22P02","message":"invalid input syntax for type integer: \"\""} |]
		// 	  else
		// 	  [json| {"hint":null,"details":null,"code":"22P02","message":"invalid input syntax for integer: \"\""} |]
		// 						  )
		// 	  { matchStatus = 400
		// 	  , matchHeaders = [matchContentTypeJson]
		// 	  }
		{
			Description: "only returns an empty result set if the in value is empty",
			Query:       "/items_with_different_col_types?int_data=in.( ,3,4)",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		// describe "Embedding when column name = table name" $ do
		//   it "works with child embeds" $
		// 	get "/being?select=*,descendant(*)&limit=1" `shouldRespondWith`
		// 	  [json|[{"being":1,"descendant":[{"descendant":1,"being":1},{"descendant":2,"being":1},{"descendant":3,"being":1}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with child embeds",
			Query:       "/being?select=*,descendant(*)&limit=1",
			Headers:     nil,
			Expected:    `[{"being":1,"descendant":[{"descendant":1,"being":1},{"descendant":2,"being":1},{"descendant":3,"being":1}]}]`,
			Status:      200,
		},
		//   it "works with many to many embeds" $
		// 	get "/being?select=*,part(*)&limit=1" `shouldRespondWith`
		// 	  [json|[{"being":1,"part":[{"part":1}]}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works with many to many embeds",
			Query:       "/being?select=*,part(*)&limit=1",
			Headers:     nil,
			Expected:    `[{"being":1,"part":[{"part":1}]}]`,
			Status:      200,
		},
		// describe "Foreign table" $ do
		//   it "can be queried by using regular filters" $
		// 	get "/projects_dump?id=in.(1,2,3)" `shouldRespondWith`
		// 	  [json| [{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}, {"id":3,"name":"IOS","client_id":2}]|]
		// 	  { matchHeaders = [matchContentTypeJson] }"
		// @@ should configure external file
		// {
		// 	Description: "Foreign table can be queried by using regular filters",
		// 	Query:       "/projects_dump?id=in.(1,2,3)",
		// 	Headers:     nil,
		// 	Expected:    `[{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}, {"id":3,"name":"IOS","client_id":2}]`,
		// 	Status:      200,
		// },
		//   it "can be queried with select, order and limit" $
		// 	get "/projects_dump?select=id,name&order=id.desc&limit=3" `shouldRespondWith`
		// 	  [json| [{"id":5,"name":"Orphan"}, {"id":4,"name":"OSX"}, {"id":3,"name":"IOS"}] |]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// it "cannot use ltree(in public schema) extension operators if no extra search path added" $
		//   get "/ltree_sample?path=cd.Top.Science.Astronomy" `shouldRespondWith` 400

		// context "VIEW that has a source FK based on a UNIQUE key" $
		//   it "can be embedded" $
		// 	get "/referrals?select=site,link:pages(url)" `shouldRespondWith`
		// 	  [json| [
		// 	   {"site":"github.com",     "link":{"url":"http://postgrest.org/en/v6.0/api.html"}},
		// 	   {"site":"hub.docker.com", "link":{"url":"http://postgrest.org/en/v6.0/admin.html"}}
		// 	  ]|]
		// 	  { matchHeaders = [matchContentTypeJson] }

		// it "shouldn't produce a Content-Profile header since only a single schema is exposed" $ do
		//   r <- get "/items"
		//   liftIO $ do
		// 	let respHeaders = simpleHeaders r
		// 	respHeaders `shouldSatisfy` noProfileHeader

		// 	context "empty embed" $ do
		//     it "works on a many-to-one relationship" $ do
		//       get "/projects?select=id,name,clients()" `shouldRespondWith`
		//         [json| [
		//           {"id":1,"name":"Windows 7"},
		//           {"id":2,"name":"Windows 10"},
		//           {"id":3,"name":"IOS"},
		//           {"id":4,"name":"OSX"},
		//           {"id":5,"name":"Orphan"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a many-to-one relationship",
			Query:       "/projects?select=id,name,clients()",
			Headers:     nil,
			Expected: `[
				          {"id":1,"name":"Windows 7"},
				          {"id":2,"name":"Windows 10"},
				          {"id":3,"name":"IOS"},
				          {"id":4,"name":"OSX"},
				          {"id":5,"name":"Orphan"}]`,
			Status: 200,
		},
		//       get "/projects?select=id,name,clients!inner()&clients.id=eq.2" `shouldRespondWith`
		//         [json|[
		//           {"id":3,"name":"IOS"},
		//           {"id":4,"name":"OSX"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a many-to-one relationship",
			Query:       "/projects?select=id,name,clients!inner()&clients.id=eq.2",
			Headers:     nil,
			Expected: `[
				          {"id":3,"name":"IOS"},
				          {"id":4,"name":"OSX"}]`,
			Status: 200,
		},
		//     it "works on a one-to-many relationship" $ do
		//       get "/clients?select=id,name,projects()" `shouldRespondWith`
		//         [json| [{"id":1,"name":"Microsoft"}, {"id":2,"name":"Apple"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a one-to-many relationship",
			Query:       "/clients?select=id,name,projects()",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Microsoft"}, {"id":2,"name":"Apple"}]`,
			Status:      200,
		},
		//       get "/clients?select=id,name,projects!inner()&projects.name=eq.IOS" `shouldRespondWith`
		//         [json|[{"id":2,"name":"Apple"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a one-to-many relationship",
			Query:       "/clients?select=id,name,projects!inner()&projects.name=eq.IOS",
			Headers:     nil,
			Expected:    `[{"id":2,"name":"Apple"}]`,
			Status:      200,
		},
		//     it "works on a many-to-many relationship" $ do
		//       get "/users?select=*,tasks!inner()" `shouldRespondWith`
		//         [json| [{"id":1,"name":"Angela Martin"}, {"id":2,"name":"Michael Scott"}, {"id":3,"name":"Dwight Schrute"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a many-to-many relationship",
			Query:       "/users?select=*,tasks!inner()",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Angela Martin"}, {"id":2,"name":"Michael Scott"}, {"id":3,"name":"Dwight Schrute"}]`,
			Status:      200,
		},
		//       get "/users?select=*,tasks!inner()&tasks.id=eq.3" `shouldRespondWith`
		//         [json|[{"id":1,"name":"Angela Martin"}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty embed works on a many-to-many relationship",
			Query:       "/users?select=*,tasks!inner()&tasks.id=eq.3",
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Angela Martin"}]`,
			Status:      200,
		},
		//   context "empty root select" $
		//     it "gives all columns" $ do
		//       get "/projects?select=" `shouldRespondWith`
		//         [json|[
		//           {"id":1,"name":"Windows 7","client_id":1},
		//           {"id":2,"name":"Windows 10","client_id":1},
		//           {"id":3,"name":"IOS","client_id":2},
		//           {"id":4,"name":"OSX","client_id":2},
		//           {"id":5,"name":"Orphan","client_id":null}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty root select gives all columns",
			Query:       "/projects?select=",
			Headers:     nil,
			Expected: `[
				          {"id":1,"name":"Windows 7","client_id":1},
				          {"id":2,"name":"Windows 10","client_id":1},
				          {"id":3,"name":"IOS","client_id":2},
				          {"id":4,"name":"OSX","client_id":2},
				          {"id":5,"name":"Orphan","client_id":null}]`,
			Status: 200,
		},
		//       get "/rpc/getallprojects?select=" `shouldRespondWith`
		//         [json|[
		//           {"id":1,"name":"Windows 7","client_id":1},
		//           {"id":2,"name":"Windows 10","client_id":1},
		//           {"id":3,"name":"IOS","client_id":2},
		//           {"id":4,"name":"OSX","client_id":2},
		//           {"id":5,"name":"Orphan","client_id":null}]|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "empty root select gives all columns",
			Query:       "/rpc/getallprojects?select=",
			Headers:     nil,
			Expected: `[
				          {"id":1,"name":"Windows 7","client_id":1},
				          {"id":2,"name":"Windows 10","client_id":1},
				          {"id":3,"name":"IOS","client_id":2},
				          {"id":4,"name":"OSX","client_id":2},
				          {"id":5,"name":"Orphan","client_id":null}]`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
