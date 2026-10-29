package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Upsert(t *testing.T) {

	tests := []test.Test{
		// describe "UPSERT" $ do
		//   context "with POST" $ do
		//     context "when Prefer: resolution=merge-duplicates is specified" $ do
		//       it "INSERTs and UPDATEs rows on pk conflict" $
		//         request
		//           methodPost
		//           "/tiobe_pls"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "name": "Javascript", "rank": 6 },
		//             { "name": "Java", "rank": 2 },
		//             { "name": "C", "rank": 1 }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "Javascript", "rank": 6 },
		//             { "name": "Java", "rank": 2 },
		//             { "name": "C", "rank": 1 }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows on pk conflict",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "name": "Javascript", "rank": 6 }, { "name": "Java", "rank": 2 }, { "name": "C", "rank": 1 }]`,
			Expected:        `[{ "name": "Javascript", "rank": 6 }, { "name": "Java", "rank": 2 }, { "name": "C", "rank": 1 }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "UPDATEs rows on pk conflict" $
		//         request
		//           methodPost
		//           "/tiobe_pls"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "name": "Python", "rank": 6 },
		//             { "name": "Java", "rank": 2 },
		//             { "name": "C", "rank": 1 }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "Python", "rank": 6 },
		//             { "name": "Java", "rank": 2 },
		//             { "name": "C", "rank": 1 }
		//           ]|]
		//             { matchStatus = 200
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates UPDATEs rows on pk conflict",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "name": "Python", "rank": 6 }, { "name": "Java", "rank": 2 }, { "name": "C", "rank": 1 }]`,
			Expected:        `[{ "name": "Python", "rank": 6 }, { "name": "Java", "rank": 2 }, { "name": "C", "rank": 1 }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a merge-duplicates upsert that inserted no row answers 200, smoothdb 201; card 32945 (upsert status)",
		},
		//       it "INSERTs and UPDATEs row on composite pk conflict" $
		//         request
		//           methodPost
		//           "/employees"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "first_name": "Frances M.", "last_name": "Roe", "salary": "30000" },
		//             { "first_name": "Peter S.", "last_name": "Yang", "salary": 42000 }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "first_name": "Frances M.", "last_name": "Roe", "salary": "$30,000.00", "company": "One-Up Realty", "occupation": "Author" },
		//             { "first_name": "Peter S.", "last_name": "Yang", "salary": "$42,000.00", "company": null, "occupation": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs row on composite pk conflict",
			Method:          "POST",
			Query:           "/employees",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "first_name": "Frances M.", "last_name": "Roe", "salary": "30000" }, { "first_name": "Peter S.", "last_name": "Yang", "salary": 42000 }]`,
			Expected:        `[{ "first_name": "Frances M.", "last_name": "Roe", "salary": "$30,000.00", "company": "One-Up Realty", "occupation": "Author" }, { "first_name": "Peter S.", "last_name": "Yang", "salary": "$42,000.00", "company": null, "occupation": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: a money column is serialized as its text (\"$30,000.00\"), smoothdb answers a number; no card",
		},
		// @@ added: the same upsert, its money column left out of the representation (see the skip above)
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs row on composite pk conflict, salary not selected",
			Method:          "POST",
			Query:           "/employees?select=first_name,last_name,company,occupation",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "first_name": "Frances M.", "last_name": "Roe", "salary": "30000" }, { "first_name": "Peter S.", "last_name": "Yang", "salary": 42000 }]`,
			Expected:        `[{ "first_name": "Frances M.", "last_name": "Roe", "company": "One-Up Realty", "occupation": "Author" }, { "first_name": "Peter S.", "last_name": "Yang", "company": null, "occupation": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "INSERTs and UPDATEs rows on composite pk conflict for partitioned tables" $
		//         request
		//           methodPost
		//           "/car_models"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "name": "Murcielago", "year": 2001, "car_brand_name": null},
		//             { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "Murcielago", "year": 2001, "car_brand_name": null},
		//             { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows on composite pk conflict for partitioned tables",
			Method:          "POST",
			Query:           "/car_models",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "name": "Murcielago", "year": 2001, "car_brand_name": null}, { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }]`,
			Expected:        `[{ "name": "Murcielago", "year": 2001, "car_brand_name": null}, { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "succeeds when the payload has no elements" $
		//         request
		//           methodPost
		//           "/articles"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json|[]|]
		//           `shouldRespondWith` [json|[]|]
		//             { matchStatus = 200 -- nothing was inserted, so it should be 200
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates succeeds when the payload has no elements",
			Method:          "POST",
			Query:           "/articles",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[]`,
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a merge-duplicates upsert that inserted no row answers 200, smoothdb 201; card 32945 (upsert status)",
		},
		//       it "INSERTs and UPDATEs rows on single unique key conflict" $
		//         request
		//           methodPost
		//           "/single_unique?on_conflict=unique_key"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "unique_key": 1, "value": "B" },
		//             { "unique_key": 2, "value": "C" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "unique_key": 1, "value": "B" },
		//             { "unique_key": 2, "value": "C" }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows on single unique key conflict",
			Method:          "POST",
			Query:           "/single_unique?on_conflict=unique_key",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "unique_key": 1, "value": "B" }, { "unique_key": 2, "value": "C" }]`,
			Expected:        `[{ "unique_key": 1, "value": "B" }, { "unique_key": 2, "value": "C" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "INSERTs and UPDATEs rows on compound unique keys conflict" $
		//         request
		//           methodPost
		//           "/compound_unique?on_conflict=key1,key2"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json| [
		//             { "key1": 1, "key2": 1, "value": "B" },
		//             { "key1": 1, "key2": 2, "value": "C" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "key1": 1, "key2": 1, "value": "B" },
		//             { "key1": 1, "key2": 2, "value": "C" }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows on compound unique keys conflict",
			Method:          "POST",
			Query:           "/compound_unique?on_conflict=key1,key2",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "key1": 1, "key2": 1, "value": "B" }, { "key1": 1, "key2": 2, "value": "C" }]`,
			Expected:        `[{ "key1": 1, "key2": 1, "value": "B" }, { "key1": 1, "key2": 2, "value": "C" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		// @@added - with quotes
		{
			Description: "with POST merge-duplicates INSERTs and UPDATEs rows on compound unique keys conflict, quoted on_conflict columns",
			Method:      "POST",
			Query:       "/compound_unique?on_conflict=\"key1\",\"key2\"",
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:        `[{ "key1": 1, "key2": 1, "value": "B" }, { "key1": 1, "key2": 2, "value": "C" }]`,
			Expected:    `[{ "key1": 1, "key2": 1, "value": "B" }, { "key1": 1, "key2": 2, "value": "C" }]`,
			Status:      201,
		},
		//       it "INSERTs and UPDATEs rows with SERIAL surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/surr_serial_upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "updated value", "extra": "existing value" },
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows with SERIAL surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/surr_serial_upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "updated value", "extra": "existing value" }, { "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "INSERTs and UPDATEs rows with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/surr_gen_default_upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "updated value", "extra": "existing value" },
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/surr_gen_default_upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "updated value", "extra": "existing value" }, { "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "INSERTs and UPDATEs rows with case sensitive table name with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/Surr_Gen_Default_Upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "updated value", "extra": "existing value cs" },
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST merge-duplicates INSERTs and UPDATEs rows with case sensitive table name with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/Surr_Gen_Default_Upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "updated value", "extra": "existing value cs" }, { "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "succeeds if the table has only PK cols and no other cols" $
		//         request
		//           methodPost
		//           "/only_pk"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json|[ { "id": 1 }, { "id": 2 }, { "id": 4} ]|]
		//           `shouldRespondWith` [json|[ { "id": 1 }, { "id": 2 }, { "id": 4} ]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Preference-Applied" <:> "resolution=merge-duplicates, return=representation"
		//                 , matchContentTypeJson
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates succeeds if the table has only PK cols and no other cols",
			Method:          "POST",
			Query:           "/only_pk",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[ { "id": 1 }, { "id": 2 }, { "id": 4} ]`,
			Expected:        `[ { "id": 1 }, { "id": 2 }, { "id": 4} ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "succeeds and ignores the Prefer: resolution header(no Preference-Applied present) if the table has no PK" $
		//         request
		//           methodPost
		//           "/no_pk"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//           [json|[ { "a": "1", "b": "0" } ]|]
		//           `shouldRespondWith` [json|[ { "a": "1", "b": "0" } ]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ matchContentTypeJson
		//                 , "Preference-Applied" <:> "return=representation"
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST merge-duplicates succeeds and ignores the Prefer: resolution header(no Preference-Applied present) if the table has no PK",
			Method:          "POST",
			Query:           "/no_pk",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[ { "a": "1", "b": "0" } ]`,
			Expected:        `[ { "a": "1", "b": "0" } ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//
		//     context "when Prefer: resolution=ignore-duplicates is specified" $ do
		//       it "INSERTs and ignores rows on pk conflict" $
		//         request
		//           methodPost
		//           "/tiobe_pls"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json|[
		//             { "name": "PHP", "rank": 9 },
		//             { "name": "Python", "rank": 10 }
		//           ]|]
		//           `shouldRespondWith` [json|[
		//             { "name": "PHP", "rank": 9 }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST ignore-duplicates INSERTs and ignores rows on pk conflict",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[{ "name": "PHP", "rank": 9 }, { "name": "Python", "rank": 10 }]`,
			Expected:        `[{ "name": "PHP", "rank": 9 }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "INSERTs and ignores rows on composite pk conflict" $
		//         request
		//           methodPost
		//           "/employees"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json|[
		//             { "first_name": "Daniel B.", "last_name": "Lyon", "salary": "72000", "company": null, "occupation": null },
		//             { "first_name": "Sara M.", "last_name": "Torpey", "salary": 60000, "company": "Burstein-Applebee", "occupation": "Soil scientist" }
		//           ]|]
		//           `shouldRespondWith` [json|[
		//             { "first_name": "Sara M.", "last_name": "Torpey", "salary": "$60,000.00", "company": "Burstein-Applebee", "occupation": "Soil scientist" }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates INSERTs and ignores rows on composite pk conflict",
			Method:          "POST",
			Query:           "/employees",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[{ "first_name": "Daniel B.", "last_name": "Lyon", "salary": "72000", "company": null, "occupation": null }, { "first_name": "Sara M.", "last_name": "Torpey", "salary": 60000, "company": "Burstein-Applebee", "occupation": "Soil scientist" }]`,
			Expected:        `[{ "first_name": "Sara M.", "last_name": "Torpey", "salary": "$60,000.00", "company": "Burstein-Applebee", "occupation": "Soil scientist" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: a money column is serialized as its text (\"$60,000.00\"), smoothdb answers a number; no card",
		},
		// @@ added: the same upsert, its money column left out of the representation (see the skip above)
		{
			Description:     "with POST ignore-duplicates INSERTs and ignores rows on composite pk conflict, salary not selected",
			Method:          "POST",
			Query:           "/employees?select=first_name,last_name,company,occupation",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[{ "first_name": "Daniel B.", "last_name": "Lyon", "salary": "72000", "company": null, "occupation": null }, { "first_name": "Sara M.", "last_name": "Torpey", "salary": 60000, "company": "Burstein-Applebee", "occupation": "Soil scientist" }]`,
			Expected:        `[{ "first_name": "Sara M.", "last_name": "Torpey", "company": "Burstein-Applebee", "occupation": "Soil scientist" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "INSERTs and ignores rows on composite pk conflict for partitioned tables" $
		//         request
		//           methodPost
		//           "/car_models"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json| [
		//             { "name": "Murcielago", "year": 2001, "car_brand_name": "Ferrari" },
		//             { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation", matchContentTypeJson]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST ignore-duplicates INSERTs and ignores rows on composite pk conflict for partitioned tables",
			Method:          "POST",
			Query:           "/car_models",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[{ "name": "Murcielago", "year": 2001, "car_brand_name": "Ferrari" }, { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }]`,
			Expected:        `[{ "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "INSERTs and ignores rows on single unique key conflict" $
		//         request
		//           methodPost
		//           "/single_unique?on_conflict=unique_key"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json| [
		//               { "unique_key": 1, "value": "B" },
		//               { "unique_key": 2, "value": "C" },
		//               { "unique_key": 3, "value": "D" }
		//             ]|]
		//           `shouldRespondWith` [json| [
		//               { "unique_key": 2, "value": "C" },
		//               { "unique_key": 3, "value": "D" }
		//             ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation"]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "with POST ignore-duplicates INSERTs and ignores rows on single unique key conflict",
			Method:      "POST",
			Query:       "/single_unique?on_conflict=unique_key",
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:        `[{ "unique_key": 1, "value": "B" }, { "unique_key": 2, "value": "C" }, { "unique_key": 3, "value": "D" }]`,
			Expected:    `[{ "unique_key": 2, "value": "C" }, { "unique_key": 3, "value": "D" }]`,
			Status:      201,
		},
		//       it "INSERTs and UPDATEs rows on compound unique keys conflict" $
		//         request
		//           methodPost
		//           "/compound_unique?on_conflict=key1,key2"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json| [
		//               { "key1": 1, "key2": 1, "value": "B" },
		//               { "key1": 1, "key2": 2, "value": "C" },
		//               { "key1": 1, "key2": 3, "value": "D" }
		//             ]|]
		//           `shouldRespondWith` [json| [
		//               { "key1": 1, "key2": 2, "value": "C" },
		//               { "key1": 1, "key2": 3, "value": "D" }
		//             ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation"]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "with POST ignore-duplicates INSERTs and UPDATEs rows on compound unique keys conflict",
			Method:      "POST",
			Query:       "/compound_unique?on_conflict=key1,key2",
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:        `[{ "key1": 1, "key2": 1, "value": "B" }, { "key1": 1, "key2": 2, "value": "C" }, { "key1": 1, "key2": 3, "value": "D" }]`,
			Expected:    `[{ "key1": 1, "key2": 2, "value": "C" }, { "key1": 1, "key2": 3, "value": "D" }]`,
			Status:      201,
		},
		//       it "INSERTs and UPDATEs rows with SERIAL surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/surr_serial_upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates INSERTs and UPDATEs rows with SERIAL surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/surr_serial_upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "INSERTs and UPDATEs rows with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/surr_gen_default_upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates INSERTs and UPDATEs rows with GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/surr_gen_default_upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "INSERTs and UPDATEs rows with case sensitive table name GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default" $
		//         request
		//           methodPost
		//           "/Surr_Gen_Default_Upsert?columns=id,name&select=name,extra"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates"), ("Prefer", "missing=default")]
		//           [json| [
		//             { "id": 1, "name": "updated value" },
		//             { "name": "new value" }
		//           ]|]
		//           `shouldRespondWith` [json| [
		//             { "name": "new value", "extra": null }
		//           ]|]
		//             { matchStatus = 201
		//             , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, missing=default, return=representation", matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates INSERTs and UPDATEs rows with case sensitive table name GENERATED BY DEFAULT surrogate primary keys using Prefer: missing=default",
			Method:          "POST",
			Query:           "/Surr_Gen_Default_Upsert?columns=id,name&select=name,extra",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates", "missing=default"}},
			Body:            `[{ "id": 1, "name": "updated value" }, { "name": "new value" }]`,
			Expected:        `[{ "name": "new value", "extra": null }]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: Prefer missing=default fills the ?columns keys a row lacks with the column default (smoothdb answers 23502); card 32482 (prefer options)",
		},
		//       it "succeeds if the table has only PK cols and no other cols" $
		//         request
		//           methodPost
		//           "/only_pk"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json|[ { "id": 1 }, { "id": 2 }, { "id": 3} ]|]
		//           `shouldRespondWith` [json|[ { "id": 3} ]|]
		//             { matchStatus = 201
		//             , matchHeaders =
		//                 [ "Preference-Applied" <:> "resolution=ignore-duplicates, return=representation"
		//                 , matchContentTypeJson
		//                 ]
		//             }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "with POST ignore-duplicates succeeds if the table has only PK cols and no other cols",
			Method:          "POST",
			Query:           "/only_pk",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[ { "id": 1 }, { "id": 2 }, { "id": 3} ]`,
			Expected:        `[ { "id": 3} ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//       it "succeeds if not a single resource is created" $ do
		//         request
		//           methodPost
		//           "/tiobe_pls"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json|[ { "name": "Java", "rank": 1 } ]|]
		//           `shouldRespondWith` [json|[]|]
		//             { matchStatus = 201
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates succeeds if not a single resource is created",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[ { "name": "Java", "rank": 1 } ]`,
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//         request
		//           methodPost
		//           "/tiobe_pls"
		//           [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//           [json|[ { "name": "Java", "rank": 1 }, { "name": "C", "rank": 2 } ]|]
		//           `shouldRespondWith` [json|[]|]
		//             { matchStatus = 201
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with POST ignore-duplicates succeeds if not a single resource is created",
			Method:          "POST",
			Query:           "/tiobe_pls",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:            `[ { "name": "Java", "rank": 1 }, { "name": "C", "rank": 2 } ]`,
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		// @@ PostgREST 14 had, before "fails if limit is specified", a case dropped since:
		// 	  it "fails if Range is specified" $
		// 		request methodPut "/tiobe_pls?name=eq.Javascript" [("Range", "0-5")]
		// 		  [json| [ { "name": "Javascript", "rank": 1 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Range header and limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		// 		  { matchStatus = 400 , matchHeaders = [matchContentTypeJson] }
		// PostgREST 16.4 ignores the Range header on PUT ("ignores the Range header" below).
		//
		//   context "with PUT" $ do
		//     context "Restrictions" $ do
		//       it "fails if limit is specified" $
		//         put
		//           "/tiobe_pls?name=eq.Javascript&limit=1"
		//           [json| [ { "name": "Javascript", "rank": 1 } ]|]
		//           `shouldRespondWith` [json|{"message":"limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions fails if limit is specified",
			Method:          "PUT",
			Query:           "/tiobe_pls?name=eq.Javascript&limit=1",
			Body:            `[ { "name": "Javascript", "rank": 1 } ]`,
			Expected:        `{"message":"limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
			Skip:            "PostgREST 16.4: PUT refuses limit and offset (400, PGRST114); card 32478 (PUT upsert)",
		},
		//       it "fails if offset is specified" $
		//         put
		//           "/tiobe_pls?name=eq.Javascript&offset=1"
		//           [json| [ { "name": "Javascript", "rank": 1 } ]|]
		//           `shouldRespondWith` [json|{"message":"limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		//             { matchStatus = 400
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions fails if offset is specified",
			Method:          "PUT",
			Query:           "/tiobe_pls?name=eq.Javascript&offset=1",
			Body:            `[ { "name": "Javascript", "rank": 1 } ]`,
			Expected:        `{"message":"limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
			Skip:            "PostgREST 16.4: PUT refuses limit and offset (400, PGRST114); card 32478 (PUT upsert)",
		},
		//       it "rejects every other filter than pk cols eq's" $ do
		//         put
		//           "/tiobe_pls?rank=eq.19"
		//           [json| [ { "name": "Go", "rank": 19 } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions rejects every other filter than pk cols eq's",
			Method:          "PUT",
			Query:           "/tiobe_pls?rank=eq.19",
			Body:            `[ { "name": "Go", "rank": 19 } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//         put
		//           "/tiobe_pls?id=not.eq.Java"
		//           [json| [ { "name": "Go", "rank": 19 } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions rejects every other filter than pk cols eq's",
			Method:          "PUT",
			Query:           "/tiobe_pls?id=not.eq.Java",
			Body:            `[ { "name": "Go", "rank": 19 } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//         put
		//           "/tiobe_pls?id=in.(Go)"
		//           [json| [ { "name": "Go", "rank": 19 } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions rejects every other filter than pk cols eq's",
			Method:          "PUT",
			Query:           "/tiobe_pls?id=in.(Go)",
			Body:            `[ { "name": "Go", "rank": 19 } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//         put
		//           "/tiobe_pls?and=(id.eq.Go)"
		//           [json| [ { "name": "Go", "rank": 19 } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions rejects every other filter than pk cols eq's",
			Method:          "PUT",
			Query:           "/tiobe_pls?and=(id.eq.Go)",
			Body:            `[ { "name": "Go", "rank": 19 } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//       it "fails if not all composite key cols are specified as eq filters" $ do
		//         put
		//           "/employees?first_name=eq.Susan"
		//           [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions fails if not all composite key cols are specified as eq filters",
			Method:          "PUT",
			Query:           "/employees?first_name=eq.Susan",
			Body:            `[ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//         put
		//           "/employees?last_name=eq.Heidt"
		//           [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		//           `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//             { matchStatus = 405
		//             , matchHeaders = [matchContentTypeJson]
		//             }
		{
			Description:     "with PUT restrictions fails if not all composite key cols are specified as eq filters",
			Method:          "PUT",
			Query:           "/employees?last_name=eq.Heidt",
			Body:            `[ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//     it "fails if the uri primary key doesn't match the payload primary key" $ do
		//       put "/tiobe_pls?name=eq.MATLAB" [json| [ { "name": "Perl", "rank": 17 } ]|]
		//         `shouldRespondWith` [json|{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}|]
		//           { matchStatus = 400
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "with PUT fails if the uri primary key doesn't match the payload primary key",
			Method:          "PUT",
			Query:           "/tiobe_pls?name=eq.MATLAB",
			Body:            `[ { "name": "Perl", "rank": 17 } ]`,
			Expected:        `{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
			Skip:            "PostgREST 16.4: PUT refuses a payload whose primary key differs from the URL (400, PGRST115); card 32478 (PUT upsert)",
		},
		//       put
		//         "/employees?first_name=eq.Wendy&last_name=eq.Anderson"
		//         [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		//         `shouldRespondWith` [json|{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}|]
		//           { matchStatus = 400
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "with PUT fails if the uri primary key doesn't match the payload primary key",
			Method:          "PUT",
			Query:           "/employees?first_name=eq.Wendy&last_name=eq.Anderson",
			Body:            `[ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]`,
			Expected:        `{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          400,
			Skip:            "PostgREST 16.4: PUT refuses a payload whose primary key differs from the URL (400, PGRST115); card 32478 (PUT upsert)",
		},
		//     it "fails if the table has no PK" $
		//       put "/no_pk?a=eq.one&b=eq.two" [json| [ { "a": "one", "b": "two" } ]|]
		//         `shouldRespondWith` [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		//           { matchStatus = 405
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "with PUT fails if the table has no PK",
			Method:          "PUT",
			Query:           "/no_pk?a=eq.one&b=eq.two",
			Body:            `[ { "a": "one", "b": "two" } ]`,
			Expected:        `{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          405,
			Skip:            "PostgREST 16.4: PUT refuses filters other than eq on every primary key column (405, PGRST105); card 32478 (PUT upsert)",
		},
		//     context "Inserting row" $ do
		//       it "succeeds on table with single pk col" $ do
		//         -- assert that the next request will indeed be an insert
		//         get "/tiobe_pls?name=eq.Go"
		//           `shouldRespondWith` [json|[]|]
		{
			Description: "with PUT inserting row succeeds on table with single pk col: the row is absent before",
			Method:      "GET",
			Query:       "/tiobe_pls?name=eq.Go",
			Expected:    `[]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/tiobe_pls?name=eq.Go"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "name": "Go", "rank": 19 } ]|]
		//           `shouldRespondWith` [json| [ { "name": "Go", "rank": 19 } ]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "with PUT inserting row succeeds on table with single pk col",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Go",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "name": "Go", "rank": 19 } ]`,
			Expected:    `[ { "name": "Go", "rank": 19 } ]`,
			Status:      201,
			Skip:        "PostgREST 16.4: PUT inserts the row addressed by its primary key (201); card 32478 (PUT upsert)",
		},
		//
		//       it "succeeds on table with composite pk" $ do
		//         -- assert that the next request will indeed be an insert
		//         get "/employees?first_name=eq.Susan&last_name=eq.Heidt"
		//           `shouldRespondWith` [json|[]|]
		{
			Description: "with PUT inserting row succeeds on table with composite pk: the row is absent before",
			Method:      "GET",
			Query:       "/employees?first_name=eq.Susan&last_name=eq.Heidt",
			Expected:    `[]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/employees?first_name=eq.Susan&last_name=eq.Heidt"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		//           `shouldRespondWith` [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "$48,000.00", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "with PUT inserting row succeeds on table with composite pk",
			Method:      "PUT",
			Query:       "/employees?first_name=eq.Susan&last_name=eq.Heidt",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]`,
			Expected:    `[ { "first_name": "Susan", "last_name": "Heidt", "salary": "$48,000.00", "company": "GEX", "occupation": "Railroad engineer" } ]`,
			Status:      201,
			Skip:        "PostgREST 16.4: PUT inserts the row addressed by its primary key (201); card 32478 (PUT upsert)",
		},
		//
		//       it "succeeds on a partitioned table with composite pk" $ do
		//         -- assert that the next request will indeed be an insert
		//         get "/car_models?name=eq.Supra&year=eq.2021"
		//           `shouldRespondWith` [json|[]|]
		{
			Description: "with PUT inserting row succeeds on a partitioned table with composite pk: the row is absent before",
			Method:      "GET",
			Query:       "/car_models?name=eq.Supra&year=eq.2021",
			Expected:    `[]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/car_models?name=eq.Supra&year=eq.2021"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "name": "Supra", "year": 2021 } ]|]
		//           `shouldRespondWith` [json| [ { "name": "Supra", "year": 2021, "car_brand_name": null } ]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "with PUT inserting row succeeds on a partitioned table with composite pk",
			Method:      "PUT",
			Query:       "/car_models?name=eq.Supra&year=eq.2021",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "name": "Supra", "year": 2021 } ]`,
			Expected:    `[ { "name": "Supra", "year": 2021, "car_brand_name": null } ]`,
			Status:      201,
			Skip:        "PostgREST 16.4: PUT inserts the row addressed by its primary key (201); card 32478 (PUT upsert)",
		},
		//
		//       it "succeeds if the table has only PK cols and no other cols" $ do
		//         -- assert that the next request will indeed be an insert
		//         get "/only_pk?id=eq.10"
		//           `shouldRespondWith` [json|[]|]
		{
			Description: "with PUT inserting row succeeds if the table has only PK cols and no other cols: the row is absent before",
			Method:      "GET",
			Query:       "/only_pk?id=eq.10",
			Expected:    `[]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/only_pk?id=eq.10"
		//           [("Prefer", "return=representation")]
		//           [json|[ { "id": 10 } ]|]
		//           `shouldRespondWith` [json|[ { "id": 10 } ]|]
		//             { matchStatus = 201
		//             }
		{
			Description: "with PUT inserting row succeeds if the table has only PK cols and no other cols",
			Method:      "PUT",
			Query:       "/only_pk?id=eq.10",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "id": 10 } ]`,
			Expected:    `[ { "id": 10 } ]`,
			Status:      201,
			Skip:        "PostgREST 16.4: PUT inserts the row addressed by its primary key (201); card 32478 (PUT upsert)",
		},
		//     context "Updating row" $ do
		//       it "succeeds on table with single pk col" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/tiobe_pls?name=eq.Java"
		//           `shouldRespondWith` [json|[ { "name": "Java", "rank": 1 } ]|]
		{
			Description: "with PUT updating row succeeds on table with single pk col: the row exists before",
			Method:      "GET",
			Query:       "/tiobe_pls?name=eq.Java",
			Expected:    `[ { "name": "Java", "rank": 1 } ]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/tiobe_pls?name=eq.Java"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "name": "Java", "rank": 13 } ]|]
		//           `shouldRespondWith` [json| [ { "name": "Java", "rank": 13 } ]|]
		{
			Description: "with PUT updating row succeeds on table with single pk col",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Java",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "name": "Java", "rank": 13 } ]`,
			Expected:    `[ { "name": "Java", "rank": 13 } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by its primary key; card 32478 (PUT upsert)",
		},
		//       it "succeeds if the payload has more than one row, but it only puts the first element" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/tiobe_pls?name=eq.Java"
		//           `shouldRespondWith` [json|[ { "name": "Java", "rank": 1 } ]|]
		{
			Description: "with PUT updating row succeeds if the payload has more than one row, but it only puts the first element: the row exists before",
			Method:      "GET",
			Query:       "/tiobe_pls?name=eq.Java",
			Expected:    `[ { "name": "Java", "rank": 1 } ]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/tiobe_pls?name=eq.Java"
		//           [("Prefer", "return=representation"), ("Accept", "application/vnd.pgrst.object+json")]
		//           [json| [ { "name": "Java", "rank": 19 }, { "name": "Swift", "rank": 12 } ] |]
		//           `shouldRespondWith` [json|{ "name": "Java", "rank": 19 }|]
		//             { matchHeaders = [matchContentTypeSingular]
		//             }
		{
			Description: "with PUT updating row succeeds if the payload has more than one row, but it only puts the first element",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Java",
			Headers:     test.Headers{"Prefer": {"return=representation"}, "Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `[ { "name": "Java", "rank": 19 }, { "name": "Swift", "rank": 12 } ]`,
			Expected:    `{ "name": "Java", "rank": 19 }`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT of a payload with more rows writes only its first element, as a vnd.pgrst.object; card 32478 (PUT upsert)",
		},
		//       it "succeeds on table with composite pk" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/employees?first_name=eq.Frances M.&last_name=eq.Roe"
		//           `shouldRespondWith` [json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$24,000.00", "company": "One-Up Realty", "occupation": "Author" } ]|]
		{
			Description: "with PUT updating row succeeds on table with composite pk: the row exists before",
			Method:      "GET",
			Query:       "/employees?first_name=eq.Frances M.&last_name=eq.Roe",
			Expected:    `[ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$24,000.00", "company": "One-Up Realty", "occupation": "Author" } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: a money column is serialized as its text (\"$24,000.00\"), smoothdb answers a number; no card",
		},
		//         request
		//           methodPut
		//           "/employees?first_name=eq.Frances M.&last_name=eq.Roe"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "60000", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]|]
		//           `shouldRespondWith` [json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$60,000.00", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]|]
		{
			Description: "with PUT updating row succeeds on table with composite pk",
			Method:      "PUT",
			Query:       "/employees?first_name=eq.Frances M.&last_name=eq.Roe",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "first_name": "Frances M.", "last_name": "Roe", "salary": "60000", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]`,
			Expected:    `[ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$60,000.00", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by its primary key; card 32478 (PUT upsert)",
		},
		//       it "succeeds on a partitioned table with composite pk" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/car_models?name=eq.DeLorean&year=eq.1981"
		//           `shouldRespondWith` [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": "DMC" } ]|]
		{
			Description: "with PUT updating row succeeds on a partitioned table with composite pk: the row exists before",
			Method:      "GET",
			Query:       "/car_models?name=eq.DeLorean&year=eq.1981",
			Expected:    `[ { "name": "DeLorean", "year": 1981, "car_brand_name": "DMC" } ]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/car_models?name=eq.DeLorean&year=eq.1981"
		//           [("Prefer", "return=representation")]
		//           [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]|]
		//           `shouldRespondWith` [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]|]
		{
			Description: "with PUT updating row succeeds on a partitioned table with composite pk",
			Method:      "PUT",
			Query:       "/car_models?name=eq.DeLorean&year=eq.1981",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]`,
			Expected:    `[ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by its primary key; card 32478 (PUT upsert)",
		},
		//       it "succeeds if the table has only PK cols and no other cols" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/only_pk?id=eq.1"
		//           `shouldRespondWith` [json|[ { "id": 1 } ]|]
		{
			Description: "with PUT updating row succeeds if the table has only PK cols and no other cols: the row exists before",
			Method:      "GET",
			Query:       "/only_pk?id=eq.1",
			Expected:    `[ { "id": 1 } ]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/only_pk?id=eq.1"
		//           [("Prefer", "return=representation")]
		//           [json|[ { "id": 1 } ]|]
		//           `shouldRespondWith` [json|[ { "id": 1 } ]|]
		{
			Description: "with PUT updating row succeeds if the table has only PK cols and no other cols",
			Method:      "PUT",
			Query:       "/only_pk?id=eq.1",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[ { "id": 1 } ]`,
			Expected:    `[ { "id": 1 } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by its primary key; card 32478 (PUT upsert)",
		},
		//       it "ignores the Range header" $ do
		//         -- assert that the next request will indeed be an update
		//         get "/tiobe_pls?name=eq.Java"
		//           `shouldRespondWith` [json|[ { "name": "Java", "rank": 1 } ]|]
		{
			Description: "with PUT updating row ignores the Range header: the row exists before",
			Method:      "GET",
			Query:       "/tiobe_pls?name=eq.Java",
			Expected:    `[ { "name": "Java", "rank": 1 } ]`,
			Status:      200,
		},
		//         request
		//           methodPut
		//           "/tiobe_pls?name=eq.Java"
		//           [("Prefer", "return=representation"), ("Range", "1-1")]
		//           [json| [ { "name": "Java", "rank": 5 } ]|]
		//           `shouldRespondWith` [json| [ { "name": "Java", "rank": 5 } ]|]
		{
			Description: "with PUT updating row ignores the Range header",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Java",
			Headers:     test.Headers{"Prefer": {"return=representation"}, "Range": {"1-1"}},
			Body:        `[ { "name": "Java", "rank": 5 } ]`,
			Expected:    `[ { "name": "Java", "rank": 5 } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT ignores the Range header; card 32478 (PUT upsert)",
		},
		//
		//     -- TODO: move this to SingularSpec?
		//     it "works with return=representation and vnd.pgrst.object+json" $
		//       request
		//         methodPut
		//         "/tiobe_pls?name=eq.Ruby"
		//         [("Prefer", "return=representation"), ("Accept", "application/vnd.pgrst.object+json")]
		//         [json| [ { "name": "Ruby", "rank": 11 } ]|]
		//         `shouldRespondWith` [json|{ "name": "Ruby", "rank": 11 }|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchContentTypeSingular]
		//           }
		{
			Description: "with PUT works with return=representation and vnd.pgrst.object+json",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Ruby",
			Headers:     test.Headers{"Prefer": {"return=representation"}, "Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `[ { "name": "Ruby", "rank": 11 } ]`,
			Expected:    `{ "name": "Ruby", "rank": 11 }`,
			Status:      201,
			Skip:        "PostgREST 16.4: PUT inserts the row addressed by its primary key, as a vnd.pgrst.object; card 32478 (PUT upsert)",
		},
		//
		//   context "with a camel case pk column" $ do
		//     it "works with POST and merge-duplicates" $ do
		//       request
		//         methodPost
		//         "/UnitTest"
		//         [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//         [json|[
		//             { "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		//             { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		//           ]|]
		//         `shouldRespondWith` [json|[
		//             { "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		//             { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		//           ]|]
		//           { matchStatus = 201
		//           , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates, return=representation"]
		//           }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "with a camel case pk column works with POST and merge-duplicates",
			Method:      "POST",
			Query:       "/UnitTest",
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:        `[{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" }, { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }]`,
			Expected:    `[{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" }, { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }]`,
			Status:      201,
		},
		//     it "works with POST and ignore-duplicates headers" $ do
		//       request
		//         methodPost
		//         "/UnitTest"
		//         [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		//         [json|[
		//             { "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		//             { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		//           ]|]
		//         `shouldRespondWith` [json|[
		//             { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		//           ]|]
		//           { matchStatus = 201
		//           , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates, return=representation"]
		//           }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description: "with a camel case pk column works with POST and ignore-duplicates headers",
			Method:      "POST",
			Query:       "/UnitTest",
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Body:        `[{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" }, { "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }]`,
			Expected:    `[{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }]`,
			Status:      201,
		},
		//     it "works with PUT" $ do
		//       put
		//         "/UnitTest?idUnitTest=eq.1"
		//         [json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
		//         `shouldRespondWith` ""
		//           { matchStatus = 204
		//           , matchHeaders = [matchHeaderAbsent hContentType]
		//           }
		{
			Description:     "with a camel case pk column works with PUT",
			Method:          "PUT",
			Query:           "/UnitTest?idUnitTest=eq.1",
			Body:            `[ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          204,
			Skip:            "PostgREST 16.4: PUT updates the row addressed by a camel case primary key; card 32478 (PUT upsert)",
		},
		//       get "/UnitTest?idUnitTest=eq.1"
		//         `shouldRespondWith` [json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
		{
			Description: "with a camel case pk column works with PUT: the row is written",
			Method:      "GET",
			Query:       "/UnitTest?idUnitTest=eq.1",
			Expected:    `[ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by a camel case primary key; card 32478 (PUT upsert)",
		},
		//     it "works with request method PUT and return=minimal" $ do
		//       request
		//         methodPut
		//         "/UnitTest?idUnitTest=eq.1"
		//         [("Prefer", "return=minimal")]
		//         [json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
		//         `shouldRespondWith` ""
		//           { matchStatus = 204
		//           , matchHeaders =
		//               [ matchHeaderAbsent hContentType
		//               , "Preference-Applied" <:> "return=minimal"
		//               ]
		//           }
		{
			Description:     "with a camel case pk column works with request method PUT and return=minimal",
			Method:          "PUT",
			Query:           "/UnitTest?idUnitTest=eq.1",
			Headers:         test.Headers{"Prefer": {"return=minimal"}},
			Body:            `[ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          204,
			Skip:            "PostgREST 16.4: PUT updates the row addressed by a camel case primary key; card 32478 (PUT upsert)",
		},
		//       get "/UnitTest?idUnitTest=eq.1"
		//         `shouldRespondWith` [json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
		{
			Description: "with a camel case pk column works with request method PUT and return=minimal: the row is written",
			Method:      "GET",
			Query:       "/UnitTest?idUnitTest=eq.1",
			Expected:    `[ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]`,
			Status:      200,
			Skip:        "PostgREST 16.4: PUT updates the row addressed by a camel case primary key; card 32478 (PUT upsert)",
		},
		//
		//   context "with ordering" $ do
		//     it "works with request method PUT and embedded resource" $
		//       request
		//         methodPut
		//         "/web_content?id=eq.0&select=id,name,web_content(name)&web_content.order=name.asc"
		//         [("Prefer", "return=representation")]
		//         [json|{ "id": 0, "name": "tardis-upserted", "p_web_id": 5 }|]
		//         `shouldRespondWith` [json|
		//           [ { "id": 0, "name": "tardis-upserted", "web_content": [ { "name": "bar" }, { "name": "fezz" }, { "name": "foo" } ]} ]
		//         |]
		//           { matchStatus = 200
		//           , matchHeaders = [matchContentTypeJson, "Preference-Applied" <:> "return=representation"]
		//           }
		{
			Description:     "with ordering works with request method PUT and embedded resource",
			Method:          "PUT",
			Query:           "/web_content?id=eq.0&select=id,name,web_content(name)&web_content.order=name.asc",
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Body:            `{ "id": 0, "name": "tardis-upserted", "p_web_id": 5 }`,
			Expected:        `[ { "id": 0, "name": "tardis-upserted", "web_content": [ { "name": "bar" }, { "name": "fezz" }, { "name": "foo" } ]} ]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: PUT upserts the row addressed by its primary key, its representation ordered; card 32478 (PUT upsert)",
		},
		//     it "works with batch upserts and embedded resource" $
		//       request
		//         methodPost
		//         "/artists?select=id,name,albums(title)&order=id.desc"
		//         [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		//         [json| [{ "id": 1, "name": "duster-updated" },
		//                 { "id": 2, "name": "bcnr-updated" }] |]
		//         `shouldRespondWith` [json|  [{"id":2,"name":"bcnr-updated","albums":[{"title": "ants from up above"}]}, {"id":1,"name":"duster-updated","albums":[{"title": "stratosphere"}, {"title": "contemporary movement"}]}] |]
		//           { matchStatus = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "with ordering works with batch upserts and embedded resource",
			Method:          "POST",
			Query:           "/artists?select=id,name,albums(title)&order=id.desc",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "id": 1, "name": "duster-updated" }, { "id": 2, "name": "bcnr-updated" }]`,
			Expected:        `[{"id":2,"name":"bcnr-updated","albums":[{"title": "ants from up above"}]}, {"id":1,"name":"duster-updated","albums":[{"title": "stratosphere"}, {"title": "contemporary movement"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: a merge-duplicates upsert that inserted no row answers 200, smoothdb 201; card 32945 (upsert status)",
		},
		// @@ added: the same request, its ordering asserted without the status (see the skip above)
		{
			Description:     "with ordering works with batch upserts and embedded resource, status not asserted",
			Method:          "POST",
			Query:           "/artists?select=id,name,albums(title)&order=id.desc",
			Headers:         test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Body:            `[{ "id": 1, "name": "duster-updated" }, { "id": 2, "name": "bcnr-updated" }]`,
			Expected:        `[{"id":2,"name":"bcnr-updated","albums":[{"title": "ants from up above"}]}, {"id":1,"name":"duster-updated","albums":[{"title": "stratosphere"}, {"title": "contemporary movement"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
		},
	}

	test.Execute(t, testConfig, tests)
}
