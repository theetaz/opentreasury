// Package demodata generates reproducible demo payment exports in the
// interchange CSV shape the generic file connector maps (see
// profiles/itmis-payments.yaml). Pilots use it to exercise the full
// ingestion → posting → anchoring pipeline before any real system is wired.
package demodata

import (
	"fmt"
	"math/rand"
	"time"
)

type Config struct {
	// Seed makes output reproducible: the same seed yields the same file.
	Seed         int64
	Rows         int
	FiscalYear   int
	Institutions []string
}

// Generate returns CSV records (header first) with one demo payment per row.
func Generate(config Config) [][]string {
	random := rand.New(rand.NewSource(config.Seed)) // #nosec G404 -- reproducible demo data, not cryptography

	records := [][]string{{"reference", "payment_date", "institution_code", "amount", "currency"}}
	yearStart := time.Date(config.FiscalYear, time.January, 1, 0, 0, 0, 0, time.UTC)

	for i := 1; i <= config.Rows; i++ {
		institution := config.Institutions[random.Intn(len(config.Institutions))]
		date := yearStart.AddDate(0, 0, random.Intn(365))
		// Amounts between 100.00 and 250,099.99 in cents, two decimals.
		amountMinor := 10000 + random.Int63n(25000000)

		records = append(records, []string{
			fmt.Sprintf("DEMO-%d-%06d", config.FiscalYear, i),
			date.Format("2006-01-02"),
			institution,
			fmt.Sprintf("%d.%02d", amountMinor/100, amountMinor%100),
			"USD",
		})
	}

	return records
}
