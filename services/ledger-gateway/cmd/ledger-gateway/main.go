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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// The anchoring backend is pluggable (ADR-0004): the transparency log is
	// the default; OPENTREASURY_ANCHOR_BACKEND=fabric commits roots to the
	// treasury chaincode on a Hyperledger Fabric network instead.
	var backend anchor.Backend = anchor.NewTransparencyLog(db)
	if os.Getenv("OPENTREASURY_ANCHOR_BACKEND") == "fabric" {
		fabricBackend, closeFabric, err := anchor.ConnectFabric(anchor.FabricConfig{
			PeerEndpoint: envOr("OPENTREASURY_FABRIC_PEER_ENDPOINT", "peer0.opentreasury.local:7051"),
			MSPID:        envOr("OPENTREASURY_FABRIC_MSP_ID", "TreasuryMSP"),
			CertPath:     os.Getenv("OPENTREASURY_FABRIC_CERT_PATH"),
			KeyPath:      os.Getenv("OPENTREASURY_FABRIC_KEY_PATH"),
			TLSCertPath:  os.Getenv("OPENTREASURY_FABRIC_TLS_CERT_PATH"),
			Channel:      envOr("OPENTREASURY_FABRIC_CHANNEL", "opentreasury"),
			Chaincode:    envOr("OPENTREASURY_FABRIC_CHAINCODE", "treasury"),
		})
		if err != nil {
			logger.Error("connecting to Fabric", "error", err)
			os.Exit(1)
		}
		defer closeFabric()
		backend = fabricBackend
	}
	service := anchor.NewService(db, backend, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Serve Prometheus metrics and health on a side listener; the anchoring
	// loop itself has no API surface.
	metricsAddr := os.Getenv("OPENTREASURY_METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9464"
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", service.Metrics().Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	metricsServer := &http.Server{Addr: metricsAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics listener failed", "error", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("ledger gateway exited", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
