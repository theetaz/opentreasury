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

## Trying it on a real cluster (kind)

The `examples/` directory holds everything needed to replicate a
production-shaped deployment on a laptop:

```sh
kind create cluster --name opentreasury

# Local images (or let the cluster pull the signed release images)
kind load docker-image --name opentreasury \
  ghcr.io/theetaz/opentreasury/core-api:0.1.0 \
  ghcr.io/theetaz/opentreasury/web:0.1.0 \
  ghcr.io/theetaz/opentreasury/ledger-gateway:0.1.0 \
  ghcr.io/theetaz/opentreasury/mcp-server:0.1.0

# Dev database + migrations (production uses a managed DB and an audited
# migration step instead)
kubectl apply -f infra/helm/examples/kind-postgres.yaml
kubectl create configmap migrations --from-file=database/migrations/
kubectl apply -f infra/helm/examples/kind-migrate-job.yaml
kubectl wait --for=condition=complete job/opentreasury-migrate

helm install opentreasury infra/helm/opentreasury \
  --set database.existingSecret=opentreasury-database

kubectl port-forward svc/opentreasury-core-api 18080:8080 &
curl http://localhost:18080/readyz   # {"status":"ready"}
```
