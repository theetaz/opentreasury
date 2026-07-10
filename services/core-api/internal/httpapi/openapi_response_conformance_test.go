package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

// Every exercised (request, response) pair is validated against the OpenAPI
// document, so schema drift between contract and implementation fails CI
// instead of surprising API consumers.

type conformanceRepository struct{ saveErr error }

func (repository conformanceRepository) Save(context.Context, treasury.Transaction) error {
	return repository.saveErr
}

func (conformanceRepository) List(context.Context, treasury.ListTransactionsFilter) (treasury.TransactionPage, error) {
	return treasury.TransactionPage{Total: 1, Transactions: []treasury.Transaction{{
		ID:              "txn-2026-0001",
		InstitutionID:   "minfin",
		FiscalYear:      2026,
		AmountMinor:     125000,
		Currency:        "USD",
		Description:     "Road maintenance payment",
		TransactionDate: "2026-06-28",
		Status:          "POSTED",
	}}}, nil
}

func (conformanceRepository) ListAuditEvents(context.Context, treasury.ListAuditEventsFilter) (treasury.AuditEventPage, error) {
	return treasury.AuditEventPage{Total: 1, Events: []treasury.AuditEvent{{
		ID:            "audit-txn-2026-0001-created",
		EventType:     "TRANSACTION_CREATED",
		TransactionID: "txn-2026-0001",
		InstitutionID: "minfin",
		OccurredAt:    "2026-06-28T10:24:28Z",
		Summary:       "Transaction txn-2026-0001 was created.",
	}}}, nil
}

func (conformanceRepository) PostEntry(context.Context, treasury.JournalEntry) error { return nil }

func (conformanceRepository) ListEntries(context.Context, treasury.ListJournalEntriesFilter) (treasury.JournalEntryPage, error) {
	return treasury.JournalEntryPage{Total: 1, Entries: []treasury.JournalEntry{{
		ID: "je-1", InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-06-28",
		Description: "Tax receipt", Status: "POSTED", EntryType: "STANDARD",
		Lines: []treasury.JournalLine{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 125000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 125000, Currency: "USD"},
		},
	}}}, nil
}

func (conformanceRepository) ListBalances(context.Context, treasury.ListBalancesFilter) (treasury.BalancePage, error) {
	return treasury.BalancePage{Total: 1, Balances: []treasury.Balance{
		{InstitutionID: "minfin", AccountCode: "6202", AccountName: "Currency and deposits", AccountType: "ASSET", Currency: "USD", BalanceMinor: 125000},
	}}, nil
}

func (conformanceRepository) ListAccounts(context.Context, treasury.ListAccountsFilter) (treasury.AccountPage, error) {
	return treasury.AccountPage{Total: 1, Accounts: []treasury.Account{{
		Code:        "1",
		Name:        "Revenue",
		AccountType: "REVENUE",
		GfsmCode:    "1",
		Active:      true,
	}}}, nil
}

func (conformanceRepository) ListInstitutions(context.Context, treasury.ListInstitutionsFilter) (treasury.InstitutionPage, error) {
	return treasury.InstitutionPage{Total: 1, Institutions: []treasury.Institution{{
		ID:          "minfin",
		Name:        "Ministry of Finance",
		Type:        "MINISTRY",
		CountryCode: "KE",
		Status:      "ACTIVE",
	}}}, nil
}

const validTransactionJSON = `{
	"id": "txn-2026-0001",
	"institutionId": "minfin",
	"fiscalYear": 2026,
	"amountMinor": 125000,
	"currency": "USD",
	"description": "Road maintenance payment",
	"transactionDate": "2026-06-28"
}`

func TestResponsesConformToOpenAPIContract(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "..", "api", "core-api", "openapi.yaml")
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(contractPath)
	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))

	specRouter, err := gorillamux.NewRouter(document)
	require.NoError(t, err)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		repository conformanceRepository
		wantStatus int
	}{
		{name: "health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{name: "readiness", method: http.MethodGet, path: "/readyz", wantStatus: http.StatusOK},
		{name: "list transactions", method: http.MethodGet, path: "/v1/transactions", wantStatus: http.StatusOK},
		{name: "list transactions bad limit", method: http.MethodGet, path: "/v1/transactions?limit=101", wantStatus: http.StatusBadRequest},
		{name: "list audit events", method: http.MethodGet, path: "/v1/audit-events", wantStatus: http.StatusOK},
		{name: "list institutions", method: http.MethodGet, path: "/v1/institutions", wantStatus: http.StatusOK},
		{name: "list accounts", method: http.MethodGet, path: "/v1/accounts?accountType=REVENUE", wantStatus: http.StatusOK},
		{name: "list accounts bad type", method: http.MethodGet, path: "/v1/accounts?accountType=CRYPTO", wantStatus: http.StatusBadRequest},
		{name: "list journal entries", method: http.MethodGet, path: "/v1/journal-entries", wantStatus: http.StatusOK},
		{name: "list balances", method: http.MethodGet, path: "/v1/balances?institutionId=minfin", wantStatus: http.StatusOK},
		{name: "validate ok", method: http.MethodPost, path: "/v1/transactions/validate", body: validTransactionJSON, wantStatus: http.StatusOK},
		{name: "validate rejects", method: http.MethodPost, path: "/v1/transactions/validate", body: `{"id":""}`, wantStatus: http.StatusBadRequest},
		{name: "create ok", method: http.MethodPost, path: "/v1/transactions", body: validTransactionJSON, wantStatus: http.StatusCreated},
		{
			name: "create duplicate", method: http.MethodPost, path: "/v1/transactions", body: validTransactionJSON,
			repository: conformanceRepository{saveErr: treasury.ErrDuplicateTransaction}, wantStatus: http.StatusConflict,
		},
		{
			name: "create unknown institution", method: http.MethodPost, path: "/v1/transactions", body: validTransactionJSON,
			repository: conformanceRepository{saveErr: treasury.ErrUnknownInstitution}, wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			apiRouter := NewRouter(
				WithTransactionRepository(testCase.repository),
				WithInstitutionRepository(testCase.repository),
				WithAccountRepository(testCase.repository),
				WithJournalRepository(testCase.repository),
			)

			var body io.Reader
			if testCase.body != "" {
				body = strings.NewReader(testCase.body)
			}
			request := httptest.NewRequest(testCase.method, "http://localhost:8080"+testCase.path, body)
			if testCase.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}

			recorder := httptest.NewRecorder()
			apiRouter.ServeHTTP(recorder, request)
			require.Equal(t, testCase.wantStatus, recorder.Code)

			route, pathParams, err := specRouter.FindRoute(request)
			require.NoError(t, err, "request %s %s is not documented in the contract", testCase.method, testCase.path)

			requestValidationInput := &openapi3filter.RequestValidationInput{
				Request:    request,
				PathParams: pathParams,
				Route:      route,
			}
			responseValidationInput := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: requestValidationInput,
				Status:                 recorder.Code,
				Header:                 recorder.Header(),
			}
			responseValidationInput.SetBodyBytes(recorder.Body.Bytes())

			err = openapi3filter.ValidateResponse(context.Background(), responseValidationInput)
			require.NoError(t, err, "response for %s %s (status %d) does not match the contract", testCase.method, testCase.path, recorder.Code)
		})
	}
}
