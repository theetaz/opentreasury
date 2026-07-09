# ADR-0001: Architecture Stance

- **Status:** Accepted
- **Date:** 2026-07-09
- **Deciders:** @theetaz

## Context

The OpenTreasury blueprint ([docs/opentreasury-blueprint.md](../opentreasury-blueprint.md))
requires a small set of framing decisions that every subsequent design inherits.
Recording them once, here, prevents each PR from re-litigating them.

## Decision

Six framing decisions are adopted:

1. **Sovereign deployment model.** One OpenTreasury deployment per government
   (on-premises or their cloud). No shared multi-government SaaS, no
   cross-border data pooling. Only the public read path is engineered for
   mass scale; operator surfaces are engineered for correctness and
   auditability.
2. **PostgreSQL is the system of record; Hyperledger Fabric is the system of
   proof.** The chain never holds operational truth, PII, or non-public data —
   only hashes, anchors, attestations, and public-safe projections.
3. **Append-only financial core.** Once posted, journal data is immutable.
   Corrections and reversals are new, linked entries. No UPDATE or DELETE on
   posted financial records, ever.
4. **Policy-as-code below every surface.** REST, MCP, dashboards, and exports
   all pass through the same OPA authorization and redaction layer. No
   consumer — human or AI agent — has a privileged side door.
5. **Event-driven integration, synchronous queries.** Writes flow through
   validated APIs into PostgreSQL and out through a transactional outbox to
   the event bus. Reads are synchronous against purpose-built projections.
6. **Contract-first.** OpenAPI documents, event JSON Schemas, the connector
   interchange format, and the proof format are the source of truth; Go and
   TypeScript types are generated from them, never hand-duplicated.

## Alternatives Considered

- **Multi-tenant SaaS** — rejected: governments will not put treasury data in
  a shared operator's hands; sovereignty is an adoption prerequisite.
- **Blockchain as system of record** — rejected: operational truth needs
  relational integrity, redactability (right-to-correct via projections), and
  DBA-familiar operations; a chain of record would make redaction impossible
  and operations exotic.
- **Mutable records with audit log** — rejected: anchoring only has meaning if
  the anchored data cannot change; append-only is what makes public
  verification honest.
- **Hand-written types on each surface** — rejected: the 2026-07 audit found
  three divergent type sources (Go structs, OpenAPI, TS types) after only 23
  PRs; drift is inevitable without generation.

## Consequences

- Easier: public verifiability, audits, horizontal read scaling, community
  review of policies.
- Harder: schema migrations must respect append-only history; corrections UX
  needs care; generated-type toolchains must be maintained.
- Revisit: the Fabric choice is isolated behind the anchoring interface
  (ledger-gateway); if operational burden blocks adoption, a transparency-log
  substitute (e.g. Trillian-style) can be proposed in a future ADR without
  touching the financial core.
