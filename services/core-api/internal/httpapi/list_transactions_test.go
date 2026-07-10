package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
	"github.com/stretchr/testify/require"
)

func TestListTransactionsEndpointReturnsTransactions(t *testing.T) {
	repository := &recordingTransactionRepository{
		transactions: []treasury.Transaction{
			{
				ID:              "txn-2026-0002",
				InstitutionID:   "minfin",
				FiscalYear:      2026,
				AmountMinor:     450075,
				Currency:        "USD",
				Description:     "Quarterly grant release",
				TransactionDate: "2026-06-29",
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions?institutionId=minfin&fiscalYear=2026&limit=25", nil)
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, treasury.ListTransactionsFilter{
		InstitutionID: "minfin",
		FiscalYear:    2026,
		Pagination:    treasury.Pagination{Page: 1, PageSize: 25},
	}, repository.filter)

	var body struct {
		Transactions []struct {
			ID              string `json:"id"`
			InstitutionID   string `json:"institutionId"`
			FiscalYear      int    `json:"fiscalYear"`
			AmountMinor     int64  `json:"amountMinor"`
			Currency        string `json:"currency"`
			Description     string `json:"description"`
			TransactionDate string `json:"transactionDate"`
		} `json:"transactions"`
	}
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Len(t, body.Transactions, 1)
	require.Equal(t, "txn-2026-0002", body.Transactions[0].ID)
	require.Equal(t, "minfin", body.Transactions[0].InstitutionID)
}

func TestListTransactionsEndpointUsesDefaultPageSize(t *testing.T) {
	repository := &recordingTransactionRepository{}
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions", nil)
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, treasury.ListTransactionsFilter{
		Pagination: treasury.Pagination{Page: 1, PageSize: 15},
	}, repository.filter)
}

func TestListTransactionsEndpointRejectsInvalidFiscalYear(t *testing.T) {
	repository := &recordingTransactionRepository{}
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions?fiscalYear=twenty-six", nil)
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestListTransactionsEndpointRequiresRepository(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions", nil)
	response := httptest.NewRecorder()

	NewRouter().ServeHTTP(response, request)

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestListTransactionsEndpointFiltersByStatus(t *testing.T) {
	repository := &recordingTransactionRepository{
		transactions: []treasury.Transaction{
			{
				ID: "txn-2026-0009", InstitutionID: "minfin", FiscalYear: 2026,
				AmountMinor: 1000, Currency: "USD", Status: "POSTED",
				TransactionDate: "2026-06-29",
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions?status=POSTED", nil)
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "POSTED", repository.filter.Status)

	var body struct {
		Transactions []struct {
			Status string `json:"status"`
		} `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.Transactions, 1)
	require.Equal(t, "POSTED", body.Transactions[0].Status)
}

func TestListTransactionsEndpointRejectsInvalidStatus(t *testing.T) {
	repository := &recordingTransactionRepository{}
	request := httptest.NewRequest(http.MethodGet, "/v1/transactions?status=bogus", nil)
	response := httptest.NewRecorder()

	NewRouter(WithTransactionRepository(repository)).ServeHTTP(response, request)

	// Strict validation: reject, never clamp or ignore.
	require.Equal(t, http.StatusBadRequest, response.Code)
}
