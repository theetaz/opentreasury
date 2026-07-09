package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
)

// principalVerifier returns a fixed principal, simulating a validated token.
type principalVerifier struct{ principal auth.Principal }

func (p principalVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return p.principal, nil
}

func newAuthorizer(t *testing.T) Authorizer {
	t.Helper()
	authorizer, err := authz.New(context.Background(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return authorizer
}

func authRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer token")
	return request
}

func TestInstitutionUserCannotReadAnotherInstitution(t *testing.T) {
	router := NewRouter(
		WithTokenVerifier(principalVerifier{auth.Principal{
			Subject: "i", Roles: []string{"institution-user"}, InstitutionID: "minfin",
		}}),
		WithAuthorizer(newAuthorizer(t)),
		WithTransactionRepository(&recordingTransactionRepository{}),
	)

	// Own institution: allowed.
	own := httptest.NewRecorder()
	router.ServeHTTP(own, authRequest("/v1/transactions?institutionId=minfin"))
	require.Equal(t, http.StatusOK, own.Code)

	// Another institution: forbidden.
	other := httptest.NewRecorder()
	router.ServeHTTP(other, authRequest("/v1/transactions?institutionId=health"))
	require.Equal(t, http.StatusForbidden, other.Code)
	require.Contains(t, other.Body.String(), "not authorized")
}

func TestTreasuryAdminReadsAnyInstitution(t *testing.T) {
	router := NewRouter(
		WithTokenVerifier(principalVerifier{auth.Principal{
			Subject: "t", Roles: []string{"treasury-admin"},
		}}),
		WithAuthorizer(newAuthorizer(t)),
		WithTransactionRepository(&recordingTransactionRepository{}),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, authRequest("/v1/transactions?institutionId=health"))
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuditorCannotWrite(t *testing.T) {
	router := NewRouter(
		WithTokenVerifier(principalVerifier{auth.Principal{
			Subject: "a", Roles: []string{"auditor"},
		}}),
		WithAuthorizer(newAuthorizer(t)),
		WithTransactionRepository(&recordingTransactionRepository{}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", validTransactionBody(t))
	request.Header.Set("Authorization", "Bearer token")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestInstitutionUserReadsSharedAccounts(t *testing.T) {
	router := NewRouter(
		WithTokenVerifier(principalVerifier{auth.Principal{
			Subject: "i", Roles: []string{"institution-user"}, InstitutionID: "minfin",
		}}),
		WithAuthorizer(newAuthorizer(t)),
		WithAccountRepository(&recordingAccountRepository{}),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, authRequest("/v1/accounts"))
	require.Equal(t, http.StatusOK, recorder.Code)
}
