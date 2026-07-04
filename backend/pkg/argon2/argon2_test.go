package argon2

import "testing"

func TestHashVerify(t *testing.T) {
	p := DefaultParams()
	hash, err := Hash("correct-horse-battery", p)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := Verify("correct-horse-battery", hash)
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}

	ok, err = Verify("wrong-password", hash)
	if err != nil {
		t.Fatalf("verify wrong: %v", err)
	}
	if ok {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	if _, err := Verify("x", "not-a-phc-string"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestHashUnique(t *testing.T) {
	p := DefaultParams()
	a, _ := Hash("same", p)
	b, _ := Hash("same", p)
	if a == b {
		t.Fatal("expected different salts to yield different hashes")
	}
}
