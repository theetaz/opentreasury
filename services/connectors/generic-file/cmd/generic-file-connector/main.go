// The generic-file connector is the universal on-ramp for government finance
// systems: it watches a drop directory for CSV exports, maps them through a
// declarative profile onto the interchange format, and delivers them to the
// staging API with at-least-once semantics (the API deduplicates by hash).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/opentreasury/opentreasury/services/connectors/generic-file/internal/connector"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dropDir := envOr("OPENTREASURY_DROP_DIR", "./dropbox")
	profilePath := envOr("OPENTREASURY_PROFILE_PATH", "./profile.yaml")
	stagingURL := envOr("OPENTREASURY_STAGING_URL", "http://localhost:8080")
	tokenURL := os.Getenv("OPENTREASURY_TOKEN_URL") // empty = unauthenticated local dev
	clientID := envOr("OPENTREASURY_CLIENT_ID", "opentreasury-connector")
	clientSecret := os.Getenv("OPENTREASURY_CLIENT_SECRET")

	profile, err := connector.LoadProfile(profilePath)
	if err != nil {
		logger.Error("loading mapping profile", "path", profilePath, "error", err)
		os.Exit(1)
	}

	deliverer := connector.NewDeliverer(stagingURL, tokenURL, clientID, clientSecret)
	watcher := connector.NewWatcher(dropDir, profile, deliverer, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := watcher.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("watcher exited", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
