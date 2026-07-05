// Package worker runs periodic background maintenance jobs.
package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

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
