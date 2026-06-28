package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	if response.Code != http.StatusNoContent {
		t.Fatalf("response.Code = %d, want %d; body = %q", response.Code, http.StatusNoContent, response.Body.String())
	}
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
