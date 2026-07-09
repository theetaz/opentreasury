package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeVerifier struct {
	principal Principal
	err       error
}

func (f fakeVerifier) Verify(context.Context, string) (Principal, error) {
	return f.principal, f.err
}

func protectedHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal, ok := PrincipalFromContext(request.Context())
		if !ok {
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte("anonymous"))
			return
		}
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(principal.Username))
	})
}

func publicOnly(paths ...string) func(*http.Request) bool {
	set := map[string]struct{}{}
	for _, p := range paths {
		set[p] = struct{}{}
	}
	return func(r *http.Request) bool {
		_, ok := set[r.URL.Path]
		return ok
	}
}

func TestMiddlewareAllowsPublicPathsWithoutToken(t *testing.T) {
	handler := Middleware(fakeVerifier{}, publicOnly("/healthz"), protectedHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "anonymous", recorder.Body.String())
}

func TestMiddlewareRejectsProtectedPathWithoutToken(t *testing.T) {
	handler := Middleware(fakeVerifier{}, publicOnly("/healthz"), protectedHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/transactions", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "missing bearer token")
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	handler := Middleware(fakeVerifier{err: errors.New("bad signature")}, publicOnly("/healthz"), protectedHandler())

	request := httptest.NewRequest(http.MethodGet, "/v1/transactions", nil)
	request.Header.Set("Authorization", "Bearer garbage")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid token")
}

func TestMiddlewarePutsPrincipalOnContextForValidToken(t *testing.T) {
	verifier := fakeVerifier{principal: Principal{
		Subject:       "sub-1",
		Username:      "institution-user",
		Roles:         []string{"institution-user"},
		InstitutionID: "minfin",
	}}
	handler := Middleware(verifier, publicOnly("/healthz"), protectedHandler())

	request := httptest.NewRequest(http.MethodGet, "/v1/transactions", nil)
	request.Header.Set("Authorization", "Bearer valid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "institution-user", recorder.Body.String())
}

func TestPrincipalRoleHelpers(t *testing.T) {
	treasury := Principal{Roles: []string{"treasury-admin"}}
	require.True(t, treasury.IsTreasury())
	require.False(t, treasury.IsAuditor())

	auditor := Principal{Roles: []string{"auditor"}}
	require.True(t, auditor.IsAuditor())
	require.False(t, auditor.IsTreasury())

	institution := Principal{Roles: []string{"institution-user"}, InstitutionID: "minfin"}
	require.False(t, institution.IsTreasury())
	require.True(t, institution.HasRole("institution-user"))
}
