# Contributing

Thanks for helping with Vouch.

## First Principle

Vouch is in beta/research-prototype mode.

It is not a code reviewer. Please do not frame contributions as "the tool reviews code and says it is safe."

The project is a compiler-like verification control plane:

1. Intent
2. Obligations
3. Evidence
4. Policy
5. Release decision

Good contributions make that pipeline more explicit, deterministic, testable, and auditable.

## How To Start

Run the tests:

```sh
GOCACHE=/private/tmp/vouch-gocache go test ./...
```

Install the CLI locally:

```sh
go install ./cmd/vouch
```

If `vouch` is not on your `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Run the demo pipeline:

```sh
vouch intent parse --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.ast.json
vouch intent compile --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.json
vouch ir build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/auth.password_reset.ir.json
vouch plan build --spec demo_repo/.vouch/specs/auth.password_reset.json --manifest demo_repo/.vouch/manifests/pass.json --out /tmp/auth.password_reset.plan.json
vouch artifacts build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/vouch-artifacts
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/blocked.json evidence
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/pass.json evidence
```

Expected demo outcomes:

| Manifest | Decision |
| --- | --- |
| `blocked.json` | `block` |
| `pass.json` | `canary` |

## Where Help Is Needed

### Compiler Front End

Useful work:

- Real YAML parser with source positions.
- Better AST validation.
- Clearer diagnostics.
- Golden diagnostic fixtures.
- Strict unknown-field behavior.
- Schema version migration.

### Schemas And Contracts

Useful work:

- JSON schemas for all artifacts.
- Schema compatibility tests.
- Example artifact bundles.
- Strict fixture validation.
- Documentation for every field.

### IR And Verification Planning

Useful work:

- Better obligation IDs.
- More obligation kinds.
- Traceability from spec fields to obligations.
- Verification-plan validation.
- Verifier-role assignment rules.
- Release-policy artifact improvements.

### Policy Engine

Useful work:

- Move hard-coded gate rules into policy files.
- Add policy simulation.
- Add risk-specific policy profiles.
- Add exception handling.
- Add policy regression tests.

### Evidence Importers

Useful work:

- Coverage report importers.
- Test result importers.
- Static-analysis importers.
- Secret-scanner importers.
- Runtime metric importers.
- Changed-file to spec traceability.

### Evidence Verifier Infrastructure

Useful work:

- Verifier packet schemas.
- Structured verifier output validation.
- Verifier disagreement handling.
- Prompt and model version pinning.
- Verifier audit logs.
- Fixtures for incomplete, misleading, or malicious evidence.

Important: AI verifiers should verify evidence against obligations. They should not be presented as generic code reviewers.

### Workflow And Developer Experience

Useful work:

- GitHub Actions examples.
- GitHub Checks output.
- SARIF or annotation output.
- Machine-readable gate result files.
- Better CLI help.
- Installation docs.

### Runtime And Rollback

Useful work:

- Canary metric binding.
- Alert validation.
- Deployment integration examples.
- Rollback hook examples.
- Post-release evidence packets.
- Incident feedback loops.

### Documentation And Examples

Useful work:

- More demo repos.
- More risk categories.
- Example specs for migrations, payments, auth, privacy, and data deletion.
- Clear diagrams of the pipeline.
- Operator docs.
- Threat model docs.

## Contribution Standards

Please keep changes narrow and testable.

For code changes:

- Add or update tests for changed behavior.
- Keep generated outputs deterministic.
- Prefer structured data over ad hoc string parsing.
- Preserve source-span diagnostics where possible.
- Avoid unrelated refactors.

For docs changes:

- Be explicit that this is beta.
- Be explicit that this is not a code reviewer.
- Avoid overclaiming correctness.
- Explain what is implemented versus planned.
- Include runnable commands when helpful.

## Good First Contributions

Good first areas:

- Add a new demo scenario.
- Add a schema draft for one artifact.
- Add golden tests for invalid intent files.
- Improve CLI usage text.
- Document artifact fields.
- Add a runner workflow example.

## What To Avoid

Avoid contributions that:

- Claim the tool proves code is correct.
- Turn the project into a generic coding agent.
- Hide uncertainty instead of reporting it.
- Depend on nondeterministic output without tests.
- Make verifiers inspect huge diffs without obligation context.
- Weaken the beta/prototype warnings.

The project should be ambitious about infrastructure and conservative about claims.

## License

By contributing to Vouch, you agree that your contributions are licensed under the Apache License, Version 2.0.
