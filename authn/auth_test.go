package authn

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// postgrestSecret is the secret of PostgREST's spec suite (SpecHelper.hs,
// generateSecret).
const postgrestSecret = "reallyreallyreallyreallyverysafe"

func newVerifier(t *testing.T, cfg JWTConfig) *JWTVerifier {
	t.Helper()
	v, err := NewJWTVerifier(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// sign signs a JSON payload with HS256, keeping its text as it is.
func sign(t *testing.T, payload string, secret string) string {
	t.Helper()
	return signWith(t, jwt.SigningMethodHS256, `{"alg":"HS256","typ":"JWT"}`, payload, []byte(secret))
}

func signWith(t *testing.T, m jwt.SigningMethod, header, payload string, key any) string {
	t.Helper()
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(header)) + "." + enc.EncodeToString([]byte(payload))
	sig, err := m.Sign(unsigned, key)
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + enc.EncodeToString(sig)
}

// A bearer token must never verify against an empty HMAC key: with no secret
// configured (allowed when LoginMode is "none"), anyone could forge a token for
// any role. authenticate must reject every token when the secret is empty,
// while a configured secret keeps working.
func TestAuthenticateRejectsEmptySecret(t *testing.T) {
	forged, err := GenerateToken("postgres", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authenticate(forged, newVerifier(t, JWTConfig{})); err == nil {
		t.Fatal("a token signed with the empty key was accepted against an empty secret")
	}

	// positive control: the same flow with a real secret authenticates
	good, err := GenerateToken("user1", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := authenticate(good, newVerifier(t, JWTConfig{Secret: "s3cret"}))
	if err != nil {
		t.Fatalf("expected the token to authenticate with its secret, got %v", err)
	}
	if claims.Role != "user1" {
		t.Errorf("expected role user1, got %q", claims.Role)
	}
}

// checkVerify verifies payloads signed with the secret at a fixed clock: an
// empty want is a success, else the error message.
func checkVerify(t *testing.T, v *JWTVerifier, now time.Time, cases map[string]string) {
	t.Helper()
	for payload, want := range cases {
		_, err := v.verify(sign(t, payload, postgrestSecret), now)
		switch {
		case want == "" && err != nil:
			t.Errorf("%s: %v, want success", payload, err)
		case want != "" && (err == nil || err.Error() != want):
			t.Errorf("%s: %v, want %q", payload, err, want)
		}
	}
}

// PostgREST allows a fixed 30 s clock skew on exp, nbf and iat
// (Auth/Jwt.hs:70-76: expired when now - 30 > exp, not yet valid when
// now + 30 < nbf, issued in the future when now + 30 < iat). Checked on a live
// PostgREST 16.4: exp now-29 accepted, now-40 "JWT expired"; nbf and iat
// now+20 accepted, now+40 refused.
func TestVerifyClockSkew(t *testing.T) {
	v := newVerifier(t, JWTConfig{Secret: postgrestSecret})
	now := time.Unix(1_800_000_000, 0)
	at := func(claim string, offset int64) string {
		return fmt.Sprintf(`{"role":"postgrest_test_author","%s":%d}`, claim, now.Unix()+offset)
	}
	checkVerify(t, v, now, map[string]string{
		at("exp", -31):  "JWT expired",
		at("exp", -30):  "",
		at("exp", -1):   "",
		at("exp", 0):    "",
		at("nbf", 31):   "JWT not yet valid",
		at("nbf", 30):   "",
		at("iat", 31):   "JWT issued at future",
		at("iat", 30):   "",
		at("iat", -1e6): "",
	})
}

// AuthSpec.hs: the claims of the wrong type, and their messages. The other
// cases were checked on a live PostgREST 16.4.
func TestVerifyClaimTypes(t *testing.T) {
	v := newVerifier(t, JWTConfig{Secret: postgrestSecret})
	now := time.Unix(1_800_000_000, 0)
	checkVerify(t, v, now, map[string]string{
		// it "fails when the exp claim is not a number"
		`{"exp":"invalid","role":"postgrest_test_author"}`: "The JWT 'exp' claim must be a number",
		// it "fails when the nbf claim is not a number"
		`{"nbf":"invalid","role":"postgrest_test_author"}`: "The JWT 'nbf' claim must be a number",
		// it "fails when the iat claim is not a number"
		`{"iat":"invalid","role":"postgrest_test_author"}`: "The JWT 'iat' claim must be a number",
		// it "fails when the aud claim has a single value and it's not a string"
		`{"aud":{"invalid":"value"},"role":"postgrest_test_author"}`: "The JWT 'aud' claim must be a string or an array of strings",
		// it "fails when the aud claim is an array but it has non-string elements"
		`{"aud":[{"invalid":"value"},"test"],"role":"postgrest_test_author"}`: "The JWT 'aud' claim must be a string or an array of strings",
		`{"aud":["x",5],"role":"postgrest_test_author"}`:                      "The JWT 'aud' claim must be a string or an array of strings",
		`{"aud":5,"role":"postgrest_test_author"}`:                            "The JWT 'aud' claim must be a string or an array of strings",
		// an exp read as 0 by Sci.toBoundedInteger: fractional, or beyond int64
		`{"exp":9999999999.5,"role":"postgrest_test_author"}`: "JWT expired",
		`{"exp":1e30,"role":"postgrest_test_author"}`:         "JWT expired",
		`{"exp":1e10,"role":"postgrest_test_author"}`:         "",
		`{"exp":null,"role":"postgrest_test_author"}`:         "",
		`{"exp":"9999999999","role":"postgrest_test_author"}`: "The JWT 'exp' claim must be a number",
		// the first failure in the order exp, nbf, iat, aud
		`{"exp":1,"nbf":1900000000,"role":"postgrest_test_author"}`: "JWT expired",
		`{"nbf":1900000000,"aud":5,"role":"postgrest_test_author"}`: "JWT not yet valid",
		// other registered claims are not checked
		`{"sub":5,"iss":7,"role":"postgrest_test_author"}`: "",
	})
}

// AudienceJwtSecretSpec.hs, with jwt-aud = "youraudience" (spec) and without
// it (disabledSpec).
func TestVerifyAudience(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	aud := newVerifier(t, JWTConfig{Secret: postgrestSecret, Aud: "youraudience"})
	checkVerify(t, aud, now, map[string]string{
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":"youraudience"}`:                                               "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":"notyouraudience"}`:                                            "JWT not in audience",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":""}`:                                                           "JWT not in audience",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["youraudience"]}`:                                             "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["notyouraudience","youraudience","anotheraudience"]}`:         "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["notyouraudience"]}`:                                          "JWT not in audience",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["notyouraudience","stillnotyouraudience","anotheraudience"]}`: "JWT not in audience",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":[]}`:                                                           "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":null}`:                                                         "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe"}`:                                                                    "",
	})
	none := newVerifier(t, JWTConfig{Secret: postgrestSecret})
	checkVerify(t, none, now, map[string]string{
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":"youraudience"}`:                                       "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":""}`:                                                   "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["youraudience"]}`:                                     "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":["notyouraudience","youraudience","anotheraudience"]}`: "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":[]}`:                                                   "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe","aud":null}`:                                                 "",
		`{"exp":9999999999,"role":"postgrest_test_author","id":"jdoe"}`:                                                            "",
	})
}

// PostgREST takes jwt-aud as a string, or as a URI when it holds a ":"
// (Config.hs, optStringOrURI; a live 16.4 starts with "urn:example:aud" and
// refuses "http://x y").
func TestJWTAudConfig(t *testing.T) {
	for _, aud := range []string{"youraudience", "urn:example:aud", "https://api.example.com/v1?x=1#f", ""} {
		if _, err := NewJWTVerifier(JWTConfig{Secret: postgrestSecret, Aud: aud}); err != nil {
			t.Errorf("%q: %v", aud, err)
		}
	}
	for _, aud := range []string{"http://x y", ":nope", "1http://x", "http://x%zz"} {
		if _, err := NewJWTVerifier(JWTConfig{Secret: postgrestSecret, Aud: aud}); err == nil {
			t.Errorf("%q: accepted, want an error", aud)
		}
	}
}

// jwt-secret-is-base64 (Config.hs, decodeSecret): "-", "_" and "." stand for
// "+", "/" and "=", blanks around are stripped, the rest is strict base64.
// Checked on a live PostgREST 16.4: padded, "."-padded, URL-safe and
// blank-surrounded values start; unpadded, multi-line and invalid ones do not.
func TestSecretIsBase64(t *testing.T) {
	std := base64.StdEncoding.EncodeToString([]byte(postgrestSecret)) // ends in "="
	token := sign(t, `{"role":"postgrest_test_author"}`, postgrestSecret)
	for _, secret := range []string{std, strings.TrimSuffix(std, "=") + ".", " " + std + "\n"} {
		v, err := NewJWTVerifier(JWTConfig{Secret: secret, SecretIsBase64: true})
		if err != nil {
			t.Errorf("%q: %v", secret, err)
			continue
		}
		if _, err := v.verify(token, time.Now()); err != nil {
			t.Errorf("%q: %v", secret, err)
		}
		if v.SigningSecret() != postgrestSecret {
			t.Errorf("%q: signs with %q", secret, v.SigningSecret())
		}
	}
	binary := []byte{0xfb, 0xff, 0x00, 0x3e, 0x10, 0xfa, 0xbf, 0xee, 0x01}
	urlSafe := base64.URLEncoding.EncodeToString(binary)
	if !strings.ContainsAny(urlSafe, "-_") {
		t.Fatal("the binary secret must exercise the URL-safe alphabet")
	}
	v, err := NewJWTVerifier(JWTConfig{Secret: urlSafe, SecretIsBase64: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.verify(sign(t, `{"role":"r"}`, string(binary)), time.Now()); err != nil {
		t.Errorf("binary secret: %v", err)
	}
	for _, secret := range []string{strings.TrimSuffix(std, "="), std[:10] + "\n" + std[10:], "!!!notbase64!!!"} {
		if _, err := NewJWTVerifier(JWTConfig{Secret: secret, SecretIsBase64: true}); err == nil {
			t.Errorf("%q: accepted, want an error", secret)
		}
	}
	// without the flag the base64 text is the secret itself
	v = newVerifier(t, JWTConfig{Secret: std})
	if _, err := v.verify(token, time.Now()); err == nil {
		t.Error("the encoded value must not verify a token signed with the decoded one")
	}
}

// The decode errors (AuthSpec.hs, and a live PostgREST 16.4 for the others):
// PGRST301, with PostgREST's messages, and a *JWTError the error writer can
// recognize as an invalid token.
func TestVerifyDecodeErrors(t *testing.T) {
	v := newVerifier(t, JWTConfig{Secret: postgrestSecret})
	good := sign(t, `{"role":"postgrest_test_author"}`, postgrestSecret)
	cases := []struct {
		name, token, message string
	}{
		// it "hides tables from users with invalid JWT"
		{"2 parts", "ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0", "Expected 3 parts in JWT; got 2"},
		{"4 parts", "a.b.c.d", "Expected 3 parts in JWT; got 4"},
		{"bad signature", sign(t, `{"role":"postgrest_test_author"}`, "otherotherotherotherotherotherxx"), "No suitable key or wrong key type"},
		{"alg none", "eyJhbGciOiJub25lIn0.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.", "Wrong or unsupported encoding algorithm"},
		{"garbage", "eyJhbGciOiJIUzI1NiJ9.!!!.abc", "JWT cryptographic operation failed"},
		{"HS512 with the HS256 secret", signWith(t, jwt.SigningMethodHS512, `{"alg":"HS512"}`, `{"role":"r"}`, []byte(postgrestSecret)), "No suitable key or wrong key type"},
		{"payload not an object", signWith(t, jwt.SigningMethodHS256, `{"alg":"HS256"}`, `["x"]`, []byte(postgrestSecret)), "Parsing claims failed"},
	}
	for _, c := range cases {
		_, err := v.verify(c.token, time.Now())
		var jerr *JWTError
		if !errors.As(err, &jerr) {
			t.Errorf("%s: %v, want a *JWTError", c.name, err)
			continue
		}
		if jerr.Message != c.message {
			t.Errorf("%s: %q, want %q", c.name, jerr.Message, c.message)
		}
	}
	if _, err := v.verify(good, time.Now()); err != nil {
		t.Errorf("positive control: %v", err)
	}
	_, err := v.verify("a.b", time.Now())
	if jerr, _ := err.(*JWTError); jerr == nil || jerr.Code != "PGRST301" {
		t.Errorf("a decode error is PGRST301: %#v", err)
	}
	_, err = v.verify(sign(t, `{"exp":1,"role":"r"}`, postgrestSecret), time.Now())
	if jerr, _ := err.(*JWTError); jerr == nil || jerr.Code != "PGRST303" {
		t.Errorf("a claims error is PGRST303: %#v", err)
	}
}

// The role is found by the role claim key, and request.jwt.claims carries it
// in "role", as PostgREST's PreQuery inserts it (checked live: with key
// $.roles[1] the claims {"role":"postgrest_test_anonymous","roles":[...]}
// reach the database with "role":"postgrest_test_author").
func TestVerifyRoleClaimKey(t *testing.T) {
	now := time.Now()
	cases := []struct {
		key, payload, role, claims string
		hasRole                    bool
	}{
		{"", `{"role":"postgrest_test_author","id":"jdoe"}`, author, `{"role":"postgrest_test_author","id":"jdoe"}`, true},
		{"$.roles[1]", `{"role":"postgrest_test_anonymous","roles":["other","postgrest_test_author"]}`, author,
			`{"role":"postgrest_test_author","roles":["other","postgrest_test_author"]}`, true},
		{"$.app.role", `{"app":{"role":"postgrest_test_author"}}`, author, `{"app":{"role":"postgrest_test_author"},"role":"postgrest_test_author"}`, true},
		{".app.role", `{"app":{"role":"postgrest_test_author"}}`, author, `{"app":{"role":"postgrest_test_author"},"role":"postgrest_test_author"}`, true},
		{"$.role", `{"role":5}`, "5", `{"role":"5"}`, true},
		{"$.role", `{"role":null}`, "null", `{"role":"null"}`, true},
		{"$.myRole", `{"id":"jdoe","role":"postgrest_test_author"}`, "", `{"id":"jdoe","role":"postgrest_test_author"}`, false},
	}
	for _, c := range cases {
		v := newVerifier(t, JWTConfig{Secret: postgrestSecret, RoleClaimKey: c.key})
		claims, err := v.verify(sign(t, c.payload, postgrestSecret), now)
		if err != nil {
			t.Errorf("%s on %s: %v", c.key, c.payload, err)
			continue
		}
		if claims.Role != c.role || claims.hasRole != c.hasRole {
			t.Errorf("%s on %s: role %q (%v), want %q (%v)", c.key, c.payload, claims.Role, claims.hasRole, c.role, c.hasRole)
		}
		if claims.RawClaims != c.claims {
			t.Errorf("%s on %s: claims %s, want %s", c.key, c.payload, claims.RawClaims, c.claims)
		}
	}
	if _, err := NewJWTVerifier(JWTConfig{Secret: postgrestSecret, RoleClaimKey: "role.other"}); err == nil {
		t.Error("an invalid role claim key must be refused")
	}
}
