// Package handlers implements HTTP handlers and the shared error plumbing.
package handlers

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 7807 problem+json body.
type Problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// WriteProblem writes an RFC 7807 error. The underlying cause is surfaced in
// `detail` only when debug errors are enabled (development); production stays
// {title,status} so internal details never leak.
func WriteProblem(w http.ResponseWriter, _ *http.Request, status int, message string, cause ...error) {
	p := Problem{Title: message, Status: status}
	if debugErrors && len(cause) > 0 && cause[0] != nil {
		p.Detail = cause[0].Error()
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
