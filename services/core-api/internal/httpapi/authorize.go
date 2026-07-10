package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
	"github.com/opentreasury/opentreasury/services/core-api/internal/authz"
)

// Authorizer is the authorization decision seam (satisfied by internal/authz).
type Authorizer interface {
	Allow(ctx context.Context, requestID string, input authz.Input) bool
}

// resourceForPath maps a request path to the policy resource name.
func resourceForPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/v1/transactions"):
		return "transactions"
	case strings.HasPrefix(path, "/v1/journal-entries"):
		return "journal"
	case strings.HasPrefix(path, "/v1/balances"):
		return "balances"
	case strings.HasPrefix(path, "/v1/accounts"):
		return "accounts"
	case strings.HasPrefix(path, "/v1/institutions"):
		return "institutions"
	case strings.HasPrefix(path, "/v1/audit-events"):
		return "audit-events"
	case strings.HasPrefix(path, "/v1/reconciliation"):
		return "reconciliation"
	case strings.HasPrefix(path, "/v1/staging-records"):
		return "staging"
	default:
		return ""
	}
}

func actionForMethod(method string) string {
	if method == http.MethodGet {
		return "read"
	}
	return "write"
}

// authorized checks the policy for the current principal, resource, and target
// institution, writing 403 if denied. Returns true when the request may proceed.
// targetInstitutionID is the institution the request acts on ("" = unscoped).
func (config routerConfig) authorized(response http.ResponseWriter, request *http.Request, targetInstitutionID string) bool {
	if config.authorizer == nil {
		return true // authorization disabled (local dev without policy)
	}

	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeError(response, http.StatusUnauthorized, "authentication required")
		return false
	}

	decision := config.authorizer.Allow(request.Context(), requestIDFromContext(request.Context()), authz.Input{
		Principal:     principal,
		Action:        actionForMethod(request.Method),
		Resource:      resourceForPath(request.URL.Path),
		InstitutionID: targetInstitutionID,
	})
	if !decision {
		writeError(response, http.StatusForbidden, "not authorized")
		return false
	}

	return true
}
