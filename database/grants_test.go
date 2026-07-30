package database

import (
	"slices"
	"testing"
)

// PostgreSQL 17 added the MAINTAIN privilege ('m'), so owner ACLs read
// "admin=arwdDxtm/admin"; the parser must accept every current aclitem letter
// or table grant introspection fails with "invalid privilege string".
func TestParsePrivilege(t *testing.T) {
	good := []struct {
		acl     string
		grantee string
		grantor string
		types   []string
	}{
		{"admin=arwdDxt/admin", "admin", "admin",
			[]string{"INSERT", "SELECT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"}},
		{"admin=arwdDxtm/admin", "admin", "admin",
			[]string{"INSERT", "SELECT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "MAINTAIN"}},
		{"=Tc/admin", "", "admin", []string{"TEMPORARY", "CONNECT"}},
		{"user1=CcTsA/admin", "user1", "admin",
			[]string{"CREATE", "CONNECT", "TEMPORARY", "SET", "ALTER SYSTEM"}},
		{"user1=XU/admin", "user1", "admin", []string{"EXECUTE", "USAGE"}},
	}
	for _, c := range good {
		var priv Privilege
		if err := parsePrivilege(c.acl, &priv); err != nil {
			t.Errorf("%s: unexpected error %q", c.acl, err)
			continue
		}
		if priv.Grantee != c.grantee || priv.Grantor != c.grantor {
			t.Errorf("%s: expected grantee %q grantor %q, got %q %q",
				c.acl, c.grantee, c.grantor, priv.Grantee, priv.Grantor)
		}
		if !slices.Equal(priv.Types, c.types) {
			t.Errorf("%s: expected types %v, got %v", c.acl, c.types, priv.Types)
		}
	}

	var priv Privilege
	if err := parsePrivilege("user1=z/admin", &priv); err == nil {
		t.Errorf("user1=z/admin: expected an error for an unknown privilege letter, got nil")
	}
}
