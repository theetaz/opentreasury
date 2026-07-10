package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
	"github.com/stretchr/testify/require"
)

type recordingCommitmentRepository struct {
	created   []treasury.Commitment
	createErr error
	filter    treasury.ListCommitmentsFilter
	page      treasury.CommitmentPage
}

func (r *recordingCommitmentRepository) CreateCommitment(_ context.Context, commitment treasury.Commitment) error {
	r.created = append(r.created, commitment)
	return r.createErr
}

func (r *recordingCommitmentRepository) ListCommitments(_ context.Context, filter treasury.ListCommitmentsFilter) (treasury.CommitmentPage, error) {
	r.filter = filter
	return r.page, nil
}

const validCommitmentJSON = `{
	"id": "com-2026-0001",
	"institutionId": "minfin",
	"fiscalYear": 2026,
	"accountCode": "22",
	"description": "Road maintenance framework contract",
	"amountMinor": 500000,
	"currency": "USD",
	"committedDate": "2026-07-01"
}`

func TestCreateCommitmentPersistsAndReturns201(t *testing.T) {
	repository := &recordingCommitmentRepository{}
	request := httptest.NewRequest(http.MethodPost, "/v1/commitments", strings.NewReader(validCommitmentJSON))
	response := httptest.NewRecorder()

	NewRouter(WithCommitmentRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, repository.created, 1)
	require.Equal(t, "com-2026-0001", repository.created[0].ID)
	require.Equal(t, int64(500000), repository.created[0].AmountMinor)
}

func TestCreateCommitmentMapsDomainErrors(t *testing.T) {
	cases := map[error]int{
		treasury.ErrDuplicateCommitment: http.StatusConflict,
		treasury.ErrUnknownInstitution:  http.StatusUnprocessableEntity,
	}
	for domainErr, wantStatus := range cases {
		repository := &recordingCommitmentRepository{createErr: domainErr}
		response := httptest.NewRecorder()
		NewRouter(WithCommitmentRepository(repository)).ServeHTTP(response,
			httptest.NewRequest(http.MethodPost, "/v1/commitments", strings.NewReader(validCommitmentJSON)))
		require.Equal(t, wantStatus, response.Code)
	}
}

func TestCreateCommitmentRejectsInvalidPayload(t *testing.T) {
	response := httptest.NewRecorder()
	NewRouter(WithCommitmentRepository(&recordingCommitmentRepository{})).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/v1/commitments", strings.NewReader(`{"id":""}`)))
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestListCommitmentsFiltersAndComputesRemaining(t *testing.T) {
	repository := &recordingCommitmentRepository{
		page: treasury.CommitmentPage{
			Total: 1,
			Commitments: []treasury.Commitment{{
				ID: "com-2026-0001", InstitutionID: "minfin", FiscalYear: 2026,
				AccountCode: "22", AmountMinor: 500000, Currency: "USD",
				CommittedDate: "2026-07-01", Status: "OPEN", SettledAmountMinor: 200000,
			}},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/commitments?institutionId=minfin&fiscalYear=2026&status=OPEN", nil)
	response := httptest.NewRecorder()

	NewRouter(WithCommitmentRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "OPEN", repository.filter.Status)
	require.Equal(t, "minfin", repository.filter.InstitutionID)

	var body struct {
		Commitments []struct {
			ID                   string `json:"id"`
			SettledAmountMinor   int64  `json:"settledAmountMinor"`
			RemainingAmountMinor int64  `json:"remainingAmountMinor"`
			Status               string `json:"status"`
		} `json:"commitments"`
		Pagination paginationResponse `json:"pagination"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.Commitments, 1)
	require.Equal(t, int64(300000), body.Commitments[0].RemainingAmountMinor)
}

func TestListCommitmentsRejectsInvalidStatus(t *testing.T) {
	response := httptest.NewRecorder()
	NewRouter(WithCommitmentRepository(&recordingCommitmentRepository{})).ServeHTTP(response,
		httptest.NewRequest(http.MethodGet, "/v1/commitments?status=bogus", nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestPostJournalEntryPassesCommitmentThrough(t *testing.T) {
	repository := &recordingJournalRepository{}
	body := `{
		"id": "je-1", "institutionId": "minfin", "fiscalYear": 2026,
		"effectiveDate": "2026-07-02", "description": "settles contract",
		"commitmentId": "com-2026-0001",
		"lines": [
			{"accountCode": "22", "direction": "DEBIT", "amountMinor": 1000, "currency": "USD"},
			{"accountCode": "6202", "direction": "CREDIT", "amountMinor": 1000, "currency": "USD"}
		]
	}`
	response := httptest.NewRecorder()
	NewRouter(WithJournalRepository(repository)).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/v1/journal-entries", strings.NewReader(body)))

	require.Equal(t, http.StatusCreated, response.Code)
	require.Equal(t, "com-2026-0001", repository.posted.CommitmentID)
}

func TestPostJournalEntryMapsCommitmentErrors(t *testing.T) {
	cases := map[error]int{
		treasury.ErrUnknownCommitment:  http.StatusUnprocessableEntity,
		treasury.ErrCommitmentClosed:   http.StatusConflict,
		treasury.ErrCommitmentExceeded: http.StatusUnprocessableEntity,
		treasury.ErrCommitmentMismatch: http.StatusUnprocessableEntity,
	}
	body := `{
		"id": "je-1", "institutionId": "minfin", "fiscalYear": 2026,
		"effectiveDate": "2026-07-02", "description": "settles contract",
		"commitmentId": "com-2026-0001",
		"lines": [
			{"accountCode": "22", "direction": "DEBIT", "amountMinor": 1000, "currency": "USD"},
			{"accountCode": "6202", "direction": "CREDIT", "amountMinor": 1000, "currency": "USD"}
		]
	}`
	for domainErr, wantStatus := range cases {
		repository := &recordingJournalRepository{postErr: domainErr}
		response := httptest.NewRecorder()
		NewRouter(WithJournalRepository(repository)).ServeHTTP(response,
			httptest.NewRequest(http.MethodPost, "/v1/journal-entries", strings.NewReader(body)))
		require.Equal(t, wantStatus, response.Code, "error %v", domainErr)
	}
}
