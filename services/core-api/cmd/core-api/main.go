package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/opentreasury/opentreasury/services/core-api/internal/httpapi"
	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type config struct {
	addr           string
	databaseDSN    string
	allowedOrigins []string
}

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
	startupPingWait   = 5 * time.Second

	dbMaxOpenConns    = 25
	dbMaxIdleConns    = 10
	dbConnMaxLifetime = 30 * time.Minute
	dbConnMaxIdleTime = 5 * time.Minute
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, loadConfig(), logger); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}

// run boots the server and blocks until ctx is cancelled (then shuts down
// gracefully, draining in-flight requests) or the listener fails.
func run(ctx context.Context, cfg config, logger *slog.Logger) error {
	db, closeDatabase, err := openDatabase(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeDatabase(); err != nil {
			logger.Error("closing database", "error", err)
		}
	}()

	if db != nil {
		pingCtx, cancel := context.WithTimeout(ctx, startupPingWait)
		if err := db.PingContext(pingCtx); err != nil {
			// Not fatal: readiness reports the outage and the database may
			// come up after us (e.g. compose startup ordering).
			logger.Warn("database unreachable at startup", "error", err)
		}
		cancel()
	}

	srv := newServer(cfg, db)

	listenErr := make(chan error, 1)
	go func() {
		logger.Info("core api listening", "addr", cfg.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}
	}()

	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining requests")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func loadConfig() config {
	addr := os.Getenv("OPENTREASURY_CORE_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return config{
		addr:           addr,
		databaseDSN:    os.Getenv("OPENTREASURY_DATABASE_DSN"),
		allowedOrigins: splitCSV(os.Getenv("OPENTREASURY_ALLOWED_ORIGINS")),
	}
}

func newServer(cfg config, db *sql.DB) *http.Server {
	options := []httpapi.RouterOption{
		httpapi.WithAllowedOrigins(cfg.allowedOrigins),
	}
	if db != nil {
		options = append(
			options,
			httpapi.WithTransactionRepository(treasury.NewPostgresTransactionRepository(db)),
			httpapi.WithInstitutionRepository(treasury.NewPostgresInstitutionRepository(db)),
			httpapi.WithAccountRepository(treasury.NewPostgresAccountRepository(db)),
			httpapi.WithReadinessCheck(db.PingContext),
		)
	}

	return &http.Server{
		Addr:              cfg.addr,
		Handler:           httpapi.NewRouter(options...),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}

func openDatabase(cfg config) (*sql.DB, func() error, error) {
	if cfg.databaseDSN == "" {
		return nil, func() error { return nil }, nil
	}

	db, err := sql.Open("pgx", cfg.databaseDSN)
	if err != nil {
		return nil, nil, err
	}

	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)
	db.SetConnMaxLifetime(dbConnMaxLifetime)
	db.SetConnMaxIdleTime(dbConnMaxIdleTime)

	return db, db.Close, nil
}
