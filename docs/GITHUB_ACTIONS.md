<p align="center">
  <img src="../assets/vouch.png" alt="Vouch logo" width="180">
</p>

# GitHub Actions

This repo uses GitHub Actions for two things:

1. publishing the Vouch docs site to GitHub Pages
2. running the current Vouch gate in pull request workflows

Published docs URL:

https://duriantaco.github.io/vouch/

Pages is configured with `build_type: workflow`, so content is published by
`.github/workflows/pages.yml` after changes land on `main`.

## Docs Deployment

The Pages workflow follows GitHub's custom Actions publishing flow:

- `actions/configure-pages`
- `actions/upload-pages-artifact`
- `actions/deploy-pages`

Workflow file:

```text
.github/workflows/pages.yml
```

Source files:

```text
mkdocs.yml
docs/requirements.txt
docs/site/
docs/site/assets/vouch.png
```

The workflow builds the MkDocs site into `_site/`, uploads that artifact, and
deploys it to the `github-pages` environment.

Official references:

- https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site
- https://github.com/actions/upload-pages-artifact
- https://github.com/actions/deploy-pages

## Vouch PR Workflow

The relevant code paths are:

- CLI command and `--github-summary` flag:
  [`internal/vouch/cli.go`](../internal/vouch/cli.go)
- `$GITHUB_STEP_SUMMARY` handling:
  [`appendGitHubSummary`](../internal/vouch/cli.go)
- Markdown summary rendering:
  [`RenderGitHubSummary`](../internal/vouch/render.go)
- Final gate result shape:
  [`GateResultFromEvidence`](../internal/vouch/render.go)
- Default release policy:
  [`DefaultReleasePolicy`](../internal/vouch/policy.go)

### Shadow Mode

Use this first. It shows Vouch's decision in the job summary but does not block
the pull request.

```yaml
name: Vouch

on:
  pull_request:

jobs:
  vouch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Install Vouch
        run: go install github.com/duriantaco/vouch/cmd/vouch@latest

      - name: Compile Vouch contracts
        run: vouch compile

      - name: Run tests
        run: pytest --junitxml .vouch/artifacts/pytest.xml

      - name: Import JUnit evidence
        run: vouch evidence import junit .vouch/artifacts/pytest.xml

      - name: Gate PR
        continue-on-error: true
        run: vouch gate --github-summary --out .vouch/build/gate-result.json

      - name: Upload Vouch result
        uses: actions/upload-artifact@v4
        with:
          name: vouch-gate-result
          path: .vouch/build/gate-result.json
```

Use shadow mode until reviewers agree that Vouch is catching real
release-readiness gaps and the false-block rate is acceptable.

### Enforced Mode

After a shadow-mode pilot, remove `continue-on-error: true` from the gate step:

```yaml
      - name: Gate PR
        run: vouch gate --github-summary --out .vouch/build/gate-result.json
```

`vouch gate` exits non-zero only when the release decision is `block`.
`human_escalation`, `canary`, and `auto_merge` are non-blocking process exits.

### What The Summary Shows

`gate --github-summary` appends a Markdown report to `$GITHUB_STEP_SUMMARY`.
The report includes:

- final decision
- risk
- obligation coverage
- policy path
- reasons
- component-level covered and missing obligations
- verifier findings and required fixes

The renderer is [`RenderGitHubSummary`](../internal/vouch/render.go).

### Contracts In CI

Do not silently generate new contracts in an enforced workflow. Commit reviewed
`.vouch/intents/*.yaml`, `.vouch/specs/*.json`, and
`.vouch/policy/release-policy.json`.

For pilots, it is acceptable to run:

```sh
vouch bootstrap --review
```

Use generated contracts as scaffolding only. A human should edit owners, paths,
risk, behavior, security, runtime signals, and rollback expectations before
Vouch becomes an enforced gate.

### Signed Evidence

For stricter runs, use:

```sh
vouch gate --require-signed --github-summary
```

The signed-evidence checks are wired through `CollectEvidenceWithOptions` and
artifact linking in [`internal/vouch/evidence.go`](../internal/vouch/evidence.go).
Allowed signers are loaded from `.vouch/config.json`.

Use this only after your runners are producing Vouch evidence bundles and cosign
signature bundles.
