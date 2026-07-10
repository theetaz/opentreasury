package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type fakeStagingRepository struct {
	ingested []treasury.StagingRecord
}

func (f *fakeStagingRepository) Ingest(_ context.Context, record treasury.StagingRecord) (treasury.StagingOutcome, error) {
	f.ingested = append(f.ingested, record)
	if record.SourceRef == "PAY-DUP" {
		return treasury.StagingOutcome{SourceRef: record.SourceRef, SourceHash: record.SourceHash, Status: "DUPLICATE", EntryID: "itmis-PAY-DUP"}, nil
	}
	if record.Lines[0].AmountMinor != record.Lines[1].AmountMinor {
		return treasury.StagingOutcome{SourceRef: record.SourceRef, SourceHash: record.SourceHash, Status: "QUARANTINED", Reason: "entry debits and credits are not balanced per currency"}, nil
	}
	return treasury.StagingOutcome{SourceRef: record.SourceRef, SourceHash: record.SourceHash, Status: "POSTED", EntryID: record.SourceSystem + "-" + record.SourceRef}, nil
}

func (f *fakeStagingRepository) ListStagingRecords(_ context.Context, filter treasury.ListStagingRecordsFilter) (treasury.StagingRecordPage, error) {
	return treasury.StagingRecordPage{
		Records: []treasury.StagingRecord{{
			ID: "sr-1", SourceSystem: "itmis", SourceRef: "PAY-002", Status: "QUARANTINED",
			Reason: "institution does not exist", InstitutionID: "ghost", OccurredAt: "2026-07-01",
			Lines: []treasury.JournalLine{}, CreatedAt: "2026-07-09T12:00:00Z",
		}},
		Total: 1,
	}, nil
}

const stagingBatch = `{
	"records": [
		{"sourceSystem":"itmis","sourceRef":"PAY-1","sourceHash":"h1","profile":"itmis@1","institutionId":"minfin","occurredAt":"2026-07-01",
		 "lines":[{"accountCode":"22","direction":"DEBIT","amountMinor":4000,"currency":"USD"},{"accountCode":"6202","direction":"CREDIT","amountMinor":4000,"currency":"USD"}]},
		{"sourceSystem":"itmis","sourceRef":"PAY-2","sourceHash":"h2","profile":"itmis@1","institutionId":"minfin","occurredAt":"2026-07-01",
		 "lines":[{"accountCode":"22","direction":"DEBIT","amountMinor":4000,"currency":"USD"},{"accountCode":"6202","direction":"CREDIT","amountMinor":999,"currency":"USD"}]},
		{"sourceSystem":"itmis","sourceRef":"PAY-DUP","sourceHash":"h3","profile":"itmis@1","institutionId":"minfin","occurredAt":"2026-07-01",
		 "lines":[{"accountCode":"22","direction":"DEBIT","amountMinor":1,"currency":"USD"},{"accountCode":"6202","direction":"CREDIT","amountMinor":1,"currency":"USD"}]},
		{"sourceSystem":"","sourceRef":"","sourceHash":"","profile":"itmis@1","institutionId":"minfin","occurredAt":"2026-07-01","lines":[]}
	]
}`

func TestSubmitStagingBatchReturnsPerRecordOutcomes(t *testing.T) {
	repo := &fakeStagingRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithStagingRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/staging-records", strings.NewReader(stagingBatch)))

	require.Equal(t, http.StatusOK, recorder.Code)

	var body struct {
		Outcomes []struct {
			SourceRef string `json:"sourceRef"`
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			EntryID   string `json:"entryId"`
		} `json:"outcomes"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Outcomes, 4)
	require.Equal(t, "POSTED", body.Outcomes[0].Status)
	require.Equal(t, "itmis-PAY-1", body.Outcomes[0].EntryID)
	require.Equal(t, "QUARANTINED", body.Outcomes[1].Status)
	require.Contains(t, body.Outcomes[1].Reason, "balanced")
	require.Equal(t, "DUPLICATE", body.Outcomes[2].Status)
	require.Equal(t, "QUARANTINED", body.Outcomes[3].Status)
	require.Contains(t, body.Outcomes[3].Reason, "source identity")
	// The malformed record never reached the repository.
	require.Len(t, repo.ingested, 3)
}

func TestSubmitStagingRejectsEmptyBatch(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(WithStagingRepository(&fakeStagingRepository{})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/staging-records", strings.NewReader(`{"records":[]}`)))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestListStagingRecordsFiltersAndReturnsReasons(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(WithStagingRepository(&fakeStagingRepository{})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/staging-records?status=QUARANTINED", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "institution does not exist")
}

func TestListStagingRecordsRejectsUnknownStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(WithStagingRepository(&fakeStagingRepository{})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/staging-records?status=WEIRD", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
