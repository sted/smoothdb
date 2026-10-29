package authn

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
)

// jwk is a verification key (RFC 7517): an HMAC secret ("oct"), or an EC, RSA
// or Ed25519 ("OKP") public key, with the algorithms it may verify. Binding
// the algorithms to the key type is what keeps an HS256 token from being
// verified with the bytes of a public key.
type jwk struct {
	kty    string
	kid    string
	hasKid bool
	algs   []string
	key    any // []byte, *ecdsa.PublicKey, *rsa.PublicKey or ed25519.PublicKey
}

// The algorithms of each key type, those PostgREST verifies with jose:
// checked on a live PostgREST 16.4, which also refuses PS256 as an
// unsupported algorithm.
var algsByType = map[string][]string{
	"oct": {"HS256", "HS384", "HS512"},
	"RSA": {"RS256", "RS384", "RS512"},
	"EC":  {"ES256", "ES384", "ES512"},
	"OKP": {"EdDSA"},
}

func supportedAlg(alg string) bool {
	for _, algs := range algsByType {
		if slices.Contains(algs, alg) {
			return true
		}
	}
	return false
}

var ecCurves = map[string]struct {
	curve elliptic.Curve
	alg   string
}{
	"P-256": {elliptic.P256(), "ES256"},
	"P-384": {elliptic.P384(), "ES384"},
	"P-521": {elliptic.P521(), "ES512"},
}

type rawJWK struct {
	Kty string  `json:"kty"`
	Kid *string `json:"kid"`
	Alg *string `json:"alg"`
	Use string  `json:"use"`
	K   string  `json:"k"`
	Crv string  `json:"crv"`
	X   string  `json:"x"`
	Y   string  `json:"y"`
	N   string  `json:"n"`
	E   string  `json:"e"`
}

// parseJWK reads one key. Strict is the JWTKeys catalog: every key has a kid,
// its alg (when given) fits its type and curve, and none is for encryption.
// Otherwise the key is read as jose reads PostgREST's jwt-secret: a key whose
// alg does not fit its type, or with "use":"enc", is kept but verifies
// nothing.
func parseJWK(data []byte, strict bool) (*jwk, error) {
	var r rawJWK
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	k := &jwk{kty: r.Kty}
	if r.Kid != nil {
		k.kid, k.hasKid = *r.Kid, true
	} else if strict {
		return nil, fmt.Errorf("a key without kid")
	}
	b64 := func(field, s string) ([]byte, error) {
		b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
		if err != nil || len(b) == 0 {
			return nil, fmt.Errorf("key %q: invalid %q", k.kid, field)
		}
		return b, nil
	}
	algs := algsByType[r.Kty]
	switch r.Kty {
	case "oct":
		secret, err := b64("k", r.K)
		if err != nil {
			return nil, err
		}
		k.key = secret
	case "EC":
		c, ok := ecCurves[r.Crv]
		if !ok {
			return nil, fmt.Errorf("key %q: unsupported curve %q", k.kid, r.Crv)
		}
		x, err := b64("x", r.X)
		if err != nil {
			return nil, err
		}
		y, err := b64("y", r.Y)
		if err != nil {
			return nil, err
		}
		size := (c.curve.Params().BitSize + 7) / 8
		if len(x) != size || len(y) != size {
			return nil, fmt.Errorf("key %q: coordinates of the wrong size for %s", k.kid, r.Crv)
		}
		pub, err := ecdsa.ParseUncompressedPublicKey(c.curve, append(append([]byte{4}, x...), y...))
		if err != nil {
			return nil, fmt.Errorf("key %q: %v", k.kid, err)
		}
		k.key = pub
		if strict {
			// the alg follows from the curve
			algs = []string{c.alg}
		}
	case "RSA":
		n, err := b64("n", r.N)
		if err != nil {
			return nil, err
		}
		e, err := b64("e", r.E)
		if err != nil {
			return nil, err
		}
		exp := new(big.Int).SetBytes(e)
		if !exp.IsInt64() || exp.Int64() < 3 || exp.Int64() > 1<<31-1 {
			return nil, fmt.Errorf("key %q: invalid RSA exponent", k.kid)
		}
		k.key = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exp.Int64())}
	case "OKP":
		if r.Crv != "Ed25519" {
			return nil, fmt.Errorf("key %q: unsupported curve %q", k.kid, r.Crv)
		}
		x, err := b64("x", r.X)
		if err != nil {
			return nil, err
		}
		if len(x) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("key %q: invalid Ed25519 key", k.kid)
		}
		k.key = ed25519.PublicKey(x)
	default:
		return nil, fmt.Errorf("key %q: unsupported key type %q", k.kid, r.Kty)
	}
	if r.Alg != nil {
		switch {
		case slices.Contains(algs, *r.Alg):
			algs = []string{*r.Alg}
		case strict:
			return nil, fmt.Errorf("key %q: alg %q does not fit a %s key", k.kid, *r.Alg, describeKey(r))
		default:
			algs = nil
		}
	}
	if r.Use == "enc" {
		if strict {
			return nil, fmt.Errorf("key %q: an encryption key (\"use\":\"enc\")", k.kid)
		}
		algs = nil
	}
	k.algs = algs
	return k, nil
}

func describeKey(r rawJWK) string {
	if r.Crv != "" {
		return r.Kty + " " + r.Crv
	}
	return r.Kty
}

// parseJWKSet reads a JWK set, {"keys": [...]}.
func parseJWKSet(data []byte, strict bool) ([]*jwk, error) {
	var set struct {
		Keys *[]json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, err
	}
	if set.Keys == nil {
		return nil, fmt.Errorf("not a JWK set: no \"keys\"")
	}
	return parseKeys(*set.Keys, strict)
}

func parseKeys(raw []json.RawMessage, strict bool) ([]*jwk, error) {
	keys := make([]*jwk, 0, len(raw))
	for _, r := range raw {
		k, err := parseJWK(r, strict)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// parseSecretKeys reads JWTSecret as PostgREST reads jwt-secret (Config.hs,
// parseSecret): a JWK set, else a JWK, else a passphrase. The passphrase is an
// HS256 key without kid, so a token with a kid never verifies against it.
// It also says whether the secret is a passphrase.
func parseSecretKeys(secret []byte) ([]*jwk, bool) {
	if keys, err := parseJWKSet(secret, false); err == nil {
		return keys, false
	}
	if k, err := parseJWK(secret, false); err == nil {
		return []*jwk{k}, false
	}
	return []*jwk{{kty: "oct", algs: []string{"HS256"}, key: secret}}, true
}

// parseKeyCatalog reads JWTKeys: a JSON array of JWKs, or a JWK set, parsed
// strictly, every kid unique.
func parseKeyCatalog(s string) ([]*jwk, error) {
	data := []byte(strings.TrimSpace(s))
	var keys []*jwk
	var err error
	if bytes.HasPrefix(data, []byte("[")) {
		var raw []json.RawMessage
		if err = json.Unmarshal(data, &raw); err == nil {
			keys, err = parseKeys(raw, true)
		}
	} else {
		keys, err = parseJWKSet(data, true)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid 'JWTKeys': %v", err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("invalid 'JWTKeys': no keys")
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k.kid] {
			return nil, fmt.Errorf("invalid 'JWTKeys': duplicate kid %q", k.kid)
		}
		seen[k.kid] = true
	}
	return keys, nil
}

// signingKey is the HMAC key that /token signs HS256 tokens with: the
// passphrase, or the one oct key of a JWK(S) secret that verifies HS256, so
// that the tokens it mints verify. Anything else has none.
func signingKey(keys []*jwk, passphrase bool) []byte {
	if passphrase {
		return keys[0].key.([]byte)
	}
	var found []byte
	for _, k := range keys {
		if k.kty == "oct" && slices.Contains(k.algs, "HS256") {
			if found != nil {
				return nil
			}
			found = k.key.([]byte)
		}
	}
	return found
}
