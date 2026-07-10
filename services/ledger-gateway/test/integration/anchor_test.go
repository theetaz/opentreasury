//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/opentreasury/opentreasury/services/ledger-gateway/internal/anchor"
	"github.com/opentreasury/opentreasury/services/ledger-gateway/internal/merkle"
)

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)
	return filepath.Join(append([]string{root}, parts...)...)
}

func startDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("opentreasury_test"),
		tcpostgres.WithUsername("opentreasury"),
		tcpostgres.WithPassword("opentreasury"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	migrator, err := migrate.New("file://"+repoPath(t, "database", "migrations"), "pgx5://"+strings.TrimPrefix(dsn, "postgres://"))
	require.NoError(t, err)
	require.NoError(t, migrator.Up())

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { _ = db.Close() })

	applySeeds(t, db)
	return db
}

func applySeeds(t *testing.T, db *sql.DB) {
	t.Helper()
	dir := repoPath(t, "database", "seeds")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		contents, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = db.Exec(string(contents))
		require.NoError(t, err)
	}
}

func postEntry(t *testing.T, db *sql.DB, id string, amount int64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO journal_entries (id, institution_id, fiscal_year, effective_date, description, status, entry_type)
		VALUES ($1, 'minfin', 2026, '2026-07-01', 'test', 'POSTED', 'STANDARD')`, id)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO journal_lines (entry_id, line_number, account_code, direction, amount_minor, currency)
		VALUES ($1, 1, '6202', 'DEBIT', $2, 'USD'), ($1, 2, '114', 'CREDIT', $2, 'USD')`, id, amount)
	require.NoError(t, err)
}

func TestAnchorAndIndependentlyVerify(t *testing.T) {
	db := startDB(t)
	ctx := context.Background()

	postEntry(t, db, "je-anchor-1", 125000)
	postEntry(t, db, "je-anchor-2", 40000)
	postEntry(t, db, "je-anchor-3", 999)

	service := anchor.NewService(db, anchor.NewTransparencyLog(db), slog.New(slog.DiscardHandler))
	count, err := service.AnchorPending(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, float64(3), testutil.ToFloat64(service.Metrics().EntriesAnchoredTotal),
		"anchoring must be observable in the gateway metrics")

	// Re-running is a no-op: everything is already anchored.
	again, err := service.AnchorPending(ctx)
	require.NoError(t, err)
	require.Zero(t, again)

	// Read the stored proof for one entry and verify it independently, exactly
	// as the standalone verifier and the public API do.
	var leafHash, root, proofJSON string
	err = db.QueryRow(`
		SELECT jea.leaf_hash, a.merkle_root, jea.proof
		FROM journal_entry_anchors jea JOIN anchors a ON a.id = jea.anchor_id
		WHERE jea.entry_id = 'je-anchor-2'
	`).Scan(&leafHash, &root, &proofJSON)
	require.NoError(t, err)

	var proof []merkle.ProofStep
	require.NoError(t, json.Unmarshal([]byte(proofJSON), &proof))

	// Recompute the leaf from the canonical entry — trusting nothing stored.
	recomputed := merkle.CanonicalHash(merkle.Entry{
		ID: "je-anchor-2", InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-07-01",
		Status: "POSTED", EntryType: "STANDARD",
		Lines: []merkle.Line{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 40000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 40000, Currency: "USD"},
		},
	})
	require.Equal(t, leafHash, recomputed, "recomputed leaf must match the anchored leaf")
	require.True(t, merkle.Verify(recomputed, proof, root), "the proof must reach the anchored root")

	// The tamper demo: a tampered entry no longer hashes to the anchored leaf,
	// so verification fails — the anchor exposes the modification.
	tampered := merkle.CanonicalHash(merkle.Entry{
		ID: "je-anchor-2", InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-07-01",
		Status: "POSTED", EntryType: "STANDARD",
		Lines: []merkle.Line{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 40001, Currency: "USD"}, // +1 cent
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 40001, Currency: "USD"},
		},
	})
	require.NotEqual(t, leafHash, tampered)
	require.False(t, merkle.Verify(tampered, proof, root), "a tampered entry must fail verification")
}

func TestTransparencyLogChainsSequentially(t *testing.T) {
	db := startDB(t)
	ctx := context.Background()
	log := anchor.NewTransparencyLog(db)

	ref1, err := log.Commit(ctx, "root-a", 1)
	require.NoError(t, err)
	ref2, err := log.Commit(ctx, "root-b", 1)
	require.NoError(t, err)
	require.NotEqual(t, ref1, ref2)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM transparency_log WHERE prev_hash != 'genesis'").Scan(&count))
	require.Equal(t, 1, count, "the second entry must chain the first")
}
