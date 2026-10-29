package authn

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	noSuitableKey = "No suitable key was found to decode the JWT"
	noKeyVerifies = "None of the keys was able to decode the JWT"
	badCrypto     = "JWT cryptographic operation failed"
)

var b64url = base64.RawURLEncoding

func octJWK(secret, kid, alg string) string {
	j := map[string]string{"kty": "oct", "k": b64url.EncodeToString([]byte(secret))}
	if kid != "" {
		j["kid"] = kid
	}
	if alg != "" {
		j["alg"] = alg
	}
	b, _ := json.Marshal(j)
	return string(b)
}

func ecJWK(k *ecdsa.PrivateKey, kid, alg string) string {
	size := (k.Curve.Params().BitSize + 7) / 8
	j := map[string]string{
		"kty": "EC",
		"crv": k.Curve.Params().Name,
		"x":   b64url.EncodeToString(k.X.FillBytes(make([]byte, size))),
		"y":   b64url.EncodeToString(k.Y.FillBytes(make([]byte, size))),
	}
	if kid != "" {
		j["kid"] = kid
	}
	if alg != "" {
		j["alg"] = alg
	}
	b, _ := json.Marshal(j)
	return string(b)
}

func newEC(t *testing.T, c elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// tok signs a token whose header carries alg, and kid when not empty.
func tok(t *testing.T, alg, kid string, key any, payload string) string {
	t.Helper()
	h := map[string]string{"alg": alg, "typ": "JWT"}
	if kid != "" {
		h["kid"] = kid
	}
	header, _ := json.Marshal(h)
	return signWith(t, jwt.GetSigningMethod(alg), string(header), payload, key)
}

const authorClaims = `{"role":"postgrest_test_author"}`

type keyCase struct {
	name, token string
	want        string // "" for success, else the message or the details
}

func checkKeys(t *testing.T, v *JWTVerifier, cases []keyCase) {
	t.Helper()
	for _, c := range cases {
		_, err := v.verify(c.token, time.Now())
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v, want success", c.name, err)
			}
			continue
		}
		var jerr *JWTError
		if !errors.As(err, &jerr) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
			continue
		}
		if jerr.Message != c.want && jerr.Details != c.want {
			t.Errorf("%s: %q (%q), want %q", c.name, jerr.Message, jerr.Details, c.want)
		}
	}
}

// JWTSecret holds a JWK or a JWK set as PostgREST's jwt-secret does
// (Config.hs, parseSecret: a JWK set, else a JWK, else a passphrase). The key
// selection is jose's, which PostgREST calls (Auth/Jwt.hs, parseToken): a
// token with a kid is tried on the keys with that kid, one without on every
// key; a key is tried only for the algorithms of its type, or its own alg.
// Every case was run on a live PostgREST 16.4 with the same keys.
func TestSecretJWKSelection(t *testing.T) {
	const s1, s2 = "secretone-secretone-secretone-12", "secrettwo-secrettwo-secrettwo-12"
	two := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK(s1, "k1", "") + `,` + octJWK(s2, "k2", "") + `]}`})
	checkKeys(t, two, []keyCase{
		{"kid k2, signed with k2", tok(t, "HS256", "k2", []byte(s2), authorClaims), ""},
		{"kid k1, signed with k2", tok(t, "HS256", "k1", []byte(s2), authorClaims), noKeyVerifies},
		{"unknown kid", tok(t, "HS256", "k3", []byte(s2), authorClaims), noSuitableKey},
		{"no kid, signed with k2", tok(t, "HS256", "", []byte(s2), authorClaims), ""},
		{"no kid, signed with k1", tok(t, "HS256", "", []byte(s1), authorClaims), ""},
		{"no kid, HS384", tok(t, "HS384", "", []byte(s1), authorClaims), ""},
		{"no kid, HS512", tok(t, "HS512", "", []byte(s2), authorClaims), ""},
		{"kid k1, HS512", tok(t, "HS512", "k1", []byte(s1), authorClaims), ""},
	})
	withAlg := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK(s1, "", "HS256") + `]}`})
	checkKeys(t, withAlg, []keyCase{
		{"a kid never matches a key without one", tok(t, "HS256", "x", []byte(s1), authorClaims), noSuitableKey},
		{"no kid", tok(t, "HS256", "", []byte(s1), authorClaims), ""},
		{"the key's alg only", tok(t, "HS384", "", []byte(s1), authorClaims), noSuitableKey},
	})
	single := newVerifier(t, JWTConfig{Secret: octJWK(s1, "", "")})
	checkKeys(t, single, []keyCase{
		{"single JWK, HS256", tok(t, "HS256", "", []byte(s1), authorClaims), ""},
		{"single JWK, HS384", tok(t, "HS384", "", []byte(s1), authorClaims), ""},
		{"single JWK, HS512", tok(t, "HS512", "", []byte(s1), authorClaims), ""},
		{"single JWK, kid", tok(t, "HS256", "x", []byte(s1), authorClaims), noSuitableKey},
	})
	hs512 := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK(s1, "", "HS512") + `]}`})
	checkKeys(t, hs512, []keyCase{
		{"alg HS512 key, HS512", tok(t, "HS512", "", []byte(s1), authorClaims), ""},
		{"alg HS512 key, HS256", tok(t, "HS256", "", []byte(s1), authorClaims), noSuitableKey},
	})
	dup := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK(s1, "d", "") + `,` + octJWK(s2, "d", "") + `]}`})
	checkKeys(t, dup, []keyCase{
		{"duplicate kid, first key", tok(t, "HS256", "d", []byte(s1), authorClaims), ""},
		{"duplicate kid, second key", tok(t, "HS256", "d", []byte(s2), authorClaims), ""},
	})
	enc := newVerifier(t, JWTConfig{Secret: `{"keys":[{"kty":"oct","k":"` + b64url.EncodeToString([]byte(s1)) + `","use":"enc"}]}`})
	checkKeys(t, enc, []keyCase{{"use enc", tok(t, "HS256", "", []byte(s1), authorClaims), noSuitableKey}})
	signOnly := newVerifier(t, JWTConfig{Secret: `{"keys":[{"kty":"oct","k":"` + b64url.EncodeToString([]byte(s1)) + `","key_ops":["sign"]}]}`})
	checkKeys(t, signOnly, []keyCase{{"key_ops is not checked", tok(t, "HS256", "", []byte(s1), authorClaims), ""}})
	empty := newVerifier(t, JWTConfig{Secret: `{"keys":[]}`})
	checkKeys(t, empty, []keyCase{{"empty set", tok(t, "HS256", "", []byte(s1), authorClaims), noSuitableKey}})
	short := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK("short", "", "") + `]}`})
	checkKeys(t, short, []keyCase{{"a short oct key", tok(t, "HS256", "", []byte("short"), authorClaims), ""}})
}

// The asymmetric keys of a JWTSecret JWK(S), and the algorithm bound to the
// key type: an HS256 token never verifies with an EC key's bytes, whatever its
// kid (the alg-confusion hole).
func TestSecretJWKAsymmetric(t *testing.T) {
	const s1 = "secretone-secretone-secretone-12"
	ec1, other := newEC(t, elliptic.P256()), newEC(t, elliptic.P256())
	ec := newVerifier(t, JWTConfig{Secret: ecJWK(ec1, "ec1", "ES256")})
	x := []byte(ecJWK(ec1, "", "")) // the public JWK text as an HMAC key
	checkKeys(t, ec, []keyCase{
		{"ES256 with its kid", tok(t, "ES256", "ec1", ec1, authorClaims), ""},
		{"ES256 without kid", tok(t, "ES256", "", ec1, authorClaims), ""},
		{"ES256 signed by another key", tok(t, "ES256", "ec1", other, authorClaims), noKeyVerifies},
		{"HS256 with the EC key's kid", tok(t, "HS256", "ec1", x, authorClaims), noSuitableKey},
		{"HS256 without kid", tok(t, "HS256", "", ec1.X.Bytes(), authorClaims), noSuitableKey},
	})
	mixed := newVerifier(t, JWTConfig{Secret: `{"keys":[` + octJWK(s1, "k1", "HS256") + `,` + ecJWK(ec1, "ec1", "ES256") + `]}`})
	checkKeys(t, mixed, []keyCase{
		{"mixed, ES256 without kid", tok(t, "ES256", "", ec1, authorClaims), ""},
		{"mixed, HS256 without kid", tok(t, "HS256", "", []byte(s1), authorClaims), ""},
		{"mixed, HS256 with the EC kid", tok(t, "HS256", "ec1", []byte(s1), authorClaims), noSuitableKey},
		{"mixed, ES256 with the oct kid", tok(t, "ES256", "k1", ec1, authorClaims), noSuitableKey},
	})
	noAlg := newVerifier(t, JWTConfig{Secret: ecJWK(ec1, "ec1", "")})
	p384 := newEC(t, elliptic.P384())
	checkKeys(t, noAlg, []keyCase{
		{"EC key without alg, ES256", tok(t, "ES256", "ec1", ec1, authorClaims), ""},
		{"EC P-256 key, ES384 token", tok(t, "ES384", "", p384, authorClaims), noKeyVerifies},
	})
	ec384 := newVerifier(t, JWTConfig{Secret: ecJWK(p384, "", "")})
	checkKeys(t, ec384, []keyCase{{"EC P-384, ES384", tok(t, "ES384", "", p384, authorClaims), ""}})
	hsAlg := newVerifier(t, JWTConfig{Secret: ecJWK(ec1, "", "HS256")})
	checkKeys(t, hsAlg, []keyCase{{"EC key with alg HS256", tok(t, "ES256", "", ec1, authorClaims), noSuitableKey}})
	esAlg := newVerifier(t, JWTConfig{Secret: octJWK(s1, "", "ES256")})
	checkKeys(t, esAlg, []keyCase{{"oct key with alg ES256", tok(t, "HS256", "", []byte(s1), authorClaims), noSuitableKey}})

	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaJWK := fmt.Sprintf(`{"kty":"RSA","n":"%s","e":"%s"}`,
		b64url.EncodeToString(rk.N.Bytes()), b64url.EncodeToString(big.NewInt(int64(rk.E)).Bytes()))
	checkKeys(t, newVerifier(t, JWTConfig{Secret: rsaJWK}), []keyCase{
		{"RSA, RS256", tok(t, "RS256", "", rk, authorClaims), ""},
		{"RSA, RS384", tok(t, "RS384", "", rk, authorClaims), ""},
		{"RSA, RS512", tok(t, "RS512", "", rk, authorClaims), ""},
		{"RSA, PS256 is not supported", tok(t, "PS256", "", rk, authorClaims), badCrypto},
	})
	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	edJWK := `{"kty":"OKP","crv":"Ed25519","x":"` + b64url.EncodeToString(edPriv.Public().(ed25519.PublicKey)) + `"}`
	checkKeys(t, newVerifier(t, JWTConfig{Secret: edJWK}), []keyCase{
		{"Ed25519, EdDSA", tok(t, "EdDSA", "", edPriv, authorClaims), ""},
	})
}

// A secret that is not a JWK set nor a JWK is a passphrase: JSON with a typo,
// or a JWK PostgREST cannot read (here a numeric kid), verifies as the raw
// text, as in PostgREST. A passphrase is an HS256 key without kid.
func TestSecretPassphraseFallback(t *testing.T) {
	broken := `{"keys":[{"kty":"oct","k":"abc", BROKEN}]}`
	numericKid := `{"kty":"oct","k":"c2VjcmV0","kid":5}`
	for _, secret := range []string{broken, numericKid} {
		checkKeys(t, newVerifier(t, JWTConfig{Secret: secret}), []keyCase{
			{secret, tok(t, "HS256", "", []byte(secret), authorClaims), ""},
		})
	}
	plain := newVerifier(t, JWTConfig{Secret: postgrestSecret})
	enc := base64.RawURLEncoding.EncodeToString
	checkKeys(t, plain, []keyCase{
		{"a kid never matches the passphrase", tok(t, "HS256", "k1", []byte(postgrestSecret), authorClaims), noSuitableKey},
		{"the passphrase is HS256 only", tok(t, "HS384", "", []byte(postgrestSecret), authorClaims), noSuitableKey},
		{"unknown alg", enc([]byte(`{"alg":"XYZ"}`)) + "." + enc([]byte(authorClaims)) + ".abc", badCrypto},
		{"kid not a string", enc([]byte(`{"alg":"HS256","kid":5}`)) + "." + enc([]byte(authorClaims)) + ".abc", badCrypto},
		{"header not JSON", enc([]byte(`nope`)) + "." + enc([]byte(authorClaims)) + ".abc", badCrypto},
		{"header without alg", enc([]byte(`{"typ":"JWT"}`)) + "." + enc([]byte(authorClaims)) + ".abc", badCrypto},
		{"padded signature", tok(t, "HS256", "", []byte(postgrestSecret), authorClaims) + "=", badCrypto},
	})
}

// Feature/Auth/AsymmetricJwtSpec.hs: the RS256 JWK and JWK set of the spec,
// and its token.
func TestAsymmetricJwtSpecKeys(t *testing.T) {
	// auth = authHeaderJWT "eyJhbGciOiJSUzI1NiJ9.eyJyb2xlIjogInBvc3RncmVzdF90ZXN0X2F1dGhvciJ9Cg.CBOY..."
	token := asymmetricSpecToken
	for _, secret := range []string{asymmetricSpecJWK, `{"keys": [` + asymmetricSpecJWK + `]}`} {
		v := newVerifier(t, JWTConfig{Secret: secret})
		claims, err := v.verify(token, time.Now())
		if err != nil {
			t.Fatalf("succeeds with jwt token signed with an asymmetric key: %v", err)
		}
		if claims.Role != author {
			t.Errorf("role %q", claims.Role)
		}
	}
}

const asymmetricSpecToken = "eyJhbGciOiJSUzI1NiJ9.eyJyb2xlIjogInBvc3RncmVzdF90ZXN0X2F1dGhvciJ9Cg.CBOYWDvqgAR0YYnZnyDGTQi6AJLc2Pds6_eV3YuBG6I36mj_h05eLhkEKNEDA5ZteMzCiY83P60rC_xtxVd7B6vo3BeF5uoanPS3rrbuHzKPwzsrgrD_CqvEuJ4n7Q9epkQiLsNkcexneENZDRqFjbwZx3DrXiCWwlK3Ytr5NAIGxmy0od-0xNpb2U1nXQyO_Q3mumWFViRt4tmFn_3goDHNKG3Ha_AzImfUNvHnWL78kAc4rbn15vLtWXD8PwtSnZaB4lY4V6RfsaW937srQsmRetvytM1i_bHBnjkjQLAqGbXPyItjtlXPs0uGNBadE8-wgkLtfmSCC4v2DjUthw"

const asymmetricSpecJWK = `{"alg":"RS256","e":"AQAB","key_ops":["verify"],"kty":"RSA","n":"0etQ2Tg187jb04MWfpuogYGV75IFrQQBxQaGH75eq_FpbkyoLcEpRUEWSbECP2eeFya2yZ9vIO5ScD-lPmovePk4Aa4SzZ8jdjhmAbNykleRPCxMg0481kz6PQhnHRUv3nF5WP479CnObJKqTVdEagVL66oxnX9VhZG9IZA7k0Th5PfKQwrKGyUeTGczpOjaPqbxlunP73j9AfnAt4XCS8epa-n3WGz1j-wfpr_ys57Aq-zBCfqP67UYzNpeI1AoXsJhD9xSDOzvJgFRvc3vm2wjAW4LEMwi48rCplamOpZToIHEPIaPzpveYQwDnB1HFTR1ove9bpKJsHmi-e2uzQ","use":"sig"}`

// JWTKeys, the deployment's key catalog (card 32265 jwk-set): oct keys for
// HS256 and EC P-256 public keys for ES256, every one with a kid. A token with
// a kid is verified by that key only; a token without one falls back to the
// oct keys, the legacy secret, while the catalog holds one. The algorithm is
// bound to the key type.
func TestKeyCatalogSelection(t *testing.T) {
	const legacy = "legacy-legacy-legacy-legacy-legacy"
	ec1, ec2, stranger := newEC(t, elliptic.P256()), newEC(t, elliptic.P256()), newEC(t, elliptic.P256())
	catalog := `[` + octJWK(legacy, "legacy", "HS256") + `,` + ecJWK(ec1, "ec1", "ES256") + `,` + ecJWK(ec2, "ec2", "") + `]`
	v := newVerifier(t, JWTConfig{Secret: "the-single-secret-is-not-used-here", Keys: catalog})
	now := time.Now().Unix()
	at := func(offset int64) string {
		return fmt.Sprintf(`{"role":"postgrest_test_author","exp":%d}`, now+offset)
	}
	checkKeys(t, v, []keyCase{
		{"kid hit, ES256", tok(t, "ES256", "ec1", ec1, authorClaims), ""},
		{"kid hit, the second EC key", tok(t, "ES256", "ec2", ec2, authorClaims), ""},
		{"kid hit, HS256 legacy", tok(t, "HS256", "legacy", []byte(legacy), authorClaims), ""},
		{"kid miss", tok(t, "ES256", "ec3", ec1, authorClaims), noSuitableKey},
		{"kid of another key", tok(t, "ES256", "ec2", ec1, authorClaims), noKeyVerifies},
		{"no kid, HS256 legacy", tok(t, "HS256", "", []byte(legacy), authorClaims), ""},
		{"no kid, ES256: no fallback to the EC keys", tok(t, "ES256", "", ec1, authorClaims), noSuitableKey},
		{"no kid, HS256 with another secret", tok(t, "HS256", "", []byte("the-single-secret-is-not-used-here"), authorClaims), noKeyVerifies},
		{"HS256 with an EC key's kid", tok(t, "HS256", "ec1", []byte(ecJWK(ec1, "ec1", "ES256")), authorClaims), noSuitableKey},
		{"HS256 with an EC key's kid, x as the secret", tok(t, "HS256", "ec1", ec1.X.Bytes(), authorClaims), noSuitableKey},
		{"ES256 with the oct key's kid", tok(t, "ES256", "legacy", ec1, authorClaims), noSuitableKey},
		{"HS384 with the HS256 legacy key", tok(t, "HS384", "legacy", []byte(legacy), authorClaims), noSuitableKey},
		{"ES256, invalid signature", tok(t, "ES256", "ec1", stranger, authorClaims), noKeyVerifies},
		{"ES256, expired within the skew", tok(t, "ES256", "ec1", ec1, at(-20)), ""},
		{"ES256, expired outside the skew", tok(t, "ES256", "ec1", ec1, at(-40)), "JWT expired"},
	})
	// without the legacy secret in the set, a token without kid has no key
	noLegacy := newVerifier(t, JWTConfig{Keys: `[` + ecJWK(ec1, "ec1", "ES256") + `]`})
	checkKeys(t, noLegacy, []keyCase{
		{"no kid, no legacy secret", tok(t, "HS256", "", []byte(legacy), authorClaims), noSuitableKey},
		{"kid hit", tok(t, "ES256", "ec1", ec1, authorClaims), ""},
	})
	// a JWK set object is read as the bare array
	set := newVerifier(t, JWTConfig{Keys: `{"keys":` + catalog + `}`})
	checkKeys(t, set, []keyCase{{"JWK set form", tok(t, "ES256", "ec1", ec1, authorClaims), ""}})
}

// JWTKeys is parsed strictly: a mistake in the catalog stops the server
// instead of degrading verification at runtime.
func TestKeyCatalogStrict(t *testing.T) {
	ec := newEC(t, elliptic.P256())
	b := b64url.EncodeToString
	zero := b(make([]byte, 32))
	bad := map[string]string{
		"not JSON":           `[{"kty":"oct",}]`,
		"not an array":       `{"kty":"oct","k":"c2VjcmV0","kid":"a"}`,
		"empty":              `[]`,
		"oct without kid":    `[` + octJWK("secret", "", "") + `]`,
		"EC without kid":     `[` + ecJWK(ec, "", "ES256") + `]`,
		"duplicate kid":      `[` + octJWK("secret", "a", "") + `,` + ecJWK(ec, "a", "") + `]`,
		"unknown kty":        `[{"kty":"XYZ","kid":"a"}]`,
		"oct with alg ES256": `[` + octJWK("secret", "a", "ES256") + `]`,
		"EC with alg HS256":  `[` + ecJWK(ec, "a", "HS256") + `]`,
		"P-256 with ES384":   `[` + ecJWK(ec, "a", "ES384") + `]`,
		"unknown alg":        `[` + octJWK("secret", "a", "XYZ") + `]`,
		"use enc":            `[{"kty":"oct","k":"c2VjcmV0","kid":"a","use":"enc"}]`,
		"EC off the curve":   `[{"kty":"EC","crv":"P-256","kid":"a","x":"` + zero + `","y":"` + zero + `"}]`,
		"EC unknown curve":   `[{"kty":"EC","crv":"P-192","kid":"a","x":"` + zero + `","y":"` + zero + `"}]`,
		"bad base64":         `[{"kty":"oct","k":"!!","kid":"a"}]`,
		"empty oct key":      `[{"kty":"oct","k":"","kid":"a"}]`,
		"kid not a string":   `[{"kty":"oct","k":"c2VjcmV0","kid":5}]`,
	}
	for name, keys := range bad {
		if _, err := NewJWTVerifier(JWTConfig{Keys: keys}); err == nil {
			t.Errorf("%s: accepted, want an error", name)
		} else if !strings.Contains(err.Error(), "JWTKeys") {
			t.Errorf("%s: the error must name JWTKeys: %v", name, err)
		}
	}
	if _, err := NewJWTVerifier(JWTConfig{Keys: `[` + octJWK("secret", "a", "") + `,` + ecJWK(ec, "b", "ES256") + `]`}); err != nil {
		t.Errorf("positive control: %v", err)
	}
}

// /token signs with HS256 and the HMAC secret: the passphrase, decoded when
// JWTSecretIsBase64 is set, or the one oct key of a JWK(S) secret that can
// sign HS256, so that its tokens verify. A JWK(S) without exactly one such key
// has no signing secret.
func TestSigningSecret(t *testing.T) {
	const s1, s2 = "secretone-secretone-secretone-12", "secrettwo-secrettwo-secrettwo-12"
	ec := newEC(t, elliptic.P256())
	cases := []struct {
		secret, want string
	}{
		{s1, s1},
		{octJWK(s1, "k1", ""), s1},
		{`{"keys":[` + octJWK(s1, "k1", "HS256") + `,` + ecJWK(ec, "ec", "") + `]}`, s1},
		{`{"keys":[` + octJWK(s1, "k1", "") + `,` + octJWK(s2, "k2", "") + `]}`, ""},
		{`{"keys":[` + octJWK(s1, "k1", "HS512") + `]}`, ""},
		{ecJWK(ec, "ec", ""), ""},
	}
	for _, c := range cases {
		v := newVerifier(t, JWTConfig{Secret: c.secret})
		if got := v.SigningSecret(); got != c.want {
			t.Errorf("%s: signs with %q, want %q", c.secret, got, c.want)
		}
		if c.want == "" {
			continue
		}
		token, err := GenerateToken("user1", v.SigningSecret())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := v.verify(token, time.Now()); err != nil {
			t.Errorf("%s: a /token token does not verify: %v", c.secret, err)
		}
	}
	// a base64 secret is decoded before it is read as a JWK, as in PostgREST
	v := newVerifier(t, JWTConfig{Secret: base64.StdEncoding.EncodeToString([]byte(octJWK(s1, "", ""))), SecretIsBase64: true})
	if v.SigningSecret() != s1 {
		t.Errorf("base64 JWK: signs with %q", v.SigningSecret())
	}
	// with JWTKeys the passphrase signs, and verifies only through the catalog
	k := newVerifier(t, JWTConfig{Secret: s1, Keys: `[` + octJWK(s1, "legacy", "") + `]`})
	if k.SigningSecret() != s1 {
		t.Errorf("with JWTKeys: signs with %q", k.SigningSecret())
	}
	if !k.SignsWithCatalogKey() {
		t.Error("the passphrase is the catalog's oct key")
	}
	if newVerifier(t, JWTConfig{Secret: s2, Keys: `[` + octJWK(s1, "legacy", "") + `]`}).SignsWithCatalogKey() {
		t.Error("a passphrase outside the catalog does not verify its tokens")
	}
}
