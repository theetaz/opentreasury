// Package auth provides OIDC bearer-token verification and the request
// principal that downstream authorization decisions consume.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// Principal is the authenticated caller. InstitutionID scopes what an
// institution user may see; treasury and auditor roles are cross-institution.
type Principal struct {
	Subject       string
	Username      string
	Roles         []string
	InstitutionID string
}

func (p Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsTreasury and IsAuditor identify cross-institution roles that are not
// limited to a single institution's data.
func (p Principal) IsTreasury() bool { return p.HasRole("treasury-admin") || p.HasRole("treasury") }
func (p Principal) IsAuditor() bool  { return p.HasRole("auditor") }

type principalKey struct{}

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal, or false if the
// request was not authenticated (only public routes reach handlers unauthenticated).
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

// TokenVerifier validates a raw bearer token and returns its principal.
// The production implementation verifies OIDC signatures against Keycloak's
// JWKS; tests supply a fake.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Principal, error)
}

// Middleware verifies bearer tokens for protected routes. Paths matched by
// isPublic are served without authentication; everything else requires a
// valid token or gets 401.
func Middleware(verifier TokenVerifier, isPublic func(*http.Request) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if isPublic(request) {
			next.ServeHTTP(response, request)
			return
		}

		token, ok := bearerToken(request)
		if !ok {
			writeUnauthorized(response, "missing bearer token")
			return
		}

		principal, err := verifier.Verify(request.Context(), token)
		if err != nil {
			writeUnauthorized(response, "invalid token")
			return
		}

		next.ServeHTTP(response, request.WithContext(withPrincipal(request.Context(), principal)))
	})
}

func bearerToken(request *http.Request) (string, bool) {
	header := request.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

func writeUnauthorized(response http.ResponseWriter, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusUnauthorized)
	_, _ = response.Write([]byte(`{"error":"` + message + `"}`))
}
