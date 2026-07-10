package demodata

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateIsDeterministicForASeed(t *testing.T) {
	a := Generate(Config{Seed: 42, Rows: 20, FiscalYear: 2026, Institutions: []string{"minfin", "health"}})
	b := Generate(Config{Seed: 42, Rows: 20, FiscalYear: 2026, Institutions: []string{"minfin", "health"}})
	require.Equal(t, a, b, "same seed must produce identical data — demos and docs stay reproducible")

	c := Generate(Config{Seed: 43, Rows: 20, FiscalYear: 2026, Institutions: []string{"minfin", "health"}})
	require.NotEqual(t, a, c, "a different seed must vary the data")
}

func TestGenerateMatchesTheInterchangeProfileColumns(t *testing.T) {
	records := Generate(Config{Seed: 1, Rows: 5, FiscalYear: 2026, Institutions: []string{"minfin"}})

	require.Len(t, records, 6, "header plus five rows")
	require.Equal(t, []string{"reference", "payment_date", "institution_code", "amount", "currency"}, records[0])

	seen := map[string]bool{}
	for _, row := range records[1:] {
		require.Len(t, row, 5)

		require.True(t, strings.HasPrefix(row[0], "DEMO-2026-"), "references are namespaced per fiscal year")
		require.False(t, seen[row[0]], "references must be unique: %s", row[0])
		seen[row[0]] = true

		date, err := time.Parse("2006-01-02", row[1])
		require.NoError(t, err)
		require.Equal(t, 2026, date.Year(), "dates fall inside the fiscal year")

		require.Equal(t, "minfin", row[2])

		amount, err := strconv.ParseFloat(row[3], 64)
		require.NoError(t, err)
		require.Greater(t, amount, 0.0)

		require.Equal(t, "USD", row[4])
	}
}
