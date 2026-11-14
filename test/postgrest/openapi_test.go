package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_OpenApi holds the cases of OpenApiSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApi(t *testing.T) {
	tests := []test.Test{
		// it "root path returns a valid openapi spec" $ do
		//   validateOpenApiResponse [("Accept", "application/openapi+json")]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		//   request
		//     methodHead
		//     "/"
		//     (acceptHdrs "application/openapi+json")
		//     ""
		//     `shouldRespondWith` ""
		//       { matchStatus = 200
		//       , matchHeaders =
		//           [ "Content-Type" <:> "application/openapi+json; charset=utf-8"
		//           , matchHeaderAbsent hContentLength
		//           ]
		//       }
		{
			Description:     "root path returns a valid openapi spec",
			Method:          "HEAD",
			Query:           "/",
			Headers:         test.Headers{"Accept": {"application/openapi+json"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/openapi+json; charset=utf-8", "Content-Length": ""},
			Status:          200,
			Skip:            "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		// it "should respond to openapi request on none root path with 406" $
		//   request
		//     methodGet
		//     "/items"
		//     (acceptHdrs "application/openapi+json")
		//     ""
		//     `shouldRespondWith` 406
		{
			Description: "should respond to openapi request on none root path with 406",
			Method:      "GET",
			Query:       "/items",
			Headers:     test.Headers{"Accept": {"application/openapi+json"}},
			Status:      406,
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		// it "should respond to openapi request with unsupported media type with 406" $
		//   request
		//     methodGet
		//     "/"
		//     (acceptHdrs "text/csv")
		//     ""
		//     `shouldRespondWith` 406
		{
			Description: "should respond to openapi request with unsupported media type with 406",
			Method:      "GET",
			Query:       "/",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Status:      406,
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		// it "includes postgrest.org current version api docs" $ do
		//   r <- get "/"
		//
		//   let
		//     headers = simpleHeaders r
		//     docsUrl = simpleBody r ^? key "externalDocs" . key "url"
		//
		//   liftIO $ do
		//     headers `shouldSatisfy` notZeroContentLength
		//     docsUrl `shouldBe` Just (String ("https://postgrest.org/en/" <> docsVersion <> "/references/api.html"))
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes postgrest.org current version api docs",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "schema" $ do
		//   it "includes title and comments to schema" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let childGetTitle = r ^? key "info" . key "title"
		//     let childGetDescription = r ^? key "info" . key "description"
		//
		//     liftIO $ do
		//       childGetTitle `shouldBe` Just "My API title"
		//
		//       childGetDescription `shouldBe` Just "My API description\nthat spans\nmultiple lines"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes title and comments to schema",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "table" $ do
		//   it "includes paths to tables" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/child_entities" . key s
		//       childGetSummary = r ^? method "get" . key "summary"
		//       childGetDescription = r ^? method "get" . key "description"
		//       getParameters = r ^? method "get" . key "parameters"
		//       postParameters = r ^? method "post" . key "parameters"
		//       postResponse = r ^? method "post" . key "responses" . key "201" . key "description"
		//       patchResponse = r ^? method "patch" . key "responses" . key "204" . key "description"
		//       deleteResponse = r ^? method "delete" . key "responses" . key "204" . key "description"
		//
		//     let
		//       grandChildGet s = key "paths" . key "/grandchild_entities" . key "get" . key s
		//       grandChildGetSummary = r ^? grandChildGet "summary"
		//       grandChildGetDescription = r ^? grandChildGet "description"
		//
		//     liftIO $ do
		//       childGetSummary `shouldBe` Just "child_entities comment"
		//
		//       childGetDescription `shouldBe` Nothing
		//
		//       grandChildGetSummary `shouldBe` Just "grandchild_entities summary"
		//
		//       grandChildGetDescription `shouldBe` Just "grandchild_entities description\nthat spans\nmultiple lines"
		//
		//       getParameters
		//         `shouldBe` Just
		//           [aesonQQ|
		//           [
		//             { "$ref": "#/parameters/rowFilter.child_entities.id" },
		//             { "$ref": "#/parameters/rowFilter.child_entities.name" },
		//             { "$ref": "#/parameters/rowFilter.child_entities.parent_id" },
		//             { "$ref": "#/parameters/select" },
		//             { "$ref": "#/parameters/order" },
		//             { "$ref": "#/parameters/range" },
		//             { "$ref": "#/parameters/rangeUnit" },
		//             { "$ref": "#/parameters/offset" },
		//             { "$ref": "#/parameters/limit" },
		//             { "$ref": "#/parameters/preferCount" }
		//           ]
		//         |]
		//
		//       postParameters
		//         `shouldBe` Just
		//           [aesonQQ|
		//           [
		//             { "$ref": "#/parameters/body.child_entities" },
		//             { "$ref": "#/parameters/select" },
		//             { "$ref": "#/parameters/preferPost" }
		//           ]
		//         |]
		//
		//       postResponse `shouldBe` Just "Created"
		//
		//       patchResponse `shouldBe` Just "No Content"
		//
		//       deleteResponse `shouldBe` Just "No Content"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes paths to tables",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes an array type for GET responses" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let childGetSchema =
		//           r
		//             ^? key "paths"
		//               . key "/child_entities"
		//               . key "get"
		//               . key "responses"
		//               . key "200"
		//               . key "schema"
		//
		//     liftIO $
		//       childGetSchema
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes an array type for GET responses",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes definitions to tables" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let def = r ^? key "definitions" . key "child_entities"
		//
		//     liftIO $
		//       def
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes definitions to tables",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes definitions to views" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let def = r ^? key "definitions" . key "child_entities_view"
		//
		//     liftIO $
		//       def
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes definitions to views",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "doesn't include privileged table for anonymous" $ do
		//     r <- simpleBody <$> get "/"
		//     let tablePath = r ^? key "paths" . key "/authors_only"
		//
		//     liftIO $ tablePath `shouldBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "doesn't include privileged table for anonymous",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes table if user has permission" $ do
		//     let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//     r <- simpleBody <$> request methodGet "/" [auth] ""
		//     let tableTag =
		//           r
		//             ^? key "paths"
		//               . key "/authors_only"
		//               . key "post"
		//               . key "tags"
		//               . nth 0
		//     liftIO $ tableTag `shouldBe` Just [aesonQQ|"authors_only"|]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes table if user has permission",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes a fk description for a O2O relationship" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let referralLink = r ^? key "definitions" . key "first" . key "properties" . key "second_id_1"
		//
		//     liftIO $
		//       referralLink
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes a fk description for a O2O relationship",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "Foreign table" $
		//   it "includes foreign table properties" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/projects_dump" . key s
		//       getSummary = r ^? method "get" . key "summary"
		//       getDescription = r ^? method "get" . key "description"
		//       getParameters = r ^? method "get" . key "parameters"
		//
		//     liftIO $ do
		//       getSummary `shouldBe` Just "A temporary projects dump"
		//
		//       getDescription `shouldBe` Just "Just a test for foreign tables"
		//
		//       getParameters
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes foreign table properties",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "Partitioned table" $
		//   it "includes partitioned table properties" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/car_models" . key s
		//       getSummary = r ^? method "get" . key "summary"
		//       getDescription = r ^? method "get" . key "description"
		//       getParameterName = r ^? method "get" . key "parameters" . nth 0 . key "$ref"
		//       getParameterYear = r ^? method "get" . key "parameters" . nth 1 . key "$ref"
		//       getParameterRef = r ^? method "get" . key "parameters" . nth 2 . key "$ref"
		//
		//     liftIO $ do
		//       getSummary `shouldBe` Just "A partitioned table"
		//
		//       getDescription `shouldBe` Just "A test for partitioned tables"
		//
		//       getParameterName `shouldBe` Just "#/parameters/rowFilter.car_models.name"
		//
		//       getParameterYear `shouldBe` Just "#/parameters/rowFilter.car_models.year"
		//
		//       getParameterRef `shouldBe` Just "#/parameters/rowFilter.car_models.car_brand_name"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes partitioned table properties",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "Materialized view" $
		//   it "includes materialized view properties" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/materialized_projects" . key s
		//       summary = r ^? method "get" . key "summary"
		//       description = r ^? method "get" . key "description"
		//       parameters = r ^? method "get" . key "parameters"
		//
		//     liftIO $ do
		//       summary `shouldBe` Just "A materialized view for projects"
		//
		//       description `shouldBe` Just "Just a test for materialized views"
		//
		//       parameters
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes materialized view properties",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "VIEW that has a source FK based on a UNIQUE key" $
		//   it "includes fk description" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let referralLink = r ^? key "definitions" . key "referrals" . key "properties" . key "link"
		//
		//     liftIO $
		//       referralLink
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes fk description",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "VIEW created for a TABLE with a O2M relationship" $ do
		//   it "fk points to destination TABLE instead of the VIEW" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let referralLink = r ^? key "definitions" . key "projects" . key "properties" . key "client_id"
		//
		//     liftIO $
		//       referralLink
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "fk points to destination TABLE instead of the VIEW",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "PostgreSQL to Swagger Type Mapping" $ do
		//   it "character varying to string" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_character_varying"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "character varying to string",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "character(1) to string" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_character"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "character(1) to string",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "text to string" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_text"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "text to string",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "boolean to boolean" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_boolean"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "boolean to boolean",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "smallint to integer" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_smallint"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "smallint to integer",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "integer to integer" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_integer"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "integer to integer",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "bigint to integer" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_bigint"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "bigint to integer",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "numeric to number" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_numeric"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "numeric to number",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "real to number" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_real"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "real to number",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "double_precision to number" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_double_precision"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "double_precision to number",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "json to any" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_json"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "json to any",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "jsonb to any" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_jsonb"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "jsonb to any",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "array types to array" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let text_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_text_arr"
		//     let int_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_int_arr"
		//     let bool_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_bool_arr"
		//     let char_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_char_arr"
		//     let varchar_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_varchar_arr"
		//     let bigint_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_bigint_arr"
		//     let numeric_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_numeric_arr"
		//     let json_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_json_arr"
		//     let jsonb_arr_types = r ^? key "definitions" . key "openapi_types" . key "properties" . key "a_jsonb_arr"
		//
		//     liftIO $ do
		//       text_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "text[]",
		//             "type": "array",
		//             "items": {
		//               "type": "string"
		//             }
		//           }
		//         |]
		//
		//       int_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "integer[]",
		//             "type": "array",
		//             "items": {
		//               "type": "integer"
		//             }
		//           }
		//         |]
		//
		//       bool_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "boolean[]",
		//             "type": "array",
		//             "items": {
		//               "type": "boolean"
		//             }
		//           }
		//         |]
		//
		//       char_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "character[]",
		//             "type": "array",
		//             "items": {
		//               "type": "string"
		//             }
		//           }
		//         |]
		//
		//       varchar_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "character varying[]",
		//             "type": "array",
		//             "items": {
		//               "type": "string"
		//             }
		//           }
		//         |]
		//
		//       bigint_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "bigint[]",
		//             "type": "array",
		//             "items": {
		//               "type": "integer"
		//             }
		//           }
		//         |]
		//
		//       numeric_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "numeric[]",
		//             "type": "array",
		//             "items": {
		//               "type": "number"
		//             }
		//           }
		//         |]
		//
		//       json_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		//           {
		//             "format": "json[]",
		//             "type": "array",
		//             "items": {}
		//           }
		//         |]
		//
		//       jsonb_arr_types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "array types to array",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "Detects default values" $ do
		//   it "text" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let defaultValue = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "text" . key "default"
		//
		//     liftIO $
		//       defaultValue `shouldBe` Just "default"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "text",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "boolean" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "boolean" . key "default"
		//
		//     liftIO $
		//       types `shouldBe` Just (Bool False)
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "boolean",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "integer" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "integer" . key "default"
		//
		//     liftIO $
		//       types `shouldBe` Just (Number 42)
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "integer",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "numeric" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "numeric" . key "default"
		//
		//     liftIO $
		//       types `shouldBe` Just (Number 42.2)
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "numeric",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "date" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "date" . key "default"
		//
		//     liftIO $
		//       types `shouldBe` Just "1900-01-01"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "date",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "time" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "openapi_defaults" . key "properties" . key "time" . key "default"
		//
		//     liftIO $
		//       types `shouldBe` Just "13:00:00"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "time",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "enum" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let types = r ^? key "definitions" . key "menagerie" . key "properties" . key "enum"
		//
		//     liftIO $
		//       types
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "enum",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "RPC" $ do
		//   it "includes function summary/description and query parameters for arguments in the get path item" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/rpc/varied_arguments_openapi" . key s
		//       args = r ^? method "get" . key "parameters"
		//       summary = r ^? method "get" . key "summary"
		//       description = r ^? method "get" . key "description"
		//
		//     liftIO $ do
		//       summary `shouldBe` Just "An RPC function"
		//
		//       description `shouldBe` Just "Just a test for RPC function arguments"
		//
		//       args
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes function summary/description and query parameters for arguments in the get path item",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes function summary/description and body schema for arguments in the post path item" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       method s = key "paths" . key "/rpc/varied_arguments_openapi" . key s
		//       args = r ^? method "post" . key "parameters" . nth 0 . key "schema"
		//       summary = r ^? method "post" . key "summary"
		//       description = r ^? method "post" . key "description"
		//
		//     liftIO $ do
		//       summary `shouldBe` Just "An RPC function"
		//
		//       description `shouldBe` Just "Just a test for RPC function arguments"
		//
		//       args
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes function summary/description and body schema for arguments in the post path item",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "doesn't include privileged function for anonymous" $ do
		//     r <- simpleBody <$> get "/"
		//     let funcPath = r ^? key "paths" . key "/rpc/privileged_hello"
		//
		//     liftIO $ funcPath `shouldBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "doesn't include privileged function for anonymous",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes function if user has permission" $ do
		//     let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//     r <- simpleBody <$> request methodGet "/" [auth] ""
		//     let funcTag =
		//           r
		//             ^? key "paths"
		//               . key "/rpc/privileged_hello"
		//               . key "post"
		//               . key "tags"
		//               . nth 0
		//
		//     liftIO $ funcTag `shouldBe` Just [aesonQQ|"(rpc) privileged_hello"|]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes function if user has permission",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "doesn't include OUT params of function as required parameters" $ do
		//     r <- simpleBody <$> get "/"
		//     let params =
		//           r
		//             ^? key "paths"
		//               . key "/rpc/many_out_params"
		//               . key "post"
		//               . key "parameters"
		//               . nth 0
		//               . key "schema"
		//               . key "required"
		//
		//     liftIO $ params `shouldBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "doesn't include OUT params of function as required parameters",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes INOUT params(with no DEFAULT) of function as required parameters" $ do
		//     r <- simpleBody <$> get "/"
		//     let params =
		//           r
		//             ^? key "paths"
		//               . key "/rpc/many_inout_params"
		//               . key "post"
		//               . key "parameters"
		//               . nth 0
		//               . key "schema"
		//               . key "required"
		//
		//     liftIO $ params `shouldBe` Just [aesonQQ|["num", "str"]|]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes INOUT params(with no DEFAULT) of function as required parameters",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "uses a multi collection format when the function has a VARIADIC parameter" $ do
		//     r <- simpleBody <$> get "/"
		//     let param =
		//           r
		//             ^? key "paths"
		//               . key "/rpc/variadic_param"
		//               . key "get"
		//               . key "parameters"
		//               . nth 0
		//
		//     liftIO $
		//       param
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "uses a multi collection format when the function has a VARIADIC parameter",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "only includes POST method for volatile functions" $ do
		//     r <- simpleBody <$> get "/"
		//     let
		//       volatileGet = r ^? key "paths" . key "/rpc/reset_table" . key "get"
		//       volatilePost = r ^? key "paths" . key "/rpc/reset_table" . key "post"
		//
		//     liftIO $ do
		//       volatileGet `shouldBe` Nothing
		//       volatilePost `shouldNotBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "only includes POST method for volatile functions",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes GET and POST methods for stable functions" $ do
		//     r <- simpleBody <$> get "/"
		//     let
		//       stableGet = r ^? key "paths" . key "/rpc/getallusers" . key "get"
		//       stablePost = r ^? key "paths" . key "/rpc/getallusers" . key "post"
		//
		//     liftIO $ do
		//       stableGet `shouldNotBe` Nothing
		//       stablePost `shouldNotBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes GET and POST methods for stable functions",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes GET and POST methods for immutable functions" $ do
		//     r <- simpleBody <$> get "/"
		//     let
		//       immutableGet = r ^? key "paths" . key "/rpc/jwt_test" . key "get"
		//       immutablePost = r ^? key "paths" . key "/rpc/jwt_test" . key "post"
		//
		//     liftIO $ do
		//       immutableGet `shouldNotBe` Nothing
		//       immutablePost `shouldNotBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes GET and POST methods for immutable functions",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "does not include empty enum in the preferParams parameter" $ do
		//     r <- simpleBody <$> get "/"
		//     let preferParams = r ^? key "parameters" . key "preferParams" . key "enum"
		//
		//     liftIO $ do
		//       preferParams `shouldBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "does not include empty enum in the preferParams parameter",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "Security" $
		//   it "does not include security or security definitions by default" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       sec = r ^? key "security"
		//       secDef = r ^? key "securityDefinitions"
		//
		//     liftIO $ do
		//       sec `shouldBe` Nothing
		//
		//       secDef `shouldBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "does not include security or security definitions by default",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_OpenApiRoot holds the cases of RootSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApiRoot(t *testing.T) {
	tests := []test.Test{
		// describe "root spec function" $ do
		//   it "accepts application/openapi+json" $ do
		//     request
		//       methodGet
		//       "/"
		//       [("Accept", "application/openapi+json")]
		//       ""
		//       `shouldRespondWith` [json|{
		//          "swagger": "2.0",
		//          "info": {"title": "PostgREST API", "description": "This is a dynamic API generated by PostgREST"}
		//         }|]
		//         { matchHeaders = ["Content-Type" <:> "application/openapi+json; charset=utf-8"]
		//         }
		{
			Description: "accepts application/openapi+json",
			Method:      "GET",
			Query:       "/",
			Headers:     test.Headers{"Accept": {"application/openapi+json"}},
			Expected: `{
             "swagger": "2.0",
             "info": {"title": "PostgREST API", "description": "This is a dynamic API generated by PostgREST"}
            }`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/openapi+json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		//   it "accepts application/json" $ do
		//     request
		//       methodGet
		//       "/"
		//       [("Accept", "application/json")]
		//       ""
		//       `shouldRespondWith` [json|{
		//          "swagger": "2.0",
		//          "info": {"title": "PostgREST API", "description": "This is a dynamic API generated by PostgREST"}
		//         }|]
		//         { matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//         }
		{
			Description: "accepts application/json",
			Method:      "GET",
			Query:       "/",
			Headers:     test.Headers{"Accept": {"application/json"}},
			Expected: `{
             "swagger": "2.0",
             "info": {"title": "PostgREST API", "description": "This is a dynamic API generated by PostgREST"}
            }`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		// describe "root spec default response" $ do
		//   it "works when schema name is mixed case" $ do
		//     r <- request methodGet "/" [] ""
		//     let
		//       resStatus = simpleStatus r
		//       resBody = simpleBody r
		//
		//     liftIO $ do
		//       resStatus `shouldBe` Status{statusCode = 200, statusMessage = "OK"}
		//       resBody ^? key "info" . key "title" `shouldBe` Just [aesonQQ| "v3 schema" |]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "works when schema name is mixed case",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_OpenApiIgnorePriv holds the cases of IgnorePrivOpenApiSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApiIgnorePriv(t *testing.T) {
	tests := []test.Test{
		// it "root path returns a valid openapi spec" $ do
		//   validateOpenApiResponse [("Accept", "application/openapi+json")]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		//   request
		//     methodHead
		//     "/"
		//     (acceptHdrs "application/openapi+json")
		//     ""
		//     `shouldRespondWith` ""
		//       { matchStatus = 200
		//       , matchHeaders =
		//           [ "Content-Type" <:> "application/openapi+json; charset=utf-8"
		//           , matchHeaderAbsent hContentLength
		//           ]
		//       }
		{
			Description:     "root path returns a valid openapi spec",
			Method:          "HEAD",
			Query:           "/",
			Headers:         test.Headers{"Accept": {"application/openapi+json"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/openapi+json; charset=utf-8", "Content-Length": ""},
			Status:          200,
			Skip:            "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
		// describe "table" $ do
		//   it "includes privileged table even if user does not have permission" $ do
		//     r <- simpleBody <$> get "/"
		//     let tableTag =
		//           r
		//             ^? key "paths"
		//               . key "/authors_only"
		//               . key "post"
		//               . key "tags"
		//               . nth 0
		//
		//     liftIO $ tableTag `shouldBe` Just [aesonQQ|"authors_only"|]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes privileged table even if user does not have permission",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "only includes tables that belong to another schema if the Accept-Profile header is used" $ do
		//     r1 <- simpleBody <$> get "/"
		//     let tableKey1 = r1 ^? key "paths" . key "/children"
		//
		//     liftIO $ tableKey1 `shouldBe` Nothing
		//
		//     r2 <- simpleBody <$> request methodGet "/" [("Accept-Profile", "v1")] ""
		//     let tableKey2 = r2 ^? key "paths" . key "/children"
		//
		//     liftIO $ tableKey2 `shouldNotBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "only includes tables that belong to another schema if the Accept-Profile header is used",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "includes comments on tables" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       grandChildGet s = key "paths" . key "/grandchild_entities" . key "get" . key s
		//       grandChildGetSummary = r ^? grandChildGet "summary"
		//       grandChildGetDescription = r ^? grandChildGet "description"
		//
		//     liftIO $ do
		//       grandChildGetSummary `shouldBe` Just "grandchild_entities summary"
		//       grandChildGetDescription `shouldBe` Just "grandchild_entities description\nthat spans\nmultiple lines"
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes comments on tables",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		// describe "RPC" $ do
		//   it "includes privileged function even if user does not have permission" $ do
		//     r <- simpleBody <$> get "/"
		//     let funcTag =
		//           r
		//             ^? key "paths"
		//               . key "/rpc/privileged_hello"
		//               . key "post"
		//               . key "tags"
		//               . nth 0
		//
		//     liftIO $ funcTag `shouldBe` Just [aesonQQ|"(rpc) privileged_hello"|]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes privileged function even if user does not have permission",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
		//   it "only includes functions that belong to another schema if the Accept-Profile header is used" $ do
		//     r1 <- simpleBody <$> get "/"
		//     let funcKey1 = r1 ^? key "paths" . key "/rpc/get_parents_below"
		//
		//     liftIO $ funcKey1 `shouldBe` Nothing
		//
		//     r2 <- simpleBody <$> request methodGet "/" [("Accept-Profile", "v1")] ""
		//     let funcKey2 = r2 ^? key "paths" . key "/rpc/get_parents_below"
		//
		//     liftIO $ funcKey2 `shouldNotBe` Nothing
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "only includes functions that belong to another schema if the Accept-Profile header is used",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_OpenApiDisabled holds the cases of DisabledOpenApiSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApiDisabled(t *testing.T) {
	tests := []test.Test{
		// describe "Disabled OpenApi" $ do
		//   it "responds with 404" $
		//     request
		//       methodGet
		//       "/"
		//       [("Accept", "application/openapi+json")]
		//       ""
		//       `shouldRespondWith` [json| {"code":"PGRST126","details":null,"hint":null,"message":"Root endpoint metadata is disabled"} |]
		//         { matchStatus = 404
		//         , matchHeaders = ["Content-Length" <:> "93"]
		//         }
		{
			Description:     "responds with 404",
			Method:          "GET",
			Query:           "/",
			Headers:         test.Headers{"Accept": {"application/openapi+json"}},
			Expected:        `{"code":"PGRST126","details":null,"hint":null,"message":"Root endpoint metadata is disabled"}`,
			ExpectedHeaders: map[string]string{"Content-Length": "93"},
			Status:          404,
			Skip:            "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi)",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_OpenApiProxy holds the cases of ProxySpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApiProxy(t *testing.T) {
	tests := []test.Test{
		// describe "GET / with proxy" $
		//   it "returns a valid openapi spec with proxy" $
		//     validateOpenApiResponse [("Accept", "application/openapi+json")]
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_OpenApiSecurity holds the cases of SecurityOpenApiSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_OpenApiSecurity(t *testing.T) {
	tests := []test.Test{
		// describe "Security active" $
		//   it "includes security and security definitions" $ do
		//     r <- simpleBody <$> get "/"
		//
		//     let
		//       sec = r ^? key "security"
		//       secDef = r ^? key "securityDefinitions"
		//
		//     liftIO $ do
		//       sec
		//         `shouldBe` Just
		//           [aesonQQ|
		//           [
		//             { "JWT": [] }
		//           ]
		//         |]
		//
		//       secDef
		//         `shouldBe` Just
		//           [aesonQQ|
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "includes security and security definitions",
			Method:      "GET",
			Query:       "/",
			Skip:        "PostgREST 16.4: the OpenAPI description on the root endpoint; card 32493 (openapi); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}
