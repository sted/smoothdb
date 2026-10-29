package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sted/smoothdb/test"
)

// The ports of PostgREST's JWT specs that need a configuration of their own:
// each runs its own server, which the test/postgrest suite cannot do (a server
// replaces the process-wide database engine of the suite's). They run on the
// pgrest fixtures of that suite (make prepare-postgrest-tests).

// postgrestSecret is generateSecret of PostgREST's SpecHelper.hs: the AuthSpec
// tokens below are signed with it.
const postgrestSecret = postgrestJWTSecret

// startJWTServer runs a server with its own JWT configuration on the pgrest
// fixtures, configured like PostgREST's baseCfg: anonymous requests run as
// postgrest_test_anonymous.
func startJWTServer(t *testing.T, port int, jwtConfig map[string]any) test.Config {
	t.Helper()
	c := map[string]any{
		"Address":                   fmt.Sprintf("localhost:%d", port),
		"AllowAnon":                 true,
		"Database.AnonRole":         "postgrest_test_anonymous",
		"Database.SchemaSearchPath": []string{"test"},
	}
	for k, v := range jwtConfig {
		c[k] = v
	}
	s := newTestServer(t, c)
	done := startTestServer(t, s)
	t.Cleanup(func() {
		s.Shutdown(context.Background())
		<-done
	})
	base := fmt.Sprintf("http://localhost:%d", port)
	return test.Config{
		BaseUrl:       base + "/api/pgrest",
		CommonHeaders: test.Headers{"Accept-Profile": {"test"}, "Content-Profile": {"test"}},
	}
}

func authHeader(token string) test.Headers {
	return test.Headers{"Authorization": {"Bearer " + token}}
}

func signPayload(t *testing.T, payload string) string {
	t.Helper()
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString([]byte(payload))
	sig, err := jwt.SigningMethodHS256.Sign(unsigned, []byte(postgrestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + enc.EncodeToString(sig)
}

// Feature/Auth/AudienceJwtSecretSpec.hs: spec, with jwt-aud = "youraudience".
func TestAudienceJwtSecretSpec(t *testing.T) {
	cfg := startJWTServer(t, 8141, map[string]any{
		"JWTSecret": postgrestSecret,
		"JWTAud":    "youraudience",
	})
	notInAudience := `{"code":"PGRST303","details":null,"hint":"","message":"JWT not in audience","position":0,"subsystem":"auth"}`
	aud := func(aud string) test.Headers {
		return authHeader(signPayload(t, `{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe"`+aud+`}`))
	}
	test.Execute(t, cfg, []test.Test{
		// it "succeeds when the audience claim matches" $ do
		//   ... "aud": "youraudience"  `shouldRespondWith` 200
		{Description: "succeeds when the audience claim matches", Query: "/authors_only",
			Headers: aud(`,"aud":"youraudience"`), Status: 200},
		// it "fails when the audience claim does not match" $ do
		//   ... `shouldRespondWith` [json|{"code":"PGRST303","details":null,"hint":null,"message":"JWT not in audience"}|]
		//     { matchStatus = 401 }
		{Description: "fails when the audience claim does not match", Query: "/authors_only",
			Headers: aud(`,"aud":"notyouraudience"`), Expected: notInAudience, Status: 401},
		// it "fails when the audience claim is empty"
		{Description: "fails when the audience claim is empty", Query: "/authors_only",
			Headers: aud(`,"aud":""`), Expected: notInAudience, Status: 401},
		// it "succeeds when the audience claim has 1 element and it matches"
		{Description: "succeeds when the audience claim has 1 element and it matches", Query: "/authors_only",
			Headers: aud(`,"aud":["youraudience"]`), Status: 200},
		// it "succeeds when the audience claim has more than 1 element and one matches"
		{Description: "succeeds when the audience claim has more than 1 element and one matches", Query: "/authors_only",
			Headers: aud(`,"aud":["notyouraudience","youraudience","anotheraudience"]`), Status: 200},
		// it "fails when the audience claim has 1 element and it doesn't match"
		{Description: "fails when the audience claim has 1 element and it doesn't match", Query: "/authors_only",
			Headers: aud(`,"aud":["notyouraudience"]`), Expected: notInAudience, Status: 401},
		// it "fails when the audience claim has more than 1 element and none matches"
		{Description: "fails when the audience claim has more than 1 element and none matches", Query: "/authors_only",
			Headers: aud(`,"aud":["notyouraudience","stillnotyouraudience","anotheraudience"]`), Expected: notInAudience, Status: 401},
		// it "ignores the audience claim and succeeds when it's empty"
		{Description: "ignores the audience claim and succeeds when it's empty", Query: "/authors_only",
			Headers: aud(`,"aud":[]`), Status: 200},
		// it "ignores the audience claim and succeeds when it's null"
		{Description: "ignores the audience claim and succeeds when it's null", Query: "/authors_only",
			Headers: aud(`,"aud":null`), Status: 200},
		// it "succeeds with a JWT with no audience claim"
		{Description: "succeeds with a JWT with no audience claim", Query: "/authors_only",
			Headers: aud(``), Status: 200},
		// it "succeeds without a JWT" $
		//   get "/has_count_column" `shouldRespondWith` 200
		{Description: "succeeds without a JWT", Query: "/has_count_column", Status: 200},
	})
}

// Feature/Auth/AuthSpec.hs: the cases about token verification, and the 30 s
// clock skew of Auth/Jwt.hs:70-76.
func TestAuthSpecJWT(t *testing.T) {
	cfg := startJWTServer(t, 8142, map[string]any{"JWTSecret": postgrestSecret})
	now := time.Now().Unix()
	at := func(claim string, offset int64) test.Headers {
		return authHeader(signPayload(t, fmt.Sprintf(`{"role":"postgrest_test_author","%s":%d}`, claim, now+offset)))
	}
	test.Execute(t, cfg, []test.Test{
		// it "succeeds with an unexpired token" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjk5OTk5OTk5OTksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.Dpss-QoLYjec5OTsOaAc3FNVsSjA89wACoV-0ra3ClA"
		//   request methodGet "/authors_only" [auth] "" `shouldRespondWith` 200
		{Description: "succeeds with an unexpired token", Query: "/authors_only",
			Headers: authHeader("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjk5OTk5OTk5OTksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.Dpss-QoLYjec5OTsOaAc3FNVsSjA89wACoV-0ra3ClA"),
			Status:  200},
		// it "succeeds with a valid iat claim in jwt" $ do
		//   currentTime <- liftIO $ relativeSeconds 0
		{Description: "succeeds with a valid iat claim in jwt", Query: "/authors_only",
			Headers: at("iat", 0), Status: 200},
		// it "fails with an expired token" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NDY2NzgxNDksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.f8__E6VQwYcDqwHmr9PG03uaZn8Zh1b0vbJ9DYS0AdM"
		//   `shouldRespondWith` [json| {"message":"JWT expired","code":"PGRST303","hint":null,"details":null} |]
		//     { matchStatus = 401 , matchHeaders = [ "WWW-Authenticate" <:> "Bearer error=\"invalid_token\", error_description=\"JWT expired\"" ] }
		{Description: "fails with an expired token", Query: "/authors_only",
			Headers:  authHeader("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NDY2NzgxNDksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.f8__E6VQwYcDqwHmr9PG03uaZn8Zh1b0vbJ9DYS0AdM"),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"JWT expired","position":0,"subsystem":"auth"}`, Status: 401},
		// it "hides tables from users with invalid JWT" $ do
		//   let auth = authHeaderJWT "ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0"
		//   `shouldRespondWith` [json| {"message":"Expected 3 parts in JWT; got 2","code":"PGRST301","hint":null,"details":null} |]
		{Description: "hides tables from users with invalid JWT", Query: "/authors_only",
			Headers:  authHeader("ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0"),
			Expected: `{"code":"PGRST301","details":null,"hint":"","message":"Expected 3 parts in JWT; got 2","position":0,"subsystem":"auth"}`, Status: 401},
		// it "fails when the exp claim is not a number" $ do
		//   ... `shouldRespondWith` [json|{"code":"PGRST303","details":null,"hint":null,"message":"The JWT 'exp' claim must be a number"}|]
		{Description: "fails when the exp claim is not a number", Query: "/authors_only",
			Headers:  authHeader(signPayload(t, `{"exp":"invalid","role":"postgrest_test_author"}`)),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"The JWT 'exp' claim must be a number","position":0,"subsystem":"auth"}`, Status: 401},
		// it "fails when the nbf claim is not a number"
		{Description: "fails when the nbf claim is not a number", Query: "/authors_only",
			Headers:  authHeader(signPayload(t, `{"nbf":"invalid","role":"postgrest_test_author"}`)),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"The JWT 'nbf' claim must be a number","position":0,"subsystem":"auth"}`, Status: 401},
		// it "fails when the iat claim is not a number"
		{Description: "fails when the iat claim is not a number", Query: "/authors_only",
			Headers:  authHeader(signPayload(t, `{"iat":"invalid","role":"postgrest_test_author"}`)),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"The JWT 'iat' claim must be a number","position":0,"subsystem":"auth"}`, Status: 401},
		// it "fails when the aud claim has a single value and it's not a string"
		{Description: "fails when the aud claim has a single value and it's not a string", Query: "/authors_only",
			Headers:  authHeader(signPayload(t, `{"aud":{"invalid":"value"},"role":"postgrest_test_author"}`)),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"The JWT 'aud' claim must be a string or an array of strings","position":0,"subsystem":"auth"}`, Status: 401},
		// it "fails when the aud claim is an array but it has non-string elements"
		{Description: "fails when the aud claim is an array but it has non-string elements", Query: "/authors_only",
			Headers:  authHeader(signPayload(t, `{"aud":[{"invalid":"value"},"test"],"role":"postgrest_test_author"}`)),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"The JWT 'aud' claim must be a string or an array of strings","position":0,"subsystem":"auth"}`, Status: 401},
		// the audience is not checked without JWTAud (AudienceJwtSecretSpec.disabledSpec)
		{Description: "ignores the audience claim without JWTAud", Query: "/authors_only",
			Headers: authHeader(signPayload(t, `{"exp":9999999999,"role":"postgrest_test_author","aud":"youraudience"}`)), Status: 200},
		// the 30 s clock skew
		{Description: "an exp 20 s ago is within the skew", Query: "/authors_only", Headers: at("exp", -20), Status: 200},
		{Description: "an exp 40 s ago is expired", Query: "/authors_only", Headers: at("exp", -40),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"JWT expired","position":0,"subsystem":"auth"}`, Status: 401},
		{Description: "an nbf 20 s ahead is within the skew", Query: "/authors_only", Headers: at("nbf", 20), Status: 200},
		{Description: "an nbf 40 s ahead is not yet valid", Query: "/authors_only", Headers: at("nbf", 40),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"JWT not yet valid","position":0,"subsystem":"auth"}`, Status: 401},
		{Description: "an iat 20 s ahead is within the skew", Query: "/authors_only", Headers: at("iat", 20), Status: 200},
		{Description: "an iat 40 s ahead is issued in the future", Query: "/authors_only", Headers: at("iat", 40),
			Expected: `{"code":"PGRST303","details":null,"hint":"","message":"JWT issued at future","position":0,"subsystem":"auth"}`, Status: 401},
	})
}

// test/io/test_config.py, test_role_claim_key and
// test_deprecated_role_claim_key: authorization follows the role the
// configured key finds in the claims (fixtures.yaml, roleclaims). The role
// also reaches request.jwt.claims, as PostgREST's PreQuery inserts it.
func TestRoleClaimKey(t *testing.T) {
	cases := []struct {
		port int
		key  string
	}{
		{8143, `$.postgrest.a_role`},
		{8144, `.postgrest.a_role`},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			cfg := startJWTServer(t, c.port, map[string]any{
				"JWTSecret":       postgrestSecret,
				"JWTRoleClaimKey": c.key,
			})
			// - key: '$.postgrest.a_role'
			//   data: { postgrest: { a_role: postgrest_test_author }, other: claims }
			//   expected_status: 200
			roleClaim := authHeader(signPayload(t, `{"postgrest":{"a_role":"postgrest_test_author"},"other":"claims"}`))
			// - key: '$.myRole' data: { role: postgrest_test_author, other: true } expected_status: 401
			// (here the key stays the same and the claim moves: the top-level
			// role is not the role claim, the request runs as the anon role)
			topLevel := authHeader(signPayload(t, `{"role":"postgrest_test_author","other":true}`))
			test.Execute(t, cfg, []test.Test{
				{Description: "the role claim key finds the role", Query: "/authors_only", Headers: roleClaim, Status: 200},
				{Description: "the role runs the request", Method: "POST", Query: "/rpc/get_current_user", Body: `{}`,
					Headers: roleClaim, Expected: `"postgrest_test_author"`, Status: 200},
				{Description: "a top-level role is not the role claim", Query: "/authors_only", Headers: topLevel, Status: 401},
				{Description: "a top-level role runs as the anon role", Method: "POST", Query: "/rpc/get_current_user", Body: `{}`,
					Headers: topLevel, Expected: `"postgrest_test_anonymous"`, Status: 200},
			})
		})
	}
}

// Feature/Auth/BinaryJwtSecretSpec.hs, with the secret given base64-encoded
// as jwt-secret-is-base64 takes it (test/io: secrets/*.b64).
func TestBinaryJwtSecretSpec(t *testing.T) {
	cfg := startJWTServer(t, 8145, map[string]any{
		"JWTSecret":         base64.StdEncoding.EncodeToString([]byte(postgrestSecret)),
		"JWTSecretIsBase64": true,
	})
	test.Execute(t, cfg, []test.Test{
		// it "succeeds with jwt token encoded with a binary secret" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjk5OTk5OTk5OTksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.Dpss-QoLYjec5OTsOaAc3FNVsSjA89wACoV-0ra3ClA"
		//   request methodGet "/authors_only" [auth] "" `shouldRespondWith` 200
		{Description: "succeeds with jwt token encoded with a binary secret", Query: "/authors_only",
			Headers: authHeader("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjk5OTk5OTk5OTksInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdF9hdXRob3IiLCJpZCI6Impkb2UifQ.Dpss-QoLYjec5OTsOaAc3FNVsSjA89wACoV-0ra3ClA"),
			Status:  200},
	})
}

// Feature/Auth/AsymmetricJwtSpec.hs: the secret as an RS256 JWK, and as a JWK
// set.
func TestAsymmetricJwtSpec(t *testing.T) {
	// auth = authHeaderJWT "eyJhbGciOiJSUzI1NiJ9.eyJyb2xlIjogInBvc3RncmVzdF90ZXN0X2F1dGhvciJ9Cg.CBOY..."
	auth := authHeader("eyJhbGciOiJSUzI1NiJ9.eyJyb2xlIjogInBvc3RncmVzdF90ZXN0X2F1dGhvciJ9Cg.CBOYWDvqgAR0YYnZnyDGTQi6AJLc2Pds6_eV3YuBG6I36mj_h05eLhkEKNEDA5ZteMzCiY83P60rC_xtxVd7B6vo3BeF5uoanPS3rrbuHzKPwzsrgrD_CqvEuJ4n7Q9epkQiLsNkcexneENZDRqFjbwZx3DrXiCWwlK3Ytr5NAIGxmy0od-0xNpb2U1nXQyO_Q3mumWFViRt4tmFn_3goDHNKG3Ha_AzImfUNvHnWL78kAc4rbn15vLtWXD8PwtSnZaB4lY4V6RfsaW937srQsmRetvytM1i_bHBnjkjQLAqGbXPyItjtlXPs0uGNBadE8-wgkLtfmSCC4v2DjUthw")
	jwk := `{"alg":"RS256","e":"AQAB","key_ops":["verify"],"kty":"RSA","n":"0etQ2Tg187jb04MWfpuogYGV75IFrQQBxQaGH75eq_FpbkyoLcEpRUEWSbECP2eeFya2yZ9vIO5ScD-lPmovePk4Aa4SzZ8jdjhmAbNykleRPCxMg0481kz6PQhnHRUv3nF5WP479CnObJKqTVdEagVL66oxnX9VhZG9IZA7k0Th5PfKQwrKGyUeTGczpOjaPqbxlunP73j9AfnAt4XCS8epa-n3WGz1j-wfpr_ys57Aq-zBCfqP67UYzNpeI1AoXsJhD9xSDOzvJgFRvc3vm2wjAW4LEMwi48rCplamOpZToIHEPIaPzpveYQwDnB1HFTR1ove9bpKJsHmi-e2uzQ","use":"sig"}`
	cases := []struct {
		name   string
		port   int
		secret string
	}{
		// context "secret provided as JWK"
		{"secret provided as JWK", 8146, jwk},
		// context "secret provided as JWKSet"
		{"secret provided as JWKSet", 8147, `{"keys": [` + jwk + `]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := startJWTServer(t, c.port, map[string]any{"JWTSecret": c.secret})
			test.Execute(t, cfg, []test.Test{
				// it "succeeds with jwt token signed with an asymmetric key"
				//   request methodGet "/authors_only" [auth] "" `shouldRespondWith` 200
				{Description: "succeeds with jwt token signed with an asymmetric key", Query: "/authors_only",
					Headers: auth, Status: 200},
			})
		})
	}
}

// JWTKeys end to end: an ES256 key selected by kid, and the legacy HS256
// secret for the tokens without kid, both at once.
func TestJWTKeys(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	ecJWK := fmt.Sprintf(`{"kty":"EC","crv":"P-256","kid":"backend-1","alg":"ES256","x":"%s","y":"%s"}`,
		enc(ec.X.FillBytes(make([]byte, 32))), enc(ec.Y.FillBytes(make([]byte, 32))))
	octJWK := fmt.Sprintf(`{"kty":"oct","kid":"legacy","alg":"HS256","k":"%s"}`, enc([]byte(postgrestSecret)))
	cfg := startJWTServer(t, 8148, map[string]any{"JWTKeys": "[" + octJWK + "," + ecJWK + "]"})
	es256 := func(kid string) test.Headers {
		token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"role": "postgrest_test_author"})
		if kid != "" {
			token.Header["kid"] = kid
		}
		s, err := token.SignedString(ec)
		if err != nil {
			t.Fatal(err)
		}
		return authHeader(s)
	}
	test.Execute(t, cfg, []test.Test{
		{Description: "an ES256 token with the key's kid", Query: "/authors_only", Headers: es256("backend-1"), Status: 200},
		{Description: "a legacy HS256 token without kid", Query: "/authors_only",
			Headers: authHeader(signPayload(t, `{"role":"postgrest_test_author"}`)), Status: 200},
		{Description: "an ES256 token without kid", Query: "/authors_only", Headers: es256(""),
			Expected: `{"code":"PGRST301","details":null,"hint":"","message":"No suitable key or wrong key type","position":0,"subsystem":"auth"}`, Status: 401},
		{Description: "an unknown kid", Query: "/authors_only", Headers: es256("backend-2"),
			Expected: `{"code":"PGRST301","details":null,"hint":"","message":"No suitable key or wrong key type","position":0,"subsystem":"auth"}`, Status: 401},
	})
}
