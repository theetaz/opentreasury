# OpenTreasury Deployment Guide

Audience: the IT team of a government or public institution deploying
OpenTreasury for a pilot or production use. It assumes familiarity with
Kubernetes and PostgreSQL administration, and complements
[development.md](development.md) (local workflow) and the ADRs in
[adr/](adr/) (why the system is shaped this way).

## 1. What you are deploying

| Component        | Role                                                             | State                         |
| ---------------- | ---------------------------------------------------------------- | ----------------------------- |
| `core-api`       | Authenticated treasury API + anonymous public read tier + `/metrics` | stateless                     |
| `web`            | Dashboard and public portal (static SPA behind nginx)            | stateless                     |
| `ledger-gateway` | Anchors posted entries (Merkle roots) into the traceability ledger | stateless worker              |
| `mcp-server`     | AI-agent access over the public tier only                        | stateless                     |
| `connector`      | Watches a drop directory, maps files, delivers to staging        | stateless worker + drop volume |
| PostgreSQL       | **System of record** (double-entry ledger, audit, staging, proofs) | external — you provide it     |
| OIDC provider    | Identity (Keycloak or any OIDC-compliant IdP)                    | external — you provide it     |
| Fabric network   | Optional traceability backend (ADR-0004)                         | external — multi-org          |

Two hard rules shape everything below:

1. **PostgreSQL is the operational truth.** Its durability, backups, and
   access control are the deployment's most important properties.
2. **Nothing secret or personal leaves the trust boundary.** The public tier,
   the MCP server, and the anchoring chain only ever see the published,
   redacted projection (journal descriptions, for example, are never
   published).

## 2. Prerequisites

- Kubernetes 1.28+ with an ingress controller and cert-manager (or an
  equivalent TLS story).
- PostgreSQL 16+, managed or operator-run (e.g. CloudNativePG), with WAL
  archiving / PITR enabled.
- An OIDC provider. For Keycloak, import a realm with the four roles
  (`treasury-admin`, `auditor`, `institution-user`, `connector`) and the
  `institution_id` claim mapper — `infra/keycloak/opentreasury-realm.json` is
  the working reference.
- Helm 3.14+.

## 3. Install

Container images are published per release as
`ghcr.io/theetaz/opentreasury/<component>:<version>`. Mirror them into your
own registry if your environment requires it (set `image.registry`).

### 3.1 Database

Create the database and a least-privilege application role, then run the
migrations from `database/migrations` with a `migrate/migrate` Job (the chart
deliberately does not run them — schema changes should be an explicit,
audited step):

```sh
docker run --rm -v $PWD/database/migrations:/migrations migrate/migrate:v4 \
  -path=/migrations -database "$DSN" up
```

Seed the reference chart of accounts from `database/seeds` and adapt it in
the mapping workshop (see the [onboarding playbook](onboarding-playbook.md)).

### 3.2 Secrets

```sh
kubectl create secret generic opentreasury-database --from-literal=dsn='postgres://…'
```

### 3.3 Helm

```yaml
# values-production.yaml
database:
  existingSecret: opentreasury-database
oidc:
  issuerUrl: https://id.example.gov/realms/opentreasury
  audience: opentreasury-web
coreApi:
  allowedOrigins: https://treasury.example.gov
ingress:
  enabled: true
  className: nginx
  webHost: treasury.example.gov
  apiHost: api.treasury.example.gov
  tls:
    - hosts: [treasury.example.gov, api.treasury.example.gov]
      secretName: opentreasury-tls
```

```sh
helm install opentreasury infra/helm/opentreasury -f values-production.yaml
```

The chart refuses nothing but warns loudly: with `oidc.issuerUrl` unset the
API runs unauthenticated, which is acceptable only on throwaway clusters.

## 4. The traceability ledger

The anchoring backend is pluggable (ADR-0004):

- **Transparency log** (default): zero extra infrastructure; anchors live in
  a hash-chained Postgres table. Right for pilots. Independent verification
  (proof endpoint, standalone verifier, in-browser check) works identically.
- **Hyperledger Fabric**: set `OPENTREASURY_ANCHOR_BACKEND=fabric` on the
  ledger gateway plus the `OPENTREASURY_FABRIC_*` variables, mounting the
  gateway's enrollment MSP and the peer TLS CA. `infra/fabric/` is the
  working single-org reference; a production network is **multi-org** — the
  point of the chain is that the finance ministry, the supreme audit
  institution, and ideally a civil-society observer each run a peer, so no
  single party can rewrite the anchor history.

Anchoring can be switched later without breaking existing proofs: old
anchors keep their backend reference; new batches commit to the new backend.

## 5. Day-2 operations

- **Observability**: `core-api` (`:8080/metrics`) and `ledger-gateway`
  (`:9464/metrics`) expose Prometheus metrics with `prometheus.io/*` pod
  annotations. Import the dashboard from `infra/docker/grafana/dashboards/`.
  Alert on: 5xx rate, p95 latency, `opentreasury_anchor_failures_total > 0`,
  and anchor lag (entries posted but unanchored).
- **Backups**: continuous WAL archiving + a tested restore procedure. The
  proofs and transparency log live in the same database — a restore that
  loses committed anchors is detectable (that is the point) but operationally
  painful; treat RPO seriously.
- **Upgrades**: `helm upgrade` with the new image tag; migrations run first,
  as their own Job. All services are stateless and roll without downtime.
- **Key rotation**: OIDC keys rotate in the IdP (the API discovers them);
  Fabric enrollment certificates rotate by re-enrolling the gateway identity
  and remounting.

## 6. Hardening checklist

- [ ] Database role for the API has no DDL rights; migrations use a separate role.
- [ ] `/metrics` is not exposed through the ingress (cluster-internal scrape only).
- [ ] Public tier rate limits reviewed for your expected traffic.
- [ ] NetworkPolicies restrict `postgres` access to `core-api` and `ledger-gateway`.
- [ ] The connector's client credentials grant the `connector` role only
      (staging writes; it can never post directly to the journal).
- [ ] Audit events and anchor verification checked end-to-end after install
      (post a test entry, run the standalone verifier against production).
- [ ] Realm/IdP admin console is not internet-exposed.
