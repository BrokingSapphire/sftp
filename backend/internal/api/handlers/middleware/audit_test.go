package middleware

import (
	"net/http"
	"testing"
)

func TestDeriveAction(t *testing.T) {
	cases := []struct {
		method, path   string
		action, cat    string
		hasObject      bool
	}{
		{http.MethodPost, "/api/v1/files/upload", "file.upload", "file", false},
		{http.MethodPost, "/api/v1/folders/", "folder.create", "folder", false},
		{http.MethodPost, "/api/v1/shares/", "share.create", "share", false},
		{http.MethodDelete, "/api/v1/shares/9d89909d-ff88-481c-935e-12135feffa2e", "share.revoke", "share", true},
		{http.MethodPost, "/api/v1/users/", "user.create", "user", false},
		{http.MethodPost, "/api/v1/auth/login", "auth.login", "auth", false},
		{http.MethodPut, "/api/v1/files/9d89909d-ff88-481c-935e-12135feffa2e/rename", "file.rename", "file", true},
	}
	for _, c := range cases {
		action, cat, obj := deriveAction(c.method, c.path)
		if action != c.action {
			t.Errorf("%s %s: action = %q, want %q", c.method, c.path, action, c.action)
		}
		if cat != c.cat {
			t.Errorf("%s %s: category = %q, want %q", c.method, c.path, cat, c.cat)
		}
		if (obj != "") != c.hasObject {
			t.Errorf("%s %s: object=%q, wantObject=%v", c.method, c.path, obj, c.hasObject)
		}
	}
}
