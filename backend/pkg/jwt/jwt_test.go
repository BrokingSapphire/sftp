package jwt

import (
	"testing"
	"time"
)

func TestIssueVerify(t *testing.T) {
	m := NewManager("test-secret-that-is-long-enough-32chars", "sftp_service", 15*time.Minute)

	token, exp, err := m.Issue("user-1", "u@example.com", "user1", "admin", "sess-1")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if exp.Before(time.Now()) {
		t.Fatal("expiry in the past")
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Sub == nil || *claims.Sub != "user-1" {
		t.Fatalf("bad sub: %+v", claims.Sub)
	}
	if claims.Role != "admin" || claims.SessionID != "sess-1" {
		t.Fatalf("bad claims: %+v", claims)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	m1 := NewManager("secret-one-that-is-long-enough-32chars", "sftp_service", time.Minute)
	m2 := NewManager("secret-two-that-is-long-enough-32chars", "sftp_service", time.Minute)

	token, _, _ := m1.Issue("u", "e", "u", "r", "s")
	if _, err := m2.Verify(token); err == nil {
		t.Fatal("expected verification failure with wrong secret")
	}
}

func TestVerifyExpired(t *testing.T) {
	m := NewManager("secret-that-is-definitely-32-chars!!", "sftp_service", -time.Minute)
	token, _, _ := m.Issue("u", "e", "u", "r", "s")
	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected expired token to fail verification")
	}
}
