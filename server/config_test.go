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
