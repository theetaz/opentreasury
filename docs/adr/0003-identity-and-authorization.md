# ADR-0003: Identity and Authorization

- **Status:** Accepted
- **Date:** 2026-07-09
- **Deciders:** @theetaz

## Context

Institution-scoped data was previously served to anonymous callers (audit
finding D2). Phase 2 introduces authentication and authorization before any
real government data flows.

## Decision

1. **Keycloak is the identity provider.** OIDC/OAuth2; the web app uses the
   authorization-code flow with PKCE. A seeded `opentreasury` realm ships three
   demo users (treasury-admin, institution-user, auditor) and an
   `institution_id` claim mapper.
2. **The core API verifies bearer tokens** against the realm JWKS (go-oidc) and
   maps claims to a `Principal` (subject, username, realm roles, institution).
   Authentication is enabled only when `OPENTREASURY_OIDC_ISSUER_URL` is set, so
   local dev still boots without Keycloak.
3. **Authorization is policy-as-code, evaluated in-process.** The Rego policy
   (`policies/opa/authz.rego`) is embedded in the API via the OPA Go SDK — the
   policy file is the single source of truth, not a re-implementation. Chosen
   over a sidecar to avoid a per-request network hop and an extra moving part in
   every deployment; the embedding is behind an interface so a sidecar can
   replace it later without touching handlers.
4. **Decision model**: treasury-admin = full cross-institution; auditor =
   read-only cross-institution; institution-user = own institution only (with
   shared chart-of-accounts reference readable by all). Deny by default.
5. **Every decision is logged** with request id, subject, roles, action,
   resource, and target institution.

## Alternatives Considered

- **OPA as a sidecar** — rejected for now: operational weight and latency for a
  policy small enough to embed; revisit if policies grow to need hot-reload or
  cross-service sharing.
- **Roles hard-coded in Go** — rejected: policy must be auditable by outside
  reviewers and changeable without recompiling (blueprint principle).
- **Audience validation on Keycloak access tokens** — Keycloak puts the client
  id in `azp`, not `aud`; issuer + signature validity is the check.

## Consequences

- Easier: outside review of authz rules; institution isolation is enforced in
  one place; adding a resource is a route→resource map entry plus policy review.
- Harder: local dev needs Keycloak for the authenticated path (mitigated: the
  API and web both no-op auth when unconfigured).
- The public read tier (Phase 2.4) and field-level redaction (2.3) build on this
  principal + policy foundation.
