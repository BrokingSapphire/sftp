// Package dlp scans extracted document text for sensitive data (PII / financial
// identifiers) and derives a sensitivity level. It reuses the text the
// ingestion pipeline already extracts, so classification is essentially free.
//
// Patterns are tuned for Indian financial context (PAN, Aadhaar, IFSC) plus
// universal ones (credit cards with Luhn validation, email, phone).
package dlp

import (
	"regexp"
	"sort"
	"strings"
)

// Sensitivity levels, most sensitive last.
const (
	Public       = "public"
	Internal     = "internal"
	Confidential = "confidential"
	Restricted   = "restricted"
)

var (
	rePAN   = regexp.MustCompile(`\b[A-Z]{5}[0-9]{4}[A-Z]\b`)
	reIFSC  = regexp.MustCompile(`\b[A-Z]{4}0[A-Z0-9]{6}\b`)
	reAadhr = regexp.MustCompile(`\b\d{4}\s\d{4}\s\d{4}\b`)
	reEmail = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
	rePhone = regexp.MustCompile(`(?:\+?91[\-\s]?)?[6-9]\d{4}[\-\s]?\d{5}`)
	reCard  = regexp.MustCompile(`\b(?:\d{4}[ \-]){3}\d{1,4}\b|\b\d{15,16}\b`)
)

// Result is the outcome of scanning a document.
type Result struct {
	PIITypes    []string // e.g. ["pan", "credit_card"]
	Sensitivity string   // public | internal | confidential | restricted
}

// Scan classifies text. Empty text yields Public with no PII.
func Scan(text string) Result {
	if strings.TrimSpace(text) == "" {
		return Result{Sensitivity: Public}
	}
	found := map[string]bool{}

	if rePAN.MatchString(text) {
		found["pan"] = true
	}
	if reIFSC.MatchString(text) {
		found["ifsc"] = true
	}
	if reAadhr.MatchString(text) {
		found["aadhaar"] = true
	}
	if reEmail.MatchString(text) {
		found["email"] = true
	}
	if rePhone.MatchString(text) {
		found["phone"] = true
	}
	if hasValidCard(text) {
		found["credit_card"] = true
	}

	types := make([]string, 0, len(found))
	for t := range found {
		types = append(types, t)
	}
	sort.Strings(types)

	return Result{PIITypes: types, Sensitivity: sensitivity(found)}
}

// sensitivity derives a level from the detected categories.
func sensitivity(found map[string]bool) string {
	switch {
	case found["credit_card"] || found["aadhaar"] || found["pan"]:
		return Restricted
	case found["ifsc"] || found["phone"]:
		return Confidential
	case found["email"]:
		return Internal
	default:
		return Public
	}
}

// hasValidCard reports whether text contains a Luhn-valid 13–19 digit number,
// which strongly indicates a payment card (cuts false positives from long IDs).
func hasValidCard(text string) bool {
	for _, m := range reCard.FindAllString(text, -1) {
		digits := strip(m)
		if len(digits) >= 13 && len(digits) <= 19 && luhn(digits) {
			return true
		}
	}
	return false
}

func strip(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func luhn(digits string) bool {
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

// AtLeast reports whether level a is at least as sensitive as b.
func AtLeast(a, b string) bool {
	return rank(a) >= rank(b)
}

func rank(level string) int {
	switch level {
	case Restricted:
		return 3
	case Confidential:
		return 2
	case Internal:
		return 1
	default:
		return 0
	}
}
