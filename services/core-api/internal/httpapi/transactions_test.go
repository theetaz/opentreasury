package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionValidationEndpointAcceptsValidTransaction(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions/validate", strings.NewReader(`{
		"id": "txn-2026-0001",
		"institutionId": "minfin",
		"fiscalYear": 2026,
		"amountMinor": 125000,
		"currency": "USD",
		"description": "Road maintenance payment",
		"transactionDate": "2026-06-28"
	}`))
	response := httptest.NewRecorder()

	NewRouter().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)

	var body struct {
		Status          string `json:"status"`
		Code            string `json:"code"`
		Message         string `json:"message"`
		TransactionID   string `json:"transactionId"`
		InstitutionID   string `json:"institutionId"`
		FiscalYear      int    `json:"fiscalYear"`
		AmountMinor     int64  `json:"amountMinor"`
		Currency        string `json:"currency"`
		TransactionDate string `json:"transactionDate"`
	}
	err := json.NewDecoder(response.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "VALID", body.Status)
	require.Equal(t, "VALIDATION_SUCCESS", body.Code)
	require.Equal(t, "Transaction is valid.", body.Message)
	require.Equal(t, "txn-2026-0001", body.TransactionID)
	require.Equal(t, "minfin", body.InstitutionID)
	require.Equal(t, 2026, body.FiscalYear)
	require.Equal(t, int64(125000), body.AmountMinor)
	require.Equal(t, "USD", body.Currency)
	require.Equal(t, "2026-06-28", body.TransactionDate)
}

func TestTransactionValidationEndpointRejectsInvalidTransaction(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions/validate", strings.NewReader(`{
		"id": "txn-2026-0001",
		"institutionId": "minfin",
		"fiscalYear": 2026,
		"amountMinor": 0,
		"currency": "USD",
		"description": "Road maintenance payment",
		"transactionDate": "2026-06-28"
	}`))
	response := httptest.NewRecorder()

	NewRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode(response.Body) error = %v", err)
	}

	if body["error"] != "invalid amount" {
		t.Fatalf("body[error] = %q, want %q", body["error"], "invalid amount")
	}
}
