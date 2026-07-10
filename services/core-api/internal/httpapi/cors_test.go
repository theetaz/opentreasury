package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouterAddsCORSHeadersForAllowedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	NewRouter(WithAllowedOrigins([]string{"http://localhost:5173"})).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Origin", response.Header().Get("Vary"))
}

func TestRouterHandlesCORSPreflightForAllowedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/v1/transactions", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	NewRouter(WithAllowedOrigins([]string{"http://localhost:5173"})).ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, response.Header().Get("Access-Control-Allow-Methods"), http.MethodPost)
	require.Contains(t, response.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

func TestRouterDoesNotAddCORSHeadersForUnknownOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "https://example.invalid")
	response := httptest.NewRecorder()

	NewRouter(WithAllowedOrigins([]string{"http://localhost:5173"})).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Empty(t, response.Header().Get("Access-Control-Allow-Origin"))
}

func TestPublicTierIsCORSOpenToAnyOrigin(t *testing.T) {
	router := NewRouter(WithAllowedOrigins([]string{"http://localhost:5173"}))

	// The anonymous public tier serves published data any site may read.
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/public/v1/institutions", nil)
	request.Header.Set("Origin", "https://citizen-tools.example.org")
	router.ServeHTTP(response, request)
	require.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"),
		"public data must be readable from any origin")

	// The authenticated tier stays restricted to configured origins.
	restricted := httptest.NewRecorder()
	restrictedRequest := httptest.NewRequest(http.MethodGet, "/v1/institutions", nil)
	restrictedRequest.Header.Set("Origin", "https://citizen-tools.example.org")
	router.ServeHTTP(restricted, restrictedRequest)
	require.Empty(t, restricted.Header().Get("Access-Control-Allow-Origin"))
}
