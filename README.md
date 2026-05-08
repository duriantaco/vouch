<p align="center">
  <img src="assets/vouch.png" alt="Vouch logo" width="240">
</p>

# Vouch (Beta)

## What Vouch Is Today

Vouch is a beta contract-and-evidence gate for agent-written changes.

It compiles human-owned intent YAML into typed specs, obligation IR, verification plans, and runner/verifier/release artifacts. It then validates a change manifest and linked evidence artifacts to produce a deterministic release decision: `block`, `human_escalation`, `canary`, or `auto_merge`.

It is **NOT** a code reviewer. It does not read a diff and decide whether the implementation is good. It does not run your tests today. External runners execute tests, probes, scanners, and deployment checks. Vouch checks whether the resulting evidence maps to the obligations compiled from explicit contracts.

Use it as experimental infrastructure for designing Vouch verification contracts, not as a replacement for production security review, code review, incident response, or human ownership of product intent.

## Quickstart

Use this when you have an existing repo and want Vouch to produce a release decision for an agent change.

### 0. Install The CLI

From the Vouch checkout:

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

Pick one suggestion as a starting point. Vouch suggestions are structural; you still need to write the real product intent.

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

The output is not generated product code, and it is not proof that the implementation is correct. The output is an auditable answer to a narrower question. For the contracts this change claims to touch, are the required obligations covered by evidence artifacts that pass Vouch's current checks?

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
- Whether release policy uses a general-purpose policy language such as Rego; the current policy evaluator is a small Vouch JSON rule engine.
- Whether signer identities should vary by contract owner, path, or risk tier; the current allowlist is repo-level.

## Validation Status

The generic flow has been validated against temp copies of real Python repos:

| Repo | Shape | Evidence Path | Decision |
| --- | --- | --- | --- |
| `sundae` | Flat Python package | pytest/JUnit evidence | `auto_merge` |
| `sago` | Python `src/` layout | high-risk builder contract | `canary` |
| `wooster` | Flat Python package | raw pytest JUnit mapped through `.vouch/test-map.json` | `auto_merge` |

Those runs exercised `init`, `contract suggest`, `contract create`, `manifest create`, `manifest check`, `junit map`, `manifest attach-artifact`, and `gate`.

This validates the generic workflow plumbing. It does not prove Vouch understands those products automatically. Vouch still needs human-owned contracts; suggestions are structural starting points based on repo shape and ownership paths.

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

The demo exercises the compiler plumbing and failure modes. It does not prove Vouch can infer or verify password reset correctness from application code.

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
vouch --repo DIR --manifest FILE policy simulate [--policy FILE] [--require-signed]
vouch --repo DIR --manifest FILE verify [--policy FILE]
vouch --repo DIR --manifest FILE gate [--policy FILE] [--out FILE] [--require-signed]
vouch --repo DIR --manifest FILE evidence [--policy FILE]
```

`--json` emits machine-readable evidence for commands that collect evidence. For `gate`, `--json` emits the compact gate result to stdout and `--out FILE` writes the same gate-result shape as a JSON artifact.

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
