package server

import (
	"strings"
	"testing"
)

// An empty JWT secret with authentication enabled means tokens are signed/verified
// with an empty HMAC key — anyone can forge a token for any role. The server must
// refuse to start rather than only warn.
func TestCheckConfigRejectsEmptyJWTSecret(t *testing.T) {
	cfg := defaultConfig()
	cfg.LoginMode = "jwt"
	cfg.JWTSecret = ""

	err := checkConfig(cfg)
	if err == nil {
		t.Fatal("expected checkConfig to reject an empty JWTSecret with auth enabled")
	}
	if !strings.Contains(err.Error(), "JWTSecret") {
		t.Errorf("expected error to mention JWTSecret, got: %v", err)
	}
}

// Debug mode is a dev convenience and should not require the operator to set a
// secret by hand — but it must not fall back to the empty (forgeable) key. It
// should auto-generate a strong random secret.
func TestDebugModeGeneratesJWTSecret(t *testing.T) {
	t.Setenv("SMOOTHDB_DEBUG", "true")
	t.Setenv("SMOOTHDB_JWT_SECRET", "")

	cfg := defaultConfig()
	cfg.JWTSecret = ""
	getEnvironment(cfg)

	if cfg.JWTSecret == "" {
		t.Fatal("debug mode should auto-generate a JWTSecret instead of leaving it empty")
	}
	if len(cfg.JWTSecret) < 32 {
		t.Errorf("auto-generated secret too short (%d chars); want a strong random value", len(cfg.JWTSecret))
	}
}

// Verbose database errors (hint, detail) are a fingerprinting aid for an
// attacker. The safe default is off; development opts in.
func TestVerboseErrorsDefaultsToFalse(t *testing.T) {
	if defaultConfig().VerboseErrors {
		t.Fatal("VerboseErrors must default to false")
	}
}

// SessionMode selects how much a session caches: nothing ("none"), the verified
// claims and the prepared connection ("role"), or the claims only ("claims").
// Anything else is a configuration error, caught before the JWT secret check.
func TestCheckConfigValidatesSessionMode(t *testing.T) {
	cfg := defaultConfig()
	cfg.LoginMode = "jwt"
	cfg.JWTSecret = ""
	cfg.SessionMode = "bogus"
	err := checkConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "SessionMode") {
		t.Fatalf("expected a SessionMode error, got %v", err)
	}
	cfg.SessionMode = "claims"
	err = checkConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "JWTSecret") {
		t.Fatalf("\"claims\" must pass the mode check and reach the secret check, got %v", err)
	}
}

// The JWT settings of PostgREST (jwt-role-claim-key, jwt-aud,
// jwt-secret-is-base64) are checked at startup, as PostgREST refuses to start
// on an invalid value, before the database check.
func TestCheckConfigRejectsInvalidJWTSettings(t *testing.T) {
	cases := []struct {
		name, key string
		set       func(*Config)
	}{
		{"role claim key", "JWTRoleClaimKey", func(c *Config) { c.JWTRoleClaimKey = "role.other" }},
		{"audience", "JWTAud", func(c *Config) { c.JWTAud = "http://x y" }},
		{"base64 secret", "JWTSecret", func(c *Config) { c.JWTSecret = "!!!notbase64!!!"; c.JWTSecretIsBase64 = true }},
	}
	for _, c := range cases {
		cfg := defaultConfig()
		cfg.JWTSecret = "secret"
		c.set(cfg)
		err := checkConfig(cfg)
		if err == nil || !strings.Contains(err.Error(), c.key) {
			t.Errorf("%s: expected an error naming %s, got %v", c.name, c.key, err)
		}
	}
}

// The new JWT settings come from the environment like SMOOTHDB_JWT_SECRET.
func TestJWTSettingsFromEnvironment(t *testing.T) {
	t.Setenv("SMOOTHDB_JWT_ROLE_CLAIM_KEY", "$.app.role")
	t.Setenv("SMOOTHDB_JWT_AUD", "youraudience")
	t.Setenv("SMOOTHDB_JWT_SECRET_IS_BASE64", "true")
	cfg := defaultConfig()
	if cfg.JWTRoleClaimKey != "$.role" || cfg.JWTAud != "" || cfg.JWTSecretIsBase64 {
		t.Fatalf("defaults: %q %q %v", cfg.JWTRoleClaimKey, cfg.JWTAud, cfg.JWTSecretIsBase64)
	}
	getEnvironment(cfg)
	if cfg.JWTRoleClaimKey != "$.app.role" || cfg.JWTAud != "youraudience" || !cfg.JWTSecretIsBase64 {
		t.Errorf("from the environment: %q %q %v", cfg.JWTRoleClaimKey, cfg.JWTAud, cfg.JWTSecretIsBase64)
	}
}

// JWTKeys is read from SMOOTHDB_JWT_KEYS and checked at startup; a JWTSecret
// that is a JWK(S) must hold the key /token signs with when LoginMode is "db".
func TestJWTKeysConfig(t *testing.T) {
	t.Setenv("SMOOTHDB_JWT_KEYS", `[{"kty":"oct","k":"c2VjcmV0","kid":"legacy"}]`)
	cfg := defaultConfig()
	getEnvironment(cfg)
	if cfg.JWTKeys != `[{"kty":"oct","k":"c2VjcmV0","kid":"legacy"}]` {
		t.Fatalf("JWTKeys from the environment: %q", cfg.JWTKeys)
	}

	cfg = defaultConfig()
	cfg.JWTSecret = "secret"
	cfg.JWTKeys = `[{"kty":"oct","k":"c2VjcmV0"}]` // no kid
	if err := checkConfig(cfg); err == nil || !strings.Contains(err.Error(), "JWTKeys") {
		t.Errorf("an invalid JWTKeys must stop the server, got %v", err)
	}

	cfg = defaultConfig()
	cfg.LoginMode = "db"
	cfg.JWTSecret = `{"kty":"EC","crv":"P-256","x":"dQkl4PP7doIatL26VhEwCWO-DLZpO5lgOJRv0DV7lwI","y":"6O9vSWaLewDQmn_aKz99BV8uHM4ksFy7BL9AbsmoPJ0"}`
	if err := checkConfig(cfg); err == nil || !strings.Contains(err.Error(), "JWTSecret") {
		t.Errorf("LoginMode db needs an HMAC key to sign with, got %v", err)
	}
}

// Database.MaxRows (PostgREST's db-max-rows) is off by default and can be
// set from the environment, like the database URL.
func TestMaxRowsFromEnvironment(t *testing.T) {
	if defaultConfig().Database.MaxRows != 0 {
		t.Fatal("Database.MaxRows must default to 0, no limit")
	}
	t.Setenv("SMOOTHDB_DATABASE_MAX_ROWS", "1000")
	cfg := defaultConfig()
	getEnvironment(cfg)
	if cfg.Database.MaxRows != 1000 {
		t.Errorf("SMOOTHDB_DATABASE_MAX_ROWS: want 1000, got %d", cfg.Database.MaxRows)
	}
}

// The tokens of /token (LoginMode "db") carry the role in a top-level "role"
// claim: a JWTRoleClaimKey that looks elsewhere could not read the role of the
// tokens the server itself issues, so that combination stops the server. The
// default key, in both its spellings, is accepted.
func TestCheckConfigLoginModeDBNeedsTopLevelRole(t *testing.T) {
	for _, key := range []string{"$.app.role", "$.roles[0]", ".app.role"} {
		cfg := defaultConfig()
		cfg.LoginMode = "db"
		cfg.JWTSecret = "secret"
		cfg.JWTRoleClaimKey = key
		err := checkConfig(cfg)
		if err == nil || !strings.Contains(err.Error(), "JWTRoleClaimKey") || !strings.Contains(err.Error(), "LoginMode") {
			t.Errorf("%q: expected an error naming JWTRoleClaimKey and LoginMode, got %v", key, err)
		}
	}
	for _, key := range []string{"$.role", ".role", ""} {
		cfg := defaultConfig()
		cfg.LoginMode = "db"
		cfg.JWTSecret = "secret"
		cfg.JWTRoleClaimKey = key
		// the check before the database one: an unreachable database fails
		// past it
		cfg.Database.URL = "postgresql://nobody@localhost:1/none"
		if err := checkConfig(cfg); err == nil || strings.Contains(err.Error(), "JWTRoleClaimKey") {
			t.Errorf("%q: expected the database error only, got %v", key, err)
		}
	}
	// a custom key is fine when /token is off
	cfg := defaultConfig()
	cfg.LoginMode = "gotrue"
	cfg.JWTSecret = "secret"
	cfg.JWTRoleClaimKey = "$.app.role"
	cfg.Database.URL = "postgresql://nobody@localhost:1/none"
	if err := checkConfig(cfg); err == nil || strings.Contains(err.Error(), "JWTRoleClaimKey") {
		t.Errorf("gotrue with a custom key: expected the database error only, got %v", err)
	}
}
