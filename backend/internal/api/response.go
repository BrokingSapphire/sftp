// Package api wires HTTP routing, handlers and the server lifecycle.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the uniform JSON response shape.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// APIError is the machine-readable error body.
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// OK writes a 200 success envelope.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

// Created writes a 201 success envelope.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

// Paginated writes a 200 envelope with pagination metadata.
func Paginated(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: meta})
}

// Fail writes an error envelope with the given HTTP status.
func Fail(c *gin.Context, status int, code, message string, details interface{}) {
	c.AbortWithStatusJSON(status, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message, Details: details},
	})
}
