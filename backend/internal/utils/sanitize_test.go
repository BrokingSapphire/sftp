package utils

import "testing"

func TestSanitizeName(t *testing.T) {
	good := []string{"report.pdf", "Q3 Report", "my_file-1.txt", "café.doc"}
	for _, n := range good {
		if _, err := SanitizeName(n); err != nil {
			t.Errorf("SanitizeName(%q) unexpected error: %v", n, err)
		}
	}
	bad := []string{"", ".", "..", "a/b", "a\\b", "../etc", "with\x00null", "  "}
	for _, n := range bad {
		if _, err := SanitizeName(n); err == nil {
			t.Errorf("SanitizeName(%q) should have errored", n)
		}
	}
}

func TestSanitizeTrims(t *testing.T) {
	got, err := SanitizeName("  spaced.txt  ")
	if err != nil || got != "spaced.txt" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestFileExtension(t *testing.T) {
	cases := map[string]string{
		"a.PDF": "pdf", "archive.tar.gz": "gz", "noext": "", "x.": "",
		".env": "env", // leading-dot files are treated as their suffix
	}
	for in, want := range cases {
		if got := FileExtension(in); got != want {
			t.Errorf("FileExtension(%q) = %q, want %q", in, got, want)
		}
	}
}
