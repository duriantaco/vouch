# Vouch Operating Brief

Use this before choosing roadmap work, designing features, writing docs, or
describing Vouch.

## Identity

Vouch is a compiler for release contracts.

Its job is to turn human-owned release intent into typed obligations, link those
obligations to evidence artifacts, and produce an auditable release decision:

```text
human-owned intent
  -> typed AST and diagnostics
  -> spec JSON
  -> obligation IR
  -> verification plan and runner artifacts
  -> evidence and policy input
  -> block | human_escalation | canary | auto_merge
```

The release gate is a runtime target for the compiler output. It is not the
whole product.

## Boundary

Vouch is not an AI code reviewer.

Do not position Vouch as a tool that reads a diff and decides whether the
implementation is good. It should not compete with AI pull-request review tools
by producing generic review comments, style suggestions, or line-by-line bug
claims.

Vouch should answer a different question:

```text
For the contracts this change touched, which compiled obligations are required,
which evidence artifacts satisfy them, and what release posture follows?
```

## Good Work

Prefer work that strengthens at least one of these surfaces:

- Contract language and source diagnostics.
- Typed AST, semantic analysis, and stable compiler stages.
- Obligation IR and stable semantic obligation IDs.
- Spec-to-file traceability for routing changed files to contracts.
- Evidence artifact import, validation, linking, and coverage.
- Policy input, policy simulation, policy semantics, and fired-rule reporting.
- Auditable gate results for CI and pull-request workflows.
- Evidence provenance, runner identity, signatures, and tamper evidence.
- Shadow-mode pilot workflows that prove Vouch catches release-readiness gaps.

The best next work makes existing CI artifacts more meaningful to Vouch:

- SARIF or Semgrep import for `security_check` evidence.
- Coverage import for `test_coverage` or behavior-adjacent evidence.
- Deployment, metric, alert, and rollback evidence importers.
- GitHub summary/check output that explains obligations, evidence, and policy.
- Case studies where tests pass but release obligations remain uncovered.

## Wrong Turns

Avoid work whose main value is generic code review:

- Inline comments about arbitrary diff quality.
- General bug-finding agents that are not tied to compiled obligations.
- Style, lint, readability, naming, or refactor suggestions as product output.
- "Approve this PR" or "this code is correct" claims.
- Chat-based code review workflows without artifact-backed evidence.
- AI verifier layers that are not bound to runner identity, evidence artifacts,
  and release policy.

Avoid work that turns Vouch into only a CI wrapper:

- Running test commands without improving obligation coverage.
- Duplicating existing scanner dashboards without mapping results to obligations.
- Blocking on tool output without explaining the touched contract and evidence
  requirement.

## Positioning

Use these phrases:

- "compiler for release contracts"
- "release-contract compiler and evidence gate"
- "the layer between CI passed and ship it"
- "obligation-oriented control plane"
- "auditable release decision for agent-written code"

Avoid these phrases as primary positioning:

- "AI code reviewer"
- "automated PR reviewer"
- "bug finder"
- "lint bot"
- "CI wrapper"
- "general coding agent"

## Decision Test

Before implementing a feature, answer these questions:

1. Which human-owned contract or typed source artifact does this strengthen?
2. Which compiled obligation, evidence kind, or policy fact does this improve?
3. Does the output remain deterministic and auditable?
4. Can the result be represented in compiler artifacts, evidence manifests, or
   gate results?
5. Would this still matter if all generic AI code reviewers already existed?
6. Are we inspecting implementation correctness, or are we linking evidence to
   declared obligations?

If the answer to question 6 is "inspecting implementation correctness," stop and
redesign the work around contracts, obligations, evidence, or policy.

## Near-Term Work

The current product direction is:

1. Ship a strong shadow-mode pull-request pilot path.
2. Add evidence connector importers, starting with SARIF/Semgrep and coverage.
3. Produce realistic case studies where tests pass but Vouch blocks or routes
   the change because release evidence is missing, invalid, or out of scope.
4. Continue trust hardening where it supports real evidence workflows: required
   high-risk hashes, commit/runner provenance, scoped signers, and signed specs
   or manifests.

Do not lead with AI review features. AI verifiers can come later only as
evidence verifiers that are tied to obligations, artifacts, provenance, and
policy.

## Integration Posture

Vouch should compose with existing tools instead of replacing them:

- Sigstore/cosign proves evidence signer identity.
- SLSA and in-toto-style metadata describe provenance and authorized steps.
- OPA/Rego can evaluate policy over structured inputs.
- SARIF, Semgrep, coverage, test reports, deployment plans, metrics, and
  rollback plans are evidence inputs.

Vouch supplies the missing semantic layer: contract language, obligation IR,
evidence mapping, and release result.
