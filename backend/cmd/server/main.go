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
	authhandler "sapphirebroking.com/sftp_service/internal/api/handlers/auth"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	apikeyhandler "sapphirebroking.com/sftp_service/internal/api/handlers/apikey"
	audithandler "sapphirebroking.com/sftp_service/internal/api/handlers/audit"
	filehandler "sapphirebroking.com/sftp_service/internal/api/handlers/file"
	notifhandler "sapphirebroking.com/sftp_service/internal/api/handlers/notification"
	sharehandler "sapphirebroking.com/sftp_service/internal/api/handlers/share"
	ssohandler "sapphirebroking.com/sftp_service/internal/api/handlers/sso"
	userhandler "sapphirebroking.com/sftp_service/internal/api/handlers/user"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/db"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	apikeysvc "sapphirebroking.com/sftp_service/internal/service/apikey"
	auditsvc "sapphirebroking.com/sftp_service/internal/service/audit"
	authsvc "sapphirebroking.com/sftp_service/internal/service/auth"
	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	sharesvc "sapphirebroking.com/sftp_service/internal/service/share"
	ssosvc "sapphirebroking.com/sftp_service/internal/service/sso"
	usersvc "sapphirebroking.com/sftp_service/internal/service/user"
	"sapphirebroking.com/sftp_service/internal/sftpserver"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/internal/worker"
	"sapphirebroking.com/sftp_service/migrations"
	"sapphirebroking.com/sftp_service/pkg/cache"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
	"sapphirebroking.com/sftp_service/pkg/mailer"
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

	// Cache for the hot RBAC/permission path: Redis when configured (shared
	// across instances), otherwise an in-process TTL cache.
	var appCache cache.Cache
	if cfg.Redis.Enabled {
		if rc, err := cache.NewRedis(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB); err != nil {
			appLogger.Warn("redis unavailable; falling back to in-memory cache", "error", err)
			appCache = cache.NewMemory()
		} else {
			appLogger.Info("redis cache connected", "addr", cfg.Redis.Address)
			appCache = rc
		}
	} else {
		appCache = cache.NewMemory()
	}
	defer func() { _ = appCache.Close() }()

	// Build the data-access, auth and HTTP layers.
	queries := sftpdb.New(pool)
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL)
	authService := authsvc.New(authsvc.Deps{
		Queries:  queries,
		JWT:      jwtManager,
		Security: cfg.Security,
		Logger:   appLogger,
	})
	storageEngine, err := storage.New(cfg.Storage.RootPath, cfg.Storage.TempPath, cfg.Storage.EncryptionKey)
	if err != nil {
		appLogger.Fatal("failed to initialise storage engine", "error", err)
	}
	if storageEngine.Encrypted() {
		appLogger.Info("file storage encryption enabled (AES-256 at rest)")
	}

	userService := usersvc.New(usersvc.Deps{
		Queries:  queries,
		Storage:  storageEngine,
		Security: cfg.Security,
		Logger:   appLogger,
	})
	fileService := filesvc.New(filesvc.Deps{
		Queries:       queries,
		Storage:       storageEngine,
		Logger:        appLogger,
		ChunkSize:     cfg.Storage.ChunkSize,
		MaxUploadSize: cfg.Storage.MaxUploadSize,
	})
	apiKeyService := apikeysvc.New(queries, appLogger)
	mailSender := mailer.New(mailer.Config{
		Enabled: cfg.Mail.Enabled, Host: cfg.Mail.Host, Port: cfg.Mail.Port,
		Username: cfg.Mail.Username, Password: cfg.Mail.Password, From: cfg.Mail.From, StartTLS: cfg.Mail.StartTLS,
	}, appLogger)
	if mailSender.Enabled() {
		appLogger.Info("smtp mailer enabled", "host", cfg.Mail.Host)
	}
	shareService := sharesvc.New(sharesvc.Deps{
		Queries: queries, Storage: storageEngine, BaseURL: cfg.App.SelfBaseURL,
		Mailer: mailSender, OrgDomains: cfg.OrgDomains, Logger: appLogger,
	})
	auditRecorder := auditsvc.New(queries, appLogger)
	defer auditRecorder.Close()

	cleaner := worker.NewCleaner(queries, storageEngine, appLogger, time.Hour, cfg.Storage.TrashRetentionDays)
	cleaner.Start()
	defer cleaner.Stop()

	// Seed the first super-admin on an empty database.
	if err := userService.EnsureSuperAdmin(ctx, cfg.Bootstrap); err != nil {
		appLogger.Error("bootstrap super-admin failed", "error", err)
	}

	// Optional Microsoft Entra ID SSO (OIDC discovery at startup).
	msSSO, err := ssosvc.NewMicrosoft(ctx, cfg.SSO.Microsoft)
	if err != nil {
		appLogger.Error("microsoft sso disabled: initialisation failed", "error", err)
	} else if msSSO != nil {
		appLogger.Info("microsoft sso enabled", "tenant", cfg.SSO.Microsoft.TenantID)
	}

	httpServer := api.NewHttpServer(cfg.App.Port, api.Deps{
		CORSConfig:    cfg.CORS,
		Logger:        appLogger,
		DebugErrors:   cfg.IsDevelopment(),
		Auth:          m.NewAuthenticator(jwtManager, apiKeyService),
		Perms:         m.NewPermissions(queries, appCache),
		Recorder:      auditRecorder,
		HealthHandler: handlers.NewHealthHandler(pool, cfg.App.Version),
		AuthHandler:   authhandler.NewHandler(authService, appLogger),
		SSOHandler:    ssohandler.NewHandler(msSSO, authService, cfg.IsProduction(), appLogger),
		UserHandler:   userhandler.NewHandler(userService, appLogger),
		FileHandler:   filehandler.NewHandler(fileService, appLogger),
		APIKeyHandler: apikeyhandler.NewHandler(apiKeyService, appLogger),
		AuditHandler:  audithandler.NewHandler(auditRecorder, appLogger),
		ShareHandler:  sharehandler.NewHandler(shareService, appLogger),
		NotifHandler:  notifhandler.NewHandler(queries, appLogger),
	})

	go httpServer.Start()

	// Optional native SFTP-over-SSH endpoint (same storage + accounts).
	var sftpSrv *sftpserver.Server
	if cfg.SFTP.Enabled {
		sftpSrv, err = sftpserver.New(sftpserver.Deps{
			Config: cfg.SFTP, Auth: authService, APIKey: apiKeyService, Files: fileService, Logger: appLogger,
		})
		if err != nil {
			appLogger.Error("sftp server disabled: initialisation failed", "error", err)
		} else {
			go func() {
				if err := sftpSrv.Start(); err != nil {
					appLogger.Error("sftp server stopped", "error", err)
				}
			}()
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	appLogger.Info("shutdown signal received", "signal", sig.String())

	if sftpSrv != nil {
		_ = sftpSrv.Close()
	}

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
