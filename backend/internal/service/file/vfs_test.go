package file

import "testing"

func TestCleanPath(t *testing.T) {
	cases := map[string]string{
		"":               "/",
		"/":              "/",
		"a/b":            "/a/b",
		"/a/b/":          "/a/b",
		"/a/../b":        "/b",
		"../../etc":      "/etc",
		"/a//b":          "/a/b",
		"./x":            "/x",
	}
	for in, want := range cases {
		if got := cleanPath(in); got != want {
			t.Errorf("cleanPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCopyName(t *testing.T) {
	cases := map[string]string{
		"report.pdf":  "report (copy).pdf",
		"noext":       "noext (copy)",
		"a.tar.gz":    "a.tar (copy).gz",
		".env":        " (copy).env",
	}
	for in, want := range cases {
		if got := copyName(in); got != want {
			t.Errorf("copyName(%q) = %q, want %q", in, got, want)
		}
	}
}
