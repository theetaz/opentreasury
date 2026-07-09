## Summary

<!-- What does this PR do, and which blueprint epic / issue does it advance? -->

## Test-first evidence

<!-- TDD is how we work: which tests were written first? What did they fail with before the change? -->

## Checklist

- [ ] Tests written first and passing (`make verify`)
- [ ] OpenAPI / event schemas / docs updated in this PR (no drift)
- [ ] New endpoints declare: authn decision, rate-limit class, audit coverage, pagination
- [ ] ADR added/updated if this constrains future contributors
- [ ] No secrets, credentials, or personal data anywhere in the diff
- [ ] Commits are signed off (DCO, `git commit -s`)
