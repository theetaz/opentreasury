# ADR-0004: Anchoring and Independent Verification

- **Status:** Accepted
- **Date:** 2026-07-10
- **Deciders:** @theetaz

## Context

The platform's trust promise is "verify, don't trust": the public must be able
to confirm a fund movement existed, unmodified, without trusting OpenTreasury's
servers. This requires tamper-evident anchoring of posted entries and a proof
format anyone can check independently. ADR-0001 anticipated this and isolated
the anchoring backend behind an interface.

## Decision

1. **Merkle batch anchoring.** The ledger gateway batches unanchored posted
   entries, hashes each to a **canonical entry hash** (a documented, stable
   SHA-256 encoding of its public-safe fields — id, institution, fiscal year,
   effective date, status, entry type, sorted lines; no free text), builds a
   Merkle tree, and commits the root to a ledger backend. Each entry stores its
   Merkle inclusion proof.
2. **Pluggable backend behind an interface.** `anchor.Backend` has two
   implementations:
   - **Transparency log** (default, ships now): an append-only, hash-chained
     table in Postgres. Real, tamper-evident, zero operational weight — right
     for local dev and small deployments.
   - **Hyperledger Fabric** (production): the `treasury-chaincode`
     `AnchorContract` records immutable Merkle roots on the permissioned
     ledger. `make fabric-up` deploys a local single-org network (Raft
     orderer + peer + chaincode-as-a-service, TLS on the Fabric nodes,
     generated crypto material never committed); the gateway selects the
     backend with `OPENTREASURY_ANCHOR_BACKEND=fabric` and commits roots via
     the Fabric Gateway API. The Merkle root doubles as the on-chain anchor
     id, so a crash-retry of the same batch converges instead of
     double-anchoring, and the chaincode rejects any recommit of an existing
     id — history cannot be rewritten.
3. **Public proof endpoint.** `GET /public/v1/entries/{id}/proof` returns the
   canonical entry, the anchor receipt (root, backend, backend ref), the leaf
   hash, and the inclusion proof — anonymous and hard-cached (anchored data is
   immutable).
4. **Standalone verifier.** A separate binary (`cmd/verify`) fetches an entry +
   proof and recomputes the canonical hash and Merkle path itself, depending on
   nothing but the math. Tampering makes recomputation diverge, so the anchor
   exposes it. This is the trust demo.

## Alternatives Considered

- **Fabric-only, now** — rejected: a full network (peers, orderer, CA,
  channels, chaincode lifecycle) is heavy to run and validate locally and would
  block shipping the trust mechanism. The interface makes it a drop-in.
- **Public blockchain anchoring** — rejected on sovereignty grounds (ADR-0001).
- **Signing entries individually** — rejected: batch Merkle anchoring is far
  cheaper per entry and gives the same per-entry inclusion proof.

## Consequences

- Easier: independent verification works today; swapping in Fabric touches only
  the gateway's backend wiring, not the proof format, the API, or the verifier.
- Harder: the canonical hash encoding is now a compatibility surface — changing
  it breaks existing proofs, so it is versioned (`v1|…`) and any change is a new
  version with re-anchoring.
- Revisit: the local network is single-org for development; a production
  deployment is multi-org (finance ministry, audit office, civil-society
  observers as endorsing organizations) with CA-issued identities replacing
  cryptogen material.
