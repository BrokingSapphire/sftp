// Command server is the entrypoint for the SFTP platform API.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/BrokingSapphire/sftp/backend/internal/api"
	"github.com/BrokingSapphire/sftp/backend/internal/config"
	"github.com/BrokingSapphire/sftp/backend/internal/database"
	"github.com/BrokingSapphire/sftp/backend/internal/logger"
	"github.com/BrokingSapphire/sftp/backend/migrations"
	"go.uber.org/zap"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("load config: " + err.Error())
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		panic("init logger: " + err.Error())
	}
	defer func() { _ = log.Sync() }()

	log.Info("starting sftp platform",
		zap.String("version", version),
		zap.String("env", cfg.Server.Env),
	)

	db, err := database.Connect(cfg.Database, log, cfg.IsProduction())
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer func() { _ = database.Close(db) }()

	if err := database.Migrate(db, migrations.FS, log); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}

	router := api.NewRouter(api.Deps{
		Config:  cfg,
		Logger:  log,
		DB:      db,
		Version: version,
	})
	srv := api.NewServer(cfg, router, log)

	// Run the HTTP server; capture a fatal startup error.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for a termination signal or a server error.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatal("server error", zap.Error(err))
	case sig := <-stop:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("server stopped")
}
