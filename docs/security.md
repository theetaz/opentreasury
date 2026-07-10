# Security

Security work for OpenTreasury starts with explicit ownership of identity, authorization, auditability, and supply-chain controls.

## Baseline

- Keycloak handles identity.
- Open Policy Agent handles authorization and redaction policy decisions.
- PostgreSQL stores operational treasury data.
- Hyperledger Fabric stores public-safe traceability records.
- CI should include dependency, vulnerability, and secret scanning before public release.

## Data Handling

- Do not commit credentials, tokens, certificates, generated local state, or private datasets.
- Keep public export and redaction behavior documented and testable.
- Prefer least-privilege service credentials and short-lived local secrets.
