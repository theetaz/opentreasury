package connector

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testProfile() Profile {
	profile := Profile{SourceSystem: "itmis", Version: 1, DebitAccount: "22", CreditAccount: "6202"}
	profile.Columns.SourceRef = "reference"
	profile.Columns.OccurredAt = "payment_date"
	profile.Columns.Institution = "institution_code"
	profile.Columns.Amount = "amount"
	profile.Columns.Currency = "currency"
	return profile
}

const sampleCSV = `reference,payment_date,institution_code,amount,currency
PAY-001,2026-07-01,minfin,1250.50,USD
PAY-002,2026-07-02,health,40.00,USD
PAY-003,2026-07-02,minfin,not-a-number,USD
,2026-07-03,minfin,10.00,USD
`

func TestMapCSVProducesBalancedRecordsAndSkipsBadRows(t *testing.T) {
	result, err := MapCSV(testProfile(), strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("mapping: %v", err)
	}

	if len(result.Records) != 2 {
		t.Fatalf("want 2 records, got %d", len(result.Records))
	}
	if len(result.Skipped) != 2 {
		t.Fatalf("want 2 skipped rows, got %d: %+v", len(result.Skipped), result.Skipped)
	}

	first := result.Records[0]
	if first.SourceRef != "PAY-001" || first.InstitutionID != "minfin" || first.Profile != "itmis@1" {
		t.Errorf("unexpected record: %+v", first)
	}
	if first.Lines[0].AmountMinor != 125050 || first.Lines[0].Direction != "DEBIT" || first.Lines[0].AccountCode != "22" {
		t.Errorf("unexpected debit line: %+v", first.Lines[0])
	}
	if first.Lines[1].AmountMinor != 125050 || first.Lines[1].Direction != "CREDIT" || first.Lines[1].AccountCode != "6202" {
		t.Errorf("unexpected credit line: %+v", first.Lines[1])
	}
	if first.SourceHash == "" {
		t.Error("source hash missing")
	}
}

func TestMappingIsDeterministic(t *testing.T) {
	first, err := MapCSV(testProfile(), strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatal(err)
	}
	second, err := MapCSV(testProfile(), strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatal(err)
	}

	for i := range first.Records {
		if first.Records[i].SourceHash != second.Records[i].SourceHash {
			t.Errorf("hash for %s not deterministic", first.Records[i].SourceRef)
		}
	}
}

func TestMapCSVRejectsFileMissingColumns(t *testing.T) {
	_, err := MapCSV(testProfile(), strings.NewReader("a,b,c\n1,2,3\n"))
	if err == nil {
		t.Fatal("expected error for missing columns")
	}
}

func TestWatcherDeliversAndMovesFile(t *testing.T) {
	var received []Record
	staging := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Records []Record `json:"records"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		received = append(received, payload.Records...)

		outcomes := make([]Outcome, 0, len(payload.Records))
		for _, record := range payload.Records {
			outcomes = append(outcomes, Outcome{SourceRef: record.SourceRef, Status: "POSTED"})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"outcomes": outcomes})
	}))
	defer staging.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "drop.csv"), []byte(sampleCSV), 0o600); err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher(dir, testProfile(), NewDeliverer(staging.URL, "", "", ""), slog.New(slog.DiscardHandler))
	for _, sub := range []string{"processed", "failed"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	watcher.processFile(context.Background(), filepath.Join(dir, "drop.csv"))

	if len(received) != 2 {
		t.Fatalf("staging received %d records, want 2", len(received))
	}
	processed, _ := os.ReadDir(filepath.Join(dir, "processed"))
	if len(processed) != 1 {
		t.Errorf("file not moved to processed/")
	}
	if _, err := os.Stat(filepath.Join(dir, "drop.csv")); !os.IsNotExist(err) {
		t.Errorf("original file still present")
	}
}

func TestWatcherLeavesFileForRetryWhenDeliveryFails(t *testing.T) {
	staging := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	}))
	defer staging.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "drop.csv")
	if err := os.WriteFile(path, []byte(sampleCSV), 0o600); err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher(dir, testProfile(), NewDeliverer(staging.URL, "", "", ""), slog.New(slog.DiscardHandler))
	watcher.processFile(context.Background(), path)

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file must stay in place for retry, got %v", err)
	}
}
