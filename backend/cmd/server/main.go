// Command server is the entrypoint for the SFTP file-transfer platform.
package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sapphirebroking.com/sftp_service/internal/api"
	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/db"
	"sapphirebroking.com/sftp_service/migrations"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx)
	if err != nil {
		stdlog.Fatalf("failed to load config: %v", err)
	}

	appLogger := buildLogger(cfg)
	defer func() { _ = appLogger.Sync() }()
	appLogger.Info("configuration loaded", "environment", cfg.App.Environment, "version", cfg.App.Version)

	pool, err := db.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		appLogger.Fatal("failed to open database pool", "error", err)
	}
	defer pool.Close()
	appLogger.Info("database pool established")

	if err := db.Migrate(ctx, pool, migrations.FS, "sftp"); err != nil {
		appLogger.Fatal("failed to run migrations", "error", err)
	}
	appLogger.Info("migrations applied")

	httpServer := api.NewHttpServer(cfg.App.Port, api.Deps{
		Config:        cfg,
		Logger:        appLogger,
		HealthHandler: handlers.NewHealthHandler(pool, cfg.App.Version),
	})

	go httpServer.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	appLogger.Info("shutdown signal received", "signal", sig.String())

	gracefulCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(gracefulCtx); err != nil {
		appLogger.Error("HTTP server shutdown error", "error", err)
	} else {
		appLogger.Info("HTTP server stopped gracefully")
	}
}

// buildLogger constructs the logger from configured sinks (default: stdout).
func buildLogger(cfg *config.Config) logger.Logger {
	var sinks []logger.Sink
	if len(cfg.Logging.Sinks) == 0 {
		sinks = append(sinks, logger.NewStdoutSink(logger.ParseLevel(cfg.Logging.Level)))
	} else {
		for _, sc := range cfg.Logging.Sinks {
			level := logger.ParseLevel(cfg.Logging.Level)
			if sc.Level != "" {
				level = logger.ParseLevel(sc.Level)
			}
			switch sc.Type {
			case "stdout":
				sinks = append(sinks, logger.NewStdoutSink(level))
			case "file":
				sinks = append(sinks, logger.NewFileSink(logger.FileSinkConfig{
					Path:       sc.Path,
					MaxSizeMB:  sc.MaxSizeMB,
					MaxBackups: sc.MaxBackups,
					MaxAgeDays: sc.MaxAgeDays,
					Compress:   sc.Compress,
				}, level))
			default:
				stdlog.Fatalf("unknown log sink type: %s", sc.Type)
			}
		}
	}

	l, err := logger.BuildLogger(&cfg.Logging, sinks)
	if err != nil {
		stdlog.Fatalf("failed to build logger: %v", err)
	}
	return l
}
