package authn

import (
	"encoding/json"
	"strings"
	"testing"
)

const author = "postgrest_test_author"

// roleOf evaluates a role claim key on claims, as the verifier does.
func roleOf(t *testing.T, key, claims string) (string, bool) {
	t.Helper()
	k, err := ParseRoleClaimKey(key)
	if err != nil {
		t.Fatalf("key %s: %v", key, err)
	}
	d := json.NewDecoder(strings.NewReader(claims))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	r, ok := k.evaluate(v)
	if !ok {
		return "", false
	}
	return roleText(r), true
}

type roleCase struct {
	key, claims string
	role        string // "" for no role found
}

func checkRoles(t *testing.T, cases []roleCase) {
	t.Helper()
	for _, c := range cases {
		role, ok := roleOf(t, c.key, c.claims)
		if c.role == "" && ok {
			t.Errorf("%s on %s: found role %q, want none", c.key, c.claims, role)
		} else if c.role != "" && role != c.role {
			t.Errorf("%s on %s: role %q (found %v), want %q", c.key, c.claims, role, ok, c.role)
		}
	}
}

// PostgREST's io tests run the server with each key and request /authors_only
// with a token carrying the claims: 200 when the key selects
// postgrest_test_author, 401 otherwise (test/io/test_config.py,
// test_role_claim_key, test_jwt_aud_in_role_claim_key; the cases are
// test/io/fixtures/fixtures.yaml, roleclaims and jwtaudroleclaims).
func TestRoleClaimKeyPostgRESTFixtures(t *testing.T) {
	checkRoles(t, []roleCase{
		// - key: '$.postgrest.a_role'  ... expected_status: 200
		{`$.postgrest.a_role`, `{"postgrest":{"a_role":"postgrest_test_author"},"other":"claims"}`, author},
		// - key: '$.customObject.manyRoles[1]'  ... expected_status: 200
		{`$.customObject.manyRoles[1]`, `{"customObject":{"manyRoles":["other","postgrest_test_author"]},"other":{}}`, author},
		// - key: '$["https://www.example.com/roles"][0].value'  ... expected_status: 200
		{`$["https://www.example.com/roles"][0].value`, `{"https://www.example.com/roles":[{"value":"postgrest_test_author"}],"other":666}`, author},
		// - key: '$.myDomain[3]'  ... expected_status: 401
		{`$.myDomain[3]`, `{"myDomain":["other","postgrest_test_author"],"other":1.23}`, ""},
		// - key: '$.myRole'  ... expected_status: 401
		{`$.myRole`, `{"role":"postgrest_test_author","other":true}`, ""},
		// - key: '$.realm_access.roles[?(@ == "postgrest_test_author")]'  ... 200
		{`$.realm_access.roles[?(@ == "postgrest_test_author")]`, `{"realm_access":{"roles":["other","postgrest_test_author"]}}`, author},
		// - key: '$.realm_access.roles[?(@ != "other")]'  ... 200
		{`$.realm_access.roles[?(@ != "other")]`, `{"realm_access":{"roles":["other","postgrest_test_author"]}}`, author},
		// jwtaudroleclaims, with jwt-aud = postgrest_test_author: '$.aud[1]'
		// passes the aud check and fails on the database (role does not exist)
		{`$.aud`, `{"aud":"postgrest_test_author"}`, author},
		{`$.aud`, `{"aud":"postgrest_test_invalid"}`, "postgrest_test_invalid"},
		{`$.aud[0]`, `{"aud":["postgrest_test_author"]}`, author},
		{`$.aud[1]`, `{"aud":["postgrest_test_author","postgrest_test_invalid"]}`, "postgrest_test_invalid"},
	})
}

// The JSPath DSL of PostgREST 15 and earlier, which 16.4 still reads
// (fixtures.yaml, deprecatedroleclaims and deprecatedjwtaudroleclaims).
func TestRoleClaimKeyDeprecatedFixtures(t *testing.T) {
	roles := `{"realm_access":{"roles":["other","postgrest_test_author"]}}`
	checkRoles(t, []roleCase{
		{`.postgrest.a_role`, `{"postgrest":{"a_role":"postgrest_test_author"},"other":"claims"}`, author},
		{`.customObject.manyRoles[1]`, `{"customObject":{"manyRoles":["other","postgrest_test_author"]},"other":{}}`, author},
		{`."https://www.example.com/roles"[0].value`, `{"https://www.example.com/roles":[{"value":"postgrest_test_author"}],"other":666}`, author},
		{`.myDomain[3]`, `{"myDomain":["other","postgrest_test_author"],"other":1.23}`, ""},
		{`.myRole`, `{"role":"postgrest_test_author","other":true}`, ""},
		// https://github.com/PostgREST/postgrest/pull/3813
		{`.realm_access.roles[?(@ == "postgrest_test_author")]`, roles, author},
		{`.realm_access.roles[?(@ != "other")]`, roles, author},
		{`.realm_access.roles[?(@ ^== "postgrest_te")]`, roles, author},
		{`.realm_access.roles[?(@ ==^ "st_test_author")]`, roles, author},
		{`.realm_access.roles[?(@ *== "_test_")]`, roles, author},
		// fails because it compares an object with a string
		{`.realm_access.roles[?(@ == "string")]`, `{"realm_access":{"roles":[{"obj_key":"obj_value"}]}}`, ""},
		{`.aud`, `{"aud":"postgrest_test_author"}`, author},
		{`.aud[0]`, `{"aud":["postgrest_test_author"]}`, author},
		{`.aud[1]`, `{"aud":["postgrest_test_author","postgrest_test_invalid"]}`, "postgrest_test_invalid"},
	})
	for _, key := range []string{`.role`, `.realm_access.roles[?(@ ^== "postgrest_te")]`} {
		k, err := ParseRoleClaimKey(key)
		if err != nil || !k.Deprecated() {
			t.Errorf("%s: want the deprecated syntax, got %v", key, err)
		}
	}
	if k, _ := ParseRoleClaimKey(`$.role`); k.Deprecated() {
		t.Error("$.role is not the deprecated syntax")
	}
}

// PostgREST exits with "failed to parse role-claim-key value (<key>)"
// (test/io/test_cli.py, test_invalid_role_claim_key and
// test_invalid_deprecated_role_claim_key; fixtures.yaml invalidroleclaimkeys
// and deprecatedinvalidroleclaimkeys). The keys after them were refused by a
// live PostgREST 16.4.
func TestRoleClaimKeyInvalid(t *testing.T) {
	keys := []string{
		`$.my_role;;domain`, `@@.role.other`,
		`role.other`, `.role##`, `.my_role;;domain`, `.#$$%&$%/`, `1234`, `.role[?(@ =)]`,
		// only search() among the function extensions
		`$.roles[?match(@, "postgrest_test_auth.*")]`,
		`$.roles[?length(@) > 6]`,
		`$.r[?count(@.*) > 4].role`,
		`$.r[?value(@.n) == 5].role`,
		// no blanks around the query, no leading zero in an index
		`$.roles[1] `, ` $.roles[1]`, `$.roles[01]`,
	}
	for _, key := range keys {
		_, err := ParseRoleClaimKey(key)
		if err == nil {
			t.Errorf("%q: accepted, want a parse error", key)
		} else if !strings.Contains(err.Error(), "failed to parse role-claim-key value ("+key+")") {
			t.Errorf("%q: error %q", key, err)
		}
	}
}

// The JSONPath subset smoothdb reads, each key checked on a live PostgREST
// 16.4 with the same claims: the role it switched to.
func TestRoleClaimKeyJSONPath(t *testing.T) {
	roles := `{"roles":["other","postgrest_test_author"],"role":"postgrest_test_anonymous"}`
	checkRoles(t, []roleCase{
		{`$.role`, roles, "postgrest_test_anonymous"},
		{``, roles, "postgrest_test_anonymous"}, // the default
		// a value that is not a string is the role as its JSON text
		{`$.role`, `{"role":5}`, "5"},
		{`$.role`, `{"role":{"x":1}}`, `{"x":1}`},
		{`$.roles[1]`, roles, author},
		{`$.roles[-1]`, roles, author},
		{`$['https://x.com/r'][0].value`, `{"https://x.com/r":[{"value":"postgrest_test_author"}]}`, author},
		{`$["https://x.com/r"][0]["value"]`, `{"https://x.com/r":[{"value":"postgrest_test_author"}]}`, author},
		{`$.roles[?(@ == "postgrest_test_author")]`, roles, author},
		{`$.roles[?@ == 'postgrest_test_author']`, roles, author},
		{`$.roles[?@ != "other"]`, roles, author},
		{`$.roles[?search(@, "^postgrest_te")]`, roles, author},
		{`$.myRole`, `{"id":"jdoe","role":"postgrest_test_author"}`, ""},
		{`.role`, `{"id":"jdoe","role":"postgrest_test_author"}`, author},
		{`.roles[?(@ ^== "postgrest_te")]`, roles, author},
		// "$" alone is a valid query, selecting the claims
		{`$`, `{"a":1}`, `{"a":1}`},
	})
}

// The examples of PostgREST's docs (docs/references/auth.rst, JWT Role
// Extraction), with the single "$" of an environment variable.
func TestRoleClaimKeyDocumentedExamples(t *testing.T) {
	postgrest := `{"postgrest":{"roles": ["other", "author"]}}`
	checkRoles(t, []roleCase{
		// # {"postgrest":{"roles": ["other", "author"]}}
		// jwt-role-claim-key = "$$.postgrest.roles[1]"
		{`$.postgrest.roles[1]`, postgrest, "author"},
		// # {"https://www.example.com/role": { "key": "author" }}
		// jwt-role-claim-key = "$$['https://www.example.com/role'].key"
		{`$['https://www.example.com/role'].key`, `{"https://www.example.com/role": { "key": "author" }}`, "author"},
		// jwt-role-claim-key = "$$.postgrest.roles[?(@ == 'author')]"
		{`$.postgrest.roles[?(@ == 'author')]`, postgrest, "author"},
		// jwt-role-claim-key = "$$.postgrest.roles[?search(@, '^au')]"
		{`$.postgrest.roles[?search(@, '^au')]`, postgrest, "author"},
	})
}

// The semantics of the subset that the live checks do not reach.
func TestRoleClaimKeySubset(t *testing.T) {
	doc := `{"o":{"z":"last","b":"first"},"a":[5,3,{"b":"x"},"y"],"s":"a\nb","u":"ü","e":"","n":null,"w":"x y"}`
	checkRoles(t, []roleCase{
		{`$.a[2].b`, doc, "x"},
		{`$.a[-4]`, doc, "5"},
		{`$.a[4]`, doc, ""},
		{`$.a[-5]`, doc, ""},
		{`$.o[?@ != "x"]`, doc, "first"},     // object members in key order, as aeson
		{`$.a[?@ != "x"]`, doc, "5"},         // != holds for a value that is not a string
		{`$.a[?@ == "y"]`, doc, "y"},         // == only for the same string
		{`$[?search(@, "^a.b$")]`, doc, ""},  // "." matches no line break
		{`$[?search(@, "x.y")]`, doc, "x y"}, //
		{`$[?search(@, "ü")]`, doc, "ü"},     //
		{`$[?search(@, "[")]`, doc, ""},      // an invalid pattern never matches
		{`$[?((@ == "x y"))]`, doc, "x y"},   // parentheses around the filter
		{`$[ ?  search( @ , "^x" ) ]`, doc, "x y"},
		{`$["ü"]`, `{"ü":"yes"}`, "yes"},
		{`$['a\'b']`, `{"a'b":"yes"}`, "yes"},
		{`$.ü`, `{"ü":"yes"}`, "yes"},
		{`$ .o .b`, doc, "first"},
		{`$[ 'o' ]['b']`, doc, "first"},
		{`$.n`, doc, "null"},
	})
	if role, ok := roleOf(t, `$.e`, `{"e":""}`); !ok || role != "" {
		t.Errorf("an empty string is a role: got %q, %v", role, ok)
	}
	for _, key := range []string{
		`$.`, `$[]`, `$['a'`, `$["\x"]`, `$[-0]`, `$[9007199254740992]`,
		`$[?@ == ]`, `$[?search(@)]`, `$[?search(@, "a") == true]`, `$[?(@ == "a"]`, `$x`,
	} {
		if _, err := ParseRoleClaimKey(key); err == nil {
			t.Errorf("%q: accepted, want an error", key)
		} else if strings.Contains(err.Error(), "not supported") {
			t.Errorf("%q: a syntax error, not an unsupported construct: %v", key, err)
		}
	}
}

// The JSONPath constructs PostgREST 16.4 accepts beyond the subset (each
// checked on a live 16.4, which switched to the role they select) stop the
// server, naming the construct: the divergence is loud, never a different
// role at runtime.
func TestRoleClaimKeyUnsupported(t *testing.T) {
	cases := []struct{ key, construct string }{
		{`$.roles[*]`, "wildcard"},
		{`$.roles.*`, "wildcard"},
		{`$.roles[0:2]`, "slice"},
		{`$.roles[1:]`, "slice"},
		{`$.roles[::-1]`, "slice"},
		{`$.a[-1:-3:-1].b`, "slice"},
		{`$..role`, "descendant"},
		{`$.a..role`, "descendant"},
		{`$.roles[1,0]`, "several selectors"},
		{`$.r[?@.n > 3].role`, "query inside a filter"},
		{`$.r[?@.flag].role`, "query inside a filter"},
		{`$.r[?@.n == $.r[1].n].role`, "query inside a filter"},
		{`$.a[?@.c[?@ == 2]].b`, "query inside a filter"},
		{`$.r[?@.n >= 5 && @.name == "y"].role`, "query inside a filter"},
		{`$.roles[?@ == "a" && @ != "b"]`, "&&"},
		{`$.roles[?@ == "a" || @ == "b"]`, "||"},
		{`$.r[?!(@ == "x")].role`, "!"},
		{`$.roles[?@ < "z"]`, "<"},
		{`$.roles[?@ >= "a"]`, ">="},
		{`$.roles[?@ == true]`, "literal"},
		{`$.roles[?@ == 5]`, "literal"},
		{`$.roles[?@]`, "existence test"},
		{`$.roles[?"x" == @]`, "literal on the left"},
		{`$.roles[?search(@, $.re)]`, "query as the search() pattern"},
	}
	for _, c := range cases {
		_, err := ParseRoleClaimKey(c.key)
		if err == nil {
			t.Errorf("%s: accepted, want it refused as not supported", c.key)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, "JWTRoleClaimKey") || !strings.Contains(msg, "not supported") || !strings.Contains(msg, c.construct) {
			t.Errorf("%s: %q, want an error naming JWTRoleClaimKey and saying %q is not supported", c.key, msg, c.construct)
		}
	}
}
