// Package middleware holds shared Gin middleware: request IDs, structured
// request logging, panic recovery and security headers.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestIDHeader is the response/propagation header for the correlation ID.
const RequestIDHeader = "X-Request-ID"

// contextKey constants stored on the gin context.
const (
	CtxRequestID = "request_id"
	CtxUserID    = "user_id"
	CtxRoleID    = "role_id"
)

// RequestID assigns a correlation ID to every request (honouring an
// inbound X-Request-ID) and echoes it back on the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(CtxRequestID, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// Logger logs one structured line per request with method, path, status,
// latency, client IP, user agent and correlation ID.
func Logger(log *zap.Logger) gin.HandlerFunc {
	log = log.Named("http")
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		fields := []zap.Field{
			zap.String("request_id", c.GetString(CtxRequestID)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.Int("bytes", c.Writer.Size()),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if uid := c.GetString(CtxUserID); uid != "" {
			fields = append(fields, zap.String("user_id", uid))
		}

		status := c.Writer.Status()
		switch {
		case len(c.Errors) > 0:
			fields = append(fields, zap.String("errors", c.Errors.String()))
			log.Error("request", fields...)
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery converts panics into a 500 JSON response and logs the stack.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	log = log.Named("recovery")
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("request_id", c.GetString(CtxRequestID)),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stack"),
				)
				c.AbortWithStatusJSON(500, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "internal_error",
						"message": "an unexpected error occurred",
					},
				})
			}
		}()
		c.Next()
	}
}
