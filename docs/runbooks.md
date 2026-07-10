# Alert Runbooks

One section per alert in `infra/docker/prometheus/alerts.yml`. Each runbook
answers: what fired, what breaks for users, how to diagnose, how to mitigate.
Logs are in Grafana (Loki datasource, filter by `service`); metrics are on
the OpenTreasury Overview dashboard.

## ApiDown

- **Impact**: the dashboard, public portal, mobile app, and MCP tools are all
  unavailable.
- **Diagnose**: `docker compose ps core-api` / pod status; then core-api logs
  in Loki (`{service="core-api"}`). Common causes: database unreachable
  (readyz fails), OIDC discovery failing at startup, crash loop after deploy.
- **Mitigate**: restore the database connection first (readyz gates on it).
  Roll back the last deploy if the crash started with it. The API is
  stateless — restart freely.

## ApiHighErrorRate

- **Impact**: users see failures; data writes may be rejected.
- **Diagnose**: split by route: `sum by (route) (rate(opentreasury_http_requests_total{status=~"5.."}[5m]))`.
  Then Loki `{service="core-api"} |= "internal error"` — every 5xx logs its
  request_id and the real error (never sent to clients).
- **Mitigate**: a single failing route after a deploy → roll back. Database
  errors → check Postgres health/connections. Errors only on writes →
  check for lock contention or a failed migration.

## ApiHighLatency

- **Impact**: degraded experience; timeouts in the web/mobile clients.
- **Diagnose**: per-route p95 (`histogram_quantile(0.95, sum by (route, le) (...))`).
  List endpoints are index-backed; a slow list usually means a missing index
  after schema changes or table bloat.
- **Mitigate**: `EXPLAIN ANALYZE` the slow route's query; check Postgres CPU
  and connection saturation; scale API replicas only if CPU-bound.

## AnchoringFailing

- **Impact**: new entries post normally but gain no proofs — the
  transparency promise degrades until fixed. No operational data is at risk.
- **Diagnose**: gateway logs (`{service="ledger-gateway"} |= "anchoring batch failed"`).
  Fabric backend: is the peer reachable, is the chaincode container running
  (`docker compose -f infra/fabric/docker-compose-fabric.yaml ps`)?
  Transparency-log backend: database errors.
- **Mitigate**: anchoring is idempotent and crash-safe — fix the backend and
  the gateway drains the backlog on the next tick, in order. Nothing needs
  manual replay. If the Fabric network must be rebuilt, `make fabric-up`
  converges; committed anchors on the old ledger stay referenced in the
  anchors table.

## AnchorLagGrowing

- **Impact**: proofs are late, not failing. Verify pages show entries as
  not-yet-anchored.
- **Diagnose**: if `AnchoringFailing` is also firing, follow that runbook.
  Otherwise the gateway may be down (`GatewayDown`) or the batch size is
  too small for a bulk import — a large backfill legitimately takes
  `entries / 500` ticks (10s each).
- **Mitigate**: during planned bulk imports, silence this alert or expect it
  to clear on its own; watch `opentreasury_anchor_entries_total` climbing.

## GatewayDown

- **Impact**: no new anchors at all; lag grows monotonically.
- **Diagnose**: gateway container state and last logs. On startup the
  gateway exits if the Fabric enrollment material is unreadable — check the
  mounted MSP paths.
- **Mitigate**: restart; the gateway is stateless. Lag drains automatically
  once it is back.
