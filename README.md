<p align="center">
  <img src="assets/vouch.png" alt="Vouch logo" width="240">
</p>

# Vouch

Vouch is a contract-and-evidence release gate for AI-written code.

It does one narrow job: given human-owned intent for a part of a repo, Vouch compiles that intent into machine-checkable obligations, links runner-produced evidence to those obligations, and returns a deterministic release decision: `block`, `human_escalation`, `canary`, or `auto_merge`.

## Contents

- [The Problem](#the-problem)
- [Why Use It](#why-use-it)
- [Why Not Just Use Another Agent?](#why-not-just-use-another-agent)
- [Current Validation](#current-validation)
- [Real Repo Walkthrough](#real-repo-walkthrough)
- [Clear Limits](#clear-limits)
- [Quickstart](#quickstart)
- [FAQ](#faq)
- [Project Docs](#project-docs)
- [Validation Status](#validation-status)
- [Demo](#demo)
- [Commands](#commands)

## The Problem

AI agents can produce code faster than humans can carefully review every line. CI can tell you whether commands passed, but it usually cannot answer the release question that matters for agent changes:

> For the contracts this change touches, are the required behavior, security, test, runtime, and rollback obligations covered by valid evidence?

Vouch adds that missing control plane. Humans declare what must remain true, existing runners produce evidence, and Vouch checks whether the evidence is complete enough for the repo's release policy.

## Why Use It

Use Vouch when you want:

- Agent changes tied to explicit, human-owned product and release contracts.
- Passing tests to be necessary but not sufficient for risky changes.
- Stable obligation IDs that connect intent, changed files, evidence artifacts, and release decisions.
- A CI-friendly gate that consumes existing JUnit, JSON, text, verifier, metric, and rollback artifacts instead of replacing your runner.
- Optional signed evidence checks that bind artifacts to approved runner identities.

Do not use Vouch for every small change. For low-risk edits, normal CI and review may be enough. Vouch is useful when the cost of a bad agent change is high enough that "tests passed" and "another agent said it looks fine" are not strong enough release criteria.

## Why Not Just Use Another Agent?

Another agent can review a diff, but it is still an opinion over code. It may miss the same release concerns a human skim misses, and its output is hard to turn into a stable merge rule.

Vouch is different because it asks for explicit release evidence:

| Question | Another Agent | Vouch |
| --- | --- | --- |
| Who owns the intended behavior? | Usually inferred from the diff or prompt. | Declared in a human-owned contract. |
| What must be checked before release? | Whatever the agent decides to inspect. | Compiled into stable obligation IDs. |
| What happens when tests pass but rollout evidence is missing? | Often depends on reviewer judgment. | Gate blocks or escalates according to policy. |
| Can the result be audited later? | Usually a prose comment or chat transcript. | Manifest, evidence artifacts, coverage, policy rule, and decision. |
| Can CI enforce it deterministically? | Not reliably without custom glue. | Yes, `vouch gate` exits non-zero on `block`. |

The benefit is not "more AI review." The benefit is a release boundary that says: for this kind of change, these obligations must have these evidence artifacts, or the change does not ship.

That is also why there are several moving parts:

- contracts define intent
- compile turns intent into stable obligations
- runners produce evidence
- gate checks evidence coverage and policy

You only pay that cost where the release risk justifies it: auth, payments, permissions, data deletion, migrations, external side effects, public APIs, production rollout, and other changes where a vague approval is not enough.

## Current Validation

Vouch includes a repo-local acceptance suite, VouchBench, for the current release-gate behavior. It builds the local CLI, runs isolated fixture repos, and fails non-zero if expected decisions, exit codes, coverage counts, missing obligation IDs, invalid evidence codes, policy rules, manifest/spec errors, or selected artifact statuses regress.

The current VouchBench corpus passes 10/10 scenarios and 114/114 assertions:

| Scenario | Baseline | Expected Vouch Decision | Validation Signal |
| --- | --- | --- | --- |
| High-risk auth with JUnit only | tests pass | `block` | Tests alone do not cover behavior, security, runtime, or rollback obligations. |
| High-risk auth with partial release evidence | tests pass | `block` | Partial artifacts are not enough when compiled obligations remain uncovered. |
| High-risk auth with full evidence but bad manifest traceability | tests pass | `block` | Full obligation coverage can still block when changed files are not owned by touched specs. |
| High-risk auth with non-zero test artifact | tests fail | `block` | Artifact exit codes are enforced. This is a negative control tests already catch. |
| High-risk auth with complete evidence and canary | tests pass | `canary` | Complete evidence routes high-risk auth to canary, not auto-merge. |
| High-risk auth with complete evidence but no canary | tests pass | `human_escalation` | High-risk complete evidence without canary escalates to a human. |
| Medium/high platform change across auth, payments, and API with partial evidence | tests pass | `block` | A 25-obligation multi-component release blocks when one high-risk component lacks security, runtime, and rollback evidence. |
| Medium/high platform change across auth, payments, and API with complete evidence | tests pass | `canary` | A larger multi-component release can pass to canary when all obligations are covered. |
| Medium-risk API-only change inside the platform repo | tests pass | `auto_merge` | Vouch scopes obligations to the touched medium-risk contract instead of gating the whole repo. |
| Low-risk docs with complete evidence | tests pass | `auto_merge` | Vouch is not just a blocker. Low-risk complete evidence can pass. |

The generic onboarding flow has also been exercised against temp copies of real Python repos (`sundae`, `sago`, and `wooster`) using `init`, `contract suggest`, `contract create`, `manifest create`, JUnit mapping/import, artifact attachment, and `gate`.

This validation is evidence of deterministic gate behavior over the stated corpus. It does not certify arbitrary code. See [Benchmarks](docs/BENCHMARKS.md) for the harness, assertions, and limits.

## Real Repo Walkthrough

This walkthrough uses [Pallets Click](https://github.com/pallets/click), a widely used Python CLI library. Pallets projects use `pytest` for tests, and Click 8.3.2 declares a `tests` dependency group with `pytest`.

The point of this walkthrough is not to claim Vouch understands Click. The point is to show the whole first-run workflow on a real repository:

```text
repo -> draft contracts -> compile obligations -> run tests -> import JUnit -> gate
```

### 1. Clone A Pinned Real Repo

Use a pinned tag so the counts below are reproducible:

```sh
mkdir -p /tmp/vouch-real-repo
cd /tmp/vouch-real-repo

git clone --depth 1 --branch 8.3.2 https://github.com/pallets/click.git
cd click
```

### 2. Install Test Dependencies

```sh
python3 -m venv .venv
. .venv/bin/activate

python -m pip install -U pip
python -m pip install -e . pytest
```

### 3. Draft And Compile Vouch Contracts

Run Vouch before running tests so generated `__pycache__` files do not become repo signals.

```sh
vouch init
vouch bootstrap --dry-run
vouch bootstrap
vouch compile
```

On Click 8.3.2, `vouch compile` currently produces:

```text
Compiled 31 contract drafts into 195 obligations.
```

This means Vouch found repo signals, wrote draft intent files under `.vouch/intents/`, compiled specs under `.vouch/specs/`, and built aggregate obligation IR under `.vouch/build/obligations.ir.json`.

At this point a human should inspect the generated intents. Bootstrap is conservative scaffolding, not product understanding. You should edit owners, risk levels, paths, behavior, security invariants, runtime signals, and rollback expectations before relying on the gate.

### 4. Run The Project Tests With JUnit

```sh
mkdir -p .vouch/artifacts
python -m pytest --junitxml .vouch/artifacts/pytest.xml
```

On one local run against Click 8.3.2, pytest reported:

```text
1384 passed, 22 skipped, 30000 deselected, 1 xfailed
```

The exact test count can vary with Python and dependency versions. The important artifact is `.vouch/artifacts/pytest.xml`.

### 5. Import Test Evidence

```sh
vouch evidence import junit .vouch/artifacts/pytest.xml
```

On the same Click run, Vouch linked:

```text
Linked obligations: 51
```

That means JUnit satisfied 51 required-test obligations. It does not satisfy behavior, security, runtime, or rollback obligations.

### 6. Run The Gate

```sh
vouch gate
```

Expected first-run result:

```text
Release decision: block
```

That block is expected. It means:

- tests passed
- Vouch imported test evidence
- required-test obligations were covered
- other generated obligations still lack accepted evidence

That is the main difference from plain CI. CI answers "did pytest pass?" Vouch asks "for the contracts touched by this release, are all required evidence classes covered?"

### 7. What You Do Next

For a real adoption pass, do not try to make every generated draft pass blindly. Tighten the contract set:

1. Delete or merge low-value generated intent files.
2. Assign real owners instead of `unowned`.
3. Keep contracts that match real release boundaries, for example `click.parser`, `click.termui`, `click.shell_completion`, and `click.testing`.
4. Replace generated behavior text with human-owned behavior obligations.
5. Keep required-test obligations mapped to real tests.
6. Add security, runtime, and rollback evidence only where those obligations matter for the component risk.
7. Re-run `vouch compile`, `vouch evidence import junit`, and `vouch gate`.

If you only want a non-destructive evaluation of another repo, use the snapshot evaluator from the Vouch checkout:

```sh
cd /path/to/vouch

scripts/vouchbench-repo.sh \
  --repo /path/to/repo \
  --test-command "mkdir -p .vouch/artifacts && python -m pytest --junitxml .vouch/artifacts/pytest.xml" \
  --junit .vouch/artifacts/pytest.xml \
  --out /tmp/vouchbench-repo
```

That copies the target repo into a temp directory and writes Vouch files only inside the snapshot.

## Clear Limits

Vouch is beta infrastructure, not a production-ready assurance system.

It does **not**:

- Review arbitrary diffs or decide whether an implementation is good.
- Infer product intent automatically. `vouch bootstrap` drafts conservative contracts from repo signals, but humans must accept, edit, or reject them.
- Run tests, probes, scanners, or deployment checks. External runners produce evidence. Vouch checks how that evidence maps to compiled obligations.
- Validate a generic JSON or text artifact as truthful beyond structure, status, obligation IDs, path, exit code, optional hash, and optional signature checks.
- Replace production code review, security review, incident response, rollout ownership, or human ownership of product intent.

## What Vouch Does Today

Vouch drafts conservative release contracts from signals already present in your repo, including tests, CODEOWNERS, OpenAPI specs, CI config, and evidence artifacts. It compiles accepted intent YAML into typed specs, obligation IR, verification plans, and runner/verifier/release artifacts. It then validates a change manifest and linked evidence artifacts to produce a release decision.

Use it today as experimental infrastructure for designing and exercising verification contracts for agent-authored changes.

## Quickstart

Use this when you have an existing repo and want Vouch to produce a release decision for an agent change.

### 0. Install The CLI

Install from GitHub:

```sh
go install github.com/duriantaco/vouch/cmd/vouch@latest
```

Or, from the Vouch checkout:

```sh
go install ./cmd/vouch
```

After that, use `vouch` directly:

```sh
vouch --repo /path/to/your/repo init
```

If your shell cannot find `vouch`, make sure Go's bin directory is on `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

If you are developing Vouch itself and do not want to install the binary, replace `vouch` with `go run ./cmd/vouch` in the commands below.

Fast v0.2 flow:

```sh
vouch bootstrap
vouch compile
pytest --junitxml .vouch/artifacts/pytest.xml
vouch evidence import junit .vouch/artifacts/pytest.xml
vouch gate
```

The mental model:

| Thing | File | What It Means |
| --- | --- | --- |
| Contract | `.vouch/intents/*.yaml` and `.vouch/specs/*.json` | Human-owned intent for one part of the repo. |
| Manifest | `.vouch/manifests/*.json` | What the agent changed in one run. |
| Evidence | `.vouch/artifacts/*` | Test/probe/scanner output that covers compiled obligation IDs. |
| Gate result | command output or `.vouch/build/gate-result.json` | Vouch's decision for that manifest. |

### 1. Initialize Vouch In Your Repo

Run `vouch` from anywhere and point `--repo` at the repo you want to protect:

```sh
vouch --repo /path/to/your/repo init
```

This creates:

- `.vouch/config.json`
- `.vouch/intents/`
- `.vouch/specs/`
- `.vouch/policy/release-policy.json`
- `.vouch/manifests/`
- `.vouch/artifacts/`
- `.vouch/build/`

### 2. Ask Vouch What Contract To Start With

```sh
vouch --repo /path/to/your/repo contract suggest --json
```

Pick one suggestion as a starting point. Vouch suggestions are structural. You still need to write the real product intent.

### 3. Create A Contract

A contract says what a part of the repo owns and what evidence must exist before changes to that area can ship.

```sh
vouch --repo /path/to/your/repo contract create \
  --name app.service \
  --owner platform \
  --risk medium \
  --paths "src/app/**,tests/test_app.py" \
  --behavior "service returns stable JSON" \
  --security "service does not expose secrets in responses" \
  --required-test "service json contract is stable" \
  --metric "app.service.requests" \
  --rollback-strategy "revert_change"
```

This writes:

- `.vouch/intents/app.service.yaml`
- `.vouch/specs/app.service.json`
- `.vouch/test-map.json` with empty stubs for required-test obligations

Build the IR once so you can see the exact obligation IDs evidence must reference:

```sh
vouch ir build \
  --spec /path/to/your/repo/.vouch/specs/app.service.json \
  --out /path/to/your/repo/.vouch/build/app.service.ir.json
```

Open `.vouch/build/app.service.ir.json` and look for `obligations[].id`. Example IDs look like:

- `app.service.behavior.service_returns_stable_json`
- `app.service.security.service_does_not_expose_secrets_in_responses`
- `app.service.required_test.service_json_contract_is_stable`
- `app.service.runtime_signal.app_service_requests`
- `app.service.rollback.revert_change`

### 4. Create A Manifest For The Agent Change

The manifest says what changed and which contract the change touches. You can pass changed files explicitly:

```sh
vouch --repo /path/to/your/repo manifest create \
  --task-id agent-123 \
  --summary "change app service" \
  --agent codex \
  --run-id run-123 \
  --changed-file src/app/service.py \
  --out .vouch/manifests/run-123.json
```

Check the manifest:

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  manifest check
```

At this point, `gate` should usually block because no evidence has been attached yet.

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate
```

That failure is useful. It tells you which obligations still need evidence.

### 5. Run Your Existing Checks

Vouch does not run your tests yet. Your runner still does that.

For pytest, generate JUnit:

```sh
cd /path/to/your/repo
pytest --junitxml .vouch/artifacts/pytest.xml
```

For the repo-level v0.2 flow, import JUnit evidence directly:

```sh
vouch --repo /path/to/your/repo evidence import junit .vouch/artifacts/pytest.xml
vouch --repo /path/to/your/repo gate
```

This writes `.vouch/evidence/manifest.json` and gates against the compiled obligation IR. JUnit only satisfies required-test obligations, so security, runtime, rollback, and behavior obligations still need their own evidence.

For a simple smoke check, write an artifact that names the obligation IDs it covers:

```json
{
  "status": "pass",
  "obligations": [
    "app.service.behavior.service_returns_stable_json"
  ]
}
```

Save artifacts under `.vouch/artifacts/`.

If an artifact does not include the exact obligation ID, Vouch will not count it as coverage.

### 6. Attach Evidence

Attach behavior, security, test, runtime, and rollback evidence as needed. The artifact kind must match the obligation kind.
`verifier_output` artifacts are the exception: they may reference any compiled obligation, import structured verifier findings, and do not satisfy required evidence coverage by themselves.

| Obligation ID Contains | Attach With `--kind` |
| --- | --- |
| `.behavior.` | `behavior_trace` |
| `.security.` | `security_check` |
| `.required_test.` | `test_coverage` |
| `.runtime_signal.` | `runtime_metric` |
| `.rollback.` | `rollback_plan` |

```sh
vouch --repo /path/to/your/repo manifest attach-artifact \
  --manifest .vouch/manifests/run-123.json \
  --id behavior \
  --kind behavior_trace \
  --path .vouch/artifacts/behavior.json \
  --producer pytest \
  --command "pytest --junitxml .vouch/artifacts/pytest.xml" \
  --exit-code 0 \
  --out .vouch/manifests/run-123.json
```

If your raw JUnit testcase names do not contain Vouch obligation IDs, map them through `.vouch/test-map.json`:

```sh
vouch --repo /path/to/your/repo manifest attach-artifact \
  --manifest .vouch/manifests/run-123.json \
  --id pytest \
  --kind test_coverage \
  --path .vouch/artifacts/pytest.xml \
  --test-map .vouch/test-map.json \
  --exit-code 0 \
  --out .vouch/manifests/run-123.json
```

Signature fields are optional in beta. To attach signed evidence, first write a
`vouch.evidence_bundle.v0` JSON file for the artifact. The bundle binds the
manifest task, touched specs, artifact ID/kind/path/hash, covered obligation
IDs, runner identity, and timestamp. Sign that evidence bundle with cosign, then
include the bundle paths and expected identity when attaching the artifact.
The signer must also be listed in `.vouch/config.json`:

```json
{
  "allowed_signers": [
    {
      "identity": "https://github.com/ORG/REPO/.github/workflows/vouch.yml@refs/heads/main",
      "oidc_issuer": "https://token.actions.githubusercontent.com"
    }
  ]
}
```

```sh
cosign sign-blob .vouch/artifacts/behavior.vouch-bundle.json \
  --bundle .vouch/artifacts/behavior.sigstore.json

vouch --repo /path/to/your/repo manifest attach-artifact \
  --manifest .vouch/manifests/run-123.json \
  --id behavior \
  --kind behavior_trace \
  --path .vouch/artifacts/behavior.json \
  --evidence-bundle .vouch/artifacts/behavior.vouch-bundle.json \
  --signature-bundle .vouch/artifacts/behavior.sigstore.json \
  --signer-identity "https://github.com/ORG/REPO/.github/workflows/vouch.yml@refs/heads/main" \
  --signer-oidc-issuer "https://token.actions.githubusercontent.com" \
  --exit-code 0 \
  --out .vouch/manifests/run-123.json
```

Production-style gates should require signed evidence:

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate --require-signed
```

`--require-signed` verifies each `evidence_bundle` with `cosign verify-blob`
using the artifact's `signature_bundle`, `signer_identity`, and
`signer_oidc_issuer` fields. It also rejects bundles whose manifest identity,
artifact hash, obligation IDs, runner identity, or runner OIDC issuer do not
match the manifest and artifact being gated, or whose signer is not present in
`config.allowed_signers`.

### 7. Gate The Change

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate
```

Vouch loads release policy from `.vouch/policy/release-policy.json` by default.
Pass `--policy` to simulate or gate with a different policy file:

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  policy simulate --policy .vouch/policy/release-policy.json

vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate --policy .vouch/policy/release-policy.json
```

Possible decisions:

- `block`: required evidence is missing or invalid.
- `human_escalation`: high-risk change passed evidence checks but has no canary.
- `canary`: high-risk change passed evidence checks and has canary enabled.
- `auto_merge`: low/medium-risk change passed required evidence checks.

For machine-readable output:

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate --json
```

To preserve a compact gate result artifact for CI upload or required-status checks:

```sh
vouch --repo /path/to/your/repo \
  --manifest .vouch/manifests/run-123.json \
  gate --out .vouch/build/gate-result.json
```

## Project Docs

- [Roadmap](ROADMAP.md) explains where the project is going and what remains before this can be production infrastructure.
- [Comparison](COMPARISON.md) explains how Vouch composes with Sigstore, SLSA, in-toto, OPA, and Conftest.
- [Benchmarks](docs/BENCHMARKS.md) explains the repo-local VouchBench harness, assertions, and current validation scope.
- `scripts/vouchbench-repo.sh --repo /path/to/repo` runs a non-destructive Vouch evaluation against an external repo snapshot.
- [Contributing](CONTRIBUTING.md) explains how to help and which areas need work.

The problem it attempts to solve is:

> If agents write more code than humans can carefully review line by line, what machine-checkable contract/evidence layer can sit before release?

Vouch's answer is:

1. Human-owned intent
2. Typed AST
3. Structured spec
4. Obligation IR
5. Verification plan
6. Generated verifier/test/release artifacts
7. Evidence manifest
8. Deterministic evidence checks
9. Release decision

The output is not generated product code, and it does not certify that the implementation is correct. The output is an auditable answer to a narrower question. For the contracts this change claims to touch, are the required obligations covered by evidence artifacts that pass Vouch's current checks?

## Current Assurance Level

Today, Vouch can check:

- Intent YAML parses into a typed AST with source-span diagnostics.
- Specs compile into stable semantic obligation IDs.
- Changed files map to touched contract ownership paths.
- Manifests cannot lower risk below the touched spec.
- Evidence artifacts reference known obligations of the right evidence kind.
- Artifact paths stay inside the repo and files exist.
- Artifact exit codes are present and zero.
- Optional artifact SHA-256 values match file contents.
- Optional `gate --require-signed` mode verifies signed `vouch.evidence_bundle.v0` files, binds manifest identity, artifact hashes, obligation IDs, and runner identity, and requires signer metadata to match `config.allowed_signers`.
- Release policy is loaded from `.vouch/policy/release-policy.json` or an explicit `--policy` file.
- Generated verifier packets pin prompt and output schema versions.
- Structured `verifier_output` artifacts parse as `vouch.verifier_output.v0` and import pass/block findings into release policy.
- `verifier_output` artifacts do not count as required behavior, security, test, runtime, or rollback evidence.
- JUnit evidence has no failure/error/skipped testcases and covers required test obligations.
- Raw pytest/JUnit testcases can be mapped through `.vouch/test-map.json`.
- Generic JSON/text artifacts contain exact obligation IDs and optional passing status.
- Compact gate results can be written to a JSON artifact file.
- Release decisions follow deterministic policy rules.

Today, Vouch cannot check:

- Whether arbitrary code semantically implements the declared behavior.
- Whether an AI verifier's judgment is correct beyond its structured output, obligation references, model/prompt pins, and artifact provenance.
- Whether a generic JSON/text artifact is truthful beyond its structure, IDs, status, path, exit code, and optional hash.
- Whether tests were actually run unless the external runner preserves and signs the artifact chain.
- Whether product intent was inferred correctly from code.
- Whether release policy uses a general-purpose policy language such as Rego. The current policy evaluator is a small Vouch JSON rule engine.
- Whether signer identities should vary by contract owner, path, or risk tier. The current allowlist is repo-level.

## Validation Status

For repeatable local validation, run:

```sh
scripts/vouchbench.sh
```

That harness compares baseline outcomes with Vouch's gate decision across fixed scenarios. The current corpus checks missing release evidence, manifest traceability, invalid artifact exit codes, high-risk canary routing, high-risk human escalation, and low-risk auto-merge. See [Benchmarks](docs/BENCHMARKS.md) for the exact scenarios, assertions, and limits.

The generic flow has been validated against temp copies of real Python repos:

| Repo | Shape | Evidence Path | Decision |
| --- | --- | --- | --- |
| `sundae` | Flat Python package | pytest/JUnit evidence | `auto_merge` |
| `sago` | Python `src/` layout | high-risk builder contract | `canary` |
| `wooster` | Flat Python package | raw pytest JUnit mapped through `.vouch/test-map.json` | `auto_merge` |

Those runs exercised `init`, `contract suggest`, `contract create`, `manifest create`, `manifest check`, `junit map`, `manifest attach-artifact`, and `gate`.

This validates the generic workflow plumbing. It does not mean Vouch understands those products automatically. Vouch still needs human-owned contracts. Suggestions are structural starting points based on repo shape and ownership paths.

## What This Solves

Traditional code review relies on humans reading diffs and noticing the important things. That model starts to fail when agents can produce changes faster than humans can inspect them.

Vouch moves part of the release boundary from informal diff reading to explicit contracts:

- Informal model: human reads diff, then decides whether the change is safe enough.
- Vouch model: human declares intent, Vouch compiles obligations, the runner produces artifacts, Vouch checks evidence coverage, and policy chooses a release posture.

This gives a team a place to enforce:

- What behavior must remain true.
- What security invariants must hold.
- What tests must cover.
- What runtime signals must exist.
- What rollback path must be available.
- What release policy applies to the risk.

That is the surface Vouch is aiming at. Vouch does not do semantic judgment over arbitrary code, but contract/evidence enforcement for the behavior humans have explicitly declared.

## Not Just CI

Vouch does not execute project checks today. It consumes a manifest and artifacts after external commands have run.

A normal CI gate answers:

> Did this command pass?

Vouch's job is to answer a different set of questions:

- What human owned contract did this change touch?
- What obligations did that contract compile into?
- Which evidence artifact covers each obligation?
- Is the evidence complete, structurally valid, and tied to this manifest?
- Does release policy allow `block`, `human_escalation`, `canary`, or `auto_merge`?

If Vouch becomes only a wrapper around `pytest`, `go test`, or a generic "merge or don't merge" check, it is not useful enough. The compiler direction is important because the value is in the typed contract, obligation IR, evidence mapping, and auditable release decision.

## Is It A Compiler?

Yes and no. Yes, in the control-plane architecture sense.

No, in the traditional programming language compiler sense.

Vouch has compiler-like stages:

| Stage | What Vouch Does Today |
| --- | --- |
| Source language | Reads intent YAML. |
| Front end | Parses into a typed AST with source-span diagnostics. |
| Semantic checks | Validates required fields, risk rules, and spec shape. |
| Middle end | Lowers specs into obligation IR. |
| Back end | Emits verification plans and runner/verifier/release artifacts. |
| Runtime checks | Collects evidence, applies policy, and returns a gate decision. |

Making it more compiler-like is useful because it makes the system more accurate in the areas that matter here. These things include deterministic parsing, typed intermediate representations, structured diagnostics, generated artifacts, and auditable policy decisions.

The current implementation is early. Policy is file-backed but still simple, generic non-JUnit artifacts are shallow, and contract suggestions are structural heuristics. The compiler direction is still the right direction because the value is in deterministic obligations and evidence linkage, not in another generic "merge or don't merge" opinion.

## Current Pipeline

| Command Area | Input | Output |
| --- | --- | --- |
| `bootstrap` | repo signals | draft intents plus bootstrap report |
| `compile` | `.vouch/intents/*.yaml` | specs, aggregate obligation IR, and verification plan |
| `intent parse` | `.vouch/intents/*.yaml` | `vouch.ast.v0` with source spans and diagnostics |
| `intent compile` | Intent AST | `vouch.spec.v0` JSON |
| `ir build` | Spec JSON | `vouch.ir.v0` obligations |
| `plan build` | IR plus change manifest | `vouch.plan.v0` verification plan |
| `artifacts build` | Spec JSON | runner/verifier/release artifacts |
| `junit map` | raw pytest/JUnit and `.vouch/test-map.json` | Vouch-compatible JUnit evidence |
| `evidence import junit` | raw pytest/JUnit and compiled obligation IR | `.vouch/evidence/manifest.json` |
| `policy simulate` | manifest, evidence, and release policy | policy input and release decision |
| `evidence`, `verify`, `gate` | manifest and linked artifacts | findings and release decision |
| Generic onboarding | repo shape and changed files | `.vouch` layout, contract suggestions, manifests, attached artifacts |

Generated artifact files:

- `verification-plan.json`
- `verifier-packets.json`
- `test-obligations.json`
- `release-policy.json`

## What It Checks Today

The MVP currently handles and checks:

- Schema versions.
- Repo profile detection for Python, Node, Go, Rust, and fallback repos.
- Contract suggestion and creation.
- Manifest creation from changed files and owned paths.
- Behavior contracts.
- Security invariants.
- Required test evidence.
- Runtime metrics.
- Rollback plans.
- Canary requirements.
- External side effects.
- Risk classification.
- Manifest risk downgrades.
- Compiler diagnostics with source locations.
- Stable semantic obligation IDs.
- Evidence artifact references.
- Evidence artifact file existence.
- Optional evidence artifact SHA-256.
- JUnit XML test evidence.
- Touched-spec compilation stats.

It blocks when:

- The spec or manifest is invalid.
- The manifest lowers risk below the touched spec.
- Behavior evidence is missing.
- Required test evidence is missing.
- Security evidence is missing.
- Runtime metrics are missing.
- Rollback evidence is missing.
- Canary config is invalid.
- External side effects have no rollback or compensation.
- Tests are reported as failing.
- Evidence artifacts reference unknown obligations.
- Evidence artifact files are missing.
- Evidence artifact hashes do not match.
- JUnit evidence has failing or missing obligation testcases.
- Changed files are not owned by any touched spec.
- Raw JUnit cannot be mapped to required-test obligations.

For high-risk changes with complete evidence, it chooses `canary`, not `auto_merge`.

## Demo

The demo uses a synthetic high-risk `auth.password_reset` change because it exercises the core risk classes:

- Account enumeration.
- Token storage.
- Rate limiting.
- Token logging.
- Token replay tests.
- Runtime metrics.
- Feature-flag rollback.
- Email side effects.

The demo exercises the compiler plumbing and failure modes. It does not mean Vouch can infer or verify password reset correctness from application code.

Run the repo-level compiler pipeline:

```sh
vouch --repo demo_repo compile
vouch --repo demo_repo compile --emit ir
```

Parse intent into a typed AST:

```sh
vouch intent parse --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.ast.json
```

Compile human intent into a structured spec:

```sh
vouch intent compile --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.json
```

Build an intermediate representation from the spec:

```sh
vouch ir build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/auth.password_reset.ir.json
```

Build a verification plan from spec + manifest:

```sh
vouch plan build --spec demo_repo/.vouch/specs/auth.password_reset.json --manifest demo_repo/.vouch/manifests/pass.json --out /tmp/auth.password_reset.plan.json
```

Generate runner/verifier/release artifacts:

```sh
vouch artifacts build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/vouch-artifacts
```

Run a blocked change:

```sh
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/blocked.json evidence
```

Expected:

```console
Decision: block
IR obligations covered: 10/16
```

Run a passing high-risk change:

```sh
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/pass.json evidence
```

Expected:

```console
Decision: canary
IR obligations covered: 16/16
```

It canaries because the change is high-risk auth behavior even though evidence is complete.

## Generic Repo Onboarding

Vouch is generic at the compiler layer. The repo supplies contracts. Vouch compiles and gates them.

The Quickstart above is the recommended path. The main idea is:

1. `init` creates `.vouch/`.
2. `contract suggest` gives structural starting points.
3. `contract create` writes intent and spec files.
4. `manifest create` links changed files to touched contracts.
5. Your runner produces test/probe/scanner artifacts.
6. `manifest attach-artifact` links artifacts to compiled obligation IDs.
7. `gate` returns `block`, `human_escalation`, `canary`, or `auto_merge`.

For normal JUnit output, map real testcases to Vouch required-test obligations with `.vouch/test-map.json`. `contract create` writes empty stubs for new required-test obligations:

```json
{
  "version": "vouch.test_map.v0",
  "mappings": {
    "app.service.required_test.service_json_contract_is_stable": [
      "tests/test_app.py::test_service_json_contract"
    ]
  }
}
```

Map raw JUnit explicitly:

```sh
vouch --repo /path/to/repo junit map \
  --manifest .vouch/manifests/run-123.json \
  --junit .vouch/artifacts/pytest.xml \
  --test-map .vouch/test-map.json \
  --out .vouch/artifacts/vouch-junit.xml
```

Or map and attach in one step:

```sh
vouch --repo /path/to/repo manifest attach-artifact \
  --manifest .vouch/manifests/run-123.json \
  --id pytest \
  --kind test_coverage \
  --path .vouch/artifacts/pytest.xml \
  --test-map .vouch/test-map.json \
  --exit-code 0 \
  --out .vouch/manifests/run-123.json
```

For the repo-level v0.2 path, import raw JUnit into an evidence manifest:

```sh
vouch --repo /path/to/repo evidence import junit .vouch/artifacts/pytest.xml
vouch --repo /path/to/repo gate
```

## GitHub PR Summary

Use `gate --github-summary` in GitHub Actions to append a Markdown decision report to the pull request job summary:

```sh
vouch gate --github-summary
```

The command writes to `$GITHUB_STEP_SUMMARY` and still exits non-zero when the release decision is `block`. A copyable workflow example lives at [`docs/github-action-example.yml`](docs/github-action-example.yml).

## Commands

```sh
vouch --repo DIR init
vouch --repo DIR bootstrap
vouch --repo DIR compile [--emit ast|spec|ir|plan]
vouch --repo DIR contract suggest
vouch --repo DIR contract create --name ID --owner OWNER --risk RISK --paths GLOB --behavior TEXT --required-test TEXT
vouch intent parse --intent FILE --out FILE
vouch intent compile --intent FILE --out FILE
vouch ir build --spec FILE --out FILE
vouch --manifest FILE plan build --spec FILE --out FILE
vouch artifacts build --spec FILE --out DIR
vouch --repo DIR spec lint
vouch --repo DIR --manifest FILE manifest check
vouch --repo DIR manifest create --task-id ID --summary TEXT --agent NAME --run-id ID [--runner-identity ID --runner-oidc-issuer URL] --out FILE
vouch --repo DIR --manifest FILE manifest attach-artifact --id ID --kind KIND --path FILE --exit-code N [--evidence-bundle FILE --signature-bundle FILE --signer-identity ID --signer-oidc-issuer URL] --out FILE
vouch --repo DIR --manifest FILE junit map --junit FILE --test-map FILE --out FILE
vouch --repo DIR evidence import junit [--out FILE] FILE
vouch --repo DIR --manifest FILE policy simulate [--policy FILE] [--require-signed]
vouch --repo DIR --manifest FILE verify [--policy FILE]
vouch --repo DIR --manifest FILE gate [--policy FILE] [--out FILE] [--github-summary] [--require-signed]
vouch --repo DIR --manifest FILE evidence [--policy FILE]
```

`--json` emits machine-readable evidence for commands that collect evidence. For `gate`, `--json` emits the compact gate result to stdout and `--out FILE` writes the same gate-result shape as a JSON artifact.

## FAQ

### Why is this more complicated than normal CI?

Because Vouch is checking a different thing. CI usually answers whether commands passed. Vouch asks whether the change has the required evidence for the contracts it touches. That needs contracts, compiled obligations, evidence artifacts, and policy.

For low-risk changes, this ceremony may not be worth it. For auth, payments, permissions, migrations, public APIs, external side effects, and production rollout, the extra structure is the point.

### Why not just ask another agent to review the code?

An agent review is still an opinion over a diff. Vouch produces a deterministic gate result from declared obligations and evidence artifacts. You can audit which obligation was required, which artifact covered it, which policy rule fired, and why the gate returned `block`, `human_escalation`, `canary`, or `auto_merge`.

### Does Vouch replace tests?

No. Tests are one evidence source. Vouch consumes test output, currently JUnit, and checks whether it covers required-test obligations. It does not run your test suite itself.

### Why did `vouch gate` block even though tests passed?

Usually because JUnit only covers required-test obligations. Behavior, security, runtime, and rollback obligations require their own accepted evidence. That is expected on a first run.

### Does Vouch infer product intent from code?

No. `vouch bootstrap` drafts conservative contracts from repo signals such as tests, paths, CODEOWNERS, and CI files. Humans still own the intent and must accept, edit, merge, or delete generated contracts.

### What do I have to write myself?

You need to review or write the release contract:

- component ownership
- risk level
- behavior obligations
- security invariants
- required tests
- runtime signals
- rollback expectations

Bootstrap can create a starting point, but the contract is only useful once a human makes it match the product.

### What is the first useful adoption step?

Pick one high-risk component instead of trying to cover the whole repo. Good first targets are auth, billing, permissions, data deletion, migrations, public APIs, and deployment-critical code.

Create or edit one contract, map its required tests, run JUnit import, and let `vouch gate` show which evidence is still missing.

### Is Vouch production-ready?

No. Treat it as beta infrastructure for experimenting with contract/evidence release gates. It is not a replacement for code review, security review, incident response, or production ownership.

### What does VouchBench actually validate?

VouchBench validates deterministic release-gate behavior over a fixed corpus. It checks expected decisions, exit codes, coverage counts, missing obligations, invalid evidence, policy rules, and manifest/spec errors.

It does not validate arbitrary product correctness.

### Can I run it against my own repo without modifying it?

Yes:

```sh
scripts/vouchbench-repo.sh --repo /path/to/repo --out /tmp/vouchbench-repo
```

That copies your repo into a temp snapshot and runs Vouch there.

### What should I look at after bootstrap?

Start with:

```text
.vouch/intents/*.yaml
.vouch/build/obligations.ir.json
.vouch/build/verification-plan.json
.vouch/evidence/manifest.json
```

The intent files are the human-editable contract drafts. The IR and plan show what Vouch will require. The evidence manifest shows which test artifacts linked to which obligations.

## Test

The sandbox may not allow Go to use the default cache path, so use a temp cache:

```sh
GOCACHE=/private/tmp/vouch-gocache go test ./...
```

## License

Vouch is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

## Why This Exists

**The compiler analogy is a direction, not a maturity claim.**

Mature software delivery relies on toolchains. These include language semantics, type systems, tests, validation, reproducible builds, monitoring, and rollback. Agent-written code needs a similar toolchain around intent and evidence.

Vouch is an early piece of that toolchain. Source intent -> AST -> spec -> obligations -> plan -> evidence -> release decision.

The output is not trust by assertion. It is a deterministic record of which declared obligations were required, which evidence artifacts covered them, what failed, and what release posture the current policy selected.
