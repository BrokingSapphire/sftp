// Package params provides validated extraction of path/query parameters for
// Fuego handlers.
package params

import (
	"strconv"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"
)

// Source is satisfied by any fuego.Context (ContextNoBody / ContextWithBody[B]).
type Source interface {
	QueryParam(name string) string
	PathParam(name string) string
}

func requiredQueryErr(name string) error {
	return fuego.BadRequestError{Title: name + " is a required query parameter"}
}

// UUIDPath validates a required, well-formed UUID path param.
func UUIDPath(c Source, name string) (uuid.UUID, error) {
	v := c.PathParam(name)
	if v == "" {
		return uuid.Nil, fuego.BadRequestError{Title: "invalid " + name}
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, fuego.BadRequestError{Title: "invalid " + name + ": must be a valid UUID"}
	}
	return id, nil
}

// StringPath validates a non-empty path segment (exactLen>0 enforces length).
func StringPath(c Source, name string, exactLen int) (string, error) {
	v := c.PathParam(name)
	if v == "" || (exactLen > 0 && len(v) != exactLen) {
		return "", fuego.BadRequestError{Title: "invalid " + name}
	}
	return v, nil
}

// UUIDQuery validates a required, well-formed UUID query param.
func UUIDQuery(c Source, name string) (uuid.UUID, error) {
	v := c.QueryParam(name)
	if v == "" {
		return uuid.Nil, requiredQueryErr(name)
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, fuego.BadRequestError{Title: "invalid " + name + ": must be a valid UUID"}
	}
	return id, nil
}

// IntQueryDefault returns an int query param or def when absent/invalid.
func IntQueryDefault(c Source, name string, def int) int {
	v := c.QueryParam(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
