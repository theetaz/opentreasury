//go:build integration

package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const foreignKeyViolationCode = "23503"

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)

	return filepath.Join(append([]string{root}, parts...)...)
}

func startPostgres(t *testing.T) (dsn string) {
	t.Helper()

	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("opentreasury_test"),
		tcpostgres.WithUsername("opentreasury"),
		tcpostgres.WithPassword("opentreasury"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	dsn, err = container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return dsn
}

func newMigrator(t *testing.T, dsn string) *migrate.Migrate {
	t.Helper()

	sourceURL := "file://" + repoPath(t, "database", "migrations")
	databaseURL := "pgx5://" + strings.TrimPrefix(dsn, "postgres://")

	migrator, err := migrate.New(sourceURL, databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		sourceErr, dbErr := migrator.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, dbErr)
	})

	return migrator
}

func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	require.NoError(t, db.Ping())

	return db
}

// applySeeds executes every database/seeds/*.sql file in order.
func applySeeds(t *testing.T, db *sql.DB) {
	t.Helper()

	seedsDir := repoPath(t, "database", "seeds")
	entries, err := os.ReadDir(seedsDir)
	require.NoError(t, err)

	var seedFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			seedFiles = append(seedFiles, filepath.Join(seedsDir, entry.Name()))
		}
	}
	sort.Strings(seedFiles)
	require.NotEmpty(t, seedFiles, "expected seed .sql files in database/seeds")

	for _, seedFile := range seedFiles {
		contents, err := os.ReadFile(seedFile)
		require.NoError(t, err)
		_, err = db.Exec(string(contents))
		require.NoErrorf(t, err, "seed file %s must apply cleanly", seedFile)
	}
}

func TestMigrationsApplyUpAndDownAgainstRealPostgres(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())

	for _, table := range []string{"treasury_transactions", "treasury_institutions"} {
		var exists bool
		err := db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		require.NoError(t, err)
		require.True(t, exists, "expected table %s to exist after up migrations", table)
	}

	require.NoError(t, migrator.Down())

	for _, table := range []string{"treasury_transactions", "treasury_institutions"} {
		var exists bool
		err := db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		require.NoError(t, err)
		require.False(t, exists, "expected table %s to be dropped after down migrations", table)
	}
}

func TestTransactionsRejectUnknownInstitution(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())

	_, err := db.Exec(
		`INSERT INTO treasury_transactions
		   (id, institution_id, fiscal_year, amount_minor, currency, description, transaction_date)
		 VALUES ('txn-fk-test', 'no-such-institution', 2026, 1000, 'USD', 'FK test', '2026-07-09')`,
	)
	require.Error(t, err, "insert referencing a missing institution must fail")
	require.Contains(t, err.Error(), foreignKeyViolationCode,
		"expected a foreign-key violation, got: %v", err)

	_, err = db.Exec(
		`INSERT INTO treasury_institutions (id, name, type, country_code, status)
		 VALUES ('inst-fk-test', 'FK Test Ministry', 'MINISTRY', 'KE', 'ACTIVE')`,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO treasury_transactions
		   (id, institution_id, fiscal_year, amount_minor, currency, description, transaction_date)
		 VALUES ('txn-fk-test', 'inst-fk-test', 2026, 1000, 'USD', 'FK test', '2026-07-09')`,
	)
	require.NoError(t, err, "insert referencing an existing institution must succeed")
}

func TestSeedsLoadAfterMigrations(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())

	applySeeds(t, db)

	var institutions int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM treasury_institutions").Scan(&institutions))
	require.GreaterOrEqual(t, institutions, 3, "seeds must provide at least the demo institutions")

	var transactions int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM treasury_transactions").Scan(&transactions))
	require.GreaterOrEqual(t, transactions, 1, "seeds must provide sample transactions")

	// Seeds must be idempotent: re-applying them is a no-op, not an error.
	applySeeds(t, db)
}
