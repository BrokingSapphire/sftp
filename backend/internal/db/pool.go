// Package db manages the pgx connection pool and transaction helpers.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a pgx connection pool to dsn and verifies reachability.
// On every fresh connection it discovers user-defined enum types and
// registers them (plus their array OIDs) with the pgx type map.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.AfterConnect = registerEnumTypes

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func registerEnumTypes(ctx context.Context, c *pgx.Conn) error {
	rows, err := c.Query(ctx, `
        SELECT t.typname
          FROM pg_type t
          JOIN pg_namespace n ON n.oid = t.typnamespace
         WHERE t.typtype = 'e' AND n.nspname = 'public'`)
	if err != nil {
		return fmt.Errorf("list enum types: %w", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("scan enum names: %w", err)
	}
	for _, name := range names {
		t, err := c.LoadType(ctx, name)
		if err != nil {
			return fmt.Errorf("load enum %q: %w", name, err)
		}
		c.TypeMap().RegisterType(t)
		if at, err := c.LoadType(ctx, "_"+name); err == nil {
			c.TypeMap().RegisterType(at)
		}
	}
	return nil
}
