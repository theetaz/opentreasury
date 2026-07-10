# OpenTreasury Helm Chart

Deploys the OpenTreasury services to Kubernetes:

| Component        | Kind                 | Default  |
| ---------------- | -------------------- | -------- |
| `core-api`       | Deployment + Service | enabled  |
| `web`            | Deployment + Service | enabled  |
| `mcp-server`     | Deployment + Service | enabled  |
| `ledger-gateway` | Deployment (worker)  | enabled  |
| `connector`      | Deployment (worker)  | disabled |

PostgreSQL (system of record) and an OIDC provider (e.g. Keycloak) are
**external dependencies** — use a managed database / identity service or
operators, and point the chart at them via values. The chart never ships
credentials; supply them through existing Secrets.

## Quick start (development cluster)

```sh
helm install opentreasury infra/helm/opentreasury \
  --set database.dsn='postgres://user:pass@my-postgres:5432/opentreasury?sslmode=require' \
  --set oidc.issuerUrl='https://keycloak.example.org/realms/opentreasury' \
  --set coreApi.allowedOrigins='https://opentreasury.example.org'
```

Run the database migrations against the target database before first use
(`make db-migrate` locally, or a `migrate/migrate` Job in-cluster); the chart
does not run them automatically.

## Production values

```yaml
database:
  existingSecret: opentreasury-database   # key: dsn
oidc:
  issuerUrl: https://id.example.org/realms/opentreasury
  audience: opentreasury-web
coreApi:
  allowedOrigins: https://opentreasury.example.org
ingress:
  enabled: true
  className: nginx
  webHost: opentreasury.example.org
  apiHost: api.opentreasury.example.org
  tls:
    - hosts: [opentreasury.example.org, api.opentreasury.example.org]
      secretName: opentreasury-tls
```

## Observability

`core-api` (`:8080/metrics`) and `ledger-gateway` (`:9464/metrics`) expose
Prometheus metrics and carry `prometheus.io/*` scrape annotations (disable
with `metrics.annotations=false` if you use ServiceMonitors instead). The
Grafana dashboard in `infra/docker/grafana/dashboards/` works unchanged
against these metrics.

## Validation

```sh
helm lint infra/helm/opentreasury
helm template opentreasury infra/helm/opentreasury
```
