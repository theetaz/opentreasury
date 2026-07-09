# Development

This document tracks local development setup for OpenTreasury.

## Prerequisites

- Go 1.25 or newer.
- Node.js 20 or newer.
- pnpm 10 or newer.
- Docker with Compose (used for the local database, migrations, and integration tests).

## Initial Commands

```sh
pnpm install
make verify
```

`make verify` runs the Go unit tests for every module, the core API integration
suite (requires Docker — it starts throwaway Postgres containers via
testcontainers), and the web app tests + typechecks.

## Full Local Stack (one command)

```sh
make up
```

Builds and starts everything in Docker: Postgres, migrations, seed data,
Keycloak (identity), the core API, and the web app.

- Dashboard (UAT): **http://localhost:5173**
- Core API: **http://localhost:8080** (`/healthz`, `/v1/...`)
- Keycloak: **http://localhost:8085** (admin `admin`/`admin`)
- MCP server: **http://localhost:8090** (streamable HTTP; tools over the public tier)

`make down` stops the stack; `make db-reset` wipes the data volume and
re-seeds.

## Authentication & Authorization

The stack runs with authentication enabled. Sign in at the dashboard with one
of the seeded demo users (realm `opentreasury`):

| User | Password | Role | Scope |
|---|---|---|---|
| `treasury-admin` | `treasury` | treasury-admin | all institutions, read + write |
| `institution-user` | `institution` | institution-user | own institution (`minfin`) only |
| `auditor` | `auditor` | auditor | all institutions, read-only |

The core API verifies Keycloak-issued bearer tokens (OIDC) and authorizes each
request against the Rego policy in `policies/opa/authz.rego` (embedded via the
OPA Go SDK). Institution users are scoped to their own institution; the chart
of accounts is shared reference data readable by all.

Authentication is opt-in by configuration: with `OPENTREASURY_OIDC_ISSUER_URL`
(API) and `VITE_OIDC_AUTHORITY` (web) unset, both run open for quick local work.

## Local Database

PostgreSQL runs in Docker via `infra/docker/docker-compose.yaml`:

```sh
make db-up            # start Postgres 16 on localhost:5433 (opentreasury/opentreasury)
make db-migrate       # apply database/migrations with golang-migrate
make db-seed          # apply demo seed data from database/seeds (idempotent)
make db-migrate-down  # roll back all migrations
make db-reset         # destroy the data volume and rebuild: up + migrate + seed
make db-down          # stop the containers
```

Connection string for local services:

```
postgres://opentreasury:opentreasury@localhost:5433/opentreasury_dev?sslmode=disable
```

Migrations follow the [golang-migrate](https://github.com/golang-migrate/migrate)
`NNNNNN_name.up.sql` / `NNNNNN_name.down.sql` convention. Every migration must
have a down pair (enforced by unit test), and the whole set is executed
against real Postgres in the integration suite
(`services/core-api/test/integration/migrations_test.go`).

## Running the Core API Locally

```sh
make db-seed
cd services/core-api
OPENTREASURY_DATABASE_DSN="postgres://opentreasury:opentreasury@localhost:5433/opentreasury_dev?sslmode=disable" \
OPENTREASURY_ALLOWED_ORIGINS="http://localhost:5173" \
go run ./cmd/core-api
```

## Running the Web App Locally

```sh
VITE_CORE_API_URL="http://localhost:8080" pnpm dev:web
```

Without `VITE_CORE_API_URL` the web app serves built-in simulated data.
