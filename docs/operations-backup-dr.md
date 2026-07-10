# Backup & Disaster Recovery

PostgreSQL is the operational system of record: the double-entry ledger,
audit trail, staging pipeline, anchors, and inclusion proofs all live there.
Its recovery properties **are** the platform's recovery properties. The
traceability layer changes the calculus in one useful way: anchors committed
to an external backend (Fabric or a mirrored transparency log) survive a
database loss and make any post-restore tampering detectable.

## Objectives (pilot baseline — tune to your SLOs)

| Objective | Target | Rationale |
|---|---|---|
| RPO | ≤ 5 minutes | continuous WAL archiving |
| RTO | ≤ 4 hours | scripted restore, stateless services |

## Backups

- **Continuous WAL archiving + periodic base backups** (pgBackRest,
  WAL-G, or your managed provider's PITR). Nothing in the schema needs
  special handling; everything is plain relational data.
- Retain backups per your fiscal-records retention law — treasury data
  commonly requires 7+ years. Archived partitions can move to cheaper
  storage classes.
- Back up the **Keycloak/IdP realm** (or rely on your IdP's own DR) and the
  **Fabric ledger volumes** if you run your own network. Losing Fabric loses
  proofs, not truth — re-anchoring rebuilds proofs from Postgres (new
  anchors, old backend refs remain recorded).

## Restore drill (rehearse quarterly)

1. Provision a scratch database instance; restore latest base backup and
   replay WAL to a chosen point in time.
2. Run migrations' `version` check (`schema_migrations`) — must match the
   deployed release.
3. Integrity checks against the restored copy:
   - **Balance invariant**: every `account_balances` row equals the sum of
     its posted `journal_lines` (the integration suite's invariant query).
   - **Audit continuity**: `audit_events` count per day has no unexplained
     gaps around the restore point.
   - **Anchor spot check**: run the standalone verifier
     (`go run ./cmd/verify -entry <id>`) against several entries anchored
     *before* the restore point. VERIFIED proves the restored data matches
     what was publicly anchored — this is the tamper-evidence dividend.
4. Point a scratch core-api at the restored database; hit `/readyz` and a
   few list endpoints.
5. Record duration (actual RTO), data loss window (actual RPO), and any
   surprises as an issue; fix the runbook in the same PR.

## Loss scenarios

| Scenario | Recovery |
|---|---|
| Database lost | PITR restore → integrity checks above → repoint services. Entries posted after the restore point are re-ingested from staging sources (connectors converge by `source_hash` — re-dropping files cannot double-post). |
| Fabric ledger lost | Postgres intact → truth intact. Rebuild the network (`make fabric-up` locally; your org's process in production); the gateway re-anchors unanchored entries automatically. Historical anchor refs to the old ledger stay in the `anchors` table for the audit trail. |
| Both lost | Restore Postgres first (it contains the proofs and anchor history), then rebuild Fabric. Third-party verifiers holding old roots can still check entries anchored before the loss. |
| Region loss | Run the same restore into a second region; all services are stateless and configured entirely by environment — the Helm values file is the recovery script. |
