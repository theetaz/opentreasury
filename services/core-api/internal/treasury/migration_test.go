package treasury

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Structural checks that keep the migration set well-formed without a database.
// Behavior against real Postgres is covered by the integration suite
// (test/integration/migrations_test.go).

func migrationsDir() string {
	return filepath.Join("..", "..", "..", "..", "database", "migrations")
}

func TestEveryMigrationHasUpAndDownPair(t *testing.T) {
	entries, err := os.ReadDir(migrationsDir())
	require.NoError(t, err)

	ups := map[string]bool{}
	downs := map[string]bool{}

	for _, entry := range entries {
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			ups[strings.TrimSuffix(name, ".up.sql")] = true
		case strings.HasSuffix(name, ".down.sql"):
			downs[strings.TrimSuffix(name, ".down.sql")] = true
		}
	}

	require.NotEmpty(t, ups, "expected at least one .up.sql migration")

	for base := range ups {
		require.True(t, downs[base], "migration %s has no .down.sql pair", base)
	}
	for base := range downs {
		require.True(t, ups[base], "migration %s has no .up.sql pair", base)
	}
}

func TestMigrationsAreNonEmptyAndSequentiallyNumbered(t *testing.T) {
	entries, err := os.ReadDir(migrationsDir())
	require.NoError(t, err)

	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(migrationsDir(), name))
		require.NoError(t, err)
		require.NotEmpty(t, strings.TrimSpace(string(contents)), "migration %s is empty", name)

		prefix, _, found := strings.Cut(name, "_")
		require.True(t, found, "migration %s does not follow NNNNNN_name convention", name)
		require.Len(t, prefix, 6, "migration %s prefix must be 6 digits", name)
		seen[prefix] = true
	}

	for i := 1; i <= len(seen); i++ {
		want := fmt.Sprintf("%06d", i)
		require.True(t, seen[want], "migration sequence gap: %s missing", want)
	}
}

func TestForeignKeyMigrationLinksTransactionsToInstitutions(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join(migrationsDir(), "000003_add_transactions_institution_fk.up.sql"))
	require.NoError(t, err)

	require.Contains(t, string(contents), "FOREIGN KEY (institution_id)")
	require.Contains(t, string(contents), "REFERENCES treasury_institutions (id)")
}
