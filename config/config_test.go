package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The config file holds secrets (JWT secret and the DB URL with its password),
// so it must not be world-readable/writable. Regression test for the previous
// 0777 mode.
func TestSaveConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.jsonc")

	type cfg struct {
		JWTSecret string `comment:"secret"`
	}
	if err := SaveConfig(&cfg{JWTSecret: "s3cr3t"}, path); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file mode = %o, want 600 (secrets must not be group/world accessible)", perm)
	}
}
