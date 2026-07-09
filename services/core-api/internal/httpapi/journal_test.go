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

type recordingJournalRepository struct {
	posted      *treasury.JournalEntry
	entryFilter treasury.ListJournalEntriesFilter
	balFilter   treasury.ListBalancesFilter
	postErr     error
}

func (r *recordingJournalRepository) PostEntry(_ context.Context, entry treasury.JournalEntry) error {
	r.posted = &entry
	return r.postErr
}

func (r *recordingJournalRepository) ListEntries(_ context.Context, filter treasury.ListJournalEntriesFilter) (treasury.JournalEntryPage, error) {
	r.entryFilter = filter
	return treasury.JournalEntryPage{
		Entries: []treasury.JournalEntry{{
			ID: "je-1", InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-06-28",
			Description: "Tax receipt", Status: "POSTED", EntryType: "STANDARD",
			Lines: []treasury.JournalLine{
				{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 125000, Currency: "USD"},
				{AccountCode: "114", Direction: "CREDIT", AmountMinor: 125000, Currency: "USD"},
			},
		}},
		Total: 1,
	}, nil
}

func (r *recordingJournalRepository) ListBalances(_ context.Context, filter treasury.ListBalancesFilter) (treasury.BalancePage, error) {
	r.balFilter = filter
	return treasury.BalancePage{
		Balances: []treasury.Balance{
			{InstitutionID: "minfin", AccountCode: "6202", AccountName: "Currency and deposits", AccountType: "ASSET", Currency: "USD", BalanceMinor: 125000},
		},
		Total: 1,
	}, nil
}

const validEntryJSON = `{
	"id": "je-2026-0001",
	"institutionId": "minfin",
	"fiscalYear": 2026,
	"effectiveDate": "2026-06-28",
	"description": "Tax receipt into the treasury account",
	"lines": [
		{"accountCode": "6202", "direction": "DEBIT", "amountMinor": 125000, "currency": "USD"},
		{"accountCode": "114", "direction": "CREDIT", "amountMinor": 125000, "currency": "USD"}
	]
}`

func TestPostJournalEntryAcceptsBalancedEntry(t *testing.T) {
	repo := &recordingJournalRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/journal-entries", strings.NewReader(validEntryJSON)))

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.NotNil(t, repo.posted)
	require.Len(t, repo.posted.Lines, 2)
	require.JSONEq(t, `{"id":"je-2026-0001"}`, recorder.Body.String())
}

func TestPostJournalEntryRejectsUnbalancedEntry(t *testing.T) {
	repo := &recordingJournalRepository{}
	body := strings.Replace(validEntryJSON, `"amountMinor": 125000, "currency": "USD"}
	]`, `"amountMinor": 100000, "currency": "USD"}
	]`, 1)
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/journal-entries", strings.NewReader(body)))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "balanced")
	require.Nil(t, repo.posted)
}

func TestPostJournalEntryDuplicateReturns409(t *testing.T) {
	repo := &recordingJournalRepository{postErr: treasury.ErrDuplicateTransaction}
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/journal-entries", strings.NewReader(validEntryJSON)))

	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestListJournalEntriesReturnsEntriesWithLinesAndPagination(t *testing.T) {
	repo := &recordingJournalRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/journal-entries?institutionId=minfin&status=POSTED", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "minfin", repo.entryFilter.InstitutionID)
	require.Equal(t, "POSTED", repo.entryFilter.Status)

	var body struct {
		Entries []struct {
			ID    string `json:"id"`
			Lines []struct {
				AccountCode string `json:"accountCode"`
				Direction   string `json:"direction"`
			} `json:"lines"`
		} `json:"entries"`
		Pagination struct{ Total int } `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Entries, 1)
	require.Len(t, body.Entries[0].Lines, 2)
	require.Equal(t, 1, body.Pagination.Total)
}

func TestListJournalEntriesRejectsInvalidStatus(t *testing.T) {
	repo := &recordingJournalRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/journal-entries?status=NONSENSE", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestListBalancesReturnsMaterializedStocks(t *testing.T) {
	repo := &recordingJournalRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repo)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/balances?institutionId=minfin", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "minfin", repo.balFilter.InstitutionID)

	var body struct {
		Balances []struct {
			AccountCode  string `json:"accountCode"`
			BalanceMinor int64  `json:"balanceMinor"`
		} `json:"balances"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Balances, 1)
	require.Equal(t, int64(125000), body.Balances[0].BalanceMinor)
}

func TestLegacyTransactionEndpointStillWorksAlongsideJournal(t *testing.T) {
	journal := &recordingJournalRepository{}
	tx := &recordingTransactionRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithTransactionRepository(tx), WithJournalRepository(journal)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{
			"id": "txn-facade-1",
			"institutionId": "minfin",
			"fiscalYear": 2026,
			"amountMinor": 125000,
			"currency": "USD",
			"description": "Road maintenance payment",
			"transactionDate": "2026-06-28"
		}`)))

	require.Equal(t, http.StatusCreated, recorder.Code)
	// The legacy single-sided transaction endpoint remains backward compatible;
	// double-entry accounting flows through /v1/journal-entries instead.
	require.Len(t, tx.saved, 1)
}
