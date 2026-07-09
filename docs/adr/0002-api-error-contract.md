# ADR-0002: API Error Contract and Strict Parameter Validation

- **Status:** Accepted
- **Date:** 2026-07-09
- **Deciders:** @theetaz

## Context

The 2026-07 audit found the implementation and the OpenAPI contract
disagreeing in several places: `limit` values above the documented maximum
were silently clamped; duplicate-ID creates surfaced as raw database errors at
500; internal error strings (including driver and network detail) leaked into
response bodies; several actually-producible status codes (500, 409, 413, 422)
were undocumented. The contract test only asserted that paths existed, so none
of this was caught.

## Decision

1. **Strict validation over silent coercion.** Out-of-range or malformed
   request parameters are rejected with 400 and a specific message (e.g.
   `invalid limit`). The API never silently rewrites a caller's input —
   silent clamping made pagination behavior unobservable to clients.
2. **Constraint violations map to semantic status codes.** Duplicate
   identifier → 409; reference to a missing institution (FK violation) → 422.
   The repository layer maps Postgres error codes (23505, 23503) to domain
   errors; HTTP handlers translate domain errors to status codes. Raw driver
   errors never choose a status.
3. **500 bodies are sanitized.** Always exactly
   `{"error":"internal server error"}`; operators get the detail from
   structured logs keyed by `X-Request-ID` (ADR follows Epic 0.4 behavior).
4. **Every producible status is documented, and conformance is tested.** The
   contract test suite drives real requests through the router and validates
   each response against the OpenAPI schemas
   (`openapi_response_conformance_test.go`), so contract drift fails CI.

## Alternatives Considered

- **Silent clamping (status quo)** — rejected: clients cannot detect that
  they received a truncated window; breaks pager implementations invisibly.
- **404 for unknown institution on create** — rejected: 404 refers to the
  request URI, not a body reference; 422 is the accurate semantic.
- **Problem-details (RFC 9457) envelope now** — deferred: planned with API v1
  consolidation (blueprint Epic 1.6) to avoid two envelope migrations; the
  current `{"error": string}` envelope stays until then.

## Consequences

- Easier: client debugging, contract-first client generation, honest pagers.
- Harder: any consumer relying on clamped limits breaks (none known; the web
  app caps its own input).
- The `{"error"}` envelope will migrate to problem-details in Epic 1.6; the
  conformance suite makes that migration mechanical.
