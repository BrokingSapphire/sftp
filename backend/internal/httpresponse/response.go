// Package httpresponse provides the shared, fluent HTTP response builder used
// by the handler and service layers so every API response has a uniform shape.
package httpresponse

import (
	"encoding/json"
	"net/http"
	"time"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/pkg/reqctx"
)

// Error-type string constants used in API responses.
const (
	ErrTypeGeneric        = "GENERIC_ERROR"
	ErrTypeValidation     = "VALIDATION_ERROR"
	ErrTypeAuthentication = "AUTHENTICATION_ERROR"
	ErrTypeAuthorization  = "AUTHORIZATION_ERROR"
	ErrTypeNotFound       = "NOT_FOUND"
	ErrTypeConflict       = "CONFLICT"
	ErrTypeRateLimit      = "RATE_LIMIT_ERROR"
	ErrTypeBusinessLogic  = "BUSINESS_LOGIC_ERROR"
	ErrTypeInternalServer = "INTERNAL_SERVER_ERROR"
)

// ErrorResponse is a single machine-readable error entry.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// APIResponse is the uniform envelope returned by all JSON endpoints.
type APIResponse struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message,omitempty"`
	Data      interface{}     `json:"data,omitempty"`
	Error     []ErrorResponse `json:"error,omitempty"`
	Meta      interface{}     `json:"meta,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

// ResponseBuilder builds API responses fluently.
type ResponseBuilder struct {
	response   APIResponse
	statusCode int
	headers    map[string]string
	w          http.ResponseWriter
	r          *http.Request
}

// NewResponse creates a ResponseBuilder bound to the current request.
func NewResponse(w http.ResponseWriter, r *http.Request) *ResponseBuilder {
	return &ResponseBuilder{
		response: APIResponse{
			RequestID: reqctx.GetRequestID(r.Context()),
			Timestamp: time.Now().UnixMilli(),
		},
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
		w:          w,
		r:          r,
	}
}

// Success marks the response successful.
func (rb *ResponseBuilder) Success() *ResponseBuilder {
	rb.response.Success = true
	rb.statusCode = http.StatusOK
	return rb
}

// ErrTypeCode returns a representative integer code for an error-type string.
func ErrTypeCode(errType string) int {
	switch errType {
	case ErrTypeValidation:
		return apperrors.CodeInvalidRequest
	case ErrTypeAuthentication:
		return apperrors.CodeUnauthorized
	case ErrTypeAuthorization:
		return apperrors.CodeForbidden
	case ErrTypeNotFound:
		return apperrors.CodeNotFound
	case ErrTypeConflict:
		return apperrors.CodeConflict
	case ErrTypeRateLimit:
		return apperrors.CodeRateLimitExceeded
	case ErrTypeBusinessLogic:
		return apperrors.CodeInvalidRequest
	default:
		return apperrors.CodeInternal
	}
}

// Error appends an error entry.
func (rb *ResponseBuilder) Error(errorCode int, errorType, message string) *ResponseBuilder {
	if errorCode == 0 {
		errorCode = ErrTypeCode(errorType)
	}
	rb.response.Success = false
	rb.response.Error = append(rb.response.Error, ErrorResponse{
		Code:    errorCode,
		Type:    errorType,
		Message: message,
	})
	if len(rb.response.Error) > 1 {
		rb.response.Message = "Multiple errors occurred"
	} else if rb.response.Message == "" {
		rb.response.Message = message
	}
	return rb
}

// Data sets the response payload.
func (rb *ResponseBuilder) Data(data interface{}) *ResponseBuilder {
	rb.response.Data = data
	return rb
}

// Meta sets pagination/metadata.
func (rb *ResponseBuilder) Meta(meta interface{}) *ResponseBuilder {
	rb.response.Meta = meta
	return rb
}

// Message sets the top-level message.
func (rb *ResponseBuilder) Message(message string) *ResponseBuilder {
	rb.response.Message = message
	return rb
}

// StatusCode overrides the HTTP status.
func (rb *ResponseBuilder) StatusCode(code int) *ResponseBuilder {
	rb.statusCode = code
	return rb
}

// Header adds a response header.
func (rb *ResponseBuilder) Header(key, value string) *ResponseBuilder {
	rb.headers[key] = value
	return rb
}

// Fail maps a domain error to status + error-type + code.
func (rb *ResponseBuilder) Fail(err error) *ResponseBuilder {
	if err == nil {
		return rb
	}
	status := apperrors.HTTPStatus(err)
	errType := apperrors.ErrTypeString(err)
	code := apperrors.ErrCode(err)
	rb.Error(code, errType, err.Error()).StatusCode(status)
	return rb
}

// Send writes the response.
func (rb *ResponseBuilder) Send() {
	rb.w.Header().Set("Content-Type", "application/json")
	for key, value := range rb.headers {
		rb.w.Header().Set(key, value)
	}
	rb.w.WriteHeader(rb.statusCode)
	_ = json.NewEncoder(rb.w).Encode(rb.response)
}

// OK sends a 200 with data.
func (rb *ResponseBuilder) OK(data interface{}) {
	rb.Success().Message("Request completed successfully").Data(data).Send()
}

// OKWithMessage sends a 200 with a custom message and data.
func (rb *ResponseBuilder) OKWithMessage(message string, data interface{}) {
	rb.Success().Message(message).Data(data).Send()
}

// Created sends a 201 with data.
func (rb *ResponseBuilder) Created(data interface{}) {
	rb.Success().Message("Resource created successfully").StatusCode(http.StatusCreated).Data(data).Send()
}

// NoContent sends a 204.
func (rb *ResponseBuilder) NoContent() {
	rb.Success().Message("Request completed successfully").StatusCode(http.StatusNoContent).Send()
}

// BadRequest sends a 400.
func (rb *ResponseBuilder) BadRequest(errorType, message string) {
	rb.Error(0, errorType, message).StatusCode(http.StatusBadRequest).Send()
}

// Unauthorized sends a 401.
func (rb *ResponseBuilder) Unauthorized(errorType, message string) {
	rb.Error(0, errorType, message).StatusCode(http.StatusUnauthorized).Send()
}

// Forbidden sends a 403.
func (rb *ResponseBuilder) Forbidden(errorType, message string) {
	rb.Error(0, errorType, message).StatusCode(http.StatusForbidden).Send()
}

// NotFound sends a 404.
func (rb *ResponseBuilder) NotFound(errorType, message string) {
	rb.Error(0, errorType, message).StatusCode(http.StatusNotFound).Send()
}

// TooManyRequests sends a 429.
func (rb *ResponseBuilder) TooManyRequests(errorType, message string) {
	rb.Error(0, errorType, message).StatusCode(http.StatusTooManyRequests).Send()
}

// InternalServerError sends a 500.
func (rb *ResponseBuilder) InternalServerError(message string) {
	rb.Error(0, ErrTypeInternalServer, message).StatusCode(http.StatusInternalServerError).Send()
}
