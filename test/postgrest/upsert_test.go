package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Upsert(t *testing.T) {

	tests := []test.Test{
		// 	context "with POST" $ do
		// 	context "when Prefer: resolution=merge-duplicates is specified" $ do
		// 	  it "INSERTs and UPDATEs rows on pk conflict" $
		// 		request methodPost "/tiobe_pls" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json| [
		// 			{ "name": "Javascript", "rank": 6 },
		// 			{ "name": "Java", "rank": 2 },
		// 			{ "name": "C", "rank": 1 }
		// 		  ]|] `shouldRespondWith` [json| [
		// 			{ "name": "Javascript", "rank": 6 },
		// 			{ "name": "Java", "rank": 2 },
		// 			{ "name": "C", "rank": 1 }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates", matchContentTypeJson]
		// 		  }
		{
			Description: "INSERTs and UPDATEs rows on pk conflict",
			Method:      "POST",
			Query:       "/tiobe_pls",
			Body: `[
						{ "name": "Javascript", "rank": 6 },
						{ "name": "Java", "rank": 2 },
						{ "name": "C", "rank": 1 }
					]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "name": "Javascript", "rank": 6 },
				 			{ "name": "Java", "rank": 2 },
				 			{ "name": "C", "rank": 1 }
						]`,
			Status: 201,
		},
		// 	  it "INSERTs and UPDATEs row on composite pk conflict" $
		// 		request methodPost "/employees" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json| [
		// 			{ "first_name": "Frances M.", "last_name": "Roe", "salary": "30000" },
		// 			{ "first_name": "Peter S.", "last_name": "Yang", "salary": 42000 }
		// 		  ]|] `shouldRespondWith` [json| [
		// 			{ "first_name": "Frances M.", "last_name": "Roe", "salary": "$30,000.00", "company": "One-Up Realty", "occupation": "Author" },
		// 			{ "first_name": "Peter S.", "last_name": "Yang", "salary": "$42,000.00", "company": null, "occupation": null }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates", matchContentTypeJson]
		// 		  }
		{
			Description: "INSERTs and UPDATEs rows on composite pk conflict",
			Method:      "POST",
			Query:       "/employees",
			Body: `[
							{ "first_name": "Frances M.", "last_name": "Roe", "salary": 30000 },
				 			{ "first_name": "Peter S.", "last_name": "Yang", "salary": 42000 }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "first_name": "Frances M.", "last_name": "Roe", "salary": 30000, "company": "One-Up Realty", "occupation": "Author" },
				 			{ "first_name": "Peter S.", "last_name": "Yang", "salary": 42000, "company": null, "occupation": null }
				 		  ]`,
			Status: 201,
		},
		// 	  when (actualPgVersion >= pgVersion110) $
		// 		it "INSERTs and UPDATEs rows on composite pk conflict for partitioned tables" $
		// 		  request methodPost "/car_models" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 			[json| [
		// 			  { "name": "Murcielago", "year": 2001, "car_brand_name": null},
		// 			  { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
		// 			]|] `shouldRespondWith` [json| [
		// 			  { "name": "Murcielago", "year": 2001, "car_brand_name": null},
		// 			  { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
		// 			]|]
		// 			{ matchStatus = 201
		// 			, matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates", matchContentTypeJson]
		// 			}
		{
			Description: "INSERTs and UPDATEs rows on composite pk conflict for partitioned tables",
			Method:      "POST",
			Query:       "/car_models",
			Body: `[
				 			  { "name": "Murcielago", "year": 2001, "car_brand_name": null},
				 			  { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
				 			]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			  { "name": "Murcielago", "year": 2001, "car_brand_name": null},
				 			  { "name": "Roma", "year": 2021, "car_brand_name": "Ferrari" }
				 			]`,
			Status: 201,
		},
		// 	  it "succeeds when the payload has no elements" $
		// 		request methodPost "/articles" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json|[]|] `shouldRespondWith`
		// 		  [json|[]|] { matchStatus = 201 , matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds when the payload has no elements",
			Method:      "POST",
			Query:       "/articles",
			Body:        `[]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected:    `[]`,
			Status:      201,
		},
		// 	  it "INSERTs and UPDATEs rows on single unique key conflict" $
		// 		request methodPost "/single_unique?on_conflict=unique_key" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json| [
		// 			{ "unique_key": 1, "value": "B" },
		// 			{ "unique_key": 2, "value": "C" }
		// 		  ]|] `shouldRespondWith` [json| [
		// 			{ "unique_key": 1, "value": "B" },
		// 			{ "unique_key": 2, "value": "C" }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates", matchContentTypeJson]
		// 		  }
		{
			Description: "INSERTs and UPDATEs rows on single unique key conflict",
			Method:      "POST",
			Query:       "/single_unique?on_conflict=unique_key",
			Body: `[
				 			{ "unique_key": 1, "value": "B" },
				 			{ "unique_key": 2, "value": "C" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "unique_key": 1, "value": "B" },
				 			{ "unique_key": 2, "value": "C" }
				 		  ]`,
			Status: 201,
		},
		// 	  it "INSERTs and UPDATEs rows on compound unique keys conflict" $
		// 		request methodPost "/compound_unique?on_conflict=key1,key2" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json| [
		// 			{ "key1": 1, "key2": 1, "value": "B" },
		// 			{ "key1": 1, "key2": 2, "value": "C" }
		// 		  ]|] `shouldRespondWith` [json| [
		// 			{ "key1": 1, "key2": 1, "value": "B" },
		// 			{ "key1": 1, "key2": 2, "value": "C" }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates", matchContentTypeJson]
		// 		  }
		{
			Description: "INSERTs and UPDATEs rows on compound unique key conflict",
			Method:      "POST",
			Query:       "/compound_unique?on_conflict=key1,key2",
			Body: `[
				 			{ "key1": 1, "key2": 1, "value": "B" },
				 			{ "key1": 1, "key2": 2, "value": "C" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "key1": 1, "key2": 1, "value": "B" },
				 			{ "key1": 1, "key2": 2, "value": "C" }
				 		  ]`,
			Status: 201,
		},
		// @@added - with quotes
		{
			Description: "INSERTs and UPDATEs rows on compound unique key conflict",
			Method:      "POST",
			Query:       "/compound_unique?on_conflict=\"key1\",\"key2\"",
			Body: `[
				 			{ "key1": 1, "key2": 1, "value": "B" },
				 			{ "key1": 1, "key2": 2, "value": "C" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "key1": 1, "key2": 1, "value": "B" },
				 			{ "key1": 1, "key2": 2, "value": "C" }
				 		  ]`,
			Status: 201,
		},
		// 	context "when Prefer: resolution=ignore-duplicates is specified" $ do
		// 	  it "INSERTs and ignores rows on pk conflict" $
		// 		request methodPost "/tiobe_pls" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		  [json|[
		// 			{ "name": "PHP", "rank": 9 },
		// 			{ "name": "Python", "rank": 10 }
		// 		  ]|] `shouldRespondWith` [json|[
		// 			{ "name": "PHP", "rank": 9 }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates", matchContentTypeJson]
		// 		  }
		{
			Description: "INSERTs and ignores rows on pk conflict",
			Method:      "POST",
			Query:       "/tiobe_pls",
			Body: `[
				 			{ "name": "PHP", "rank": 9 },
				 			{ "name": "Python", "rank": 10 }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected: `[
				 			{ "name": "PHP", "rank": 9 }
				 		  ]`,
			Status: 201,
		},
		// 	  it "INSERTs and ignores rows on composite pk conflict" $
		// 		request methodPost "/employees" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		  [json|[
		// 			{ "first_name": "Daniel B.", "last_name": "Lyon", "salary": "72000", "company": null, "occupation": null },
		// 			{ "first_name": "Sara M.", "last_name": "Torpey", "salary": 60000, "company": "Burstein-Applebee", "occupation": "Soil scientist" }
		// 		  ]|] `shouldRespondWith` [json|[
		// 			{ "first_name": "Sara M.", "last_name": "Torpey", "salary": "$60,000.00", "company": "Burstein-Applebee", "occupation": "Soil scientist" }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates", matchContentTypeJson]
		// 		  }

		// 	  when (actualPgVersion >= pgVersion110) $
		// 		it "INSERTs and ignores rows on composite pk conflict for partitioned tables" $
		// 		  request methodPost "/car_models" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 			[json| [
		// 			  { "name": "Murcielago", "year": 2001, "car_brand_name": "Ferrari" },
		// 			  { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
		// 			]|] `shouldRespondWith` [json| [
		// 			  { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
		// 			]|]
		// 			{ matchStatus = 201
		// 			, matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates", matchContentTypeJson]
		// 			}
		{
			Description: "INSERTs and ignores rows on composite pk conflict for partitioned tables",
			Method:      "POST",
			Query:       "/car_models",
			Body: `[
				 			  { "name": "Murcielago", "year": 2001, "car_brand_name": "Ferrari" },
				 			  { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
				 			]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected: `[
				 			  { "name": "Huracán", "year": 2021, "car_brand_name": "Lamborghini" }
				 			]`,
			Status: 201,
		},
		// 	  it "INSERTs and ignores rows on single unique key conflict" $
		// 		request methodPost "/single_unique?on_conflict=unique_key"
		// 			[("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 			[json| [
		// 			  { "unique_key": 1, "value": "B" },
		// 			  { "unique_key": 2, "value": "C" },
		// 			  { "unique_key": 3, "value": "D" }
		// 			]|]
		// 		  `shouldRespondWith`
		// 			[json| [
		// 			  { "unique_key": 2, "value": "C" },
		// 			  { "unique_key": 3, "value": "D" }
		// 			]|]
		// 			{ matchStatus = 201
		// 			, matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates"]
		// 			}
		{
			Description: "INSERTs and ignores rows on single unique key conflict",
			Method:      "POST",
			Query:       "/single_unique?on_conflict=unique_key",
			Body: `[
				 			{ "unique_key": 1, "value": "B" },
				 			{ "unique_key": 2, "value": "C" },
							{ "unique_key": 3, "value": "D" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected: `[
				 			{ "unique_key": 2, "value": "C" },
							{ "unique_key": 3, "value": "D" }
				 		  ]`,
			Status: 201,
		},
		// 	  it "INSERTs and UPDATEs rows on compound unique keys conflict" $
		// 		request methodPost "/compound_unique?on_conflict=key1,key2"
		// 			[("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 			[json| [
		// 			  { "key1": 1, "key2": 1, "value": "B" },
		// 			  { "key1": 1, "key2": 2, "value": "C" },
		// 			  { "key1": 1, "key2": 3, "value": "D" }
		// 			]|]
		// 		  `shouldRespondWith`
		// 			[json| [
		// 			  { "key1": 1, "key2": 2, "value": "C" },
		// 			  { "key1": 1, "key2": 3, "value": "D" }
		// 			]|]
		// 			{ matchStatus = 201
		// 			, matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates"]
		// 			}
		{
			Description: "INSERTs and ignores rows on compound unique key conflict",
			Method:      "POST",
			Query:       "/compound_unique?on_conflict=key1,key2",
			Body: `[
				 			  { "key1": 1, "key2": 1, "value": "B" },
				 			  { "key1": 1, "key2": 2, "value": "C" },
				 			  { "key1": 1, "key2": 3, "value": "D" }
				 			]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected: `[
				 			  { "key1": 1, "key2": 2, "value": "C" },
				 			  { "key1": 1, "key2": 3, "value": "D" }
				 			]`,
			Status: 201,
		},
		// 	it "succeeds if the table has only PK cols and no other cols" $ do
		// 	  request methodPost "/only_pk" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		[json|[ { "id": 1 }, { "id": 2 }, { "id": 3} ]|]
		// 		`shouldRespondWith`
		// 		[json|[ { "id": 3} ]|]
		// 		{ matchStatus = 201 ,
		// 		  matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates",
		// 		  matchContentTypeJson] }
		{
			Description: "succeeds if the table has only PK cols and no other cols",
			Method:      "POST",
			Query:       "/only_pk",
			Body:        `[ { "id": 1 }, { "id": 2 }, { "id": 3} ]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected:    `[ { "id": 3} ]`,
			Status:      201,
		},
		// 	  request methodPost "/only_pk" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		[json|[ { "id": 1 }, { "id": 2 }, { "id": 4} ]|]
		// 		`shouldRespondWith`
		// 		[json|[ { "id": 1 }, { "id": 2 }, { "id": 4} ]|]
		// 		{ matchStatus = 201 ,
		// 		  matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates",
		// 		  matchContentTypeJson] }
		{
			Description: "succeeds if the table has only PK cols and no other cols",
			Method:      "POST",
			Query:       "/only_pk",
			Body:        `[ { "id": 1 }, { "id": 2 }, { "id": 4} ]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected:    `[ { "id": 1 }, { "id": 2 }, { "id": 4} ]`,
			Status:      201,
		},
		// 	it "succeeds and ignores the Prefer: resolution header(no Preference-Applied present) if the table has no PK" $
		// 	  request methodPost "/no_pk" [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		[json|[ { "a": "1", "b": "0" } ]|]
		// 		`shouldRespondWith`
		// 		[json|[ { "a": "1", "b": "0" } ]|] { matchStatus = 201 , matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds and ignores the Prefer: resolution header(no Preference-Applied present) if the table has no PK",
			Method:      "POST",
			Query:       "/no_pk",
			Body:        `[ { "a": "1", "b": "0" } ]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected:    `[ { "a": "1", "b": "0" } ]`,
			Status:      201,
		},
		// 	it "succeeds if not a single resource is created" $ do
		// 	  request methodPost "/tiobe_pls" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		[json|[ { "name": "Java", "rank": 1 } ]|] `shouldRespondWith`
		// 		[json|[]|] { matchStatus = 201 , matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds if not a single resource is created",
			Method:      "POST",
			Query:       "/tiobe_pls",
			Body:        `[ { "name": "Java", "rank": 1 } ]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected:    `[]`,
			Status:      201,
		},
		// 	  request methodPost "/tiobe_pls" [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		[json|[ { "name": "Java", "rank": 1 }, { "name": "C", "rank": 2 } ]|] `shouldRespondWith`
		// 		[json|[]|] { matchStatus = 201 , matchHeaders = [matchContentTypeJson] }
		{
			Description: "succeeds if not a single resource is created",
			Method:      "POST",
			Query:       "/tiobe_pls",
			Body:        `[ { "name": "Java", "rank": 1 }, { "name": "C", "rank": 2 } ]`,
			Headers:     test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected:    `[]`,
			Status:      201,
		},
		//   context "with PUT" $ do
		// 	context "Restrictions" $ do
		// 	  it "fails if Range is specified" $
		// 		request methodPut "/tiobe_pls?name=eq.Javascript" [("Range", "0-5")]
		// 		  [json| [ { "name": "Javascript", "rank": 1 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Range header and limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		// 		  { matchStatus = 400 , matchHeaders = [matchContentTypeJson] }

		// 	  it "fails if limit is specified" $
		// 		put "/tiobe_pls?name=eq.Javascript&limit=1"
		// 		  [json| [ { "name": "Javascript", "rank": 1 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Range header and limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		// 		  { matchStatus = 400 , matchHeaders = [matchContentTypeJson] }

		// 	  it "fails if offset is specified" $
		// 		put "/tiobe_pls?name=eq.Javascript&offset=1"
		// 		  [json| [ { "name": "Javascript", "rank": 1 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Range header and limit/offset querystring parameters are not allowed for PUT","code":"PGRST114","details":null,"hint":null}|]
		// 		  { matchStatus = 400 , matchHeaders = [matchContentTypeJson] }

		// 	  it "rejects every other filter than pk cols eq's" $ do
		// 		put "/tiobe_pls?rank=eq.19"
		// 		  [json| [ { "name": "Go", "rank": 19 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }

		// 		put "/tiobe_pls?id=not.eq.Java"
		// 		  [json| [ { "name": "Go", "rank": 19 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }
		// 		put "/tiobe_pls?id=in.(Go)"
		// 		  [json| [ { "name": "Go", "rank": 19 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }
		// 		put "/tiobe_pls?and=(id.eq.Go)"
		// 		  [json| [ { "name": "Go", "rank": 19 } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }

		// 	  it "fails if not all composite key cols are specified as eq filters" $ do
		// 		put "/employees?first_name=eq.Susan"
		// 		  [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }
		// 		put "/employees?last_name=eq.Heidt"
		// 		  [json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		// 		  `shouldRespondWith`
		// 		  [json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		  { matchStatus = 405 , matchHeaders = [matchContentTypeJson] }

		// 	it "fails if the uri primary key doesn't match the payload primary key" $ do
		// 	  put "/tiobe_pls?name=eq.MATLAB" [json| [ { "name": "Perl", "rank": 17 } ]|]
		// 		`shouldRespondWith`
		// 		[json|{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}|]
		// 		{ matchStatus = 400 , matchHeaders = [matchContentTypeJson] }
		// 	  put "/employees?first_name=eq.Wendy&last_name=eq.Anderson"
		// 		[json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		// 		`shouldRespondWith`
		// 		[json|{"message":"Payload values do not match URL in primary key column(s)","code":"PGRST115","details":null,"hint":null}|]
		// 		{ matchStatus = 400 , matchHeaders = [matchContentTypeJson] }

		// 	it "fails if the table has no PK" $
		// 	  put "/no_pk?a=eq.one&b=eq.two" [json| [ { "a": "one", "b": "two" } ]|]
		// 		`shouldRespondWith`
		// 		[json|{"message":"Filters must include all and only primary key columns with 'eq' operators","code":"PGRST105","details":null,"hint":null}|]
		// 		{ matchStatus = 405 , matchHeaders = [matchContentTypeJson] }

		// 	context "Inserting row" $ do
		// 	  it "succeeds on table with single pk col" $ do
		// 		-- assert that the next request will indeed be an insert
		// 		get "/tiobe_pls?name=eq.Go"
		// 		  `shouldRespondWith`
		// 			[json|[]|]

		// 		request methodPut "/tiobe_pls?name=eq.Go"
		// 			[("Prefer", "return=representation")]
		// 			[json| [ { "name": "Go", "rank": 19 } ]|]
		// 		  `shouldRespondWith`
		// 			[json| [ { "name": "Go", "rank": 19 } ]|]

		// 	  it "succeeds on table with composite pk" $ do
		// 		-- assert that the next request will indeed be an insert
		// 		get "/employees?first_name=eq.Susan&last_name=eq.Heidt"
		// 		  `shouldRespondWith`
		// 			[json|[]|]

		// 		request methodPut "/employees?first_name=eq.Susan&last_name=eq.Heidt"
		// 			[("Prefer", "return=representation")]
		// 			[json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "48000", "company": "GEX", "occupation": "Railroad engineer" } ]|]
		// 		  `shouldRespondWith`
		// 			[json| [ { "first_name": "Susan", "last_name": "Heidt", "salary": "$48,000.00", "company": "GEX", "occupation": "Railroad engineer" } ]|]

		// 	  when (actualPgVersion >= pgVersion110) $
		// 		it "succeeds on a partitioned table with composite pk" $ do
		// 		  -- assert that the next request will indeed be an insert
		// 		  get "/car_models?name=eq.Supra&year=eq.2021"
		// 			`shouldRespondWith`
		// 			  [json|[]|]

		// 		  request methodPut "/car_models?name=eq.Supra&year=eq.2021"
		// 			  [("Prefer", "return=representation")]
		// 			  [json| [ { "name": "Supra", "year": 2021 } ]|]
		// 			`shouldRespondWith`
		// 			  [json| [ { "name": "Supra", "year": 2021, "car_brand_name": null } ]|]

		// 	  it "succeeds if the table has only PK cols and no other cols" $ do
		// 		-- assert that the next request will indeed be an insert
		// 		get "/only_pk?id=eq.10"
		// 		  `shouldRespondWith`
		// 			[json|[]|]

		// 		request methodPut "/only_pk?id=eq.10"
		// 			[("Prefer", "return=representation")]
		// 			[json|[ { "id": 10 } ]|]
		// 		  `shouldRespondWith`
		// 			[json|[ { "id": 10 } ]|]

		// 	context "Updating row" $ do
		// 	  it "succeeds on table with single pk col" $ do
		// 		-- assert that the next request will indeed be an update
		// 		get "/tiobe_pls?name=eq.Java"
		// 		  `shouldRespondWith`
		// 			[json|[ { "name": "Java", "rank": 1 } ]|]

		// 		request methodPut "/tiobe_pls?name=eq.Java"
		// 			[("Prefer", "return=representation")]
		// 			[json| [ { "name": "Java", "rank": 13 } ]|]
		// 		  `shouldRespondWith`
		// 			[json| [ { "name": "Java", "rank": 13 } ]|]

		// 	  -- TODO: move this to SingularSpec?
		// 	  it "succeeds if the payload has more than one row, but it only puts the first element" $ do
		// 		-- assert that the next request will indeed be an update
		// 		get "/tiobe_pls?name=eq.Java"
		// 		  `shouldRespondWith`
		// 			[json|[ { "name": "Java", "rank": 1 } ]|]

		// 		request methodPut "/tiobe_pls?name=eq.Java"
		// 			[("Prefer", "return=representation"), ("Accept", "application/vnd.pgrst.object+json")]
		// 			[json| [ { "name": "Java", "rank": 19 }, { "name": "Swift", "rank": 12 } ] |]
		// 		  `shouldRespondWith`
		// 			[json|{ "name": "Java", "rank": 19 }|]
		// 			{ matchHeaders = [matchContentTypeSingular] }

		// 	  it "succeeds on table with composite pk" $ do
		// 		-- assert that the next request will indeed be an update
		// 		get "/employees?first_name=eq.Frances M.&last_name=eq.Roe"
		// 		  `shouldRespondWith`
		// 			[json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$24,000.00", "company": "One-Up Realty", "occupation": "Author" } ]|]

		// 		request methodPut "/employees?first_name=eq.Frances M.&last_name=eq.Roe"
		// 			[("Prefer", "return=representation")]
		// 			[json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "60000", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]|]
		// 		  `shouldRespondWith`
		// 			[json| [ { "first_name": "Frances M.", "last_name": "Roe", "salary": "$60,000.00", "company": "Gamma Gas", "occupation": "Railroad engineer" } ]|]

		// 	  when (actualPgVersion >= pgVersion110) $
		// 		it "succeeds on a partitioned table with composite pk" $ do
		// 		  -- assert that the next request will indeed be an update
		// 		  get "/car_models?name=eq.DeLorean&year=eq.1981"
		// 			`shouldRespondWith`
		// 			  [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": "DMC" } ]|]

		// 		  request methodPut "/car_models?name=eq.DeLorean&year=eq.1981"
		// 			  [("Prefer", "return=representation")]
		// 			  [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]|]
		// 			`shouldRespondWith`
		// 			  [json| [ { "name": "DeLorean", "year": 1981, "car_brand_name": null } ]|]

		// 	  it "succeeds if the table has only PK cols and no other cols" $ do
		// 		-- assert that the next request will indeed be an update
		// 		get "/only_pk?id=eq.1"
		// 		  `shouldRespondWith`
		// 			[json|[ { "id": 1 } ]|]

		// 		request methodPut "/only_pk?id=eq.1"
		// 			[("Prefer", "return=representation")]
		// 			[json|[ { "id": 1 } ]|]
		// 		  `shouldRespondWith`
		// 			[json|[ { "id": 1 } ]|]

		// 	-- TODO: move this to SingularSpec?
		// 	it "works with return=representation and vnd.pgrst.object+json" $
		// 	  request methodPut "/tiobe_pls?name=eq.Ruby"
		// 		[("Prefer", "return=representation"), ("Accept", "application/vnd.pgrst.object+json")]
		// 		[json| [ { "name": "Ruby", "rank": 11 } ]|]
		// 		`shouldRespondWith` [json|{ "name": "Ruby", "rank": 11 }|] { matchHeaders = [matchContentTypeSingular] }

		//   context "with a camel case pk column" $ do
		// 	it "works with POST and merge-duplicates" $ do
		// 	  request methodPost "/UnitTest"
		// 		  [("Prefer", "return=representation"), ("Prefer", "resolution=merge-duplicates")]
		// 		  [json|[
		// 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		// 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		// 		  ]|]
		// 		`shouldRespondWith`
		// 		  [json|[
		// 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		// 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=merge-duplicates"]
		// 		  }
		{
			Description: "with a camel case pk column works with POST and merge-duplicates",
			Method:      "POST",
			Query:       "/UnitTest",
			Body: `[
				 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
				 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=merge-duplicates"}},
			Expected: `[
				 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
				 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
				 		  ]`,
			Status: 201,
		},
		// 	it "works with POST and ignore-duplicates headers" $ do
		// 	  request methodPost "/UnitTest"
		// 		  [("Prefer", "return=representation"), ("Prefer", "resolution=ignore-duplicates")]
		// 		  [json|[
		// 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
		// 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		// 		  ]|]
		// 		`shouldRespondWith`
		// 		  [json|[
		// 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
		// 		  ]|]
		// 		  { matchStatus = 201
		// 		  , matchHeaders = ["Preference-Applied" <:> "resolution=ignore-duplicates"]
		// 		  }
		{
			Description: "with a camel case pk column works with POST and ignore-duplicates",
			Method:      "POST",
			Query:       "/UnitTest",
			Body: `[
				 			{ "idUnitTest": 1, "nameUnitTest": "name of unittest 1" },
				 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
				 		  ]`,
			Headers: test.Headers{"Prefer": {"return=representation", "resolution=ignore-duplicates"}},
			Expected: `[
				 			{ "idUnitTest": 2, "nameUnitTest": "name of unittest 2" }
				 		  ]`,
			Status: 201,
		},
		// 	it "works with PUT" $ do
		// 	  put "/UnitTest?idUnitTest=eq.1"
		// 		  [json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
		// 		`shouldRespondWith`
		// 		  ""
		// 		  { matchStatus = 204
		// 		  , matchHeaders = [matchHeaderAbsent hContentType]
		// 		  }
		// 	  get "/UnitTest?idUnitTest=eq.1" `shouldRespondWith`
		// 		[json| [ { "idUnitTest": 1, "nameUnitTest": "unit test 1" } ]|]
	}

	test.Execute(t, testConfig, tests)
}
