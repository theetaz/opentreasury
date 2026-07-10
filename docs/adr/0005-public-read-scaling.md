# ADR-0005: Public Read-Path Scaling Posture

- **Status:** Accepted
- **Date:** 2026-07-10
- **Deciders:** @theetaz

## Context

The blueprint (6.3) anticipated CQRS read models with a search engine
(e.g. OpenSearch) for the public tier, sized for national-scale traffic.
The public tier now exists: anonymous, redacted, rate-limited, CORS-open,
with `Cache-Control` on responses. The question is when to add dedicated
read infrastructure.

## Decision

**Defer dedicated read models until a deployment's measured load demands
them.** The shipped posture scales a long way on simpler mechanics:

1. **Statelessness** — `core-api` scales horizontally behind the ingress;
   list endpoints are index-backed with server-side pagination capped at
   100 rows.
2. **HTTP caching as the first read model.** Public responses carry
   `Cache-Control`; anchored data is immutable by construction, so proof
   responses are hard-cacheable. A CDN or reverse-proxy cache in front of
   `/public/v1` is configuration, not code, and absorbs the
   thundering-herd case (budget-day traffic) far more cheaply than a
   search cluster.
3. **Rate limiting** protects the origin independently of caching.

Trigger for revisiting: sustained origin p95 above the SLO with caching in
place, or a product need for full-text/faceted search across entries. At
that point a projection consumer feeding a search index is additive — the
public API contract does not change.

## Consequences

- Easier: no second data store to secure, back up, or reconcile during
  pilots; the DR story stays "PostgreSQL is the system of record".
- Harder: no full-text search over published data yet; institution/fiscal
  filters must suffice until the trigger is hit.
- The vendor-neutrality rule applies to the future choice: prefer
  OpenSearch or another open-source engine, documented in
  `docs/development.md` when introduced.
