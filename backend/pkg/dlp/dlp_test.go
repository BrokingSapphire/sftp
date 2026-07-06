package dlp

import (
	"slices"
	"testing"
)

func TestPAN(t *testing.T) {
	r := Scan("My PAN is ABCDE1234F please keep safe")
	if !slices.Contains(r.PIITypes, "pan") || r.Sensitivity != Restricted {
		t.Fatalf("got %+v", r)
	}
}

func TestAadhaar(t *testing.T) {
	r := Scan("Aadhaar 1234 5678 9012 on file")
	if !slices.Contains(r.PIITypes, "aadhaar") || r.Sensitivity != Restricted {
		t.Fatalf("got %+v", r)
	}
}

func TestCreditCardLuhn(t *testing.T) {
	// 4111111111111111 is a Luhn-valid test Visa number.
	r := Scan("card 4111 1111 1111 1111 exp 12/29")
	if !slices.Contains(r.PIITypes, "credit_card") || r.Sensitivity != Restricted {
		t.Fatalf("valid card not detected: %+v", r)
	}
	// A long but Luhn-invalid number should NOT be flagged as a card.
	r2 := Scan("order 1234567890123456 shipped")
	if slices.Contains(r2.PIITypes, "credit_card") {
		t.Fatalf("false positive card: %+v", r2)
	}
}

func TestEmailInternal(t *testing.T) {
	r := Scan("contact john@corp.com for details")
	if !slices.Contains(r.PIITypes, "email") || r.Sensitivity != Internal {
		t.Fatalf("got %+v", r)
	}
}

func TestPhoneConfidential(t *testing.T) {
	r := Scan("call me on +91 98765 43210")
	if !slices.Contains(r.PIITypes, "phone") || r.Sensitivity != Confidential {
		t.Fatalf("got %+v", r)
	}
}

func TestClean(t *testing.T) {
	r := Scan("just some ordinary text with no secrets")
	if len(r.PIITypes) != 0 || r.Sensitivity != Public {
		t.Fatalf("expected clean, got %+v", r)
	}
	if Scan("").Sensitivity != Public {
		t.Fatal("empty should be public")
	}
}

func TestAtLeast(t *testing.T) {
	if !AtLeast(Restricted, Confidential) || AtLeast(Internal, Restricted) {
		t.Fatal("AtLeast ordering wrong")
	}
}
