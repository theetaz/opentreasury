# OpenTreasury

OpenTreasury is an open-source public-finance transparency and treasury-management platform.

It connects existing government finance systems, normalizes transactions into a standard treasury data model, records public traceability events on Hyperledger Fabric, and exposes dashboards, APIs, mobile views, and integration tools for institutions, auditors, and the public.

## Project Goals

- Provide an open-source foundation for public treasury transparency.
- Normalize treasury transactions across different government finance systems.
- Keep PostgreSQL as the operational system of record.
- Use Hyperledger Fabric as a tamper-evident public traceability layer.
- Support web, mobile, backend services, workers, chaincode, policies, and deployment assets in one repository.
- Keep identity, authorization, redaction, auditability, and observability explicit from the start.

## Core Architecture

```text
External Government Systems
  -> Connector Services
  -> Event Topics
  -> Core API and Workers
  -> PostgreSQL
  -> Ledger Gateway
  -> Hyperledger Fabric
  -> Public Explorer, Dashboards, Mobile App, and APIs
```

OpenTreasury separates operational truth from public traceability:

- PostgreSQL stores canonical treasury data, reconciliation state, reporting views, and audit metadata.
- Hyperledger Fabric records public-safe finalized events, reversal events, correction events, document hashes, and balance proofs.
- The core API validates treasury lifecycle rules before data is persisted or published.
- Connector services isolate source-system mapping logic.
- Policy modules govern authorization, publication, export, and redaction decisions.

## Technology Stack

- Web: React, Vite, TypeScript, Tailwind CSS, and shadcn/ui.
- Mobile: Expo, React Native, and TypeScript.
- Backend: Go services and workers.
- Database: PostgreSQL.
- Ledger: Hyperledger Fabric.
- Policy: Open Policy Agent.
- Identity: Keycloak.
- Local development: Docker Compose, pnpm workspaces, Go workspaces, and Make.
- Production deployment: Kubernetes, Helm, and Kustomize.

## Repository Layout

- `apps/web`: React and Vite web frontend.
- `apps/mobile`: Expo React Native mobile app.
- `services`: Go backend services.
- `workers`: Go background workers.
- `chaincode`: Hyperledger Fabric chaincode.
- `packages`: Shared TypeScript packages.
- `database`: PostgreSQL migrations and seeds.
- `policies`: Open Policy Agent policies.
- `infra`: Local and production infrastructure.

## Current Status

OpenTreasury is in early project scaffolding. The current repository provides the initial monorepo layout, starter package metadata, Go module boundaries, and verification commands.

## Development

Prerequisites:

- Go 1.25 or newer.
- Node.js 20 or newer.
- pnpm 9 or newer.
- Docker and Docker Compose.

Install JavaScript dependencies:

```sh
pnpm install
```

Run repository verification:

```sh
make verify
```

The initial verification runs Go tests for scaffolded Go modules. JavaScript checks run after dependencies are installed.

## Planned Service Boundaries

- `services/core-api`: treasury rules, transaction lifecycle, public/private API contracts, and persistence boundaries.
- `services/connectors/generic-file`: initial file-based ingestion connector.
- `services/ledger-gateway`: ledger publication gateway for Hyperledger Fabric.
- `services/mcp-server`: integration server for read-only analytical and operational tooling.
- `workers/treasury-worker`: durable imports, reconciliation, ledger submission, and background workflows.
- `chaincode/treasury`: public-safe traceability records for Hyperledger Fabric.

## Security Principles

- Do not commit credentials, certificates, private keys, local environment files, or generated infrastructure state.
- Keep authorization and redaction behavior policy-driven and testable.
- Treat public export decisions as security-sensitive.
- Prefer least-privilege service credentials.
- Include dependency, vulnerability, and secret scanning in CI before production use.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow (test-driven, feature branches into `develop`), and [GOVERNANCE.md](GOVERNANCE.md) for how decisions are made. Community standards are set by the [Code of Conduct](CODE_OF_CONDUCT.md).

Guiding expectations:

- Keep changes focused and reviewable, with tests written first.
- Document public APIs with OpenAPI in the same change.
- Avoid proprietary hosted-only dependencies unless there is a clear fallback and documented rationale.
- Record decisions that constrain future contributors as ADRs in [docs/adr/](docs/adr/).

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md). Never open public issues for security problems.

## Roadmap

The product blueprint, current-state audit, and phased implementation plan live in [docs/opentreasury-blueprint.md](docs/opentreasury-blueprint.md).

## License

OpenTreasury is licensed under the Apache License 2.0.
