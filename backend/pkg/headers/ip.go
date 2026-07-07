// Package headers extracts common request metadata (client IP, device info).
package headers

import (
	"net/http"
	"strings"
)

// GetClientIP resolves the originating client IP, honouring proxy headers
// set by a trusted reverse proxy (Nginx) in front of the service.
func GetClientIP(r *http.Request) string {
	// Prefer X-Real-IP: our nginx sets it to the direct peer ($remote_addr),
	// which a client cannot spoof. X-Forwarded-For's left-most entry is
	// client-supplied and only used as a fallback.
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}
