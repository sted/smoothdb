package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Insert(t *testing.T) {

	tests := []test.Test{
		//   describe "Posting new record" $ do
		//     context "disparate json types" $ do
		//       it "accepts disparate json types" $ do
		//         post
		//           "/menagerie"
		//           [json| {
		//             "integer": 13, "double": 3.14159, "varchar": "testing!"
		//           , "boolean": false, "date": "1900-01-01", "money": "$3.99"
		//           , "enum": "foo"
		//           } |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , -- should not have content type set when body is empty
		//               matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Content-Length" <:> "0"
		//                 ]
		//             }
		{
			Description: "accepts disparate json types",
			Method:      "POST",
			Query:       "/menagerie",
			Body: `{
				 		"integer": 13, "double": 3.14159, "varchar": "testing!"
				 		, "boolean": false, "date": "1900-01-01", "money": "$3.99"
				 		, "enum": "foo"
				 	}`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "0"},
			Status:          201,
		},
		//       it "filters columns in result using &select" $
		//         request
		//           methodPost
		//           "/menagerie?select=integer,varchar"
		//           [("Prefer", "return=representation")]
		//           [json| [{
		//             "integer": 14, "double": 3.14159, "varchar": "testing!"
		//           , "boolean": false, "date": "1900-01-01", "money": "$3.99"
		//           , "enum": "foo"
		//           }] |]
		//           `shouldRespondWith` [json|[{"integer":14,"varchar":"testing!"}]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , "Content-Length" <:> "37"
		//                 , "Preference-Applied" <:> "return=representation"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "filters columns in result using &select",
			Method:      "POST",
			Query:       "/menagerie?select=integer,varchar",
			Body: `[{
				 		"integer": 14, "double": 3.14159, "varchar": "testing!"
				 		 , "boolean": false, "date": "1900-01-01", "money": "$3.99"
				 		 , "enum": "foo"
				 	}]`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"integer":14,"varchar":"testing!"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Length": "37"},
			Status:          201,
		},
		//       it "ignores &select when return not set or using return=minimal" $ do
		//         request
		//           methodPost
		//           "/menagerie?select=integer,varchar"
		//           []
		//           [json| [{
		//               "integer": 15, "double": 3.14159, "varchar": "testing!",
		//               "boolean": false, "date": "1900-01-01", "money": "$3.99",
		//               "enum": "foo"
		//             }] |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description: "ignores &select when return not set or using return=minimal",
			Method:      "POST",
			Query:       "/menagerie?select=integer,varchar",
			Body: `[{
				 		"integer": 15, "double": 3.14159, "varchar": "testing!"
				 		 , "boolean": false, "date": "1900-01-01", "money": "$3.99"
				 		 , "enum": "foo"
				 	}]`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          201,
		},
		//         request
		//           methodPost
		//           "/menagerie?select=integer,varchar"
		//           [("Prefer", "return=minimal")]
		//           [json| [{
		//               "integer": 16, "double": 3.14159, "varchar": "testing!",
		//               "boolean": false, "date": "1900-01-01", "money": "$3.99",
		//               "enum": "foo"
		//             }] |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Preference-Applied" <:> "return=minimal"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "ignores &select when return not set or using return=minimal",
			Method:      "POST",
			Query:       "/menagerie?select=integer,varchar",
			Body: `[{
				 		"integer": 16, "double": 3.14159, "varchar": "testing!"
				 		 , "boolean": false, "date": "1900-01-01", "money": "$3.99"
				 		 , "enum": "foo"
				 	}]`,
			Headers:         test.Headers{"Prefer": {"return=minimal"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          201,
		},
		//     context "non uniform json array" $ do
		//       it "rejects json array that isn't exclusivily composed of objects" $
		//         post
		//           "/articles"
		//           [json| [{"id": 100, "body": "xxxxx"}, 123, "xxxx", {"id": 111, "body": "xxxx"}] |]
		//           `shouldRespondWith` [json| {"message":"All object keys must match","code":"PGRST102","hint":null,"details":null} |]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST102 error body ("All object keys must match"); card 32494 (error body)
		{
			Description:     "rejects json array that isn't exclusivily composed of objects",
			Method:          "POST",
			Query:           "/articles",
			Body:            `[{"id": 100, "body": "xxxxx"}, 123, "xxxx", {"id": 111, "body": "xxxx"}]`,
			Headers:         nil,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//       it "rejects json array that has objects with different keys" $
		//         post
		//           "/articles"
		//           [json| [{"id": 100, "body": "xxxxx"}, {"id": 111, "body": "xxxx", "owner": "me"}] |]
		//           `shouldRespondWith` [json| {"message":"All object keys must match","code":"PGRST102","hint":null,"details":null} |]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST102 error body (code, null hint); card 32494 (error body).
		// The message is PostgREST's; the body asserted is smoothdb's shape.
		{
			Description:     "rejects json array that has objects with different keys",
			Method:          "POST",
			Query:           "/articles",
			Body:            `[{"id": 100, "body": "xxxxx"}, {"id": 111, "body": "xxxx", "owner": "me"}]`,
			Headers:         nil,
			Expected:        `{"subsystem":"network","message":"All object keys must match","code":"","hint":"","details":null,"position":0}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//     context "requesting full representation" $ do
		//       it "includes related data after insert" $
		//         request
		//           methodPost
		//           "/projects?select=id,name,clients(id,name)"
		//           [("Prefer", "return=representation"), ("Prefer", "count=exact")]
		//           [json|{"id":6,"name":"New Project","client_id":2}|]
		//           `shouldRespondWith` [json|[{"id":6,"name":"New Project","clients":{"id":2,"name":"Apple"}}]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , matchHeaderAbsent hLocation
		//                 , "Content-Range" <:> "*/1"
		//                 , "Preference-Applied" <:> "return=representation, count=exact"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "requesting full representation includes related data after insert",
			Method:          "POST",
			Query:           "/projects?select=id,name,clients(id,name)",
			Body:            `{"id":6,"name":"New Project","client_id":2}`,
			Headers:         test.Headers{"Prefer": {"return=representation", "count=exact"}},
			Expected:        `[{"id":6,"name":"New Project","clients":{"id":2,"name":"Apple"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Location": "", "Content-Range": "*/1"},
			Status:          201,
		},
		//       it "can rename and cast the selected columns" $
		//         request
		//           methodPost
		//           "/projects?select=pId:id::text,pName:name,cId:client_id::text"
		//           [("Prefer", "return=representation")]
		//           [json|{"id":7,"name":"New Project","client_id":2}|]
		//           `shouldRespondWith` [json|[{"pId":"7","pName":"New Project","cId":"2"}]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , matchHeaderAbsent hLocation
		//                 , "Content-Range" <:> "*/*"
		//                 , "Preference-Applied" <:> "return=representation"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "can rename and cast the selected columns",
			Method:          "POST",
			Query:           "/projects?select=pId:id::text,pName:name,cId:client_id::text",
			Body:            `{"id":7,"name":"New Project","client_id":2}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"pId":"7","pName":"New Project","cId":"2"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Location": "", "Content-Range": "*/*"},
			Status:          201,
		},
		//       it "should not throw and return location header when selecting without PK" $
		//         request
		//           methodPost
		//           "/projects?select=name,client_id"
		//           [("Prefer", "return=representation")]
		//           [json|{"id":10,"name":"New Project","client_id":2}|]
		//           `shouldRespondWith` [json|[{"name":"New Project","client_id":2}]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , matchHeaderAbsent hLocation
		//                 , "Content-Range" <:> "*/*"
		//                 , "Preference-Applied" <:> "return=representation"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "should not throw and return location header when selecting without PK",
			Method:          "POST",
			Query:           "/projects?select=name,client_id",
			Body:            `{"id":10,"name":"New Project","client_id":2}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"name":"New Project","client_id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Location": "", "Content-Range": "*/*"},
			Status:          201,
		},
		//     context "requesting headers only representation" $ do
		//       it "should not throw and return location header when selecting without PK" $
		//         request
		//           methodPost
		//           "/projects?select=name,client_id"
		//           [("Prefer", "return=headers-only")]
		//           [json|{"id":11,"name":"New Project","client_id":2}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/projects?id=eq.11"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "requesting headers only representation returns the location header when selecting without PK",
			Method:          "POST",
			Query:           "/projects?select=name,client_id",
			Body:            `{"id":11,"name":"New Project","client_id":2}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/projects?id=eq.11", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//       it "should not throw and return location header for partitioned tables when selecting without PK" $
		//         request
		//           methodPost
		//           "/car_models"
		//           [("Prefer", "return=headers-only")]
		//           [json|{"name":"Enzo","year":2021}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/car_models?name=eq.Enzo&year=eq.2021"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "returns the location header for partitioned tables when selecting without PK",
			Method:          "POST",
			Query:           "/car_models",
			Body:            `{"name":"Enzo","year":2021}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/car_models?name=eq.Enzo&year=eq.2021", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//     context "requesting no representation" $
		//       it "should not throw and return no location header when selecting without PK" $
		//         request
		//           methodPost
		//           "/projects?select=name,client_id"
		//           []
		//           [json|{"id":12,"name":"New Project","client_id":2}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hLocation
		//                 , "Content-Length" <:> "0"
		//                 ]
		//             }
		{
			Description:     "requesting no representation returns no location header when selecting without PK",
			Method:          "POST",
			Query:           "/projects?select=name,client_id",
			Body:            `{"id":12,"name":"New Project","client_id":2}`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "", "Content-Length": "0"},
			Status:          201,
		},
		//     context "from an html form" $
		//       it "accepts disparate json types" $ do
		//         request
		//           methodPost
		//           "/menagerie"
		//           [("Content-Type", "application/x-www-form-urlencoded")]
		//           ( "integer=7&double=2.71828&varchar=forms+are+fun&"
		//               <> "boolean=false&date=1900-01-01&money=$3.99&enum=foo"
		//           )
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Content-Length" <:> "0"
		//                 ]
		//             }
		{
			Description:     "from an html form accepts disparate json types",
			Method:          "POST",
			Query:           "/menagerie",
			Body:            `integer=7&double=2.71828&varchar=forms+are+fun&boolean=false&date=1900-01-01&money=$3.99&enum=foo`,
			Headers:         test.Headers{"Content-Type": {"application/x-www-form-urlencoded"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "0"},
			Status:          201,
		},
		//     context "with no pk supplied" $ do
		//       context "into a table with auto-incrementing pk" $
		//         it "succeeds with 201 and location header" $ do
		//           -- reset pk sequence first to make test repeatable
		//           request
		//             methodPost
		//             "/rpc/reset_sequence"
		//             [("Prefer", "tx=commit")]
		//             [json|{"name": "auto_incrementing_pk_id_seq", "value": 2}|]
		//             `shouldRespondWith` ""
		//               { matchStatus = 204
		//               , matchHeaders =
		//                   [ matchHeaderAbsent hContentType
		//                   , matchHeaderAbsent hContentLength
		//                   ]
		//               }
		// @@ the reset only prepares the Location of the next request: skipped with it
		{
			Description:     "into a table with auto-incrementing pk: reset the sequence",
			Method:          "POST",
			Query:           "/rpc/reset_sequence",
			Body:            `{"name": "auto_incrementing_pk_id_seq", "value": 2}`,
			Headers:         test.Headers{"Prefer": {"tx=commit"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": ""},
			Status:          204,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//           request
		//             methodPost
		//             "/auto_incrementing_pk"
		//             [("Prefer", "return=headers-only")]
		//             [json| { "non_nullable_string":"not null"} |]
		//             `shouldRespondWith` ""
		//               { matchStatus = 201
		//               , matchHeaders =
		//                   [ matchHeaderAbsent hContentType
		//                   , "Content-Length" <:> "0"
		//                   , "Location" <:> "/auto_incrementing_pk?id=eq.2"
		//                   , "Preference-Applied" <:> "return=headers-only"
		//                   ]
		//               }
		{
			Description:     "into a table with auto-incrementing pk succeeds with 201 and location header",
			Method:          "POST",
			Query:           "/auto_incrementing_pk",
			Body:            `{ "non_nullable_string":"not null"}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "0", "Location": "/auto_incrementing_pk?id=eq.2", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//       context "into a table with simple pk" $
		//         it "fails with 400 and error" $
		//           post "/simple_pk" [json| { "extra":"foo"} |]
		//             `shouldRespondWith` [json|{"hint":null,"details":"Failing row contains (null, foo).","code":"23502","message":"null value in column \"k\" of relation \"simple_pk\" violates not-null constraint"}|]
		//               { matchStatus = 400
		//               , matchHeaders = [matchContentTypeJson]
		//               }
		// @@ not asserted, PostgREST 16.4: the error body shape (null hint, no subsystem/position); card 32494 (error body)
		{
			Description:     "into a table with simple pk fails with 400 and error",
			Method:          "POST",
			Query:           "/simple_pk",
			Body:            `{ "extra":"foo"}`,
			Expected:        `{"subsystem":"database","message":"null value in column \"k\" of relation \"simple_pk\" violates not-null constraint","code":"23502","hint":"","details":"Failing row contains (null, foo).","position":0}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//       context "into a table with no pk" $ do
		//         it "succeeds with 201 but no location header" $ do
		//           post
		//             "/no_pk"
		//             [json| { "a":"foo", "b":"bar" } |]
		//             `shouldRespondWith` ""
		//               { matchStatus = 201
		//               , matchHeaders =
		//                   [ matchHeaderAbsent hContentType
		//                   , matchHeaderAbsent hLocation
		//                   ]
		//               }
		{
			Description:     "succeeds with 201 but no location header",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            `{ "a":"foo", "b":"bar" }`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": ""},
			Status:          201,
		},
		//         it "returns full details of inserted record if asked" $ do
		//           request
		//             methodPost
		//             "/no_pk"
		//             [("Prefer", "return=representation")]
		//             [json| { "a":"bar", "b":"baz" } |]
		//             `shouldRespondWith` [json| [{ "a":"bar", "b":"baz" }] |]
		//               { matchStatus = 201
		//               , matchHeaders =
		//                   [ matchHeaderAbsent hLocation
		//                   , "Preference-Applied" <:> "return=representation"
		//                   ]
		//               }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "returns full details of inserted record if asked",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            `{ "a":"bar", "b":"baz" }`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{ "a":"bar", "b":"baz" }]`,
			ExpectedHeaders: map[string]string{"Location": ""},
			Status:          201,
		},
		//         it "returns empty array when no items inserted, and return=rep" $ do
		//           request
		//             methodPost
		//             "/no_pk"
		//             [("Prefer", "return=representation")]
		//             [json| [] |]
		//             `shouldRespondWith` [json| [] |]
		//               { matchStatus = 201
		//               }
		{
			Description: "returns empty array when no items inserted, and return=rep",
			Method:      "POST",
			Query:       "/no_pk",
			Body:        `[]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      201,
		},
		//         it "can post nulls" $ do
		//           request
		//             methodPost
		//             "/no_pk"
		//             [("Prefer", "return=representation")]
		//             [json| { "a":null, "b":"foo" } |]
		//             `shouldRespondWith` [json| [{ "a":null, "b":"foo" }] |]
		//               { matchStatus = 201
		//               , matchHeaders = [matchHeaderAbsent hLocation]
		//               }
		{
			Description:     "can post nulls",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            `{ "a":null, "b":"foo" }`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{ "a":null, "b":"foo" }]`,
			ExpectedHeaders: map[string]string{"Location": ""},
			Status:          201,
		},
		//     context "with compound pk supplied" $
		//       it "builds response location header appropriately" $ do
		//         request
		//           methodPost
		//           "/compound_pk"
		//           [("Prefer", "return=representation")]
		//           [json| { "k1":12, "k2":"Rock & R+ll" } |]
		//           `shouldRespondWith` [json|[ { "k1":12, "k2":"Rock & R+ll", "extra": null } ]|]
		//             { matchStatus = 201
		//             , matchHeaders = [matchHeaderAbsent hLocation]
		//             }
		{
			Description:     "with compound pk supplied builds response location header appropriately",
			Method:          "POST",
			Query:           "/compound_pk",
			Body:            `{ "k1":12, "k2":"Rock & R+ll" }`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[ { "k1":12, "k2":"Rock & R+ll", "extra": null } ]`,
			ExpectedHeaders: map[string]string{"Location": ""},
			Status:          201,
		},
		//     context "with bulk insert" $
		//       it "returns 201 but no location header" $ do
		//         let bulkData =
		//               [json| [ {"k1":21, "k2":"hello world"}
		//                               , {"k1":22, "k2":"bye for now"}]
		//                             |]
		//         request
		//           methodPost
		//           "/compound_pk"
		//           []
		//           bulkData
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hLocation
		//                 ]
		//             }
		{
			Description:     "with bulk insert returns 201 but no location header",
			Method:          "POST",
			Query:           "/compound_pk",
			Body:            `[ {"k1":21, "k2":"hello world"}, {"k1":22, "k2":"bye for now"}]`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": ""},
			Status:          201,
		},
		//     context "with invalid json payload" $
		//       it "fails with 400 and error" $
		//         post "/simple_pk" "}{ x = 2"
		//           `shouldRespondWith` [json|{"message":"Empty or invalid json","code":"PGRST102","details":null,"hint":null}|]
		//             { matchStatus = 400
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , "Content-Length" <:> "80"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST102 error body ("Empty or invalid json"); card 32494 (error body)
		{
			Description:     "with invalid json payload fails with 400 and error",
			Method:          "POST",
			Query:           "/simple_pk",
			Body:            `}{ x = 2`,
			Headers:         nil,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//     context "with no payload" $
		//       it "fails with 400 and error" $
		//         post "/simple_pk" ""
		//           `shouldRespondWith` [json|{"message":"Empty or invalid json","code":"PGRST102","details":null,"hint":null}|]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST102 error body ("Empty or invalid json"); card 32494 (error body)
		{
			Description:     "with no payload fails with 400 and error",
			Method:          "POST",
			Query:           "/simple_pk",
			Body:            ``,
			Headers:         nil,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//     context "with valid json payload" $
		//       it "succeeds and returns 201 created" $
		//         post
		//           "/simple_pk"
		//           [json| { "k":"k1", "extra":"e1" } |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description:     "with valid json payload succeeds and returns 201 created",
			Method:          "POST",
			Query:           "/simple_pk",
			Body:            `{ "k":"k1", "extra":"e1" }`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          201,
		},
		//     context "attempting to insert a row with the same primary key" $
		//       it "fails returning a 409 Conflict" $
		//         post
		//           "/simple_pk"
		//           [json| { "k":"xyyx", "extra":"e1" } |]
		//           `shouldRespondWith` [json|{"hint":null,"details":"Key (k)=(xyyx) already exists.","code":"23505","message":"duplicate key value violates unique constraint \"simple_pk_pkey\""}|]
		//             { matchStatus = 409
		//             }
		// @@ not asserted, PostgREST 16.4: the error body shape (null hint, no subsystem/position); card 32494 (error body)
		{
			Description: "attempting to insert a row with the same primary key",
			Method:      "POST",
			Query:       "/simple_pk",
			Body:        `{ "k":"xyyx", "extra":"e1" }`,
			Headers:     nil,
			Expected:    `{"subsystem":"database","message":"duplicate key value violates unique constraint \"simple_pk_pkey\"","code":"23505","hint":"","details":"Key (k)=(xyyx) already exists.","position":0}`,
			Status:      409,
		},
		//     context "attempting to insert a row with conflicting unique constraint" $
		//       it "fails returning a 409 Conflict" $
		//         post "/withUnique" [json| { "uni":"nodup", "extra":"e2" } |] `shouldRespondWith` 409
		{
			Description: "attempting to insert a row with conflicting unique constraint",
			Method:      "POST",
			Query:       "/withUnique",
			Body:        `{ "uni":"nodup", "extra":"e2" }`,
			Headers:     nil,
			Expected:    ``,
			Status:      409,
		},
		//     context "jsonb" $ do
		//       it "serializes nested object" $ do
		//         let inserted = [json| { "data": { "foo":"bar" } } |]
		//         request
		//           methodPost
		//           "/json_table"
		//           [("Prefer", "return=representation")]
		//           inserted
		//           `shouldRespondWith` [json|[{"data":{"foo":"bar"}}]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "jsonb serializes nested object",
			Method:      "POST",
			Query:       "/json_table",
			Body:        `{ "data": { "foo":"bar" } }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"data":{"foo":"bar"}}]`,
			Status:      201,
		},
		//       it "serializes nested array" $ do
		//         let inserted = [json| { "data": [1,2,3] } |]
		//         request
		//           methodPost
		//           "/json_table"
		//           [("Prefer", "return=representation")]
		//           inserted
		//           `shouldRespondWith` [json|[{"data":[1,2,3]}]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "jsonb serializes nested array",
			Method:      "POST",
			Query:       "/json_table",
			Body:        `{ "data": [1,2,3] }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"data":[1,2,3]}]`,
			Status:      201,
		},
		//     context "empty objects" $ do
		//       it "successfully inserts a row with all-default columns" $ do
		//         post
		//           "/items"
		//           [json|{}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description:     "empty objects successfully inserts a row with all-default columns",
			Method:          "POST",
			Query:           "/items",
			Body:            `{}`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          201,
		},
		//         post "/items" "[{}]"
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders = []
		//             }
		{
			Description:   "empty objects successfully inserts a row with all-default columns",
			Method:        "POST",
			Query:         "/items",
			Body:          `[{}]`,
			Headers:       nil,
			ExpectedEmpty: true,
			Status:        201,
		},
		//       it "successfully inserts two rows with all-default columns" $
		//         post
		//           "/items"
		//           [json|[{}, {}]|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description:     "successfully inserts two rows with all-default columns",
			Method:          "POST",
			Query:           "/items",
			Body:            `[{}, {}]`,
			Headers:         nil,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          201,
		},
		//       it "successfully inserts a row with all-default columns with prefer=rep" $ do
		//         -- reset pk sequence first to make test repeatable
		//         request
		//           methodPost
		//           "/rpc/reset_sequence"
		//           [("Prefer", "tx=commit")]
		//           [json|{"name": "items2_id_seq", "value": 20}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 204
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hContentLength
		//                 ]
		//             }
		// @@ the committed reset touches only items2_id_seq: the other suites read
		// items2 by id=eq.1 and never insert into it
		// @@ not asserted, PostgREST 16.4: a function RETURNS void answers 204 with no body
		// (smoothdb answers 200 ""); card 32490 (rpc details). Run for its effect: the reset.
		{
			Description: "all-default columns with prefer=rep: reset the sequence",
			Method:      "POST",
			Query:       "/rpc/reset_sequence",
			Body:        `{"name": "items2_id_seq", "value": 20}`,
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
		},
		//         request
		//           methodPost
		//           "/items2"
		//           [("Prefer", "return=representation")]
		//           [json|{}|]
		//           `shouldRespondWith` [json|[{ id: 20 }]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "successfully inserts a row with all-default columns with prefer=rep",
			Method:      "POST",
			Query:       "/items2",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 20 }]`,
			Status:      201,
		},
		//       it "successfully inserts a row with all-default columns with prefer=rep and &select=" $ do
		//         -- reset pk sequence first to make test repeatable
		//         request
		//           methodPost
		//           "/rpc/reset_sequence"
		//           [("Prefer", "tx=commit")]
		//           [json|{"name": "items3_id_seq", "value": 20}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 204
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hContentLength
		//                 ]
		//             }
		// @@ the committed reset touches only items3_id_seq, used by no other case
		// @@ not asserted, PostgREST 16.4: a function RETURNS void answers 204 with no body
		// (smoothdb answers 200 ""); card 32490 (rpc details). Run for its effect: the reset.
		{
			Description: "all-default columns with prefer=rep and &select=: reset the sequence",
			Method:      "POST",
			Query:       "/rpc/reset_sequence",
			Body:        `{"name": "items3_id_seq", "value": 20}`,
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
		},
		//         request
		//           methodPost
		//           "/items3?select=id"
		//           [("Prefer", "return=representation")]
		//           [json|{}|]
		//           `shouldRespondWith` [json|[{ id: 20 }]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "successfully inserts a row with all-default columns with prefer=rep and &select=",
			Method:      "POST",
			Query:       "/items3?select=id",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 20 }]`,
			Status:      201,
		},
		//     context "insignificant whitespace" $ do
		//       it "ignores it and successfuly inserts with json payload" $ do
		//         request
		//           methodPost
		//           "/json_table"
		//           [("Prefer", "return=representation")]
		//           "\t \n \r { \"data\": { \"foo\":\"bar\" } }\t \n \r "
		//           `shouldRespondWith` [json|[{"data":{"foo":"bar"}}]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "insignificant whitespace is ignored around a json object",
			Method:      "POST",
			Query:       "/json_table",
			Body:        "\t \n \r { \"data\": { \"foo\":\"bar\" } }\t \n \r ",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"data":{"foo":"bar"}}]`,
			Status:      201,
		},
		//         request
		//           methodPost
		//           "/json_table"
		//           [("Prefer", "return=representation")]
		//           "\t \n \r [{ \"data\": { \"foo\":\"bar\" } }, \t \n \r {\"data\": 34}]\t \n \r "
		//           `shouldRespondWith` [json|[{"data":{"foo":"bar"}}, {"data":34}]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "insignificant whitespace is ignored around a json array",
			Method:      "POST",
			Query:       "/json_table",
			Body:        "\t \n \r [{ \"data\": { \"foo\":\"bar\" } }, \t \n \r {\"data\": 34}]\t \n \r ",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"data":{"foo":"bar"}}, {"data":34}]`,
			Status:      201,
		},
		//     -- https://github.com/PostgREST/postgrest/issues/2861
		//     context "bit and char columns with length" $ do
		//       it "should insert to a bit column with length" $
		//         request
		//           methodPost
		//           "/bitchar_with_length?select=bit"
		//           [("Prefer", "return=representation")]
		//           [json|{"bit": "10101"}|]
		//           `shouldRespondWith` [json|[{ "bit": "10101" }]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "bit and char columns with length: should insert to a bit column with length",
			Method:      "POST",
			Query:       "/bitchar_with_length?select=bit",
			Body:        `{"bit": "10101"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "bit": "10101" }]`,
			Status:      201,
		},
		//       it "should insert to a char column with length" $
		//         request
		//           methodPost
		//           "/bitchar_with_length?select=char"
		//           [("Prefer", "return=representation")]
		//           [json|{"char": "abcde"}|]
		//           `shouldRespondWith` [json|[{ "char": "abcde" }]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "bit and char columns with length: should insert to a char column with length",
			Method:      "POST",
			Query:       "/bitchar_with_length?select=char",
			Body:        `{"char": "abcde"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "char": "abcde" }]`,
			Status:      201,
		},
		//     context "POST with ?columns parameter" $ do
		//       it "ignores json keys not included in ?columns" $ do
		//         request
		//           methodPost
		//           "/articles?columns=id,body"
		//           [("Prefer", "return=representation")]
		//           [json| {"id": 200, "body": "xxx", "smth": "here", "other": "stuff", "fake_id": 13} |]
		//           `shouldRespondWith` [json|[{"id": 200, "body": "xxx", "owner": "postgrest_test_anonymous"}]|]
		//             { matchStatus = 201
		//             , matchHeaders = []
		//             }
		{
			Description: "ignores json keys not included in ?columns",
			Method:      "POST",
			Query:       "/articles?columns=id,body",
			Body:        `{"id": 200, "body": "xxx", "smth": "here", "other": "stuff", "fake_id": 13}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id": 200, "body": "xxx", "owner": "postgrest_test_anonymous"}]`,
			Status:      201,
		},
		// @@added - with quotes
		{
			Description: "ignores json keys not included in ?columns",
			Method:      "POST",
			Query:       "/articles?columns=\"id\",\"body\"",
			Body:        `{"id": 200, "body": "xxx", "smth": "here", "other": "stuff", "fake_id": 13}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id": 200, "body": "xxx", "owner": "postgrest_test_anonymous"}]`,
			Status:      201,
		},
		//         request
		//           methodPost
		//           "/articles?columns=id,body&select=id,body"
		//           [("Prefer", "return=representation")]
		//           [json| [
		//             {"id": 201, "body": "yyy", "smth": "here", "other": "stuff", "fake_id": 13},
		//             {"id": 202, "body": "zzz", "garbage": "%%$&", "kkk": "jjj"},
		//             {"id": 203, "body": "aaa", "hey": "ho"} ]|]
		//           `shouldRespondWith` [json|[
		//             {"id": 201, "body": "yyy"},
		//             {"id": 202, "body": "zzz"},
		//             {"id": 203, "body": "aaa"} ]|]
		//             { matchStatus = 201
		//             , matchHeaders = []
		//             }
		{
			Description: "ignores json keys not included in ?columns",
			Method:      "POST",
			Query:       "/articles?columns=id,body&select=id,body",
			Body: `[
				 		{"id": 201, "body": "yyy", "smth": "here", "other": "stuff", "fake_id": 13},
				 		{"id": 202, "body": "zzz", "garbage": "%%$&", "kkk": "jjj"},
				 		{"id": 203, "body": "aaa", "hey": "ho"} ]`,
			Headers: test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				 			{"id": 201, "body": "yyy"},
				 			{"id": 202, "body": "zzz"},
				 			{"id": 203, "body": "aaa"} ]`,
			Status: 201,
		},
		//       -- TODO parse columns error message needs to be improved
		//       it "disallows blank ?columns" $
		//         post
		//           "/articles?columns="
		//           [json|[
		//             {"id": 204, "body": "yyy"},
		//             {"id": 205, "body": "zzz"}]|]
		//           `shouldRespondWith` [json|  {"details":"unexpected end of input expecting field name (* or [a..z0..9_$])","message":"\"failed to parse columns parameter ()\" (line 1, column 1)","code":"PGRST100","hint":null} |]
		//             { matchStatus = 400
		//             , matchHeaders = []
		//             }
		{
			Description: "disallows blank ?columns",
			Method:      "POST",
			Query:       "/articles?columns=",
			Body:        `[{"id": 204, "body": "yyy"}, {"id": 205, "body": "zzz"}]`,
			Expected:    `{"details":"unexpected end of input expecting field name (* or [a..z0..9_$])","message":"\"failed to parse columns parameter ()\" (line 1, column 1)","code":"PGRST100","hint":null}`,
			Status:      400,
			Skip:        "PostgREST 16.4: a blank ?columns is refused with 400 (PGRST100); no card",
		},
		//       it "disallows ?columns which don't exist" $
		//         post
		//           "/articles?columns=helicopter"
		//           [json|[
		//             {"id": 204, "body": "yyy"},
		//             {"id": 205, "body": "zzz"}]|]
		//           `shouldRespondWith` [json|{"code":"PGRST204","details":null,"hint":null,"message":"Could not find the 'helicopter' column of 'articles' in the schema cache"} |]
		//             { matchStatus = 400
		//             , matchHeaders = []
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST204 error body (code, null hint); card 32494 (error body).
		// The message is PostgREST's; the body asserted is smoothdb's shape.
		{
			Description: "disallows ?columns which don't exist",
			Method:      "POST",
			Query:       "/articles?columns=helicopter",
			Body:        `[{"id": 204, "body": "yyy"}, {"id": 205, "body": "zzz"}]`,
			Expected:    `{"subsystem":"network","message":"Could not find the 'helicopter' column of 'articles' in the schema cache","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
		//       it "returns missing table error even if also has invalid ?columns" $
		//         post
		//           "/garlic?columns=helicopter"
		//           [json|[
		//             {"id": 204, "body": "yyy"},
		//             {"id": 205, "body": "zzz"}]|]
		//           `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.garlic' in the schema cache"} |]
		//             { matchStatus = 404
		//             , matchHeaders = []
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST205 error body; card 32494 (error body)
		{
			Description: "returns missing table error even if also has invalid ?columns",
			Method:      "POST",
			Query:       "/garlic?columns=helicopter",
			Body:        `[{"id": 204, "body": "yyy"}, {"id": 205, "body": "zzz"}]`,
			Status:      404,
		},
		// @@ added: an unknown embed in the select of the representation is
		// refused as on a GET, instead of building a broken statement
		{
			Description: "errs on an unknown embed in the select of the representation",
			Method:      "POST",
			Query:       "/articles?select=id,nosuch(*)",
			Body:        `{"id": 204, "body": "yyy"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `{"subsystem":"network","message":"cannot find relationship for table articles with table nosuch","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
		// @@ added: a filter on a relation that is not embedded is refused on a
		// mutation too (PostgREST addFilters applies the non-root filters to the
		// read plan of the representation)
		{
			Description: "errs on a filter for a relation not embedded in the representation",
			Method:      "POST",
			Query:       "/articles?select=id&nosuch.id=eq.1",
			Body:        `{"id": 204, "body": "yyy"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `{"subsystem":"network","message":"'nosuch' is not an embedded resource in this request","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
		//       it "disallows array elements that are not json objects" $
		//         post
		//           "/articles?columns=id,body"
		//           [json|[
		//             {"id": 204, "body": "yyy"},
		//             333,
		//             "asdf",
		//             {"id": 205, "body": "zzz"}]|]
		//           `shouldRespondWith` 400
		{
			Description: "disallows array elements that are not json objects",
			Method:      "POST",
			Query:       "/articles?columns=id,body",
			Body:        `[{"id": 204, "body": "yyy"}, 333, "asdf", {"id": 205, "body": "zzz"}]`,
			Status:      400,
		},
		//       context "apply defaults on missing values" $ do
		//         it "inserts table default values(field-with_sep) when json keys are undefined" $
		//           request
		//             methodPost
		//             "/complex_items?columns=id,name,field-with_sep,arr_data"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json|[
		//                 {"id": 4, "name": "Vier"},
		//                 {"id": 5, "name": "Funf", "arr_data": null},
		//                 {"id": 6, "name": "Sechs", "field-with_sep": 6, "arr_data": "{1,2,3}"}
		//               ]|]
		//             `shouldRespondWith` [json|[
		//                 {"id": 4, "name": "Vier", "field-with_sep": 1, "settings":null,"arr_data":null},
		//                 {"id": 5, "name": "Funf", "field-with_sep": 1, "settings":null,"arr_data":null},
		//                 {"id": 6, "name": "Sechs", "field-with_sep": 6, "settings":null,"arr_data":[1,2,3]}
		//               ]|]
		//               { matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description: "inserts table default values(field-with_sep) when json keys are undefined",
			Method:      "POST",
			Query:       "/complex_items?columns=id,name,field-with_sep,arr_data",
			Body: `[
				{"id": 4, "name": "Vier"},
				{"id": 5, "name": "Funf", "arr_data": null},
				{"id": 6, "name": "Sechs", "field-with_sep": 6, "arr_data": "{1,2,3}"}
			]`,
			Headers: test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected: `[
				{"id": 4, "name": "Vier", "field-with_sep": 1, "settings":null,"arr_data":null},
				{"id": 5, "name": "Funf", "field-with_sep": 1, "settings":null,"arr_data":null},
				{"id": 6, "name": "Sechs", "field-with_sep": 6, "settings":null,"arr_data":[1,2,3]}
			]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
		},
		//         it "inserts view default values(field-with_sep) when json keys are undefined" $
		//           request
		//             methodPost
		//             "/complex_items_view?columns=id,name"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json|[
		//                 {"id": 7, "name": "Sieben"},
		//                 {"id": 8}
		//               ]|]
		//             `shouldRespondWith` [json|[
		//                 {"id": 7, "name": "Sieben", "field-with_sep": 1, "settings":null,"arr_data":null},
		//                 {"id": 8, "name": "Default", "field-with_sep": 1, "settings":null,"arr_data":null}
		//               ]|]
		//               { matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description: "inserts view default values(field-with_sep) when json keys are undefined",
			Method:      "POST",
			Query:       "/complex_items_view?columns=id,name",
			Body: `[
				{"id": 7, "name": "Sieben"},
				{"id": 8}
			]`,
			Headers: test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected: `[
				{"id": 7, "name": "Sieben", "field-with_sep": 1, "settings":null,"arr_data":null},
				{"id": 8, "name": "Default", "field-with_sep": 1, "settings":null,"arr_data":null}
			]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
		},
		//         it "doesn't insert json duplicate keys(since it uses jsonb)" $
		//           request
		//             methodPost
		//             "/tbl_w_json?columns=id,data"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json| { "data": { "a": 1, "a": 2 }, "id": 3 } |]
		//             `shouldRespondWith` [json| [ { "data": { "a": 2 }, "id": 3 } ] |]
		//               { matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description:     "doesn't insert json duplicate keys(since it uses jsonb)",
			Method:          "POST",
			Query:           "/tbl_w_json?columns=id,data",
			Body:            `{ "data": { "a": 1, "a": 2 }, "id": 3 }`,
			Headers:         test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected:        `[ { "data": { "a": 2 }, "id": 3 } ]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default reads the payload through jsonb, so duplicate keys collapse; card 32482 (prefer options)",
		},
		//         it "inserts a default on a generated by default as identity column" $
		//           request
		//             methodPost
		//             "/channels?columns=id,data,slug&select=data,slug"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json| { "slug": "foo" } |]
		//             `shouldRespondWith` [json| [{"data":{"foo": "bar"},"slug":"foo"}] |]
		//               { -- id 1 was inserted here, we don't get it for idempotence in the tests
		//                 matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description:     "inserts a default on a generated by default as identity column",
			Method:          "POST",
			Query:           "/channels?columns=id,data,slug&select=data,slug",
			Body:            `{ "slug": "foo" }`,
			Headers:         test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected:        `[{"data":{"foo": "bar"},"slug":"foo"}]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
		},
		//         it "fails with a good error message on generated always columns" $
		//           request
		//             methodPost
		//             "/foo?columns=a,b"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json| [
		//                 {"a": "val"},
		//                 {"a": "val", "b": "val"}
		//               ]|]
		//             `shouldRespondWith` [json| {
		//                 "code": "428C9",
		//                 "details": "Column \"b\" is a generated column.",
		//                 "hint": null,
		//                 "message": "cannot insert a non-DEFAULT value into column \"b\""
		//               }|]
		//               { matchStatus = 400
		//               }
		// @@ missing=default is not implemented: with ?columns= a key absent from an
		// object is inserted as NULL, which the generated column refuses all the same
		// @@ not asserted, PostgREST 16.4: the error body shape (null hint, no subsystem/position); card 32494 (error body)
		{
			Description: "fails with a good error message on generated always columns",
			Method:      "POST",
			Query:       "/foo?columns=a,b",
			Body: `[
				{"a": "val"},
				{"a": "val", "b": "val"}
			]`,
			Headers:  test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected: `{"subsystem":"database","message":"cannot insert a non-DEFAULT value into column \"b\"","code":"428C9","hint":"","details":"Column \"b\" is a generated column.","position":0}`,
			Status:   400,
		},
		//         it "inserts a default on a DOMAIN with default" $
		//           request
		//             methodPost
		//             "/evil_friends?columns=id,name"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json| { "name": "Lu" } |]
		//             `shouldRespondWith` [json| [{"id": 666, "name": "Lu"}] |]
		//               { matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description:     "inserts a default on a DOMAIN with default",
			Method:          "POST",
			Query:           "/evil_friends?columns=id,name",
			Body:            `{ "name": "Lu" }`,
			Headers:         test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected:        `[{"id": 666, "name": "Lu"}]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
		},
		//         it "inserts a COLUMN default before a DOMAIN default with missing=default" $
		//           request
		//             methodPost
		//             "/evil_friends_with_column_default?columns=id,name"
		//             [("Prefer", "return=representation"), ("Prefer", "missing=default")]
		//             [json| { "name": "Demon" } |]
		//             `shouldRespondWith` [json| [{"id": 420, "name": "Demon"}] |]
		//               { matchStatus = 201
		//               , matchHeaders = ["Preference-Applied" <:> "missing=default, return=representation"]
		//               }
		{
			Description:     "inserts a COLUMN default before a DOMAIN default with missing=default",
			Method:          "POST",
			Query:           "/evil_friends_with_column_default?columns=id,name",
			Body:            `{ "name": "Demon" }`,
			Headers:         test.Headers{"Prefer": {"return=representation", "missing=default"}},
			Expected:        `[{"id": 420, "name": "Demon"}]`,
			ExpectedHeaders: map[string]string{"Preference-Applied": "missing=default, return=representation"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default inserts the column default for an absent key; card 32482 (prefer options)",
		},
		//     it "inserts json that has duplicate keys" $ do
		//       request
		//         methodPost
		//         "/tbl_w_json"
		//         [("Prefer", "return=representation")]
		//         [json| { "data": { "a": 1, "a": 2 }, "id": 3 } |]
		//         `shouldRespondWith` [json| [ { "data": { "a": 1, "a": 2 }, "id": 3 } ] |]
		//           { matchStatus = 201
		//           }
		{
			Description: "inserts json that has duplicate keys",
			Method:      "POST",
			Query:       "/tbl_w_json",
			Body:        `{ "data": { "a": 1, "a": 2 }, "id": 3 }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "data": { "a": 1, "a": 2 }, "id": 3 } ]`,
			Status:      201,
		},
		//       request
		//         methodPost
		//         "/tbl_w_json?columns=id,data"
		//         [("Prefer", "return=representation")]
		//         [json| { "data": { "a": 1, "a": 2 }, "id": 3 } |]
		//         `shouldRespondWith` [json| [ { "data": { "a": 1, "a": 2 }, "id": 3 } ] |]
		//           { matchStatus = 201
		//           }
		{
			Description: "inserts json that has duplicate keys with ?columns",
			Method:      "POST",
			Query:       "/tbl_w_json?columns=id,data",
			Body:        `{ "data": { "a": 1, "a": 2 }, "id": 3 }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "data": { "a": 1, "a": 2 }, "id": 3 } ]`,
			Status:      201,
		},
		//     context "with unicode values" $ do
		//       it "succeeds and returns full representation" $
		//         request
		//           methodPost
		//           "/simple_pk2?select=extra,k"
		//           [("Prefer", "return=representation")]
		//           [json| { "k":"棋圍", "extra":"￥" } |]
		//           `shouldRespondWith` [json|[ { "k":"棋圍", "extra":"￥" } ]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "with unicode values it succeeds and returns full representation",
			Method:      "POST",
			Query:       "/simple_pk2?select=extra,k",
			Body:        `{ "k":"棋圍", "extra":"￥" }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "k":"棋圍", "extra":"￥" } ]`,
			Status:      201,
		},
		//       it "succeeds and returns usable location header" $ do
		//         p <-
		//           request
		//             methodPost
		//             "/simple_pk2?select=extra,k"
		//             [("Prefer", "tx=commit"), ("Prefer", "return=headers-only")]
		//             [json| { "k":"圍棋", "extra":"￥" } |]
		//         pure p
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             }
		//
		//         Just location <- pure $ lookup hLocation $ simpleHeaders p
		//         get location
		//           `shouldRespondWith` [json|[ { "k":"圍棋", "extra":"￥" } ]|]
		//
		//         request
		//           methodDelete
		//           location
		//           [("Prefer", "tx=commit")]
		//           ""
		//           `shouldRespondWith` ""
		//             { matchStatus = 204
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hContentLength
		//                 ]
		//             }
		// @@ the GET and the DELETE address the Location of the first answer
		// (percent-encoded by PostgREST), so all three are skipped together
		{
			Description:     "with unicode values it succeeds and returns usable location header",
			Method:          "POST",
			Query:           "/simple_pk2?select=extra,k",
			Body:            `{ "k":"圍棋", "extra":"￥" }`,
			Headers:         test.Headers{"Prefer": {"tx=commit", "return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Location": "/simple_pk2?k=eq.%E5%9C%8D%E6%A3%8B"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		{
			Description: "with unicode values the location header is usable by a GET",
			Query:       "/simple_pk2?k=eq.%E5%9C%8D%E6%A3%8B",
			Expected:    `[ { "k":"圍棋", "extra":"￥" } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		{
			Description:     "with unicode values the location header is usable by a DELETE",
			Method:          "DELETE",
			Query:           "/simple_pk2?k=eq.%E5%9C%8D%E6%A3%8B",
			Headers:         test.Headers{"Prefer": {"tx=commit"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": ""},
			Status:          204,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//   describe "CSV insert" $ do
		//     context "disparate csv types" $
		//       it "succeeds with multipart response" $ do
		//         pendingWith "Decide on what to do with CSV insert"
		//         let inserted =
		//               [str|integer,double,varchar,boolean,date,money,enum
		//             |13,3.14159,testing!,false,1900-01-01,$3.99,foo
		//             |12,0.1,a string,true,1929-10-01,12,bar
		//             |]
		//         request methodPost "/menagerie" [("Content-Type", "text/csv"), ("Accept", "text/csv"), ("Prefer", "return=representation")] inserted
		//           `shouldRespondWith` ResponseMatcher
		//             { matchStatus = 201
		//             , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//             , matchBody = bodyEquals inserted
		//             }
		// @@ pending upstream (pendingWith): this is smoothdb's own variant, with the
		// money values left empty so that the round trip is deterministic
		{
			Description:     "succeeds with multipart response",
			Method:          "POST",
			Query:           "/menagerie",
			Body:            "integer,double,varchar,boolean,date,money,enum\n13,3.14159,testing!,false,1900-01-01,,foo\n12,0.1,a string,true,1929-10-01,,bar",
			Headers:         test.Headers{"Accept": {"text/csv"}, "Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:        "integer,double,varchar,boolean,date,money,enum\n13,3.14159,testing!,false,1900-01-01,$0.00,foo\n12,0.1,a string,true,1929-10-01,$0.00,bar",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          201,
		},
		//     context "requesting full representation" $ do
		//       it "returns full details of inserted record" $
		//         request
		//           methodPost
		//           "/no_pk"
		//           [("Content-Type", "text/csv"), ("Accept", "text/csv"), ("Prefer", "return=representation")]
		//           "a,b\nbar,baz"
		//           `shouldRespondWith` "a,b\nbar,baz"
		//             { matchStatus = 201
		//             , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//             }
		{
			Description:     "returns full details of inserted record",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            "a,b\nbar,baz",
			Headers:         test.Headers{"Accept": {"text/csv"}, "Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:        "a,b\nbar,baz",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          201,
		},
		//       it "can post nulls" $
		//         request
		//           methodPost
		//           "/no_pk"
		//           [("Content-Type", "text/csv"), ("Accept", "text/csv"), ("Prefer", "return=representation")]
		//           "a,b\nNULL,foo"
		//           `shouldRespondWith` "a,b\n,foo"
		//             { matchStatus = 201
		//             , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//             }
		{
			Description:     "can post nulls",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            "a,b\nNULL,foo",
			Headers:         test.Headers{"Accept": {"text/csv"}, "Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:        "a,b\n,foo",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          201,
		},
		//       it "only returns the requested column header with its associated data" $
		//         request
		//           methodPost
		//           "/projects?select=id"
		//           [("Content-Type", "text/csv"), ("Accept", "text/csv"), ("Prefer", "return=representation")]
		//           "id,name,client_id\n8,Xenix,1\n9,Windows NT,1"
		//           `shouldRespondWith` "id\n8\n9"
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "text/csv; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "only returns the requested column header with its associated data",
			Method:          "POST",
			Query:           "/projects?select=id",
			Body:            "id,name,client_id\n8,Xenix,1\n9,Windows NT,1",
			Headers:         test.Headers{"Accept": {"text/csv"}, "Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:        "id\n8\n9",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
		},
		//     context "with wrong number of columns" $
		//       it "fails for too few" $
		//         request methodPost "/no_pk" [("Content-Type", "text/csv")] "a,b\nfoo,bar\nbaz"
		//           `shouldRespondWith` [json|{"message":"All lines must have same number of fields","code":"PGRST102","details":null,"hint":null}|]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST102 error body; card 32494 (error body)
		{
			Description:     "with wrong number of columns fails for too few",
			Method:          "POST",
			Query:           "/no_pk",
			Body:            "a,b\nfoo,bar\nbaz",
			Headers:         test.Headers{"Content-Type": {"text/csv"}},
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//   describe "Row level permission" $
		//     it "set user_id when inserting rows" $ do
		//       request
		//         methodPost
		//         "/authors_only"
		//         [authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0", ("Prefer", "return=representation")]
		//         [json| { "secret": "nyancat" } |]
		//         `shouldRespondWith` [json|[{"owner":"jdoe","secret":"nyancat"}]|]
		//           { matchStatus = 201
		//           }
		{
			Description: "set user_id when inserting rows",
			Method:      "POST",
			Query:       "/authors_only",
			Body:        `{ "secret": "nyancat" }`,
			Headers: test.Headers{"Prefer": {"return=representation"},
				"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"}},
			Expected: `[{"owner":"jdoe","secret":"nyancat"}]`,
			Status:   201,
		},
		//       request
		//         methodPost
		//         "/authors_only"
		//         -- jwt token for jroe
		//         [authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqcm9lIn0.2e7mx0U4uDcInlbJVOBGlrRufwqWLINDIEDC1vS0nw8", ("Prefer", "return=representation")]
		//         [json| { "secret": "lolcat", "owner": "hacker" } |]
		//         `shouldRespondWith` [json|[{"owner":"jroe","secret":"lolcat"}]|]
		//           { matchStatus = 201
		//           }
		{
			Description: "set user_id when inserting rows",
			Method:      "POST",
			Query:       "/authors_only",
			Body:        `{ "secret": "lolcat", "owner": "hacker" }`,
			Headers: test.Headers{"Prefer": {"return=representation"},
				"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqcm9lIn0.2e7mx0U4uDcInlbJVOBGlrRufwqWLINDIEDC1vS0nw8"}},
			Expected: `[{"owner":"jroe","secret":"lolcat"}]`,
			Status:   201,
		},
		//   context "tables with self reference foreign keys" $ do
		//     it "embeds parent after insert" $
		//       request
		//         methodPost
		//         "/web_content?select=id,name,parent_content:p_web_id(name)"
		//         [("Prefer", "return=representation")]
		//         [json|{"id":6, "name":"wot", "p_web_id":4}|]
		//         `shouldRespondWith` [json|[{"id":6,"name":"wot","parent_content":{"name":"wut"}}]|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchContentTypeJson, matchHeaderAbsent hLocation]
		//           }
		{
			Description:     "tables with self reference foreign keys embeds parent after insert",
			Method:          "POST",
			Query:           "/web_content?select=id,name,parent_content:p_web_id(name)",
			Body:            `{"id":6, "name":"wot", "p_web_id":4}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":6,"name":"wot","parent_content":{"name":"wut"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Location": ""},
			Status:          201,
		},
		//   context "table with limited privileges" $ do
		//     it "succeeds inserting if correct select is applied" $
		//       request
		//         methodPost
		//         "/limited_article_stars?select=article_id,user_id"
		//         [("Prefer", "return=representation")]
		//         [json| {"article_id": 2, "user_id": 1} |]
		//         `shouldRespondWith` [json|[{"article_id":2,"user_id":1}]|]
		//           { matchStatus = 201
		//           , matchHeaders = []
		//           }
		{
			Description: "table with limited privileges succeeds inserting if correct select is applied",
			Method:      "POST",
			Query:       "/limited_article_stars?select=article_id,user_id",
			Body:        `{"article_id": 2, "user_id": 1}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"article_id":2,"user_id":1}]`,
			Status:      201,
		},
		//     it "fails inserting if more columns are selected" $
		//       request
		//         methodPost
		//         "/limited_article_stars?select=article_id,user_id,created_at"
		//         [("Prefer", "return=representation")]
		//         [json| {"article_id": 2, "user_id": 2} |]
		//         `shouldRespondWith` [json|{"hint":null,"details":null,"code":"42501","message":"permission denied for view limited_article_stars"}|]
		//           { matchStatus = 401
		//           , matchHeaders = []
		//           }
		// @@ not asserted, PostgREST 16.4: the error body shape (null hint, no subsystem/position); card 32494 (error body)
		{
			Description: "table with limited privileges fails inserting if more columns are selected",
			Method:      "POST",
			Query:       "/limited_article_stars?select=article_id,user_id,created_at",
			Body:        `{"article_id": 2, "user_id": 2}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    ``,
			Status:      401,
		},
		//     it "fails inserting if select is not specified" $
		//       request
		//         methodPost
		//         "/limited_article_stars"
		//         [("Prefer", "return=representation")]
		//         [json| {"article_id": 3, "user_id": 1} |]
		//         `shouldRespondWith` [json|{"hint":null,"details":null,"code":"42501","message":"permission denied for view limited_article_stars"}|]
		//           { matchStatus = 401
		//           , matchHeaders = []
		//           }
		// @@ not asserted, PostgREST 16.4: the error body shape (null hint, no subsystem/position); card 32494 (error body)
		{
			Description: "table with limited privileges fails inserting if select is not specified",
			Method:      "POST",
			Query:       "/limited_article_stars",
			Body:        `{"article_id": 3, "user_id": 1}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Status:      401,
		},
		//     it "can insert in a table with no select and return=minimal" $ do
		//       request
		//         methodPost
		//         "/insertonly"
		//         [("Prefer", "return=minimal")]
		//         [json| { "v":"some value" } |]
		//         `shouldRespondWith` ""
		//           { matchStatus = 201
		//           , matchHeaders =
		//               [ matchHeaderAbsent hContentType
		//               , "Content-Length" <:> "0"
		//               , "Preference-Applied" <:> "return=minimal"
		//               ]
		//           }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "can insert in a table with no select and return=minimal",
			Method:          "POST",
			Query:           "/insertonly",
			Body:            `{ "v":"some value" }`,
			Headers:         test.Headers{"Prefer": {"return=minimal"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "0"},
			Status:          201,
		},
		//   describe "Inserting into VIEWs" $ do
		//     context "requesting no representation" $ do
		//       it "succeeds with 201" $
		//         post
		//           "/compound_pk_view"
		//           [json|{"k1":1,"k2":"test","extra":2}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , matchHeaderAbsent hLocation
		//                 , "Content-Length" <:> "0"
		//                 ]
		//             }
		{
			Description:     "inserting into VIEWs requesting no representation succeeds with 201",
			Method:          "POST",
			Query:           "/compound_pk_view",
			Body:            `{"k1":1,"k2":"test","extra":2}`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "", "Content-Length": "0"},
			Status:          201,
		},
		//       it "returns a location header with pks from both tables" $
		//         request
		//           methodPost
		//           "/with_multiple_pks"
		//           [("Prefer", "return=headers-only")]
		//           [json|{"pk1":1,"pk2":2}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/with_multiple_pks?pk1=eq.1&pk2=eq.2"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "returns a location header with pks from both tables",
			Method:          "POST",
			Query:           "/with_multiple_pks",
			Body:            `{"pk1":1,"pk2":2}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/with_multiple_pks?pk1=eq.1&pk2=eq.2", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//     context "requesting header only representation" $ do
		//       it "returns a location header with a composite PK col" $
		//         request
		//           methodPost
		//           "/compound_pk_view"
		//           [("Prefer", "return=headers-only")]
		//           [json|{"k1":1,"k2":"test","extra":2}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/compound_pk_view?k1=eq.1&k2=eq.test"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "returns a location header with a composite PK col",
			Method:          "POST",
			Query:           "/compound_pk_view",
			Body:            `{"k1":1,"k2":"test","extra":2}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/compound_pk_view?k1=eq.1&k2=eq.test", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//       it "should not throw and return location header when a PK is null" $
		//         request
		//           methodPost
		//           "/test_null_pk_competitors_sponsors"
		//           [("Prefer", "return=headers-only")]
		//           [json|{"id":1}|]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/test_null_pk_competitors_sponsors?id=eq.1&sponsor_id=is.null"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "returns the location header when a PK is null",
			Method:          "POST",
			Query:           "/test_null_pk_competitors_sponsors",
			Body:            `{"id":1}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/test_null_pk_competitors_sponsors?id=eq.1&sponsor_id=is.null", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: return=headers-only answers a Location header; card 32479 (write headers)",
		},
		//   describe "Data representations" $ do
		//     context "on regular table" $ do
		//       it "parses values in POST body" $
		//         -- we don't check that the parsing is correct here, just that it's happening. If it doesn't happen we'll get a
		//         -- an "invalid input syntax for type integer:" error.
		//         request
		//           methodPost
		//           "/datarep_todos"
		//           [("Prefer", "return=headers-only")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"} |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/datarep_todos?id=eq.5"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "data representations parse values in POST body",
			Method:          "POST",
			Query:           "/datarep_todos",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/datarep_todos?id=eq.5", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "parses values in POST body and formats individually selected values in return=representation" $
		//         request
		//           methodPost
		//           "/datarep_todos?select=id,label_color"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"} |]
		//           `shouldRespondWith` [json| [{"id":5, "label_color": "#001100"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations parse values in POST body and format individually selected values",
			Method:          "POST",
			Query:           "/datarep_todos?select=id,label_color",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5, "label_color": "#001100"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "parses values in POST body and formats values in return=representation" $
		//         request
		//           methodPost
		//           "/datarep_todos"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00", "icon_image": "3q2+7w", "created_at":-15, "budget": "-100000000000000.13"} |]
		//           `shouldRespondWith` [json| [{"id":5,"name": "party", "label_color": "#001100", "due_at":"2018-01-03T11:00:00Z", "icon_image": "3q2+7w==", "created_at":-15, "budget": "-100000000000000.13"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations parse values in POST body and format values",
			Method:          "POST",
			Query:           "/datarep_todos",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00", "icon_image": "3q2+7w", "created_at":-15, "budget": "-100000000000000.13"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5,"name": "party", "label_color": "#001100", "due_at":"2018-01-03T11:00:00Z", "icon_image": "3q2+7w==", "created_at":-15, "budget": "-100000000000000.13"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//     context "with ?columns parameter" $ do
		//       it "ignores json keys not included in ?columns; parses only the ones specified" $
		//         request
		//           methodPost
		//           "/datarep_todos?columns=id,label_color&select=id,name,label_color,due_at"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "invalid but should be ignored"} |]
		//           `shouldRespondWith` [json| [{"id":5, "name":null, "label_color": "#001100", "due_at": "2018-01-01T00:00:00Z"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations with ?columns parse only the columns specified",
			Method:          "POST",
			Query:           "/datarep_todos?columns=id,label_color&select=id,name,label_color,due_at",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "invalid but should be ignored"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5, "name":null, "label_color": "#001100", "due_at": "2018-01-01T00:00:00Z"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "fails without parsing anything if at least one specified column doesn't exist" $
		//         request
		//           methodPost
		//           "/datarep_todos?columns=id,label_color,helicopters&select=id,name,label_color,due_at"
		//           [("Prefer", "return=representation")]
		//           [json| {"due_at": "2019-01-03T11:00:00+00", "smth": "here", "label_color": "invalid", "fake_id": 13} |]
		//           `shouldRespondWith` [json| {"code":"PGRST204","details":null,"hint":null,"message":"Could not find the 'helicopters' column of 'datarep_todos' in the schema cache"} |]
		//             { matchStatus = 400
		//             , matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST204 error body; card 32494 (error body)
		{
			Description:     "data representations fail without parsing anything if a ?columns column doesn't exist",
			Method:          "POST",
			Query:           "/datarep_todos?columns=id,label_color,helicopters&select=id,name,label_color,due_at",
			Body:            `{"due_at": "2019-01-03T11:00:00+00", "smth": "here", "label_color": "invalid", "fake_id": 13}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		//     context "on updatable view" $ do
		//       it "parses values in POST body" $
		//         -- we don't check that the parsing is correct here, just that it's happening. If it doesn't happen we'll get a
		//         -- an "invalid input syntax for type integer:" error.
		//         request
		//           methodPost
		//           "/datarep_todos_computed"
		//           [("Prefer", "return=headers-only")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"} |]
		//           `shouldRespondWith` ""
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchHeaderAbsent hContentType
		//                 , "Location" <:> "/datarep_todos_computed?id=eq.5"
		//                 , "Content-Range" <:> "*/*"
		//                 , "Content-Length" <:> "0"
		//                 , "Preference-Applied" <:> "return=headers-only"
		//                 ]
		//             }
		{
			Description:     "data representations on an updatable view parse values in POST body",
			Method:          "POST",
			Query:           "/datarep_todos_computed",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"}`,
			Headers:         test.Headers{"Prefer": {"return=headers-only"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Location": "/datarep_todos_computed?id=eq.5", "Content-Range": "*/*", "Content-Length": "0", "Preference-Applied": "return=headers-only"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "parses values in POST body and formats individually selected values in return=representation" $
		//         request
		//           methodPost
		//           "/datarep_todos_computed?select=id,label_color"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"} |]
		//           `shouldRespondWith` [json| [{"id":5, "label_color": "#001100"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations on an updatable view format individually selected values",
			Method:          "POST",
			Query:           "/datarep_todos_computed?select=id,label_color",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5, "label_color": "#001100"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "parses values in POST body and formats values in return=representation" $
		//         request
		//           methodPost
		//           "/datarep_todos_computed"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"} |]
		//           `shouldRespondWith` [json| [{"id":5,"name": "party", "label_color": "#001100", "due_at":"2018-01-03T11:00:00Z", "dark_color":"#000880"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations on an updatable view format values",
			Method:          "POST",
			Query:           "/datarep_todos_computed",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "2018-01-03T11:00:00+00"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5,"name": "party", "label_color": "#001100", "due_at":"2018-01-03T11:00:00Z", "dark_color":"#000880"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//     context "on updatable views with ?columns parameter" $ do
		//       it "ignores json keys not included in ?columns; parses only the ones specified" $
		//         request
		//           methodPost
		//           "/datarep_todos_computed?columns=id,label_color&select=id,name,label_color,due_at"
		//           [("Prefer", "return=representation")]
		//           [json| {"id":5, "name": "party", "label_color": "#001100", "due_at": "invalid but should be ignored"} |]
		//           `shouldRespondWith` [json| [{"id":5, "name":null, "label_color": "#001100", "due_at": "2018-01-01T00:00:00Z"}] |]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Content-Type" <:> "application/json; charset=utf-8"
		//                 , "Content-Range" <:> "*/*"
		//                 ]
		//             }
		{
			Description:     "data representations on an updatable view with ?columns parse only the columns specified",
			Method:          "POST",
			Query:           "/datarep_todos_computed?columns=id,label_color&select=id,name,label_color,due_at",
			Body:            `{"id":5, "name": "party", "label_color": "#001100", "due_at": "invalid but should be ignored"}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Expected:        `[{"id":5, "name":null, "label_color": "#001100", "due_at": "2018-01-01T00:00:00Z"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "*/*"},
			Status:          201,
			Skip:            "PostgREST 16.4: data representations (a domain's casts from json and text) parse the values written; no card",
		},
		//       it "fails without parsing anything if at least one specified column doesn't exist" $
		//         request
		//           methodPost
		//           "/datarep_todos_computed?columns=id,label_color,helicopters&select=id,name,label_color,due_at"
		//           [("Prefer", "return=representation")]
		//           [json| {"due_at": "2019-01-03T11:00:00+00", "smth": "here", "label_color": "invalid", "fake_id": 13} |]
		//           `shouldRespondWith` [json| {"code":"PGRST204","details":null,"hint":null,"message":"Could not find the 'helicopters' column of 'datarep_todos_computed' in the schema cache"} |]
		//             { matchStatus = 400
		//             , matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//             }
		// @@ not asserted, PostgREST 16.4: the PGRST204 error body; card 32494 (error body)
		{
			Description:     "data representations on an updatable view fail without parsing anything if a ?columns column doesn't exist",
			Method:          "POST",
			Query:           "/datarep_todos_computed?columns=id,label_color,helicopters&select=id,name,label_color,due_at",
			Body:            `{"due_at": "2019-01-03T11:00:00+00", "smth": "here", "label_color": "invalid", "fake_id": 13}`,
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
		},
		// @@ added
		{
			Description: "with camel case columns works",
			Method:      "POST",
			Query:       "/UnitTest",
			Body:        `{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }]`,
			Status:      201,
		},
		// @@ added: regression for 42702 "column reference is ambiguous" — with select=*
		// plus an embed, the fk column must not be added twice to the RETURNING clause
		{
			Description: "star select with embed does not duplicate the fk in returning",
			Method:      "POST",
			Query:       "/projects?select=*,clients(id,name)",
			Body:        `{"id":8,"name":"Star Embed","client_id":2}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":8,"name":"Star Embed","client_id":2,"clients":{"id":2,"name":"Apple"}}]`,
			Status:      201,
		},
		{
			Description: "casted fk select with embed does not duplicate the fk in returning",
			Method:      "POST",
			Query:       "/projects?select=id,name,client_id::text,clients(id,name)",
			Body:        `{"id":9,"name":"Cast Embed","client_id":2}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":9,"name":"Cast Embed","client_id":"2","clients":{"id":2,"name":"Apple"}}]`,
			Status:      201,
		},
		{
			Description: "aliased fk select with embed still returns the embed",
			Method:      "POST",
			Query:       "/projects?select=id,name,cid:client_id,clients(id,name)",
			Body:        `{"id":10,"name":"Alias Embed","client_id":2}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":10,"name":"Alias Embed","cid":2,"clients":{"id":2,"name":"Apple"}}]`,
			Status:      201,
		},
		// @@ added: order= applies to the representation of a POST too (PostgREST
		// 13.0.0, #3013 "Fix order= with POST, PATCH, PUT and DELETE requests"); the
		// spec only covers it through the batch upsert in upsert_test.go
		{
			Description: "order applies to the representation of a POST",
			Method:      "POST",
			Query:       "/no_pk?order=a.desc",
			Body:        `[{ "a": "8", "b": "x" }, { "a": "9", "b": "x" }]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "a": "9", "b": "x" }, { "a": "8", "b": "x" }]`,
			Status:      201,
		},
	}

	test.Execute(t, testConfig, tests)
}
