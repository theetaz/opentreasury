package connector

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Line mirrors the interchange journal line.
type Line struct {
	AccountCode string `json:"accountCode"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}

// Record is one interchange staging record.
type Record struct {
	SourceSystem  string `json:"sourceSystem"`
	SourceRef     string `json:"sourceRef"`
	SourceHash    string `json:"sourceHash"`
	Profile       string `json:"profile"`
	InstitutionID string `json:"institutionId"`
	OccurredAt    string `json:"occurredAt"`
	Lines         []Line `json:"lines"`
}

// MapResult separates mappable rows from rows the connector itself must
// quarantine (missing columns, unparseable amounts). Mapping is deterministic:
// the same input always produces the same records and hashes.
type MapResult struct {
	Records []Record
	Skipped []SkippedRow
}

type SkippedRow struct {
	RowNumber int
	Reason    string
}

// MapCSV converts a CSV export into interchange records using the profile.
func MapCSV(profile Profile, reader io.Reader) (MapResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		return MapResult{}, fmt.Errorf("reading header: %w", err)
	}

	index := map[string]int{}
	for i, name := range header {
		index[strings.TrimSpace(name)] = i
	}
	for _, required := range []string{
		profile.Columns.SourceRef, profile.Columns.OccurredAt,
		profile.Columns.Institution, profile.Columns.Amount, profile.Columns.Currency,
	} {
		if _, ok := index[required]; !ok {
			return MapResult{}, fmt.Errorf("column %q not present in file", required)
		}
	}

	result := MapResult{}
	rowNumber := 1
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		rowNumber++
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRow{RowNumber: rowNumber, Reason: err.Error()})
			continue
		}

		field := func(column string) string { return strings.TrimSpace(row[index[column]]) }

		amountMinor, err := parseMajorAmount(field(profile.Columns.Amount))
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRow{RowNumber: rowNumber, Reason: err.Error()})
			continue
		}

		sourceRef := field(profile.Columns.SourceRef)
		if sourceRef == "" {
			result.Skipped = append(result.Skipped, SkippedRow{RowNumber: rowNumber, Reason: "empty source reference"})
			continue
		}

		currency := strings.ToUpper(field(profile.Columns.Currency))
		record := Record{
			SourceSystem:  profile.SourceSystem,
			SourceRef:     sourceRef,
			Profile:       profile.Name(),
			InstitutionID: field(profile.Columns.Institution),
			OccurredAt:    field(profile.Columns.OccurredAt),
			Lines: []Line{
				{AccountCode: profile.DebitAccount, Direction: "DEBIT", AmountMinor: amountMinor, Currency: currency},
				{AccountCode: profile.CreditAccount, Direction: "CREDIT", AmountMinor: amountMinor, Currency: currency},
			},
		}
		record.SourceHash = hashRecord(record)
		result.Records = append(result.Records, record)
	}

	return result, nil
}

// hashRecord derives the end-to-end idempotency key from the record's source
// identity and mapped content.
func hashRecord(record Record) string {
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "%s|%s|%s|%s|%s|%s",
		record.SourceSystem, record.SourceRef, record.Profile,
		record.InstitutionID, record.OccurredAt, canonicalLines(record.Lines))
	return hex.EncodeToString(hasher.Sum(nil))
}

func canonicalLines(lines []Line) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%s:%s:%d:%s", line.AccountCode, line.Direction, line.AmountMinor, line.Currency))
	}
	return strings.Join(parts, ";")
}

// parseMajorAmount converts a decimal major-unit amount ("1250.50") into
// minor units without floating-point drift.
func parseMajorAmount(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty amount")
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("unparseable amount %q", raw)
	}
	if value <= 0 {
		return 0, fmt.Errorf("amount must be positive, got %q", raw)
	}

	minor := math.Round(value * 100)
	if minor > math.MaxInt64/2 {
		return 0, fmt.Errorf("amount out of range: %q", raw)
	}

	return int64(minor), nil
}
