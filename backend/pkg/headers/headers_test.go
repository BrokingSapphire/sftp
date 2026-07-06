package headers

import (
	"net/http"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	t.Run("x-forwarded-for first", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
		if ip := GetClientIP(r); ip != "203.0.113.1" {
			t.Fatalf("got %q", ip)
		}
	})
	t.Run("x-real-ip", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("X-Real-IP", "198.51.100.7")
		if ip := GetClientIP(r); ip != "198.51.100.7" {
			t.Fatalf("got %q", ip)
		}
	})
	t.Run("remote addr fallback", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.RemoteAddr = "192.0.2.9:5555"
		if ip := GetClientIP(r); ip != "192.0.2.9" {
			t.Fatalf("got %q", ip)
		}
	})
}

func TestParseDevice(t *testing.T) {
	d := ParseDevice("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0 Safari/537")
	if d.Browser != "Chrome" || d.OS != "Windows" {
		t.Fatalf("got %+v", d)
	}
	d = ParseDevice("curl/8.7.1")
	if d.Browser != "curl" {
		t.Fatalf("got %+v", d)
	}
	d = ParseDevice("")
	if d.Browser != "unknown" || d.OS != "unknown" {
		t.Fatalf("empty UA: %+v", d)
	}
}
