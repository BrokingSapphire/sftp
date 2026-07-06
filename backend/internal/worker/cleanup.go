// Package worker runs periodic background maintenance jobs.
package worker

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/pkg/textextract"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

func itoa(n int) string { return strconv.Itoa(n) }

// Cleaner runs storage/DB housekeeping on an interval:
//   - purge recycle-bin files older than the retention window (free storage)
//   - delete abandoned/expired upload sessions and their chunks
//   - deactivate expired share links
type Cleaner struct {
	q             *sftpdb.Queries
	store         *storage.Engine
	log           logger.Logger
	interval      time.Duration
	trashRetention time.Duration
	stop          chan struct{}
	done          chan struct{}
}

// NewCleaner builds the cleanup worker.
func NewCleaner(q *sftpdb.Queries, store *storage.Engine, log logger.Logger, interval time.Duration, trashRetentionDays int) *Cleaner {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Cleaner{
		q: q, store: store, log: log.Named("worker.cleaner"),
		interval:       interval,
		trashRetention: time.Duration(trashRetentionDays) * 24 * time.Hour,
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

// Start launches the worker loop (runs once immediately, then on the interval).
func (c *Cleaner) Start() {
	go func() {
		defer close(c.done)
		t := time.NewTicker(c.interval)
		defer t.Stop()
		c.runOnce()
		for {
			select {
			case <-c.stop:
				return
			case <-t.C:
				c.runOnce()
			}
		}
	}()
}

// Stop signals the worker and waits for it to finish the current cycle.
func (c *Cleaner) Stop() {
	close(c.stop)
	<-c.done
}

func (c *Cleaner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c.purgeTrash(ctx)
	c.purgeUploads(ctx)
	if err := c.q.DeactivateExpiredShares(ctx); err != nil {
		c.log.Error("deactivate expired shares failed", "err", err)
	}
	c.transferSweep(ctx)
	c.textIndexSweep(ctx)
}

// textIndexSweep extracts and stores searchable text for any files that were
// uploaded before indexing existed, or whose async index failed. Bounded per
// cycle to keep the worker light.
func (c *Cleaner) textIndexSweep(ctx context.Context) {
	rows, err := c.q.ListFilesMissingText(ctx, 100)
	if err != nil {
		c.log.Error("list unindexed files failed", "err", err)
		return
	}
	indexed := 0
	for _, r := range rows {
		var text string
		if textextract.Supported(r.Extension, r.MimeType) {
			rc, err := c.store.Open(r.StorageKey)
			if err != nil {
				c.log.Warn("open for indexing failed", "file", r.ID, "err", err)
				continue
			}
			text, err = textextract.Extract(r.Extension, r.MimeType, rc)
			rc.Close()
			if err != nil {
				c.log.Warn("extract failed", "file", r.ID, "err", err)
				continue
			}
		}
		if err := c.q.UpsertFileText(ctx, sftpdb.UpsertFileTextParams{
			FileID: r.ID, Content: text, Ocr: false, Bytes: int64(len(text)),
		}); err != nil {
			c.log.Warn("store file text failed", "file", r.ID, "err", err)
			continue
		}
		indexed++
	}
	if indexed > 0 {
		c.log.Info("text index backfill", "indexed", indexed)
	}
}

// transferSweep drives the inherited-files workflow: it reminds heirs every
// couple of days, escalates to a strict warning near the deadline, and disables
// accounts whose deadline has lapsed with files still un-actioned. It NEVER
// deletes the files themselves.
func (c *Cleaner) transferSweep(ctx context.Context) {
	rows, err := c.q.PendingTransfersByUser(ctx)
	if err != nil {
		c.log.Error("pending transfers query failed", "err", err)
		return
	}
	now := time.Now()
	for _, r := range rows {
		deadline, ok := r.EarliestDeadline.(time.Time)
		if !ok {
			continue
		}

		if now.After(deadline) {
			// Deadline lapsed with files still pending → disable the account.
			_ = c.q.SetUserActive(ctx, sftpdb.SetUserActiveParams{ID: r.OwnerID, IsActive: false})
			_ = c.q.RevokeAllUserSessions(ctx, r.OwnerID)
			c.notifyThrottled(ctx, r.OwnerID, "account_disabled", 24*time.Hour,
				"Account disabled",
				"Your account has been disabled because inherited files were not actioned in time. Contact a super admin to re-enable it.")
			c.log.Warn("disabled account for un-actioned inherited files", "user_id", r.OwnerID, "pending", r.PendingCount)
			continue
		}

		daysLeft := int(deadline.Sub(now).Hours() / 24)
		if deadline.Sub(now) <= 3*24*time.Hour {
			c.notifyThrottled(ctx, r.OwnerID, "transfer_warning", 2*24*time.Hour,
				"Action required — account will be disabled soon",
				"You still have inherited files to review. Your account will be disabled in "+plural(daysLeft, "day")+" if you do not keep or delete them.")
		} else {
			c.notifyThrottled(ctx, r.OwnerID, "transfer_reminder", 2*24*time.Hour,
				"Inherited files need your action",
				"You have "+itoa(int(r.PendingCount))+" inherited file(s) to keep or delete within "+plural(daysLeft, "day")+".")
		}
	}
}

// notifyThrottled creates a notification only if none of the same type was
// created within window (avoids spamming on every sweep).
func (c *Cleaner) notifyThrottled(ctx context.Context, user uuid.UUID, ntype string, window time.Duration, title, body string) {
	since := pgtype.Timestamptz{Time: time.Now().Add(-window), Valid: true}
	n, err := c.q.CountRecentNotifications(ctx, sftpdb.CountRecentNotificationsParams{UserID: user, Type: ntype, CreatedAt: since})
	if err != nil || n > 0 {
		return
	}
	link := "/inherited"
	_ = c.q.CreateNotification(ctx, sftpdb.CreateNotificationParams{
		UserID: user, Type: ntype, Title: title, Body: body, Link: &link, Metadata: []byte("{}"),
	})
}

func plural(n int, unit string) string {
	if n < 0 {
		n = 0
	}
	if n == 1 {
		return "1 " + unit
	}
	return itoa(n) + " " + unit + "s"
}

func (c *Cleaner) purgeTrash(ctx context.Context) {
	cutoff := pgtype.Timestamptz{Time: time.Now().Add(-c.trashRetention), Valid: true}
	keys, err := c.q.PurgeExpiredTrash(ctx, cutoff)
	if err != nil {
		c.log.Error("purge trash failed", "err", err)
		return
	}
	for _, key := range keys {
		if err := c.store.Delete(key); err != nil {
			c.log.Error("delete purged object failed", "key", key, "err", err)
		}
	}
	if len(keys) > 0 {
		c.log.Info("purged expired trash", "count", len(keys))
	}
}

func (c *Cleaner) purgeUploads(ctx context.Context) {
	rows, err := c.q.DeleteExpiredUploads(ctx)
	if err != nil {
		c.log.Error("delete expired uploads failed", "err", err)
		return
	}
	for _, u := range rows {
		c.store.CleanupUpload(u.ID.String())
	}
	if len(rows) > 0 {
		c.log.Info("cleaned expired uploads", "count", len(rows))
	}
}
