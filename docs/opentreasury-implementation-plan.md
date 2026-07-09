# OpenTreasury Open-Source Implementation Plan

Date: 2026-06-28

## 1. Project Identity

**Project name:** OpenTreasury

**Repository name:** `opentreasury`

**Product description:** OpenTreasury is an open-source public-finance transparency and treasury-management platform. It connects existing government finance systems, normalizes transactions into a standard treasury data model, records public traceability events on Hyperledger Fabric, and exposes dashboards, APIs, mobile views, and AI-ready MCP tools.

**Core principle:** PostgreSQL is the operational system of record. Hyperledger Fabric is the tamper-evident public audit and traceability layer.

## 2. Goals and Non-Goals

### Goals

- Build a mono-repo project using only open-source technologies.
- Support web, mobile, backend services, blockchain chaincode, infrastructure, and documentation in one repo.
- Use React + Vite + shadcn/ui for the web frontend.
- Use React Native + Expo for the mobile app.
- Use Go for backend services, workers, connectors, MCP, and Hyperledger Fabric chaincode.
- Use PostgreSQL for operational treasury data.
- Use Hyperledger Fabric for permissioned public-finance audit records.
- Use Keycloak for identity and access management.
- Use Open Policy Agent for treasury-specific authorization and redaction rules.
- Use Docker Compose for local development.
- Use Kubernetes for production deployment.
- Provide clear folder structure, endpoint design, naming conventions, coding standards, and implementation phases.

### Non-Goals for the First Release

- No WSO2 technologies.
- No proprietary cloud-only dependency.
- No public cryptocurrency chain.
- No direct AI-driven financial execution.
- No attempt to integrate every government system in the first release.
- No full mobile feature parity in MVP; mobile starts with viewing, alerts, and traceability.

## 3. Recommended Open-Source Stack

| Layer                   | Technology                                                | Why                                                                     |
| ----------------------- | --------------------------------------------------------- | ----------------------------------------------------------------------- |
| Mono-repo orchestration | pnpm workspaces, Go workspaces, Makefile                  | Simple, open-source, works across JS and Go                             |
| Web app                 | React, Vite, TypeScript, shadcn/ui, Tailwind CSS          | Fast frontend, strong component model, practical admin dashboard UI     |
| Mobile app              | React Native, Expo, TypeScript                            | Shared TypeScript types and fast mobile delivery                        |
| Backend                 | Go                                                        | Strong concurrency, static binaries, good service ergonomics            |
| API transport           | REST + OpenAPI first; Server-Sent Events for live updates | Clear contracts, easy integration, good public API fit                  |
| Internal async events   | Kafka-compatible Redpanda or Apache Kafka                 | Durable ingestion and event-driven workflows                            |
| Workflows               | Temporal                                                  | Durable imports, retries, approvals, ledger submission, reconciliation  |
| Database                | PostgreSQL                                                | Canonical treasury data, reporting, reconciliation, audit metadata      |
| Cache/rate state        | Redis                                                     | Sessions, rate limits, short-lived query cache, background coordination |
| Search                  | OpenSearch, optional after MVP                            | Public explorer and dashboard search at scale                           |
| Object storage          | MinIO locally, S3-compatible storage in production        | Supporting documents, signed exports, attachments                       |
| Identity                | Keycloak                                                  | OIDC/SAML, SSO, roles, groups, federation                               |
| Authorization policy    | Open Policy Agent                                         | Public/private/redaction policy as code                                 |
| Blockchain              | Hyperledger Fabric                                        | Permissioned open-source blockchain for known institutions              |
| API gateway             | Kong Gateway OSS or Envoy Gateway                         | Routing, rate limits, public/private API entry point                    |
| Observability           | OpenTelemetry, Prometheus, Grafana, Loki, Tempo           | Metrics, logs, traces, dashboards                                       |
| Containerization        | Docker, Docker Compose                                    | Local reproducible development                                          |
| Kubernetes              | Kubernetes, Helm, Kustomize                               | Production deployment                                                   |
| CI/CD                   | GitHub Actions or GitLab CI                               | Tests, builds, scans, deployments                                       |
| Security scanning       | Trivy, Gitleaks, govulncheck, npm audit/pnpm audit        | Open-source supply-chain checks                                         |

## 4. High-Level Architecture

```text
External Government Systems
  -> Go Connector Services
  -> Redpanda/Kafka Topics
  -> OpenTreasury Core API + Workers
  -> PostgreSQL Operational Database
  -> Temporal Workflows
  -> Hyperledger Fabric Ledger Gateway
  -> Hyperledger Fabric Network
  -> Public Explorer / Treasury Dashboard / Institution Portal / Mobile App / MCP Server
```

### Main Boundary Decisions

- **Core API owns treasury rules.** It validates chart of accounts, institution access, transaction lifecycle, and data integrity.
- **Connectors are isolated.** Each source system has its own connector service and mapping logic.
- **PostgreSQL owns operational truth.** Dashboards, reconciliation, and reporting should query PostgreSQL or read models derived from PostgreSQL.
- **Fabric owns traceability.** Fabric records public-safe finalized events, reversal events, correction events, document hashes, and balance proofs.
- **OPA owns policy decisions.** The backend asks OPA whether a user/service can read a field, see a transaction, publish a public event, or export data.
- **Keycloak owns identity.** Keycloak authenticates users and issues JWTs with roles, groups, and institution claims.

## 5. Mono-Repo Structure

```text
opentreasury/
  README.md
  LICENSE
  Makefile
  Taskfile.yml
  pnpm-workspace.yaml
  package.json
  go.work
  .editorconfig
```
