# Data Onboarding Playbook

How a pilot team takes OpenTreasury from "deployed" to "publishing real
fiscal data": adapt the chart of accounts, author a connector profile for an
existing source system, validate through staging, and go live. Companion to
the [deployment guide](deployment-guide.md).

## 1. Chart of accounts mapping workshop

OpenTreasury ships a reference chart (`database/seeds`) modeled on
GFS-style classifications. Adapting it is a half-day workshop with the
treasury's accountants:

1. **Inventory** the codes actually used by the source systems being
   onboarded (payments exports usually reference a handful of expense and
   funding codes, not the whole chart).
2. **Map** each source code to a reference account, or add accounts under
   the right parent. Keep codes hierarchical (children extend the parent
   code) — the UI derives indentation and rollups from the hierarchy.
3. **Decide the publication level.** Everything in the chart is publishable
   metadata; sensitivity lives in journal descriptions and source documents,
   which are never published (publication policy, ADR-0003).
4. Capture the result as seed SQL in `database/seeds` so it is reviewable,
   versioned, and reproducible in every environment.

Outputs: an adapted chart, and for each source system a **debit/credit
account decision** the connector profile will encode (e.g. payments post as
expense `22` against funding `6202`).

## 2. Authoring a connector profile

The generic file connector maps CSV exports declaratively — a domain expert
can author a profile without touching Go. Full schema (all fields required):

```yaml
# profiles/<system>-<flow>.yaml
source_system: itmis   # stable id; becomes part of every record's identity
version: 1             # bump on ANY mapping change
columns:               # source CSV header names for the interchange fields
  source_ref: reference          # unique per row in the source system
  occurred_at: payment_date      # YYYY-MM-DD
  institution: institution_code  # must match an institution id
  amount: amount                 # major units, e.g. 1250.00
  currency: currency             # ISO 4217
debit_account: "22"    # from the mapping workshop
credit_account: "6202"
```

Rules that keep ingestion honest:

- **`source_ref` is the dedup key.** Rows are hashed
  (system + version + mapped fields); redelivering the same file converges
  as duplicates instead of double-posting. Bump `version` when a mapping
  change alters meaning — the same row then hashes differently on purpose.
- Rows that fail mapping are **skipped and counted**, never guessed at.
  Rows that map but fail validation are **quarantined** in staging with the
  reason, visible on the Ingestion page.
- The connector authenticates with client credentials carrying the
  `connector` role: it can only write to staging. Posting to the journal is
  the staging pipeline's decision, inside the core API's validation.

## 3. Rehearse with demo data

Before touching a real export, exercise the full pipeline with generated
data shaped exactly like the interchange format:

```sh
cd services/connectors/generic-file
go run ./cmd/demo-data -rows 250 -fiscal-year 2026 \
  -institutions minfin,health,educ -out dropbox/demo-payments.csv
```

The same seed reproduces the same file (`-seed`), so a rehearsal is
repeatable. Within seconds the connector picks the file up, delivers it to
staging, and moves it to `processed/`. Then confirm, in order:

1. **Ingestion page**: the batch appears; POSTED / QUARANTINED / DUPLICATE
   counts make sense.
2. **Journal**: posted entries are balanced and carry the workshop's
   debit/credit accounts.
3. **Balances**: stocks moved by exactly the flows (the invariant is
   enforced, but look anyway).
4. **Verification**: after the next anchoring tick, entries show a Verify
   chip; `go run ./cmd/verify -entry <id>` prints `VERIFIED` from an
   independent process.
5. Re-drop the identical file: everything converges as DUPLICATE, nothing
   double-posts.

## 4. First real export

- Start with **one month of one flow** (e.g. payments) from one system.
- Have the source-system operator produce the export with the agreed
  columns; drop it into the connector directory (or mount the directory
  where the system already writes exports).
- Reconcile totals: staging batch total vs. the source system's own report
  for the period. Investigate quarantined rows — they are usually institution
  codes missing from the registry or malformed dates, both fixable in the
  source or the registry, not by editing data in transit.
- Only after reconciliation, repeat for prior periods (backfill) and then
  schedule the recurring export.

## 5. Go-live checklist

- [ ] Adapted chart of accounts merged as seeds and loaded.
- [ ] Institution registry covers every code the source system emits.
- [ ] Connector profile reviewed by both a treasury accountant and an engineer.
- [ ] Demo-data rehearsal (section 3) passed end to end, including re-drop convergence.
- [ ] One real period reconciled against the source system's own totals.
- [ ] Anchoring is running (anchor lag near zero) and a spot-checked entry verifies independently.
- [ ] Public portal reviewed: published fields only, no descriptions, correct institution names.
- [ ] Operators know where quarantined rows surface and who owns triage.
