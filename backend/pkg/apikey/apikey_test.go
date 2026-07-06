package apikey

import (
	"strings"
	"testing"
)

func TestNewFormatAndHash(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(g.Plaintext, "sftp_") {
		t.Fatalf("plaintext missing prefix: %q", g.Plaintext)
	}
	if !Valid(g.Plaintext) {
		t.Fatal("generated key should be valid")
	}
	if g.Prefix == "" || !strings.Contains(g.Plaintext, g.Prefix) {
		t.Fatal("prefix should appear in plaintext")
	}
	if Hash(g.Plaintext) != g.Hash {
		t.Fatal("hash mismatch")
	}
	if len(g.Hash) != 64 {
		t.Fatalf("expected hex sha256, got %d chars", len(g.Hash))
	}
}

func TestUnique(t *testing.T) {
	a, _ := New()
	b, _ := New()
	if a.Plaintext == b.Plaintext || a.Hash == b.Hash {
		t.Fatal("keys should be unique")
	}
}

func TestValid(t *testing.T) {
	cases := map[string]bool{
		"sftp_abc_def":  true,
		"sftp_abc":      false,
		"nope_abc_def":  false,
		"":              false,
	}
	for in, want := range cases {
		if Valid(in) != want {
			t.Errorf("Valid(%q) = %v, want %v", in, Valid(in), want)
		}
	}
}
