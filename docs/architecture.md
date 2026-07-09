# Architecture

OpenTreasury uses PostgreSQL as the operational system of record and Hyperledger Fabric as the tamper-evident traceability layer.

## Main Flow

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

## Boundary Decisions

- The core API owns treasury validation rules.
- Connectors isolate source-system mapping logic.
- PostgreSQL owns operational truth.
- Hyperledger Fabric records public-safe traceability events.
- Open Policy Agent owns authorization, publication, export, and redaction decisions.
- Keycloak owns identity.
