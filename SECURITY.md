# Security Policy

OpenTreasury handles public-finance data for governments. Security reports are treated with the highest priority.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Report privately via [GitHub Security Advisories](https://github.com/theetaz/opentreasury/security/advisories/new) ("Report a vulnerability"). Include reproduction steps, affected components, and impact assessment if you can.

You will receive an acknowledgment within **48 hours** and a triage decision within **7 days**.

## Coordinated Disclosure

- We follow a 90-day coordinated disclosure window, negotiable for complex fixes.
- Fix SLAs by severity: **Critical ≤ 48 h** (patch release), High ≤ 7 d, Medium ≤ 30 d, Low ≤ 90 d.
- Advisories are published as GitHub Security Advisories with CVE where applicable; reporters are credited unless they prefer anonymity.
- Because governments run OpenTreasury in production, advisories are also announced on the repository's Releases feed so deployment operators can subscribe.

## Scope

In scope: everything in this repository — services, web/mobile apps, chaincode, OPA policies, infrastructure manifests, CI workflows, and the published container images.

Of special interest (highest-impact areas):

- Redaction/classification bypasses — any way to read non-PUBLIC data through public surfaces
- Authorization bypasses across institution boundaries
- Integrity attacks — any way to mutate posted financial records or forge anchoring proofs
- Injection through connector inputs (malicious source files/records)

Out of scope: vulnerabilities requiring physical access, social engineering of maintainers, and denial-of-service findings without a specific amplification defect.

## Security Design Baseline

See [docs/security.md](docs/security.md) and the threat model in [docs/opentreasury-blueprint.md](docs/opentreasury-blueprint.md) (Section 9). Key invariants any report can measure against:

1. No secrets/credentials/certs are ever committed to the repository.
2. Public consumers can only reach PUBLIC-classified projections, enforced server-side.
3. Financial records are append-only after posting; corrections are new linked entries.
4. Every mutation carries an audit event with actor and request identity.
