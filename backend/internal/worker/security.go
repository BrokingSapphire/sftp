package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// dedupeWindow suppresses repeat alerts of the same type+actor.
const dedupeWindow = time.Hour

// Detector scans the audit stream on an interval and raises security alerts for
// anomalous behaviour (data exfiltration, brute force, bulk destruction).
type Detector struct {
	q        *sftpdb.Queries
	log      logger.Logger
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewDetector builds the anomaly-detection worker.
func NewDetector(q *sftpdb.Queries, log logger.Logger, interval time.Duration) *Detector {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Detector{q: q, log: log.Named("worker.detector"), interval: interval, stop: make(chan struct{}), done: make(chan struct{})}
}

// Start runs the detector loop.
func (d *Detector) Start() {
	go func() {
		defer close(d.done)
		t := time.NewTicker(d.interval)
		defer t.Stop()
		d.runOnce()
		for {
			select {
			case <-d.stop:
				return
			case <-t.C:
				d.runOnce()
			}
		}
	}()
}

// Stop halts the detector.
func (d *Detector) Stop() {
	close(d.stop)
	<-d.done
}

// actionRule describes a volume-based rule over a set of audit actions.
type actionRule struct {
	typ       string
	severity  string
	actions   []string
	window    time.Duration
	threshold int
	label     string // human phrase, e.g. "downloaded"
}

var actionRules = []actionRule{
	{"mass_download", "high", []string{"file.download"}, 10 * time.Minute, 30, "downloaded"},
	{"bulk_delete", "high", []string{"file.delete", "file.trash"}, 10 * time.Minute, 20, "deleted"},
	{"share_spike", "medium", []string{"share.create"}, 15 * time.Minute, 15, "created shares"},
}

func (d *Detector) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for _, r := range actionRules {
		d.evalActionRule(ctx, r)
	}
	d.evalFailedLogins(ctx)
}

func (d *Detector) evalActionRule(ctx context.Context, r actionRule) {
	since := time.Now().Add(-r.window)
	rows, err := d.q.CountActionsByActor(ctx, sftpdb.CountActionsByActorParams{
		Since:     ts(since),
		Actions:   r.actions,
		Threshold: r.threshold,
	})
	if err != nil {
		d.log.Error("detector query failed", "type", r.typ, "err", err)
		return
	}
	for _, row := range rows {
		summary := fmt.Sprintf("%s %s %d files in %s", emailOr(row.ActorEmail, "a user"), r.label, row.N, humanDur(r.window))
		d.raise(ctx, r.typ, r.severity, row.ActorID, row.ActorEmail, summary, int(row.N), since)
	}
}

func (d *Detector) evalFailedLogins(ctx context.Context) {
	const window = 10 * time.Minute
	const threshold = 8
	since := time.Now().Add(-window)
	rows, err := d.q.CountFailedLoginsByEmail(ctx, sftpdb.CountFailedLoginsByEmailParams{
		Since: ts(since), Threshold: threshold,
	})
	if err != nil {
		d.log.Error("detector failed-login query failed", "err", err)
		return
	}
	for _, row := range rows {
		summary := fmt.Sprintf("%d failed sign-in attempts for %s in %s", row.N, emailOr(row.ActorEmail, "an account"), humanDur(window))
		d.raise(ctx, "failed_login_burst", "high", nil, row.ActorEmail, summary, int(row.N), since)
	}
}

// raise inserts an alert (deduped) and notifies super admins.
func (d *Detector) raise(ctx context.Context, typ, severity string, actorID *uuid.UUID, actorEmail *string, summary string, count int, windowStart time.Time) {
	exists, err := d.q.RecentAlertExists(ctx, sftpdb.RecentAlertExistsParams{
		Type: typ, ActorEmail: actorEmail, Since: ts(time.Now().Add(-dedupeWindow)),
	})
	if err != nil || exists {
		return
	}
	meta, _ := json.Marshal(map[string]any{"rule": typ})
	if _, err := d.q.InsertSecurityAlert(ctx, sftpdb.InsertSecurityAlertParams{
		Type: typ, Severity: severity, ActorID: actorID, ActorEmail: actorEmail,
		Summary: summary, EventCount: int32(count),
		WindowStart: ts(windowStart), WindowEnd: ts(time.Now()), Metadata: meta,
	}); err != nil {
		d.log.Error("insert alert failed", "type", typ, "err", err)
		return
	}
	d.log.Warn("security alert raised", "type", typ, "severity", severity, "summary", summary)
	d.notifySuperAdmins(ctx, summary)
}

func (d *Detector) notifySuperAdmins(ctx context.Context, summary string) {
	ids, err := d.q.ListSuperAdminIDs(ctx)
	if err != nil {
		return
	}
	link := "/admin/security"
	for _, id := range ids {
		_ = d.q.CreateNotification(ctx, sftpdb.CreateNotificationParams{
			UserID: id, Type: "security", Title: "Security alert", Body: summary, Link: &link, Metadata: []byte("{}"),
		})
	}
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func emailOr(e *string, fallback string) string {
	if e != nil && *e != "" {
		return *e
	}
	return fallback
}

func humanDur(d time.Duration) string {
	m := int(d.Minutes())
	if m == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", m)
}
