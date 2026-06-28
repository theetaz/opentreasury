package treasury

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionMigrationDefinesTreasuryTransactionsTable(t *testing.T) {
	migrationPath := filepath.Join("..", "..", "..", "..", "database", "migrations", "000001_create_treasury_transactions.sql")

	contents, err := os.ReadFile(migrationPath)

	require.NoError(t, err)
	require.Contains(t, string(contents), "CREATE TABLE treasury_transactions")
	require.Contains(t, string(contents), "id TEXT PRIMARY KEY")
	require.Contains(t, string(contents), "institution_id TEXT NOT NULL")
	require.Contains(t, string(contents), "amount_minor BIGINT NOT NULL")
	require.Contains(t, string(contents), "transaction_date DATE NOT NULL")
}
