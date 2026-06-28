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
