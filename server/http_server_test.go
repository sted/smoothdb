package server

import "testing"

// A configured-but-unloadable TLS certificate must be a hard error, not a silent
// fall-through to plaintext HTTP (which would expose Bearer tokens and login
// credentials on the wire).
func TestLoadTLSConfigFailsClosed(t *testing.T) {
	_, err := loadTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected an error when the certificate cannot be loaded")
	}
}

// The dev certificate in the repo root should load into a usable TLS config.
func TestLoadTLSConfigValidCert(t *testing.T) {
	cfg, err := loadTLSConfig("../localhost.pem", "../localhost-key.pem")
	if err != nil {
		t.Fatalf("expected valid cert to load, got: %v", err)
	}
	if cfg == nil || len(cfg.Certificates) == 0 {
		t.Fatal("expected a TLS config with a certificate")
	}
}
