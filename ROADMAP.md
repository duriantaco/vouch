# Roadmap

## Status

Vouch is in beta/research-prototype mode.

It is not a code reviewer. It does not inspect implementation diffs and decide whether the code is good. The roadmap is about building a compiler-like verification control plane: intent becomes obligations, obligations require evidence, and policy decides whether a change can ship.

## North Star

The long-term goal is a serious replacement surface for line-by-line review when agents produce more code than humans can inspect:

1. Human-owned intent
2. Typed compiler pipeline
3. Generated verification artifacts
4. Artifact-backed verifiers and deterministic checks
5. Runtime enforcement
6. Auditable release decision

## Positioning Guardrails

Vouch should not be positioned as another CI/CD gate.

CI can execute Vouch, but the product is the compiler-like control plane:

- Contract language.
- Typed AST and diagnostics.
- Obligation IR.
- Evidence manifest.
- Artifact-backed verifier inputs.
- Deterministic policy decision.
- Auditable release result.

If a feature only makes Vouch a nicer wrapper around existing test commands, it should not be a roadmap priority unless it also strengthens obligation coverage, evidence quality, policy semantics, or auditability.

## Current Beta

Implemented today:

- Intent YAML parser for a small supported subset.
- Typed AST with source spans and diagnostics.
- Intent-to-spec compilation.
- Spec-to-obligation IR lowering.
- Stable semantic obligation IDs.
- Verification-plan generation.
- Verifier/test/release artifact generation.
- Evidence artifact reference resolution from change manifests.
- Artifact path/hash verification.
- JUnit XML importer for test evidence.
- Deterministic verifier findings.
- Touched-spec compilation for faster PR checks.
- Generic repo init for Python, Node, Go, Rust, and fallback repos.
- Contract suggestion and creation commands.
- Manifest creation from changed files and owned paths.
- Artifact attachment with obligation inference.
- JUnit test-map adapter for raw pytest-style JUnit evidence.
- Release decisions: `block`, `human_escalation`, `canary`, `auto_merge`.
- Demo repo with blocked and passing manifests.
- Unit tests for the current pipeline.

## Validation Status

Recent validation runs used temp copies of real repos, so Vouch exercised the generic path without mutating the source projects:

| Repo | Shape | Evidence Path | Decision |
| --- | --- | --- | --- |
| `sundae` | Flat Python package | pytest/JUnit evidence | `auto_merge` |
| `sago` | Python `src/` layout | high-risk builder contract | `canary` |
| `wooster` | Flat Python package | pytest JUnit mapped through `.vouch/test-map.json` | `auto_merge` |

These runs show the current beta can initialize unfamiliar repos, create contracts, create manifests from changed files, map raw test output to required-test obligations, attach evidence, and produce deterministic release decisions.

It does not yet prove that Vouch understands arbitrary product intent. The contract remains the source of truth, and the next product work should reduce the manual work needed to create and maintain those contracts.

## Roadmap Phases

### Phase 1: Compiler Front End Hardening

Make the source language and diagnostics reliable enough for real users.

Planned work:

- Replace the minimal YAML subset parser with a real source-position-preserving parser.
- Publish JSON schemas for AST, spec, IR, plan, manifest, and evidence.
- Add schema compatibility tests.
- Add golden diagnostic fixtures.
- Define strict unknown-field behavior.
- Add version migration hooks.
- Improve error messages for nested sections.

### Phase 2: Policy Engine

Separate release policy from hard-coded Go logic.

Planned work:

- Policy-as-code files.
- Risk-specific policy profiles.
- Team-specific override rules.
- Exception handling with audit trails.
- Policy simulation command.
- Policy regression tests.

### Phase 3: Workflow Integration

Make Vouch useful in real pull-request workflows while keeping CI as the runner, not the product identity.

Planned work:

- GitHub Checks integration.
- Machine-readable gate result artifact.
- SARIF or annotation output for diagnostics.
- Artifact upload conventions.
- Required status check examples.
- Sample runner workflow.

### Phase 4: Evidence Verifiers

Turn generated verifier packets into first-class verifier inputs.

Planned work:

- Verifier input packet schema.
- Structured verifier output schema.
- Prompt and model version pinning.
- Verifier confidence and disagreement handling.
- Verifier isolation rules.
- Audit log for every verifier decision.
- Test fixtures for malicious or incomplete evidence.

Important constraint: AI verifiers should verify evidence against obligations. They should not be presented as a generic code-review replacement.

### Phase 5: Code-Aware Evidence Importers

Connect the obligation system to real code, tests, and tooling.

Implemented base:

- Changed-file ownership checks.
- Manifest creation from changed files and owned paths.
- JUnit XML import.
- Test-map adapter for raw pytest-style JUnit output.

Planned work:

- Spec-to-file traceability beyond owned path globs.
- Test discovery and suggested test-map generation.
- Coverage report import.
- Static analysis import.
- SARIF import.
- Secret scanning import.
- Logging and PII scanner import.
- Migration and external-effect detectors.

### Phase 6: Runtime Enforcement

Make release policy continue after merge.

Planned work:

- Deployment integration.
- Canary metric binding.
- Alert binding validation.
- Automatic rollback hooks.
- Post-release evidence packets.
- Incident feedback into specs.
- Production drift detection.

### Phase 7: Trust, Governance, And Scale

Make the system usable by teams.

Planned work:

- Signed specs and manifests.
- Agent identity and run provenance.
- Tamper-evident evidence bundles.
- Role-based approval exceptions.
- Organization-level policy packs.
- Multi-repo spec registry.
- Long-term audit storage.

## Near-Term Priorities

The next useful contributions are:

- Reference workflow for `init -> manifest -> pytest -> junit map -> attach -> gate`.
- Machine-readable gate result file for required status checks.
- Test-map discovery to reduce manual required-test mapping.
- Coverage XML importer for required-test and behavior evidence.
- Static-analysis/SARIF importer for security and quality evidence.
- JSON schemas plus compatibility tests for AST, spec, IR, plan, manifest, and evidence.
- Golden diagnostics for parser and compiler failures.
- Policy-as-code design and simulation command.
- Signed or hashed evidence bundle format.
- More demo scenarios beyond password reset, including ordinary library changes.

## Productionization Track

Before calling this production-ready, Vouch needs:

- Stable CLI contract and installable release binary.
- Sample runner workflow with artifact upload conventions.
- Documented evidence kinds and required fields.
- Schema version compatibility tests.
- Policy files instead of hard-coded release decisions.
- Auditable gate result output for GitHub Checks or similar systems.
- Tamper-evident evidence bundles with agent/run provenance.
- Fixture repos that cover Python flat layout, Python `src/` layout, Node, Go, Rust, and generic fallback.

## Explicit Non-Goals

Vouch should not claim to:

- Prove arbitrary code correct.
- Replace production security review.
- Replace incident response.
- Read any diff and declare it good.
- Act as a general-purpose coding agent.

The project is valuable only if it stays honest about that boundary.
