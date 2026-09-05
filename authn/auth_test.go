package authn

import "testing"

// A bearer token must never verify against an empty HMAC key: with no secret
// configured (allowed when LoginMode is "none"), anyone could forge a token for
// any role. authenticate must reject every token when the secret is empty,
// while a configured secret keeps working.
func TestAuthenticateRejectsEmptySecret(t *testing.T) {
	forged, err := GenerateToken("postgres", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authenticate(forged, ""); err == nil {
		t.Fatal("a token signed with the empty key was accepted against an empty secret")
	}

	// positive control: the same flow with a real secret authenticates
	good, err := GenerateToken("user1", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := authenticate(good, "s3cret")
	if err != nil {
		t.Fatalf("expected the token to authenticate with its secret, got %v", err)
	}
	if claims.Role != "user1" {
		t.Errorf("expected role user1, got %q", claims.Role)
	}
}
