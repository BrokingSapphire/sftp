// Package audit provides a compliance-grade, append-only audit trail and a
// user-activity (click/telemetry) stream. Audit writes are asynchronous and
// best-effort-durable: entries queue to a background writer and fall back to a
// synchronous write if the queue is full, so records are never silently dropped.
package audit

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Result values.
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultDenied  = "denied"
)

// Entry is a single audit record.
type Entry struct {
	ActorID    *uuid.UUID
	ActorEmail string
	Action     string
	Category   string
	ObjectType string
	ObjectID   string
	ObjectName string
	Result     string
	IP         string
	UserAgent  string
	Browser    string
	OS         string
	RequestID  string
	Metadata   map[string]any
}

// Recorder writes audit and activity records.
type Recorder struct {
	q    *sftpdb.Queries
	log  logger.Logger
	ch   chan Entry
	done chan struct{}
}

// New starts the background audit writer.
func New(q *sftpdb.Queries, log logger.Logger) *Recorder {
	r := &Recorder{
		q:    q,
		log:  log.Named("audit"),
		ch:   make(chan Entry, 1024),
		done: make(chan struct{}),
	}
	go r.worker()
	return r
}

// Record enqueues an audit entry, falling back to a synchronous write if the
// buffer is full (never drops).
func (r *Recorder) Record(e Entry) {
	select {
	case r.ch <- e:
	default:
		r.write(context.Background(), e)
	}
}

// Close drains the queue and stops the writer.
func (r *Recorder) Close() {
	close(r.ch)
	<-r.done
}

func (r *Recorder) worker() {
	defer close(r.done)
	for e := range r.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		r.write(ctx, e)
		cancel()
	}
}

func (r *Recorder) write(ctx context.Context, e Entry) {
	if e.Result == "" {
		e.Result = ResultSuccess
	}
	meta := []byte("{}")
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			meta = b
		}
	}
	err := r.q.InsertAuditLog(ctx, sftpdb.InsertAuditLogParams{
		Action:     e.Action,
		Category:   e.Category,
		Result:     e.Result,
		Metadata:   meta,
		ActorID:    e.ActorID,
		ActorEmail: strPtr(e.ActorEmail),
		ObjectType: strPtr(e.ObjectType),
		ObjectID:   strPtr(e.ObjectID),
		ObjectName: strPtr(e.ObjectName),
		IpAddress:  parseAddr(e.IP),
		UserAgent:  strPtr(e.UserAgent),
		Browser:    strPtr(e.Browser),
		Os:         strPtr(e.OS),
		RequestID:  strPtr(e.RequestID),
	})
	if err != nil {
		r.log.Error("audit write failed", "action", e.Action, "err", err)
	}
}

// ── Activity (telemetry) ──────────────────────────────────

// ActivityEntry is a UI interaction event.
type ActivityEntry struct {
	UserID    *uuid.UUID
	SessionID *uuid.UUID
	EventType string
	Element   string
	Path      string
	IP        string
	UserAgent string
	Metadata  map[string]any
}

// RecordActivity persists a UI telemetry event (best-effort).
func (r *Recorder) RecordActivity(ctx context.Context, e ActivityEntry) {
	meta := []byte("{}")
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			meta = b
		}
	}
	err := r.q.InsertActivity(ctx, sftpdb.InsertActivityParams{
		EventType: e.EventType,
		Metadata:  meta,
		UserID:    e.UserID,
		SessionID: e.SessionID,
		Element:   strPtr(e.Element),
		Path:      strPtr(e.Path),
		IpAddress: parseAddr(e.IP),
		UserAgent: strPtr(e.UserAgent),
	})
	if err != nil {
		r.log.Error("activity write failed", "event", e.EventType, "err", err)
	}
}

// ── reads ──────────────────────────────────────────────────

// List returns recent audit logs.
func (r *Recorder) List(ctx context.Context, limit, offset int) ([]sftpdb.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return r.q.ListAuditLogs(ctx, sftpdb.ListAuditLogsParams{Limit: int32(limit), Offset: int32(offset)})
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseAddr(ip string) *netip.Addr {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	return &addr
}
