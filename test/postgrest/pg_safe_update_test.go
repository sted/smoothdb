package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// PgSafeUpdateSpec runs its first part with db-pre-request set to
// test.load_safeupdate, which LOADs the pg-safeupdate library so that an
// UPDATE or DELETE without a WHERE clause fails (21000). smoothdb has no
// pre-request hook, and the test Postgres has no safeupdate library: those
// cases are skipped. The second part (disabledSpec) runs on the base config.
func TestPostgREST_PgSafeUpdate(t *testing.T) {

	tests := []test.Test{
		// spec withConfig = withConfig (baseCfg{configDbPreRequest = Just $ QualifiedIdentifier "test" "load_safeupdate"}) $
		//   describe "Enabling pg-safeupdate" $ do
		//     context "Full table update" $ do
		//       it "does not update and throws error if no condition is present" $
		//         request
		//           methodPatch
		//           "/safe_update_items"
		//           [("Prefer", "count=exact")]
		//           [json| {"name": "New name"} |]
		//           `shouldRespondWith` [json|{
		//               "code": "21000",
		//               "details": null,
		//               "hint": null,
		//               "message": "UPDATE requires a WHERE clause"
		//             }|]
		//             { matchStatus = 400
		//             }
		{
			Description: "Enabling pg-safeupdate: full table update does not update and throws error if no condition is present",
			Method:      "PATCH",
			Query:       "/safe_update_items",
			Body:        `{"name": "New name"}`,
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Expected:    `{"code": "21000", "details": null, "hint": null, "message": "UPDATE requires a WHERE clause"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: db-pre-request (test.load_safeupdate, which also needs the safeupdate library) refuses an UPDATE without WHERE; card 31811 (pre-request)",
		},
		//       it "allows full table update if a filter is present" $
		//         request methodPatch "/safe_update_items?id=gt.0" mempty [json| {"name": "updated-item"} |]
		//           `shouldRespondWith` 204
		{
			Description: "Enabling pg-safeupdate: full table update allows full table update if a filter is present",
			Method:      "PATCH",
			Query:       "/safe_update_items?id=gt.0",
			Body:        `{"name": "updated-item"}`,
			Status:      204,
			Skip:        "PostgREST 16.4: db-pre-request (test.load_safeupdate, which also needs the safeupdate library) lets a filtered UPDATE through; card 31811 (pre-request)",
		},
		//     context "Full table delete" $ do
		//       it "does not delete and throws error if no condition is present" $
		//         request methodDelete "/safe_delete_items" [] mempty
		//           `shouldRespondWith` [json|{
		//               "code": "21000",
		//               "details": null,
		//               "hint": null,
		//               "message": "DELETE requires a WHERE clause"
		//             }|]
		//             { matchStatus = 400
		//             }
		{
			Description: "Enabling pg-safeupdate: full table delete does not delete and throws error if no condition is present",
			Method:      "DELETE",
			Query:       "/safe_delete_items",
			Expected:    `{"code": "21000", "details": null, "hint": null, "message": "DELETE requires a WHERE clause"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: db-pre-request (test.load_safeupdate, which also needs the safeupdate library) refuses a DELETE without WHERE; card 31811 (pre-request)",
		},
		//       it "allows full table delete if a filter is present" $
		//         request methodDelete "/safe_delete_items?id=gt.0" mempty mempty
		//           `shouldRespondWith` 204
		{
			Description: "Enabling pg-safeupdate: full table delete allows full table delete if a filter is present",
			Method:      "DELETE",
			Query:       "/safe_delete_items?id=gt.0",
			Status:      204,
			Skip:        "PostgREST 16.4: db-pre-request (test.load_safeupdate, which also needs the safeupdate library) lets a filtered DELETE through; card 31811 (pre-request)",
		},
		// disabledSpec withConfig = withConfig baseCfg $
		//   describe "Disabling pg-safeupdate" $ do
		//     context "Full table update" $ do
		//       it "works if no condition is present" $
		//         request methodPatch "/unsafe_update_items" mempty [json| {"name": "updated-item"} |]
		//           `shouldRespondWith` 204
		{
			Description: "Disabling pg-safeupdate: full table update works if no condition is present",
			Method:      "PATCH",
			Query:       "/unsafe_update_items",
			Body:        `{"name": "updated-item"}`,
			Status:      204,
		},
		//     context "Full table delete" $ do
		//       it "works if no condition is present" $
		//         request methodDelete "/unsafe_delete_items" mempty mempty
		//           `shouldRespondWith` 204
		{
			Description: "Disabling pg-safeupdate: full table delete works if no condition is present",
			Method:      "DELETE",
			Query:       "/unsafe_delete_items",
			Status:      204,
		},
	}

	test.Execute(t, testConfig, tests)
}
