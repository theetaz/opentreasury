// Package authz evaluates the Rego authorization policy in-process. The policy
// file is the single source of truth; this package is the seam the HTTP layer
// calls, and it logs every decision.
package authz

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/opentreasury/opentreasury/services/core-api/internal/auth"
)

//go:embed authz.rego
var policySource string

// Input is the authorization question posed to the policy.
type Input struct {
	Principal     auth.Principal
	Action        string // "read" or "write"
	Resource      string // transactions | journal | balances | accounts | institutions | audit-events
	InstitutionID string // institution the request targets, or "" for unscoped
}

// Authorizer evaluates authorization decisions against the embedded policy.
type Authorizer struct {
	query  rego.PreparedEvalQuery
	logger *slog.Logger
}

// New compiles the policy once at startup; a compile failure is fatal because
// running without enforceable policy would be a security hole.
func New(ctx context.Context, logger *slog.Logger) (*Authorizer, error) {
	query, err := rego.New(
		rego.Query("data.opentreasury.authz.allow"),
		rego.Module("authz.rego", policySource),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("compiling authorization policy: %w", err)
	}

	return &Authorizer{query: query, logger: logger}, nil
}

// Allow returns whether the principal may perform the action, logging the
// decision for audit.
func (a *Authorizer) Allow(ctx context.Context, requestID string, input Input) bool {
	regoInput := map[string]any{
		"principal": map[string]any{
			"roles":         input.Principal.Roles,
			"institutionId": input.Principal.InstitutionID,
		},
		"action":        input.Action,
		"resource":      input.Resource,
		"institutionId": input.InstitutionID,
	}

	results, err := a.query.Eval(ctx, rego.EvalInput(regoInput))
	allowed := err == nil && results.Allowed()

	a.logger.Info("authorization decision",
		"request_id", requestID,
		"subject", input.Principal.Subject,
		"roles", input.Principal.Roles,
		"action", input.Action,
		"resource", input.Resource,
		"target_institution", input.InstitutionID,
		"allowed", allowed,
	)

	return allowed
}
