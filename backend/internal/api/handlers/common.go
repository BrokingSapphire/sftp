package handlers

import (
	"net/http"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/httpresponse"
)

// ResponseBuilder is re-exported for handler convenience.
type ResponseBuilder = httpresponse.ResponseBuilder

// NewResponse creates a ResponseBuilder bound to the request.
func NewResponse(w http.ResponseWriter, r *http.Request) *ResponseBuilder {
	return httpresponse.NewResponse(w, r)
}

// NotFoundHandler handles unmatched routes (404).
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	httpresponse.NewResponse(w, r).NotFound(httpresponse.ErrTypeNotFound, "The requested resource was not found")
}

// MethodNotAllowedHandler handles unsupported methods (405).
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	httpresponse.NewResponse(w, r).
		Error(apperrors.CodeInvalidRequest, httpresponse.ErrTypeGeneric, "The requested method is not allowed").
		StatusCode(http.StatusMethodNotAllowed).Send()
}
