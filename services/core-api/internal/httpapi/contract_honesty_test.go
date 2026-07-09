package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type duplicateRejectingRepository struct{}

func (duplicateRejectingRepository) Save(context.Context, treasury.Transaction) error {
	return fmt.Errorf("saving transaction: %w", treasury.ErrDuplicateTransaction)
}

func (duplicateRejectingRepository) List(context.Context, treasury.ListTransactionsFilter) (treasury.TransactionPage, error) {
	return treasury.TransactionPage{}, nil
}

type unknownInstitutionRepository struct{}

func (unknownInstitutionRepository) Save(context.Context, treasury.Transaction) error {
	return fmt.Errorf("saving transaction: %w", treasury.ErrUnknownInstitution)
}

func (unknownInstitutionRepository) List(context.Context, treasury.ListTransactionsFilter) (treasury.TransactionPage, error) {
	return treasury.TransactionPage{}, nil
}

func TestCreateTransactionReturns409ForDuplicateID(t *testing.T) {
	router := NewRouter(WithTransactionRepository(duplicateRejectingRepository{}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/transactions", validTransactionBody(t)))

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.JSONEq(t, `{"error":"transaction already exists"}`, recorder.Body.String())
}

func TestCreateTransactionReturns422ForUnknownInstitution(t *testing.T) {
	router := NewRouter(WithTransactionRepository(unknownInstitutionRepository{}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/transactions", validTransactionBody(t)))

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.JSONEq(t, `{"error":"institution does not exist"}`, recorder.Body.String())
}

func TestListEndpointsRejectLimitAboveMaximum(t *testing.T) {
	router := NewRouter(
		WithTransactionRepository(duplicateRejectingRepository{}),
		WithInstitutionRepository(stubInstitutionRepository{}),
		WithAuditEventRepository(stubAuditEventRepository{}),
	)

	for _, path := range []string{
		"/v1/transactions?limit=101",
		"/v1/audit-events?limit=101",
		"/v1/institutions?limit=101",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "path %s", path)
		require.JSONEqf(t, `{"error":"invalid limit"}`, recorder.Body.String(), "path %s", path)
	}
}

func TestListEndpointsRejectMalformedLimitWithInvalidLimitError(t *testing.T) {
	router := NewRouter(WithTransactionRepository(duplicateRejectingRepository{}))

	for _, limit := range []string{"abc", "0", "-5"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/transactions?limit="+limit, nil))

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "limit %q", limit)
		require.JSONEqf(t, `{"error":"invalid limit"}`, recorder.Body.String(), "limit %q", limit)
	}
}

type stubInstitutionRepository struct{}

func (stubInstitutionRepository) ListInstitutions(context.Context, treasury.ListInstitutionsFilter) (treasury.InstitutionPage, error) {
	return treasury.InstitutionPage{}, nil
}

type stubAuditEventRepository struct{}

func (stubAuditEventRepository) ListAuditEvents(context.Context, treasury.ListAuditEventsFilter) (treasury.AuditEventPage, error) {
	return treasury.AuditEventPage{}, nil
}
