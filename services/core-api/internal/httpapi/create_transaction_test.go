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

func TestCreateTransactionEndpointSavesValidTransaction(t *testing.T) {
	repository := &recordingTransactionRepository{}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{
		"id": "txn-2026-0001",
		"institutionId": "minfin",
		"fiscalYear": 2026,
		"amountMinor": 125000,
		"currency": "USD",
		"description": "Road maintenance payment",
		"transactionDate": "2026-06-28"
	}`))
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, repository.saved, 1)
	require.Equal(t, "txn-2026-0001", repository.saved[0].ID)

	var body map[string]string
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "txn-2026-0001", body["id"])
}

func TestCreateTransactionEndpointRejectsInvalidTransaction(t *testing.T) {
	repository := &recordingTransactionRepository{}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{
		"id": "txn-2026-0001",
		"institutionId": "minfin",
		"fiscalYear": 2026,
		"amountMinor": 0,
		"currency": "USD",
		"description": "Road maintenance payment",
		"transactionDate": "2026-06-28"
	}`))
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Empty(t, repository.saved)

	var body map[string]string
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "invalid amount", body["error"])
}

type recordingTransactionRepository struct {
	saved []treasury.Transaction
}

func (repository *recordingTransactionRepository) Save(ctx context.Context, tx treasury.Transaction) error {
	repository.saved = append(repository.saved, tx)
	return nil
}
