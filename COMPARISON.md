# Comparison With Supply-Chain And Policy Tools

Vouch should be honest about where it fits. It is not a replacement for
Sigstore, SLSA, in-toto, OPA, or Conftest. The defensible position is that
Vouch is an obligation-oriented control plane that can compose with those tools.

One-sentence version: Vouch is the layer between "tests passed" and "ship it"
for AI-written code; it maps the contract a change touched to the exact evidence
required to release it, then composes with supply-chain identity, policy engines,
and runner artifacts to decide whether the change can ship.

## What Existing Tools Cover

| Tool | Primary Job | What It Gives Vouch | What It Does Not Give Vouch |
| --- | --- | --- | --- |
| [Sigstore/cosign](https://docs.sigstore.dev/cosign/signing/overview/) | Keyless signing, identity-bound verification, transparency logging. | A way to prove an artifact was signed by an expected runner identity and recorded with auditable signing metadata. | A product contract language, obligation IR, or mapping from feature intent to required evidence. |
| [SLSA](https://slsa.dev/spec/v1.2/) | Supply-chain security levels and provenance expectations for source/build artifacts. | A vocabulary for provenance quality and build trustworthiness. | A release contract for behavior, security invariants, runtime metrics, and rollback obligations. |
| [in-toto](https://in-toto.io/docs/getting-started/) | Layouts and signed link metadata for expected supply-chain steps. | A mature model for authorized steps, materials/products, and signed supply-chain metadata. | A compiler from product intent into obligations or policy decisions over changed feature contracts. |
| [OPA/Rego](https://www.openpolicyagent.org/docs/policy-language) | General policy evaluation over structured input. | A policy engine that can replace hard-coded release-decision logic. | Vouch-specific IR, evidence collection, artifact linking, or contract authoring. |
| [Conftest](https://www.conftest.dev/) | Rego tests for structured configuration data. | A simple way to run policy checks over JSON/YAML/TOML inputs. | Vouch-specific evidence import, obligation coverage, or release-result semantics. |

## Why Not Just Compose Existing Tools Directly?

For some workflows, direct composition is enough:

```sh
cosign verify-blob artifact.json \
  --bundle artifact.sigstore.json \
  --certificate-identity "$RUNNER_IDENTITY" \
  --certificate-oidc-issuer "$OIDC_ISSUER"
conftest test manifest.json
```

That proves useful things: the artifact may have a valid signer, and the
manifest may satisfy a policy. It does not answer the Vouch-specific question:

> For the contracts this change touches, which compiled obligations are required,
> which evidence artifacts cover them, and what release posture follows from the
> contract risk and evidence quality?

Vouch should use existing tools for signing, provenance, and policy execution
where possible. The unique surface is the typed contract-to-obligation pipeline:

1. Human-owned intent.
2. Spec and obligation IR.
3. Changed-file to touched-contract traceability.
4. Evidence artifact linking by obligation ID and evidence kind.
5. Deterministic release result.

## How Vouch Should Compose With Them

### Sigstore/cosign

Vouch should not invent a long-term signing system. The production path should
verify detached evidence bundles signed by approved runner identities.
`gate --require-signed` already rejects unsigned artifacts and asks cosign to
verify the artifact blob against its Sigstore bundle and expected signer
identity. The next hardening step is to sign a canonical evidence bundle that
also binds the manifest, artifact hashes, and covered obligation IDs.

### SLSA

Vouch should treat SLSA provenance as evidence about the runner/build chain, not
as a substitute for product obligations. A high-quality SLSA provenance record
can support the claim that evidence came from an expected builder. It does not
say whether `auth.password_reset.security.reset_token_is_never_logged` was
covered.

### in-toto

in-toto is the closest conceptual neighbor for supply-chain step integrity.
Vouch should borrow the idea of authorized steps and signed metadata, while
keeping its own obligation IR as the semantic layer for agent-change release
decisions.

### OPA/Rego

OPA is the best near-term candidate for moving policy out of Go. Vouch should
prepare a compact policy input that includes manifest data, spec risk, coverage,
findings, invalid evidence, and signer/provenance status. The policy output
should be the release decision and reasons. The current implementation uses a
small Vouch JSON rule engine for that policy boundary; Rego remains the likely
next step if the policy input shape proves stable.

### Conftest

Conftest is useful as a reference workflow and local policy runner. Vouch should
not become only a wrapper around Conftest; the value is in generating the policy
input from contracts, obligations, and evidence.

## Boundary

Vouch is not trying to prove that code is correct. It is trying to make release
decisions auditable against explicit contracts. The strongest architecture is
therefore compositional:

- Sigstore/cosign proves who signed evidence.
- SLSA/in-toto-style metadata proves where evidence came from and which steps ran.
- OPA/Rego decides release posture from structured facts.
- Vouch supplies the contract language, obligation IR, evidence linkage, and gate
  result.
