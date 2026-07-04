package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var startedAt = time.Now()

// HealthHandler serves liveness/readiness probes.
type HealthHandler struct {
	db      *gorm.DB
	version string
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(db *gorm.DB, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// Live is a cheap liveness probe (process is up).
func (h *HealthHandler) Live(c *gin.Context) {
	OK(c, gin.H{"status": "alive"})
}

// Ready verifies dependencies (database) are reachable.
func (h *HealthHandler) Ready(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err == nil {
		err = sqlDB.Ping()
	}
	if err != nil {
		Fail(c, http.StatusServiceUnavailable, "not_ready", "database unreachable", nil)
		return
	}
	OK(c, gin.H{"status": "ready", "database": "ok"})
}

// Info returns build/runtime metadata.
func (h *HealthHandler) Info(c *gin.Context) {
	OK(c, gin.H{
		"version":     h.version,
		"uptime":      time.Since(startedAt).String(),
		"go_version":  runtime.Version(),
		"goroutines":  runtime.NumGoroutine(),
		"num_cpu":     runtime.NumCPU(),
		"started_at":  startedAt.UTC().Format(time.RFC3339),
	})
}
