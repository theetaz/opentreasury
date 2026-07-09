package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
)

type stubVerifier struct {
	principal auth.Principal
	ok        bool
}

func (s stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	if !s.ok {
		return auth.Principal{}, context.Canceled
	}
	return s.principal, nil
}

func TestRouterEnforcesAuthWhenVerifierSet(t *testing.T) {
	router := NewRouter(WithTokenVerifier(stubVerifier{ok: false}))

	// Health stays public.
	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, health.Code)

	// A protected route without a token is 401.
	protected := httptest.NewRecorder()
	router.ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/v1/accounts", nil))
	require.Equal(t, http.StatusUnauthorized, protected.Code)
}

func TestRouterUnauthenticatedByDefault(t *testing.T) {
	// No verifier configured: local-dev mode leaves routes open (503 because
	// no repository is wired, not 401).
	router := NewRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/accounts", nil))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
