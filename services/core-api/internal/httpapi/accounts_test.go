package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type recordingAccountRepository struct {
	filter treasury.ListAccountsFilter
}

func (repository *recordingAccountRepository) ListAccounts(_ context.Context, filter treasury.ListAccountsFilter) (treasury.AccountPage, error) {
	repository.filter = filter
	return treasury.AccountPage{
		Accounts: []treasury.Account{
			{Code: "1", Name: "Revenue", AccountType: "REVENUE", GfsmCode: "1", Active: true},
			{Code: "11", Name: "Taxes", AccountType: "REVENUE", ParentCode: "1", GfsmCode: "11", Active: true},
		},
		Total: 2,
	}, nil
}

func TestListAccountsReturnsAccountsWithPagination(t *testing.T) {
	repository := &recordingAccountRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithAccountRepository(repository)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/accounts?accountType=REVENUE&pageSize=50", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "REVENUE", repository.filter.AccountType)
	require.Equal(t, 50, repository.filter.PageSize)

	var body struct {
		Accounts []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			AccountType string `json:"accountType"`
			ParentCode  string `json:"parentCode"`
		} `json:"accounts"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Accounts, 2)
	require.Equal(t, "1", body.Accounts[0].Code)
	require.Equal(t, "1", body.Accounts[1].ParentCode)
	require.Equal(t, 2, body.Pagination.Total)
}

func TestListAccountsRejectsUnknownAccountType(t *testing.T) {
	repository := &recordingAccountRepository{}
	recorder := httptest.NewRecorder()
	NewRouter(WithAccountRepository(repository)).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/accounts?accountType=CRYPTO", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"error":"invalid account type"}`, recorder.Body.String())
}

func TestListAccountsWithoutRepositoryReturns503(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/accounts", nil))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
