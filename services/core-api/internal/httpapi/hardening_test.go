package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type panickingTransactionRepository struct{}

func (panickingTransactionRepository) Save(context.Context, treasury.Transaction) error {
	panic("boom: sensitive internal state")
}

func (panickingTransactionRepository) List(context.Context, treasury.ListTransactionsFilter) ([]treasury.Transaction, error) {
	panic("boom: sensitive internal state")
}

type failingTransactionRepository struct{}

func (failingTransactionRepository) Save(context.Context, treasury.Transaction) error {
	return errors.New("pq: duplicate key value violates unique constraint on table treasury_transactions")
}

func (failingTransactionRepository) List(context.Context, treasury.ListTransactionsFilter) ([]treasury.Transaction, error) {
	return nil, errors.New("pq: connection to server at 10.0.0.7 failed")
}

func validTransactionBody(t *testing.T) *bytes.Reader {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"id":              "txn-hardening-1",
		"institutionId":   "minfin",
		"fiscalYear":      2026,
		"amountMinor":     1000,
		"currency":        "USD",
		"description":     "hardening test",
		"transactionDate": "2026-07-09",
	})
	require.NoError(t, err)

	return bytes.NewReader(payload)
}

func TestRequestIDIsGeneratedWhenAbsent(t *testing.T) {
	router := NewRouter()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	requestID := recorder.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)
	require.GreaterOrEqual(t, len(requestID), 16)
}

func TestRequestIDIsEchoedWhenProvided(t *testing.T) {
	router := NewRouter()

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "caller-supplied-id-42")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, "caller-supplied-id-42", recorder.Header().Get("X-Request-ID"))
}

func TestMalformedInboundRequestIDIsReplaced(t *testing.T) {
	router := NewRouter()

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "bad id\nwith newline and \x00 control")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)
	require.NotContains(t, requestID, "\n")
	require.NotContains(t, requestID, " ")
}

func TestPanicsAreConvertedToClean500(t *testing.T) {
	logs := &bytes.Buffer{}
	router := NewRouter(
		WithLogger(slog.New(slog.NewJSONHandler(logs, nil))),
		WithTransactionRepository(panickingTransactionRepository{}),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/transactions", validTransactionBody(t)))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "sensitive internal state")
	require.Contains(t, logs.String(), "panic")
}

func TestInternalErrorsAreNotLeakedToClients(t *testing.T) {
	logs := &bytes.Buffer{}
	router := NewRouter(
		WithLogger(slog.New(slog.NewJSONHandler(logs, nil))),
		WithTransactionRepository(failingTransactionRepository{}),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/transactions", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "10.0.0.7")
	// The real error must be preserved in the logs for operators.
	require.Contains(t, logs.String(), "10.0.0.7")
}

func TestOversizedRequestBodiesAreRejected(t *testing.T) {
	router := NewRouter(WithMaxBodyBytes(256))

	oversized := strings.NewReader(`{"description":"` + strings.Repeat("x", 512) + `"}`)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/transactions/validate", oversized))

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func TestRequestsAreLoggedWithRequestID(t *testing.T) {
	logs := &bytes.Buffer{}
	router := NewRouter(WithLogger(slog.New(slog.NewJSONHandler(logs, nil))))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "log-assert-id")
	router.ServeHTTP(httptest.NewRecorder(), request)

	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.Split(strings.TrimSpace(logs.String()), "\n")[0]), &entry))
	require.Equal(t, "log-assert-id", entry["request_id"])
	require.Equal(t, "GET", entry["method"])
	require.Equal(t, "/healthz", entry["path"])
	require.Equal(t, float64(http.StatusOK), entry["status"])
	require.Contains(t, entry, "duration_ms")
}

func TestReadyzReportsReadyWithoutChecks(t *testing.T) {
	router := NewRouter()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"status":"ready"}`, recorder.Body.String())
}

func TestReadyzReports503WhenReadinessCheckFails(t *testing.T) {
	router := NewRouter(WithReadinessCheck(func(context.Context) error {
		return errors.New("dial tcp 10.0.0.7:5432: connect: connection refused")
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.JSONEq(t, `{"status":"unavailable"}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "10.0.0.7")
}

func TestReadyzReports200WhenReadinessCheckPasses(t *testing.T) {
	router := NewRouter(WithReadinessCheck(func(context.Context) error { return nil }))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
}
