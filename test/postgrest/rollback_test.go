package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from test/spec/Feature/RollbackSpec.hs. The spec builds its cases
// from four helpers, each run under three configurations; the Go helpers below
// mirror them, quoting the hspec they port. The suite runs with
// TransactionMode rollback-allow-override, PostgREST's baseCfg
// (tx-rollback-all = true, tx-allow-override = true): the "allowed" block
// runs, the other two need a TransactionMode TestMain does not set.
//
// PostgREST answers Preference-Applied: tx=... when it honours a Prefer tx;
// smoothdb sends no Preference-Applied. A case asserting it runs without that
// header, and a skipped twin (same request, " (Preference-Applied)" in the
// description) counts the missing header.

const (
	rollbackNoPrefApplied = "PostgREST 16.4: Preference-Applied names the tx preference it honoured; card 32479 (write headers)"
	rollbackNoPut         = "PostgREST 16.4: PUT upserts the row its primary key filter addresses; card 32478 (put)"
	rollbackDeferred      = "PostgREST 16.4: a transaction that ends in rollback checks the deferred constraints first (SET CONSTRAINTS ALL IMMEDIATE, MainTx.hs optionalRollback); no card"
	rollbackCommitError   = "PostgREST 16.4: a deferred constraint failing at COMMIT is the response, 409 (smoothdb commits after the response is sent and only logs it); card 35458 (tx after response)"
	rollbackEmptyBody     = "PostgREST 16.4: POST /rpc with an empty body calls the function with its defaults (smoothdb answers EOF); no card"
)

// -- two helpers functions to make sure that each test can setup and cleanup properly
//
// -- creates Item to work with for PATCH and DELETE
// postItem =
//
//	request
//	  methodPost
//	  "/items"
//	  [("Prefer", "tx=commit"), ("Prefer", "resolution=ignore-duplicates")]
//	  [json|{"id":0}|]
//	  `shouldRespondWith` ""
//	    { matchStatus = 201
//	    , matchHeaders =
//	        [ matchHeaderAbsent hContentType
//	        , "Content-Length" <:> "0"
//	        ]
//	    }
func rbPostItem(ctx, skip string) test.Test {
	return test.Test{
		Description:     ctx + "postItem",
		Method:          "POST",
		Query:           "/items",
		Headers:         test.Headers{"Prefer": {"tx=commit", "resolution=ignore-duplicates"}},
		Body:            `{"id":0}`,
		ExpectedEmpty:   true,
		ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "0"},
		Status:          201,
		Skip:            skip,
	}
}

// -- removes Items left over from POST, PUT, and PATCH
// deleteItems =
//
//	request
//	  methodDelete
//	  "/items?id=lte.0"
//	  [("Prefer", "tx=commit")]
//	  ""
//	  `shouldRespondWith` ""
//	    { matchStatus = 204
//	    , matchHeaders =
//	        [ matchHeaderAbsent hContentType
//	        , matchHeaderAbsent hContentLength
//	        ]
//	    }
func rbDeleteItems(ctx, skip string) test.Test {
	return test.Test{
		Description:     ctx + "deleteItems",
		Method:          "DELETE",
		Query:           "/items?id=lte.0",
		Headers:         test.Headers{"Prefer": {"tx=commit"}},
		ExpectedEmpty:   true,
		ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": ""},
		Status:          204,
		Skip:            skip,
	}
}

// rbGet is the plain `get` of the spec, which checks what a mutation left.
func rbGet(ctx, query, expected, skip string) test.Test {
	return test.Test{
		Description: ctx + "get " + query,
		Query:       query,
		Expected:    expected,
		Status:      200,
		Skip:        skip,
	}
}

// preferDefault = [("Prefer", "return=representation")]
//
// preferCommit = [("Prefer", "return=representation"), ("Prefer", "tx=commit")]
//
// preferRollback = [("Prefer", "return=representation"), ("Prefer", "tx=rollback")]
//
// withoutPreferenceApplied = []
//
// withPreferenceCommitApplied = [matchHeaderValuePresent "Preference-Applied" "tx=commit"]
//
// withPreferenceRollbackApplied = [matchHeaderValuePresent "Preference-Applied" "tx=rollback"]
var (
	preferDefault  = []string{"return=representation"}
	preferCommit   = []string{"return=representation", "tx=commit"}
	preferRollback = []string{"return=representation", "tx=rollback"}
)

// rbBlock is one describe block: its description, the Prefer values of its
// requests, the tx preference PostgREST reports applied ("" for none), when
// the block needs a configuration the suite lacks, why it is skipped, and
// why its deferred constraint case is, when smoothdb fails it.
type rbBlock struct {
	ctx          string
	prefer       []string
	applied      string
	skip         string
	deferredSkip string
}

func (b rbBlock) skipOr(reason string) string {
	if b.skip != "" {
		return b.skip
	}
	return reason
}

// withApplied returns the case and, when PostgREST would report the tx
// preference applied, its skipped twin asserting Preference-Applied: appliedValue
// is the whole header PostgREST sends (a mutation lists return= too).
func (b rbBlock) withApplied(tc test.Test, appliedValue string) []test.Test {
	tc.Skip = b.skipOr(tc.Skip)
	if b.applied == "" {
		return []test.Test{tc}
	}
	twin := tc
	twin.Description += " (Preference-Applied)"
	twin.ExpectedHeaders = map[string]string{"Preference-Applied": appliedValue}
	if twin.Skip == "" {
		twin.Skip = rollbackNoPrefApplied
	}
	return []test.Test{tc, twin}
}

// shouldRespondToReads reqHeaders respHeaders = do
func (b rbBlock) shouldRespondToReads() []test.Test {
	var tests []test.Test
	//   it "responds to GET" $ do
	//     request
	//       methodGet
	//       "/items?id=eq.1"
	//       reqHeaders
	//       ""
	//       `shouldRespondWith` [json|[{"id":1}]|]
	//         { matchHeaders = respHeaders
	//         }
	tests = append(tests, b.withApplied(test.Test{
		Description: b.ctx + "responds to GET",
		Query:       "/items?id=eq.1",
		Headers:     test.Headers{"Prefer": b.prefer},
		Expected:    `[{"id":1}]`,
		Status:      200,
	}, b.applied)...)
	//   it "responds to HEAD" $ do
	//     request
	//       methodHead
	//       "/items?id=eq.1"
	//       reqHeaders
	//       ""
	//       `shouldRespondWith` ""
	//         { matchHeaders = matchContentTypeJson : respHeaders
	//         }
	tests = append(tests, b.withApplied(test.Test{
		Description:     b.ctx + "responds to HEAD",
		Method:          "HEAD",
		Query:           "/items?id=eq.1",
		Headers:         test.Headers{"Prefer": b.prefer},
		ExpectedEmpty:   true,
		ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
		Status:          200,
	}, b.applied)...)
	//   it "responds to GET on RPC" $ do
	//     request
	//       methodGet
	//       "/rpc/search?id=1"
	//       reqHeaders
	//       ""
	//       `shouldRespondWith` [json|[{"id":1}]|]
	//         { matchHeaders = respHeaders
	//         }
	tests = append(tests, b.withApplied(test.Test{
		Description: b.ctx + "responds to GET on RPC",
		Query:       "/rpc/search?id=1",
		Headers:     test.Headers{"Prefer": b.prefer},
		Expected:    `[{"id":1}]`,
		Status:      200,
	}, b.applied)...)
	//   it "responds to POST on RPC" $ do
	//     request
	//       methodPost
	//       "/rpc/search"
	//       reqHeaders
	//       [json|{"id":1}|]
	//       `shouldRespondWith` [json|[{"id":1}]|]
	//         { matchHeaders = respHeaders
	//         }
	tests = append(tests, b.withApplied(test.Test{
		Description: b.ctx + "responds to POST on RPC",
		Method:      "POST",
		Query:       "/rpc/search",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":1}`,
		Expected:    `[{"id":1}]`,
		Status:      200,
	}, b.applied)...)
	return tests
}

// shouldRaiseExceptions reqHeaders respHeaders = do
//
// Exceptions are always without preference applied (the spec passes
// withoutPreferenceApplied in every block), so these have no twin.
func (b rbBlock) shouldRaiseExceptions() []test.Test {
	const conflict = `{"subsystem":"database","message":"duplicate key value violates unique constraint \"deferrable_unique_constraint_col_key\"","code":"23505","hint":"","details":"Key (col)=(1) already exists.","position":0}`
	return []test.Test{
		//   it "raises immediate constraints" $ do
		//     request
		//       methodPost
		//       "/rpc/raise_constraint"
		//       reqHeaders
		//       ""
		//       `shouldRespondWith` [json|{
		//           "hint":null,
		//           "details":"Key (col)=(1) already exists.",
		//           "code":"23505",
		//           "message":"duplicate key value violates unique constraint \"deferrable_unique_constraint_col_key\""
		//         }|]
		//         { matchStatus = 409
		//         , matchHeaders = respHeaders
		//         }
		{
			Description: b.ctx + "raises immediate constraints",
			Method:      "POST",
			Query:       "/rpc/raise_constraint",
			Headers:     test.Headers{"Prefer": b.prefer},
			Expected:    conflict,
			Status:      409,
			Skip:        b.skipOr(rollbackEmptyBody),
		},
		//   it "raises deferred constraints" $ do
		//     request
		//       methodPost
		//       "/rpc/raise_constraint"
		//       reqHeaders
		//       [json|{"deferred": true}|]
		//       `shouldRespondWith` [json|{
		//           "hint":null,
		//           "details":"Key (col)=(1) already exists.",
		//           "code":"23505",
		//           "message":"duplicate key value violates unique constraint \"deferrable_unique_constraint_col_key\""
		//         }|]
		//         { matchStatus = 409
		//         , matchHeaders = respHeaders
		//         }
		{
			Description: b.ctx + "raises deferred constraints",
			Method:      "POST",
			Query:       "/rpc/raise_constraint",
			Headers:     test.Headers{"Prefer": b.prefer},
			Body:        `{"deferred": true}`,
			Expected:    conflict,
			Status:      409,
			Skip:        b.skipOr(b.deferredSkip),
		},
	}
}

// shouldPersistMutations reqHeaders respHeaders = do
func (b rbBlock) shouldPersistMutations() []test.Test {
	var tests []test.Test
	mutation := "return=representation, " + b.applied
	//   it "does persist post" $ do
	//     request
	//       methodPost
	//       "/items"
	//       reqHeaders
	//       [json|{"id":0}|]
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchStatus = 201
	//         , matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//     deleteItems
	ctx := b.ctx + "does persist post: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "post",
		Method:      "POST",
		Query:       "/items",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":0}`,
		Expected:    `[{"id":0}]`,
		Status:      201,
	}, mutation)...)
	tests = append(tests,
		rbGet(ctx, "/items?id=eq.0", `[{"id":0}]`, b.skip),
		rbDeleteItems(ctx, b.skip))
	//   it "does persist put" $ do
	//     request
	//       methodPut
	//       "/items?id=eq.0"
	//       reqHeaders
	//       [json|{"id":0}|]
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchStatus = 201
	//         , matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//     deleteItems
	ctx = b.ctx + "does persist put: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "put",
		Method:      "PUT",
		Skip:        rollbackNoPut,
		Query:       "/items?id=eq.0",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":0}`,
		Expected:    `[{"id":0}]`,
		Status:      201,
	}, mutation)...)
	tests = append(tests,
		rbGet(ctx, "/items?id=eq.0", `[{"id":0}]`, b.skipOr(rollbackNoPut)),
		rbDeleteItems(ctx, b.skipOr(rollbackNoPut)))
	//   it "does persist patch" $ do
	//     postItem
	//     request
	//       methodPatch
	//       "/items?id=eq.0"
	//       reqHeaders
	//       [json|{"id":-1}|]
	//       `shouldRespondWith` [json|[{"id":-1}]|]
	//         { matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[]|]
	//     get "/items?id=eq.-1"
	//       `shouldRespondWith` [json|[{"id":-1}]|]
	//     deleteItems
	ctx = b.ctx + "does persist patch: "
	tests = append(tests, rbPostItem(ctx, b.skip))
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "patch",
		Method:      "PATCH",
		Query:       "/items?id=eq.0",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":-1}`,
		Expected:    `[{"id":-1}]`,
		Status:      200,
	}, mutation)...)
	tests = append(tests,
		rbGet(ctx, "/items?id=eq.0", `[]`, b.skip),
		rbGet(ctx, "/items?id=eq.-1", `[{"id":-1}]`, b.skip),
		rbDeleteItems(ctx, b.skip))
	//   it "does persist delete" $ do
	//     postItem
	//     request
	//       methodDelete
	//       "/items?id=eq.0"
	//       reqHeaders
	//       ""
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[]|]
	ctx = b.ctx + "does persist delete: "
	tests = append(tests, rbPostItem(ctx, b.skip))
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "delete",
		Method:      "DELETE",
		Query:       "/items?id=eq.0",
		Headers:     test.Headers{"Prefer": b.prefer},
		Expected:    `[{"id":0}]`,
		Status:      200,
	}, mutation)...)
	tests = append(tests, rbGet(ctx, "/items?id=eq.0", `[]`, b.skip))
	return tests
}

// shouldNotPersistMutations reqHeaders respHeaders = do
func (b rbBlock) shouldNotPersistMutations() []test.Test {
	var tests []test.Test
	mutation := "return=representation, " + b.applied
	//   it "does not persist post" $ do
	//     request
	//       methodPost
	//       "/items"
	//       reqHeaders
	//       [json|{"id":0}|]
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchStatus = 201
	//         , matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[]|]
	ctx := b.ctx + "does not persist post: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "post",
		Method:      "POST",
		Query:       "/items",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":0}`,
		Expected:    `[{"id":0}]`,
		Status:      201,
	}, mutation)...)
	tests = append(tests, rbGet(ctx, "/items?id=eq.0", `[]`, b.skip))
	//   it "does not persist put" $ do
	//     request
	//       methodPut
	//       "/items?id=eq.0"
	//       reqHeaders
	//       [json|{"id":0}|]
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchStatus = 201
	//         , matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[]|]
	ctx = b.ctx + "does not persist put: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "put",
		Method:      "PUT",
		Skip:        rollbackNoPut,
		Query:       "/items?id=eq.0",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":0}`,
		Expected:    `[{"id":0}]`,
		Status:      201,
	}, mutation)...)
	tests = append(tests, rbGet(ctx, "/items?id=eq.0", `[]`, b.skipOr(rollbackNoPut)))
	//   it "does not persist patch" $ do
	//     request
	//       methodPatch
	//       "/items?id=eq.1"
	//       reqHeaders
	//       [json|{"id":0}|]
	//       `shouldRespondWith` [json|[{"id":0}]|]
	//         { matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.0"
	//       `shouldRespondWith` [json|[]|]
	//     get "items?id=eq.1"
	//       `shouldRespondWith` [json|[{"id":1}]|]
	ctx = b.ctx + "does not persist patch: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "patch",
		Method:      "PATCH",
		Query:       "/items?id=eq.1",
		Headers:     test.Headers{"Prefer": b.prefer},
		Body:        `{"id":0}`,
		Expected:    `[{"id":0}]`,
		Status:      200,
	}, mutation)...)
	tests = append(tests,
		rbGet(ctx, "/items?id=eq.0", `[]`, b.skip),
		rbGet(ctx, "/items?id=eq.1", `[{"id":1}]`, b.skip))
	//   it "does not persist delete" $ do
	//     request
	//       methodDelete
	//       "/items?id=eq.1"
	//       reqHeaders
	//       ""
	//       `shouldRespondWith` [json|[{"id":1}]|]
	//         { matchHeaders = respHeaders
	//         }
	//     get "/items?id=eq.1"
	//       `shouldRespondWith` [json|[{"id":1}]|]
	ctx = b.ctx + "does not persist delete: "
	tests = append(tests, b.withApplied(test.Test{
		Description: ctx + "delete",
		Method:      "DELETE",
		Query:       "/items?id=eq.1",
		Headers:     test.Headers{"Prefer": b.prefer},
		Expected:    `[{"id":1}]`,
		Status:      200,
	}, mutation)...)
	tests = append(tests, rbGet(ctx, "/items?id=eq.1", `[{"id":1}]`, b.skip))
	return tests
}

// allowed :: SpecWithConfig
// allowed withConfig = withConfig baseCfg $ describe "tx-allow-override = true" $ do
//
//	describe "without Prefer tx" $ do
//	  preferDefault `shouldRespondToReads` withoutPreferenceApplied
//	  preferDefault `shouldNotPersistMutations` withoutPreferenceApplied
//	  preferDefault `shouldRaiseExceptions` withoutPreferenceApplied
//
//	describe "Prefer tx=commit" $ do
//	  preferCommit `shouldRespondToReads` withPreferenceCommitApplied
//	  preferCommit `shouldPersistMutations` withPreferenceCommitApplied
//	  -- Exceptions are always without preference applied,
//	  -- because they return before the end of the transaction.
//	  preferCommit `shouldRaiseExceptions` withoutPreferenceApplied
//
//	describe "Prefer tx=rollback" $ do
//	  preferRollback `shouldRespondToReads` withPreferenceRollbackApplied
//	  preferRollback `shouldNotPersistMutations` withPreferenceRollbackApplied
//	  -- Exceptions are always without preference applied,
//	  -- because they return before the end of the transaction.
//	  preferRollback `shouldRaiseExceptions` withoutPreferenceApplied
func TestPostgREST_RollbackAllowed(t *testing.T) {
	var tests []test.Test

	b := rbBlock{ctx: "tx-allow-override = true, without Prefer tx: ", prefer: preferDefault, deferredSkip: rollbackDeferred}
	tests = append(tests, b.shouldRespondToReads()...)
	tests = append(tests, b.shouldNotPersistMutations()...)
	tests = append(tests, b.shouldRaiseExceptions()...)

	b = rbBlock{ctx: "tx-allow-override = true, Prefer tx=commit: ", prefer: preferCommit, applied: "tx=commit", deferredSkip: rollbackCommitError}
	tests = append(tests, b.shouldRespondToReads()...)
	tests = append(tests, b.shouldPersistMutations()...)
	b.applied = ""
	tests = append(tests, b.shouldRaiseExceptions()...)

	b = rbBlock{ctx: "tx-allow-override = true, Prefer tx=rollback: ", prefer: preferRollback, applied: "tx=rollback", deferredSkip: rollbackDeferred}
	tests = append(tests, b.shouldRespondToReads()...)
	tests = append(tests, b.shouldNotPersistMutations()...)
	b.applied = ""
	tests = append(tests, b.shouldRaiseExceptions()...)

	test.Execute(t, testConfig, tests)
}

// disallowed :: SpecWithConfig
// disallowed withConfig = withConfig
//
//	( baseCfg
//	    { configDbTxAllowOverride = False
//	    , configDbTxRollbackAll = False
//	    }
//	)
//	$ describe "tx-rollback-all = false, tx-allow-override = false"
//	$ do
//	  describe "without Prefer tx" $ do
//	    preferDefault `shouldRespondToReads` withoutPreferenceApplied
//	    preferDefault `shouldPersistMutations` withoutPreferenceApplied
//	    preferDefault `shouldRaiseExceptions` withoutPreferenceApplied
//
//	  describe "Prefer tx=commit" $ do
//	    preferCommit `shouldRespondToReads` withoutPreferenceApplied
//	    preferCommit `shouldPersistMutations` withoutPreferenceApplied
//	    preferCommit `shouldRaiseExceptions` withoutPreferenceApplied
//
//	  describe "Prefer tx=rollback" $ do
//	    preferRollback `shouldRespondToReads` withoutPreferenceApplied
//	    preferRollback `shouldPersistMutations` withoutPreferenceApplied
//	    preferRollback `shouldRaiseExceptions` withoutPreferenceApplied
func TestPostgREST_RollbackDisallowed(t *testing.T) {
	const skip = "needs configuration Database.TransactionMode commit (PostgREST tx-rollback-all = false, tx-allow-override = false); TestMain sets rollback-allow-override"
	var tests []test.Test
	for _, b := range []rbBlock{
		{ctx: "tx-rollback-all = false, tx-allow-override = false, without Prefer tx: ", prefer: preferDefault, skip: skip},
		{ctx: "tx-rollback-all = false, tx-allow-override = false, Prefer tx=commit: ", prefer: preferCommit, skip: skip},
		{ctx: "tx-rollback-all = false, tx-allow-override = false, Prefer tx=rollback: ", prefer: preferRollback, skip: skip},
	} {
		tests = append(tests, b.shouldRespondToReads()...)
		tests = append(tests, b.shouldPersistMutations()...)
		tests = append(tests, b.shouldRaiseExceptions()...)
	}
	test.Execute(t, testConfig, tests)
}

// forced :: SpecWithConfig
// forced withConfig = withConfig
//
//	( baseCfg
//	    { configDbTxAllowOverride = False
//	    , configDbTxRollbackAll = True
//	    }
//	)
//	$ describe "tx-rollback-all = true, tx-allow-override = false"
//	$ do
//	  describe "without Prefer tx" $ do
//	    preferDefault `shouldRespondToReads` withoutPreferenceApplied
//	    preferDefault `shouldNotPersistMutations` withoutPreferenceApplied
//	    preferDefault `shouldRaiseExceptions` withoutPreferenceApplied
//
//	  describe "Prefer tx=commit" $ do
//	    preferCommit `shouldRespondToReads` withoutPreferenceApplied
//	    preferCommit `shouldNotPersistMutations` withoutPreferenceApplied
//	    preferCommit `shouldRaiseExceptions` withoutPreferenceApplied
//
//	  describe "Prefer tx=rollback" $ do
//	    preferRollback `shouldRespondToReads` withoutPreferenceApplied
//	    preferRollback `shouldNotPersistMutations` withoutPreferenceApplied
//	    preferRollback `shouldRaiseExceptions` withoutPreferenceApplied
func TestPostgREST_RollbackForced(t *testing.T) {
	const skip = "needs configuration Database.TransactionMode rollback (PostgREST tx-rollback-all = true, tx-allow-override = false); TestMain sets rollback-allow-override"
	var tests []test.Test
	for _, b := range []rbBlock{
		{ctx: "tx-rollback-all = true, tx-allow-override = false, without Prefer tx: ", prefer: preferDefault, skip: skip},
		{ctx: "tx-rollback-all = true, tx-allow-override = false, Prefer tx=commit: ", prefer: preferCommit, skip: skip},
		{ctx: "tx-rollback-all = true, tx-allow-override = false, Prefer tx=rollback: ", prefer: preferRollback, skip: skip},
	} {
		tests = append(tests, b.shouldRespondToReads()...)
		tests = append(tests, b.shouldNotPersistMutations()...)
		tests = append(tests, b.shouldRaiseExceptions()...)
	}
	test.Execute(t, testConfig, tests)
}
