package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
)

func publicRouter(t *testing.T, options ...RouterOption) http.Handler {
	t.Helper()

	publication, err := authz.NewPublication(context.Background())
	require.NoError(t, err)

	base := []RouterOption{
		WithPublication(publication),
		WithTransactionRepository(&conformanceRepository{}),
		WithInstitutionRepository(&conformanceRepository{}),
		WithAccountRepository(&conformanceRepository{}),
		WithJournalRepository(&conformanceRepository{}),
		// Auth enabled: proves the public tier bypasses it by design.
		WithTokenVerifier(stubVerifier{ok: false}),
	}

	return NewRouter(append(base, options...)...)
}

func TestPublicTierServesAnonymouslyWhileOperatorTierRequiresAuth(t *testing.T) {
	router := publicRouter(t)

	public := httptest.NewRecorder()
	router.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/public/v1/institutions", nil))
	require.Equal(t, http.StatusOK, public.Code)
	require.Equal(t, "public, max-age=30", public.Header().Get("Cache-Control"))

	operator := httptest.NewRecorder()
	router.ServeHTTP(operator, httptest.NewRequest(http.MethodGet, "/v1/institutions", nil))
	require.Equal(t, http.StatusUnauthorized, operator.Code)
}

func TestPublicJournalEntriesAreRedacted(t *testing.T) {
	router := publicRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/v1/journal-entries", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	// The conformance fixture's description must not appear publicly.
	require.NotContains(t, body, "Tax receipt")
	require.NotContains(t, body, "idempotencyKey")

	var payload struct {
		Entries []map[string]any `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.Entries)
	entry := payload.Entries[0]
	require.NotContains(t, entry, "description")
	require.Contains(t, entry, "lines")
	lines := entry["lines"].([]any)
	require.NotEmpty(t, lines)
	line := lines[0].(map[string]any)
	require.Contains(t, line, "amountMinor")
	require.Contains(t, line, "accountCode")
}

func TestPublicBalancesExposeAggregates(t *testing.T) {
	router := publicRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/v1/balances", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Balances []map[string]any `json:"balances"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.Balances)
	require.EqualValues(t, 125000, payload.Balances[0]["balanceMinor"])
}

func TestPublicTierRateLimitsPerClient(t *testing.T) {
	router := publicRouter(t, WithPublicRateLimit(1, 2))

	statuses := map[int]int{}
	for range 5 {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/public/v1/institutions", nil)
		request.RemoteAddr = "203.0.113.9:1234"
		router.ServeHTTP(recorder, request)
		statuses[recorder.Code]++
	}

	require.Positive(t, statuses[http.StatusOK], "burst should serve some requests")
	require.Positive(t, statuses[http.StatusTooManyRequests], "excess requests must be limited")

	// A different client is unaffected by the first client's bucket.
	other := httptest.NewRecorder()
	otherRequest := httptest.NewRequest(http.MethodGet, "/public/v1/institutions", nil)
	otherRequest.RemoteAddr = "198.51.100.7:9999"
	router.ServeHTTP(other, otherRequest)
	require.Equal(t, http.StatusOK, other.Code)
}

func TestPublicWritesDoNotExist(t *testing.T) {
	router := publicRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/public/v1/journal-entries", nil))
	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}
