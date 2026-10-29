package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from test/spec/Feature/OptionsSpec.hs: OPTIONS on a resource answers
// with an Allow header listing the methods that resource accepts. smoothdb
// routes OPTIONS only through the CORS catch-all (server/cors.go), a bare 200
// with no Allow, registered when CORSAllowedOrigins is not empty: TestMain sets
// none, so whether the route exists depends on the git-ignored config.jsonc
// TestMain reads (the shipped default leaves it empty).
func TestPostgREST_Options(t *testing.T) {

	const noAllow = "PostgREST 16.4: OPTIONS answers an Allow header with the methods the resource accepts; card 32497 (options cors)"

	tests := []test.Test{
		// spec withConfig = withConfig baseCfg $ describe "Allow header" $ do
		//   context "a table" $ do
		//     it "includes read/write methods for writeable table" $ do
		//       r <- request methodOptions "/items" [] ""
		//       liftIO $ do
		//         simpleHeaders r `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"
		//         simpleHeaders r `shouldSatisfy` matchHeader "Content-Length" "0"
		{
			Description:     "includes read/write methods for writeable table",
			Method:          "OPTIONS",
			Query:           "/items",
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE", "Content-Length": "0"},
			Skip:            noAllow,
		},
		//     it "fails with 404 for an unknown table" $
		//       request methodOptions "/unknown" [] "" `shouldRespondWith` 404
		{
			Description: "fails with 404 for an unknown table",
			Method:      "OPTIONS",
			Query:       "/unknown",
			Status:      404,
			Skip:        "PostgREST 16.4: OPTIONS is routed per resource, 404 only for an unknown one (smoothdb answers every path with the CORS catch-all's 200, or 404 for every path when CORS is off); card 32497 (options cors)",
		},
		//   context "a partitioned table" $ do
		//     it "includes read/write methods for writeable partitioned tables" $ do
		//       r <- request methodOptions "/car_models" [] ""
		//       liftIO $
		//         simpleHeaders r
		//           `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"
		{
			Description:     "includes read/write methods for writeable partitioned tables",
			Method:          "OPTIONS",
			Query:           "/car_models",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"},
			Skip:            noAllow,
		},
		//   context "a view" $ do
		//     context "auto updatable" $ do
		//       it "includes read/write methods for auto updatable views with pk" $ do
		//         r <- request methodOptions "/projects_auto_updatable_view_with_pk" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"
		{
			Description:     "includes read/write methods for auto updatable views with pk",
			Method:          "OPTIONS",
			Query:           "/projects_auto_updatable_view_with_pk",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"},
			Skip:            noAllow,
		},
		//       it "includes read/write methods for auto updatable views without pk" $ do
		//         r <- request methodOptions "/projects_auto_updatable_view_without_pk" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PATCH,DELETE"
		{
			Description:     "includes read/write methods for auto updatable views without pk",
			Method:          "OPTIONS",
			Query:           "/projects_auto_updatable_view_without_pk",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PATCH,DELETE"},
			Skip:            noAllow,
		},
		//     context "non auto updatable" $ do
		//       it "includes read methods for non auto updatable views" $ do
		//         r <- request methodOptions "/projects_view_without_triggers" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD"
		{
			Description:     "includes read methods for non auto updatable views",
			Method:          "OPTIONS",
			Query:           "/projects_view_without_triggers",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD"},
			Skip:            noAllow,
		},
		//       it "includes read/write methods for insertable, updatable and deletable views with pk" $ do
		//         r <- request methodOptions "/projects_view_with_all_triggers_with_pk" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"
		{
			Description:     "includes read/write methods for insertable, updatable and deletable views with pk",
			Method:          "OPTIONS",
			Query:           "/projects_view_with_all_triggers_with_pk",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PUT,PATCH,DELETE"},
			Skip:            noAllow,
		},
		//       it "includes read/write methods for insertable, updatable and deletable views without pk" $ do
		//         r <- request methodOptions "/projects_view_with_all_triggers_without_pk" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST,PATCH,DELETE"
		{
			Description:     "includes read/write methods for insertable, updatable and deletable views without pk",
			Method:          "OPTIONS",
			Query:           "/projects_view_with_all_triggers_without_pk",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST,PATCH,DELETE"},
			Skip:            noAllow,
		},
		//       it "includes read and insert methods for insertable views" $ do
		//         r <- request methodOptions "/projects_view_with_insert_trigger" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST"
		{
			Description:     "includes read and insert methods for insertable views",
			Method:          "OPTIONS",
			Query:           "/projects_view_with_insert_trigger",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST"},
			Skip:            noAllow,
		},
		//       it "includes read and update methods for updatable views" $ do
		//         r <- request methodOptions "/projects_view_with_update_trigger" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,PATCH"
		{
			Description:     "includes read and update methods for updatable views",
			Method:          "OPTIONS",
			Query:           "/projects_view_with_update_trigger",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,PATCH"},
			Skip:            noAllow,
		},
		//       it "includes read and delete methods for deletable views" $ do
		//         r <- request methodOptions "/projects_view_with_delete_trigger" [] ""
		//         liftIO $
		//           simpleHeaders r
		//             `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,DELETE"
		{
			Description:     "includes read and delete methods for deletable views",
			Method:          "OPTIONS",
			Query:           "/projects_view_with_delete_trigger",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,DELETE"},
			Skip:            noAllow,
		},
		//   context "a function" $ do
		//     it "includes the POST method for a volatile function" $ do
		//       r <- request methodOptions "/rpc/reset_table" [] ""
		//       liftIO $
		//         simpleHeaders r
		//           `shouldSatisfy` matchHeader "Allow" "OPTIONS,POST"
		{
			Description:     "includes the POST method for a volatile function",
			Method:          "OPTIONS",
			Query:           "/rpc/reset_table",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,POST"},
			Skip:            noAllow,
		},
		//     it "includes the GET/HEAD/POST method for a stable function" $ do
		//       r <- request methodOptions "/rpc/getallusers" [] ""
		//       liftIO $
		//         simpleHeaders r
		//           `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST"
		{
			Description:     "includes the GET/HEAD/POST method for a stable function",
			Method:          "OPTIONS",
			Query:           "/rpc/getallusers",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST"},
			Skip:            noAllow,
		},
		//     it "includes the GET/HEAD/POST method for a immutable function" $ do
		//       r <- request methodOptions "/rpc/jwt_test" [] ""
		//       liftIO $
		//         simpleHeaders r
		//           `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD,POST"
		{
			Description:     "includes the GET/HEAD/POST method for a immutable function",
			Method:          "OPTIONS",
			Query:           "/rpc/jwt_test",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD,POST"},
			Skip:            noAllow,
		},
		//   context "root endpoint" $ do
		//     it "includes the GET/HEAD method " $ do
		//       r <- request methodOptions "/" [] ""
		//       liftIO $
		//         simpleHeaders r
		//           `shouldSatisfy` matchHeader "Allow" "OPTIONS,GET,HEAD"
		{
			Description:     "includes the GET/HEAD method",
			Method:          "OPTIONS",
			Query:           "/",
			ExpectedHeaders: map[string]string{"Allow": "OPTIONS,GET,HEAD"},
			Skip:            noAllow,
		},
	}

	test.Execute(t, testConfig, tests)
}
