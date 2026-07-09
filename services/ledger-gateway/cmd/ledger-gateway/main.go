// The ledger gateway anchors posted journal entries into a tamper-evident
// ledger. It batches unanchored entries, builds a Merkle tree over their
// canonical hashes, commits the root to the backend (a transparency log
// locally; Hyperledger Fabric in production), and records per-entry inclusion
// proofs the public API serves and the standalone verifier checks.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opentreasury/opentreasury/services/ledger-gateway/internal/anchor"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("OPENTREASURY_DATABASE_DSN")
	if dsn == "" {
		logger.Error("OPENTREASURY_DATABASE_DSN is required")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error("opening database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// The anchoring backend is pluggable; the transparency log is the default.
	// A Fabric backend satisfies the same interface (ADR-0004).
	backend := anchor.NewTransparencyLog(db)
	service := anchor.NewService(db, backend, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("ledger gateway exited", "error", err)
		os.Exit(1)
	}
}
