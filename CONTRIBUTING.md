# Contributing to OpenTreasury

Thank you for helping build open, verifiable public-finance infrastructure. This guide covers everything you need to land your first change.

## Ground Rules

- Be respectful. We follow the [Code of Conduct](CODE_OF_CONDUCT.md).
- All contributions are licensed under [Apache-2.0](LICENSE). We use the [Developer Certificate of Origin](https://developercertificate.org/) — sign your commits with `git commit -s`.
- Security issues go through [SECURITY.md](SECURITY.md), never public issues.

## Development Setup

Prerequisites: Go 1.22+, Node 20+, pnpm 10+, Docker with Compose.

```bash
git clone https://github.com/theetaz/opentreasury.git
cd opentreasury
pnpm install
make verify          # run the full test suite
```

See [docs/development.md](docs/development.md) for the Docker-based local environment.

## How We Work

### Test-driven development

We write tests first. A PR that adds behavior without tests that fail before the change and pass after it will be asked to add them. Financial-core logic (journal balancing, balances, lifecycle) additionally requires invariant/property tests.

### Branching & PRs

- `develop` is the integration branch. `main` is reserved for releases.
- Every change lands via a feature branch → PR into `develop`. Branch names: `feature/<area>-<slug>`, `bugfix/<slug>`, `chore/<slug>`.
- Commits follow [Conventional Commits](https://www.conventionalcommits.org/): `feat(core-api): …`, `fix(web): …`, `docs: …`, `chore: …`.
- Keep PRs focused. One epic-sized change is fine; unrelated drive-by fixes are not.

### PR checklist (enforced in review)

- [ ] Tests written first and passing (`make verify`)
- [ ] OpenAPI / event schemas / docs updated in the same PR (doc drift blocks merge)
- [ ] No new endpoint without an authn decision, rate-limit class, audit coverage, and pagination
- [ ] ADR added for any decision that constrains future contributors (see `docs/adr/`)
- [ ] No secrets, credentials, or personal data in code, tests, or fixtures

### Architecture Decision Records

Significant decisions are recorded as ADRs in [docs/adr/](docs/adr/). Propose one by PR; discussion happens on the PR. Start from `docs/adr/template.md`.

## Where to Contribute

The roadmap lives in [docs/opentreasury-blueprint.md](docs/opentreasury-blueprint.md). High-leverage areas for community contributors:

- **Connectors** — self-contained integrations with government finance systems, verified by the conformance suite (Phase 3+)
- **Mapping profiles & reference charts of accounts** — YAML-only, no Go required; country-specific domain knowledge is the scarce resource
- **Translations** — the public portal must speak each deployment country's languages
- **Dashboards & explorer UX** — React + shadcn/ui
- **OPA policies** — authorization and redaction rules, fully testable

Issues labeled `good-first-issue` are curated and scoped for newcomers; `community-friendly` marks bounded epics.

## Getting Help

Open a GitHub Discussion for questions, an Issue for bugs/features. Maintainers triage weekly.
