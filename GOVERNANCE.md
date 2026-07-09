# OpenTreasury Governance

## Current Model: Maintainer Group

OpenTreasury is in its founding phase and is governed by a small maintainer
group with final decision authority. This is a deliberate, temporary
arrangement documented here so the path beyond it is explicit.

### Maintainers

| Role | GitHub |
|---|---|
| Founding maintainer | [@theetaz](https://github.com/theetaz) |

Maintainers merge PRs, cut releases, triage security reports, and steward the
roadmap ([docs/opentreasury-blueprint.md](docs/opentreasury-blueprint.md)).

### How Decisions Are Made

- **Code-level decisions** happen on PRs. Two approvals required once the
  maintainer group has ≥ 3 members; today, one maintainer approval.
- **Architecture decisions** require an ADR in [docs/adr/](docs/adr/), proposed
  by PR and open for community comment for at least 7 days before merge (except
  security-urgent changes).
- **Roadmap changes** are amendments to the blueprint document, via PR.
- Disagreements are resolved by consensus-seeking; the founding maintainer
  holds the tie-break until the TSC exists (below).

## Planned Evolution: Technical Steering Committee

When contributors from **three or more independent organizations** are
regularly landing changes, governance moves to an elected Technical Steering
Committee (TSC):

- 5 seats, 12-month terms, elected by contributors with merged commits in the
  prior 12 months.
- No single organization may hold more than 2 seats.
- The TSC owns roadmap, releases, maintainer appointments, and this document.

This transition is itself an ADR when it happens.

## Becoming a Maintainer

Sustained, high-quality contribution (code, review, docs, or community) over
~3 months; nominated by an existing maintainer; public announcement with a
7-day comment window.

## Values That Bind All Governance

1. **Sovereignty of adopters** — no decision may create dependence on a
   proprietary or cloud-locked component.
2. **Verifiability first** — features that publish data must ship the means to
   verify it.
3. **The non-goals are load-bearing** — OpenTreasury does not execute financial
   transactions and does not become an FMIS. Changing a non-goal requires an
   ADR with an extended 30-day comment window.
