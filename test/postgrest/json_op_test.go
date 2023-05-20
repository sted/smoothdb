package postgrest

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestPostgREST_JSON_Op(t *testing.T) {

	tests := []test.Test{
		// 		spec actualPgVersion = describe "json and jsonb operators" $ do
		//   context "Shaping response with select parameter" $ do
		//     it "obtains a json subfield one level with casting" $
		//       get "/complex_items?id=eq.1&select=settings->>foo::json" `shouldRespondWith`
		//         [json| [{"foo":{"int":1,"bar":"baz"}}] |] -- the value of foo here is of type "text"
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "obtains a json subfield one level with casting",
			Query:       "/complex_items?id=eq.1&select=settings->>foo::json",
			Headers:     nil,
			Expected:    `[{"foo":{"int":1,"bar":"baz"}}]`,
			Status:      200,
		},
		//     it "renames json subfield one level with casting" $
		//       get "/complex_items?id=eq.1&select=myFoo:settings->>foo::json" `shouldRespondWith`
		//         [json| [{"myFoo":{"int":1,"bar":"baz"}}] |] -- the value of foo here is of type "text"
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "renames json subfield one level with casting",
			Query:       "/complex_items?id=eq.1&select=myFoo:settings->>foo::json",
			Headers:     nil,
			Expected:    `[{"myFoo":{"int":1,"bar":"baz"}}]`,
			Status:      200,
		},
		//     it "fails on bad casting (data of the wrong format)" $
		//       get "/complex_items?select=settings->foo->>bar::integer"
		//         `shouldRespondWith` (
		//         if actualPgVersion >= pgVersion121 then
		//         [json| {"hint":null,"details":null,"code":"22P02","message":"invalid input syntax for type integer: \"baz\""} |]
		//         else
		//         [json| {"hint":null,"details":null,"code":"22P02","message":"invalid input syntax for integer: \"baz\""} |]
		//                             )
		//         { matchStatus  = 400 , matchHeaders = [] }
		{
			Description: "fails on bad casting (data of the wrong format)",
			Query:       "/complex_items?select=settings->foo->>bar::integer",
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		//     it "obtains a json subfield two levels (string)" $
		//       get "/complex_items?id=eq.1&select=settings->foo->>bar" `shouldRespondWith`
		//         [json| [{"bar":"baz"}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "obtains a json subfield two levels (string)",
			Query:       "/complex_items?id=eq.1&select=settings->foo->>bar",
			Headers:     nil,
			Expected:    `[{"bar":"baz"}]`,
			Status:      200,
		},
		//     it "renames json subfield two levels (string)" $
		//       get "/complex_items?id=eq.1&select=myBar:settings->foo->>bar" `shouldRespondWith`
		//         [json| [{"myBar":"baz"}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "renames json subfield two levels (string)",
			Query:       "/complex_items?id=eq.1&select=myBar:settings->foo->>bar",
			Headers:     nil,
			Expected:    `[{"myBar":"baz"}]`,
			Status:      200,
		},
		//     it "obtains a json subfield two levels with casting (int)" $
		//       get "/complex_items?id=eq.1&select=settings->foo->>int::integer" `shouldRespondWith`
		//         [json| [{"int":1}] |] -- the value in the db is an int, but here we expect a string for now
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "obtains a json subfield two levels with casting (int)",
			Query:       "/complex_items?id=eq.1&select=settings->foo->>int::integer",
			Headers:     nil,
			Expected:    `[{"int":1}]`,
			Status:      200,
		},
		//     it "renames json subfield two levels with casting (int)" $
		//       get "/complex_items?id=eq.1&select=myInt:settings->foo->>int::integer" `shouldRespondWith`
		//         [json| [{"myInt":1}] |] -- the value in the db is an int, but here we expect a string for now
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "renames json subfield two levels with casting (int)",
			Query:       "/complex_items?id=eq.1&select=myInt:settings->foo->>int::integer",
			Headers:     nil,
			Expected:    `[{"myInt":1}]`,
			Status:      200,
		},
		//     -- TODO the status code for the error is 404, this is because 42883 represents undefined function
		//     -- this works fine for /rpc/unexistent requests, but for this case a 500 seems more appropriate
		//     it "fails when a double arrow ->> is followed with a single arrow ->" $ do
		//       get "/json_arr?select=data->>c->1"
		//         `shouldRespondWith` (
		//         if actualPgVersion >= pgVersion112 then
		//         [json|
		//           {"hint":"No operator matches the given name and argument types. You might need to add explicit type casts.",
		//            "details":null,"code":"42883","message":"operator does not exist: text -> integer"} |]
		//            else
		//         [json|
		//           {"hint":"No operator matches the given name and argument type(s). You might need to add explicit type casts.",
		//            "details":null,"code":"42883","message":"operator does not exist: text -> integer"} |]
		//                             )
		//         { matchStatus  = 404 , matchHeaders = [] }
		//       get "/json_arr?select=data->>c->b"
		//         `shouldRespondWith` (
		//         if actualPgVersion >= pgVersion112 then
		//         [json|
		//           {"hint":"No operator matches the given name and argument types. You might need to add explicit type casts.",
		//            "details":null,"code":"42883","message":"operator does not exist: text -> unknown"} |]
		//            else
		//         [json|
		//           {"hint":"No operator matches the given name and argument type(s). You might need to add explicit type casts.",
		//            "details":null,"code":"42883","message":"operator does not exist: text -> unknown"} |]
		//                             )
		//         { matchStatus  = 404 , matchHeaders = [] }

		//     context "with array index" $ do
		//       it "can get array of ints and alias/cast it" $ do
		//         get "/json_arr?select=data->>0::int&id=in.(1,2)" `shouldRespondWith`
		//           [json| [{"data":1}, {"data":4}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array of ints and alias/cast it",
			Query:       "/json_arr?select=data->>0::int&id=in.(1,2)",
			Headers:     nil,
			Expected:    `[{"data":1}, {"data":4}]`,
			Status:      200,
		},
		//         get "/json_arr?select=idx0:data->>0::int,idx1:data->>1::int&id=in.(1,2)" `shouldRespondWith`
		//           [json| [{"idx0":1,"idx1":2}, {"idx0":4,"idx1":5}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array of ints and alias/cast it",
			Query:       "/json_arr?select=idx0:data->>0::int,idx1:data->>1::int&id=in.(1,2)",
			Headers:     nil,
			Expected:    `[{"idx0":1,"idx1":2}, {"idx0":4,"idx1":5}]`,
			Status:      200,
		},
		//       it "can get nested array of ints" $ do
		//         get "/json_arr?select=data->0->>1::int&id=in.(3,4)" `shouldRespondWith`
		//           [json| [{"data":8}, {"data":7}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get nested array of ints",
			Query:       "/json_arr?select=data->0->>1::int&id=in.(3,4)",
			Headers:     nil,
			Expected:    `[{"data":8}, {"data":7}]`,
			Status:      200,
		}, //         get "/json_arr?select=data->0->0->>1::int&id=in.(3,4)" `shouldRespondWith`
		//           [json| [{"data":null}, {"data":6}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get nested array of ints",
			Query:       "/json_arr?select=data->0->0->>1::int&id=in.(3,4)",
			Headers:     nil,
			Expected:    `[{"data":null}, {"data":6}]`,
			Status:      200,
		},
		//       it "can get array of objects" $ do
		//         get "/json_arr?select=data->0->>a&id=in.(5,6)" `shouldRespondWith`
		//           [json| [{"a":"A"}, {"a":"[1, 2, 3]"}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array of objects",
			Query:       "/json_arr?select=data->0->>a&id=in.(5,6)",
			Headers:     nil,
			Expected:    `[{"a":"A"}, {"a":"[1,2,3]"}]`,
			Status:      200,
		},
		//         get "/json_arr?select=data->0->a->>2&id=in.(5,6)" `shouldRespondWith`
		//           [json| [{"a":null}, {"a":"3"}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array of objects",
			Query:       "/json_arr?select=data->0->a->>2&id=in.(5,6)",
			Headers:     nil,
			Expected:    `[{"a":null}, {"a":"3"}]`,
			Status:      200,
		},
		//       it "can get array in object keys" $ do
		//         get "/json_arr?select=data->c->>0::json&id=in.(7,8)" `shouldRespondWith`
		//           [json| [{"c":1}, {"c":{"d": [4,5,6,7,8]}}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array in object keys",
			Query:       "/json_arr?select=data->c->>0::json&id=in.(7,8)",
			Headers:     nil,
			Expected:    `[{"c":1}, {"c":{"d": [4,5,6,7,8]}}]`,
			Status:      200,
		},
		//         get "/json_arr?select=data->c->0->d->>4::int&id=in.(7,8)" `shouldRespondWith`
		//           [json| [{"d":null}, {"d":8}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index can get array in object keys",
			Query:       "/json_arr?select=data->c->0->d->>4::int&id=in.(7,8)",
			Headers:     nil,
			Expected:    `[{"d":null}, {"d":8}]`,
			Status:      200,
		},
		//       it "only treats well formed numbers as indexes" $
		//         get "/json_arr?select=data->0->0xy1->1->23-xy-45->1->xy-6->>0::int&id=eq.9" `shouldRespondWith`
		//           [json| [{"xy-6":3}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with array index only treats well formed numbers as indexes",
			Query:       "/json_arr?select=data->0->0xy1->1->23-xy-45->1->xy-6->>0::int&id=eq.9",
			Headers:     nil,
			Expected:    `[{"xy-6":3}]`,
			Status:      200,
		},
		//     context "finishing json path with single arrow ->" $ do
		//       it "works when finishing with a key" $ do
		//         get "/json_arr?select=data->c&id=in.(7,8)" `shouldRespondWith`
		//           [json| [{"c":[1,2,3]}, {"c":[{"d": [4,5,6,7,8]}]}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finishing json path with single arrow -> works when finishing with a key",
			Query:       "/json_arr?select=data->c&id=in.(7,8)",
			Headers:     nil,
			Expected:    `[{"c":[1,2,3]}, {"c":[{"d": [4,5,6,7,8]}]}]`,
			Status:      200,
		},
		//         get "/json_arr?select=data->0->a&id=in.(5,6)" `shouldRespondWith`
		//           [json| [{"a":"A"}, {"a":[1,2,3]}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finishing json path with single arrow -> works when finishing with a key",
			Query:       "/json_arr?select=data->0->a&id=in.(5,6)",
			Headers:     nil,
			Expected:    `[{"a":"A"}, {"a":[1,2,3]}]`,
			Status:      200,
		},
		//         get "/json_arr?select=data->c->0->d&id=eq.8" `shouldRespondWith`
		//           [json| [{"d":[4,5,6,7,8]}] |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "finishing json path with single arrow -> works when finishing with a index",
			Query:       "/json_arr?select=data->c->0->d&id=eq.8",
			Headers:     nil,
			Expected:    `[{"d":[4,5,6,7,8]}]`,
			Status:      200,
		},
		// 	it "obtains a composite type field" $ do
		// 	get "/fav_numbers?select=num->i"
		// 		`shouldRespondWith`
		// 		[json| [{"i":0.5},{"i":0.6}] |]
		{
			Description: "obtains a composite type field",
			Query:       "/fav_numbers?select=num->i",
			Expected:    `[{"i":0.5},{"i":0.6}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/fav_numbers?select=num->>i"
		// 		`shouldRespondWith`
		// 		[json| [{"i":"0.5"},{"i":"0.6"}] |]
		{
			Description: "obtains a composite type field",
			Query:       "/fav_numbers?select=num->>i",
			Expected:    `[{"i":"0.5"},{"i":"0.6"}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	it "obtains an array item" $ do
		// 	get "/arrays?select=a:numbers->0,b:numbers->1,c:numbers_mult->0->0,d:numbers_mult->1->2"
		// 		`shouldRespondWith`
		// 		[json| [{"a":1,"b":2,"c":1,"d":6},{"a":11,"b":12,"c":11,"d":16}] |]
		{
			Description: "obtains an array item",
			Query:       "/arrays?select=a:numbers->0,b:numbers->1,c:numbers_mult->0->0,d:numbers_mult->1->2",
			Expected:    `[{"a":1,"b":2,"c":1,"d":6},{"a":11,"b":12,"c":11,"d":16}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/arrays?select=a:numbers->>0,b:numbers->>1,c:numbers_mult->0->>0,d:numbers_mult->1->>2"
		// 		`shouldRespondWith`
		// 		[json| [{"a":"1","b":"2","c":"1","d":"6"},{"a":"11","b":"12","c":"11","d":"16"}] |]
		{
			Description: "obtains an array item",
			Query:       "/arrays?select=a:numbers->>0,b:numbers->>1,c:numbers_mult->0->>0,d:numbers_mult->1->>2",
			Expected:    `[{"a":"1","b":"2","c":"1","d":"6"},{"a":"11","b":"12","c":"11","d":"16"}]`,
			Headers:     nil,
			Status:      200,
		},
		//   context "filtering response" $ do
		//     it "can filter by properties inside json column" $ do
		//       get "/json_table?data->foo->>bar=eq.baz" `shouldRespondWith`
		//         [json| [{"data": {"id": 1, "foo": {"bar": "baz"}}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "filtering response can filter by properties inside json column",
			Query:       "/json_table?data->foo->>bar=eq.baz",
			Headers:     nil,
			Expected:    `[{"data": {"id": 1, "foo": {"bar": "baz"}}}]`,
			Status:      200,
		},
		//       get "/json_table?data->foo->>bar=eq.fake" `shouldRespondWith`
		//         [json| [] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "filtering response can filter by properties inside json column",
			Query:       "/json_table?data->foo->>bar=eq.fake",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},
		//     it "can filter by properties inside json column using not" $
		//       get "/json_table?data->foo->>bar=not.eq.baz" `shouldRespondWith`
		//         [json| [] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by properties inside json column using not",
			Query:       "/json_table?data->foo->>bar=not.eq.baz",
			Headers:     nil,
			Expected:    `[]`,
			Status:      200,
		},
		//     it "can filter by properties inside json column using ->>" $
		//       get "/json_table?data->>id=eq.1" `shouldRespondWith`
		//         [json| [{"data": {"id": 1, "foo": {"bar": "baz"}}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by properties inside json column using ->>",
			Query:       "/json_table?data->>id=eq.1",
			Headers:     nil,
			Expected:    `[{"data": {"id": 1, "foo": {"bar": "baz"}}}]`,
			Status:      200,
		},
		//     it "can be filtered with and/or" $
		//       get "/grandchild_entities?or=(jsonb_col->a->>b.eq.foo, jsonb_col->>b.eq.bar)&select=id" `shouldRespondWith`
		//         [json|[{id: 4}, {id: 5}]|] { matchStatus = 200, matchHeaders = [matchContentTypeJson] }
		{
			Description: "can be filtered with and/or",
			Query:       "/grandchild_entities?or=(jsonb_col->a->>b.eq.foo, jsonb_col->>b.eq.bar)&select=id",
			Headers:     nil,
			Expected:    `[{"id": 4}, {"id": 5}]`,
			Status:      200,
		},
		//     it "can filter by array indexes" $ do
		//       get "/json_arr?select=data&data->>0=eq.1" `shouldRespondWith`
		//         [json| [{"data":[1, 2, 3]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by array indexes",
			Query:       "/json_arr?select=data&data->>0=eq.1",
			Headers:     nil,
			Expected:    `[{"data":[1, 2, 3]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->1->>2=eq.13" `shouldRespondWith`
		//         [json| [{"data":[[9, 8, 7], [11, 12, 13]]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by array indexes",
			Query:       "/json_arr?select=data&data->1->>2=eq.13",
			Headers:     nil,
			Expected:    `[{"data":[[9, 8, 7], [11, 12, 13]]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->1->>b=eq.B" `shouldRespondWith`
		//         [json| [{"data":[{"a": "A"}, {"b": "B"}]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by array indexes",
			Query:       "/json_arr?select=data&data->1->>b=eq.B",
			Headers:     nil,
			Expected:    `[{"data":[{"a": "A"}, {"b": "B"}]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->1->b->>1=eq.5" `shouldRespondWith`
		//         [json| [{"data":[{"a": [1,2,3]}, {"b": [4,5]}]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter by array indexes",
			Query:       "/json_arr?select=data&data->1->b->>1=eq.5",
			Headers:     nil,
			Expected:    `[{"data":[{"a": [1,2,3]}, {"b": [4,5]}]}]`,
			Status:      200,
		},
		//     it "can filter jsonb" $ do
		//       get "/jsonb_test?data=eq.{\"e\":1}" `shouldRespondWith`
		//         [json| [{"id":4,"data":{"e": 1}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter jsonb",
			Query:       "/jsonb_test?data=eq.{\"e\":1}",
			Headers:     nil,
			Expected:    `[{"id":4,"data":{"e": 1}}]`,
			Status:      200,
		},
		//       get "/jsonb_test?data->a=eq.{\"b\":2}" `shouldRespondWith`
		//         [json| [{"id":1,"data":{"a": {"b": 2}}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter jsonb",
			Query:       "/jsonb_test?data->a=eq.{\"b\":2}",
			Headers:     nil,
			Expected:    `[{"id":1,"data":{"a": {"b": 2}}}]`,
			Status:      200,
		},
		//       get "/jsonb_test?data->c=eq.[1,2,3]" `shouldRespondWith`
		//         [json| [{"id":2,"data":{"c": [1, 2, 3]}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter jsonb",
			Query:       "/jsonb_test?data->c=eq.[1,2,3]",
			Headers:     nil,
			Expected:    `[{"id":2,"data":{"c": [1, 2, 3]}}]`,
			Status:      200,
		},
		//       get "/jsonb_test?data->0=eq.{\"d\":\"test\"}" `shouldRespondWith`
		//         [json| [{"id":3,"data":[{"d": "test"}]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter jsonb",
			Query:       "/jsonb_test?data->0=eq.{\"d\":\"test\"}",
			Headers:     nil,
			Expected:    `[{"id":3,"data":[{"d": "test"}]}]`,
			Status:      200,
		},
		// 	it "can filter composite type field" $
		// 	get "/fav_numbers?num->>i=gt.0.5"
		// 		`shouldRespondWith`
		// 		[json| [{"num":{"r":0.6,"i":0.6},"person":"B"}] |]
		{
			Description: "can filter composite type field",
			Query:       "/fav_numbers?num->>i=gt.0.5",
			Expected:    `[{"num":{"r":0.6,"i":0.6},"person":"B"}]`,
			Headers:     nil,
			Status:      200,
		},

		// 	it "can filter array item" $ do
		// 	get "/arrays?select=id&numbers->0=eq.1"
		// 		`shouldRespondWith`
		// 		[json| [{"id":0}] |]
		{
			Description: "can filter array item",
			Query:       "/arrays?select=id&numbers->0=eq.1",
			Expected:    `[{"id":0}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/arrays?select=id&numbers->>0=eq.11"
		// 		`shouldRespondWith`
		// 		[json| [{"id":1}] |]
		{
			Description: "can filter array item",
			Query:       "/arrays?select=id&numbers->>0=eq.11",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/arrays?select=id&numbers_mult->1->1=eq.5"
		// 		`shouldRespondWith`
		// 		[json| [{"id":0}] |]
		{
			Description: "can filter array item",
			Query:       "/arrays?select=id&numbers_mult->1->1=eq.5",
			Expected:    `[{"id":0}]`,
			Headers:     nil,
			Status:      200,
		},
		// 	get "/arrays?select=id&numbers_mult->2->>2=eq.19"
		// 		`shouldRespondWith`
		// 		[json| [{"id":1}] |]
		{
			Description: "can filter array item",
			Query:       "/arrays?select=id&numbers_mult->2->>2=eq.19",
			Expected:    `[{"id":1}]`,
			Headers:     nil,
			Status:      200,
		},

		//   context "ordering response" $ do
		//     it "orders by a json column property asc" $
		//       get "/json_table?order=data->>id.asc" `shouldRespondWith`
		//         [json| [{"data": {"id": 0}}, {"data": {"id": 1, "foo": {"bar": "baz"}}}, {"data": {"id": 3}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "orders by a json column property asc",
			Query:       "/json_table?order=data->>id.asc",
			Headers:     nil,
			Expected:    `[{"data": {"id": 0}}, {"data": {"id": 1, "foo": {"bar": "baz"}}}, {"data": {"id": 3}}]`,
			Status:      200,
		},
		//     it "orders by a json column with two level property nulls first" $
		//       get "/json_table?order=data->foo->>bar.nullsfirst" `shouldRespondWith`
		//         [json| [{"data": {"id": 3}}, {"data": {"id": 0}}, {"data": {"id": 1, "foo": {"bar": "baz"}}}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "orders by a json column with two level property nulls first",
			Query:       "/json_table?order=data->foo->>bar.nullsfirst",
			Headers:     nil,
			Expected:    `[{"data": {"id": 3}}, {"data": {"id": 0}}, {"data": {"id": 1, "foo": {"bar": "baz"}}}]`,
			Status:      200,
		},
		//     it "orders by composite type field" $ do
		//       get "/fav_numbers?order=num->i.asc"
		//         `shouldRespondWith`
		//           [json| [{"num":{"r":0.5,"i":0.5},"person":"A"}, {"num":{"r":0.6,"i":0.6},"person":"B"}] |]
		{
			Description: "orders by composite type field",
			Query:       "/fav_numbers?order=num->i.asc",
			Headers:     nil,
			Expected:    `[{"num":{"r":0.5,"i":0.5},"person":"A"}, {"num":{"r":0.6,"i":0.6},"person":"B"}]`,
			Status:      200,
		},
		//       get "/fav_numbers?order=num->>i.desc"
		//         `shouldRespondWith`
		//           [json| [{"num":{"r":0.6,"i":0.6},"person":"B"}, {"num":{"r":0.5,"i":0.5},"person":"A"}] |]
		{
			Description: "orders by composite type field",
			Query:       "/fav_numbers?order=num->>i.desc",
			Headers:     nil,
			Expected:    `[{"num":{"r":0.6,"i":0.6},"person":"B"}, {"num":{"r":0.5,"i":0.5},"person":"A"}]`,
			Status:      200,
		},
		//     it "orders by array item" $ do
		//       get "/arrays?select=id&order=numbers->0.desc"
		//         `shouldRespondWith`
		//           [json| [{"id":1},{"id":0}] |]
		{
			Description: "orders by array item",
			Query:       "/arrays?select=id&order=numbers->0.desc",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":0}]`,
			Status:      200,
		},
		//       get "/arrays?select=id&order=numbers->1.asc"
		//         `shouldRespondWith`
		//           [json| [{"id":0},{"id":1}] |]
		{
			Description: "orders by array item",
			Query:       "/arrays?select=id&order=numbers->1.asc",
			Headers:     nil,
			Expected:    `[{"id":0},{"id":1}]`,
			Status:      200,
		},
		//       get "/arrays?select=id&order=numbers_mult->0->0.desc"
		//         `shouldRespondWith`
		//           [json| [{"id":1},{"id":0}] |]
		{
			Description: "orders by array item",
			Query:       "/arrays?select=id&order=numbers_mult->0->0.desc",
			Headers:     nil,
			Expected:    `[{"id":1},{"id":0}]`,
			Status:      200,
		},
		//       get "/arrays?select=id&order=numbers_mult->2->2.asc"
		//         `shouldRespondWith`
		//           [json| [{"id":0},{"id":1}] |]
		{
			Description: "orders by array item",
			Query:       "/arrays?select=id&order=numbers_mult->2->2.asc",
			Headers:     nil,
			Expected:    `[{"id":0},{"id":1}]`,
			Status:      200,
		},
		//   context "Patching record, in a nonempty table" $
		//     it "can set a json column to escaped value" $ do
		//       request methodPatch "/json_table?data->>id=eq.3"
		//           [("Prefer", "return=representation")]
		//           [json| { "data": { "id":" \"escaped" } } |]
		//         `shouldRespondWith`
		//           [json| [{ "data": { "id":" \"escaped" } }] |]
		{
			Description: "Patching record, in a nonempty table can set a json column to escaped value",
			Method:      "PATCH",
			Query:       "/json_table?data->>id=eq.3",
			Body:        `{ "data": { "id":" \"escaped" } }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "data": { "id":" \"escaped" } }]`,
			Status:      200,
		},
		//   context "json array negative index" $ do
		//     it "can select with negative indexes" $ do
		//       get "/json_arr?select=data->>-1::int&id=in.(1,2)" `shouldRespondWith`
		//         [json| [{"data":3}, {"data":6}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can select with negative indexes",
			Query:       "/json_arr?select=data->>-1::int&id=in.(1,2)",
			Headers:     nil,
			Expected:    `[{"data":3}, {"data":6}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data->0->>-2::int&id=in.(3,4)" `shouldRespondWith`
		//         [json| [{"data":8}, {"data":7}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can select with negative indexes",
			Query:       "/json_arr?select=data->0->>-2::int&id=in.(3,4)",
			Headers:     nil,
			Expected:    `[{"data":8}, {"data":7}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data->-2->>a&id=in.(5,6)" `shouldRespondWith`
		//         [json| [{"a":"A"}, {"a":"[1, 2, 3]"}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can select with negative indexes",
			Query:       "/json_arr?select=data->-2->>a&id=in.(5,6)",
			Headers:     nil,
			Expected:    `[{"a":"A"}, {"a":"[1,2,3]"}]`,
			Status:      200,
		},
		//     it "can filter with negative indexes" $ do
		//       get "/json_arr?select=data&data->>-3=eq.1" `shouldRespondWith`
		//         [json| [{"data":[1, 2, 3]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter with negative indexes",
			Query:       "/json_arr?select=data&data->>-3=eq.1",
			Headers:     nil,
			Expected:    `[{"data":[1, 2, 3]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->-1->>-3=eq.11" `shouldRespondWith`
		//         [json| [{"data":[[9, 8, 7], [11, 12, 13]]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter with negative indexes",
			Query:       "/json_arr?select=data&data->-1->>-3=eq.11",
			Headers:     nil,
			Expected:    `[{"data":[[9, 8, 7], [11, 12, 13]]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->-1->>b=eq.B" `shouldRespondWith`
		//         [json| [{"data":[{"a": "A"}, {"b": "B"}]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter with negative indexes",
			Query:       "/json_arr?select=data&data->-1->>b=eq.B",
			Headers:     nil,
			Expected:    `[{"data":[{"a": "A"}, {"b": "B"}]}]`,
			Status:      200,
		},
		//       get "/json_arr?select=data&data->-1->b->>-1=eq.5" `shouldRespondWith`
		//         [json| [{"data":[{"a": [1,2,3]}, {"b": [4,5]}]}] |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter with negative indexes",
			Query:       "/json_arr?select=data&data->-1->b->>-1=eq.5",
			Headers:     nil,
			Expected:    `[{"data":[{"a": [1,2,3]}, {"b": [4,5]}]}]`,
			Status:      200,
		},
		//   it "gives a meaningful error on bad syntax" $
		//     get "/json_arr?select=data->>--34" `shouldRespondWith`
		//       [json|
		//         {"details": "unexpected \"-\" expecting digit",
		//          "message": "\"failed to parse select parameter (data->>--34)\" (line 1, column 9)",
		//          "code": "PGRST100",
		//          "hint": null} |]
		//       { matchStatus = 400, matchHeaders = [matchContentTypeJson] }
	}

	test.Execute(t, testConfig, tests)
}
