package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sapphirebroking.com/sftp_service/pkg/ratelimit"
)

func TestRateLimitMiddleware(t *testing.T) {
	l := ratelimit.New(0, 2) // burst 2, no refill
	defer l.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := RateLimit(l)(next)

	call := func() int {
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "203.0.113.5:1111"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if call() != 200 || call() != 200 {
		t.Fatal("first two requests should pass (burst)")
	}
	if code := call(); code != http.StatusTooManyRequests {
		t.Fatalf("third request = %d, want 429", code)
	}
}

func TestRateLimitNilIsNoop(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := RateLimit(nil)(next)
	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("nil limiter should be a no-op, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	h := rec.Header()
	if h.Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q, want SAMEORIGIN", h.Get("X-Frame-Options"))
	}
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff")
	}
}
