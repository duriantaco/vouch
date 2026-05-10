# GitHub Actions

This repo uses GitHub Actions for two things:

1. publishing the Vouch docs site to GitHub Pages
2. running the current Vouch gate in pull request workflows

Published docs URL:

```text
https://duriantaco.github.io/vouch/
```

GitHub Pages is configured with `build_type: workflow`, so content is published
by `.github/workflows/pages.yml` after changes land on `main`.

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
docs/site/
docs/site/assets/vouch.png
```

The workflow builds the MkDocs site into `_site/`, uploads that artifact, and
deploys it to the `github-pages` environment.

Official references:

- [Configuring a publishing source for your GitHub Pages site](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site)
- [actions/upload-pages-artifact](https://github.com/actions/upload-pages-artifact)
- [actions/deploy-pages](https://github.com/actions/deploy-pages)

## Vouch PR Workflow

The current Vouch PR workflow should start in shadow mode. It appends a job
summary and uploads `.vouch/build/gate-result.json` without blocking the PR.

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

## Code References

- CLI command and `--github-summary` flag:
  [`internal/vouch/cli.go`](https://github.com/duriantaco/vouch/blob/main/internal/vouch/cli.go)
- `$GITHUB_STEP_SUMMARY` handling:
  [`appendGitHubSummary`](https://github.com/duriantaco/vouch/blob/main/internal/vouch/cli.go)
- Markdown summary rendering:
  [`RenderGitHubSummary`](https://github.com/duriantaco/vouch/blob/main/internal/vouch/render.go)
- Gate result JSON:
  [`GateResultFromEvidence`](https://github.com/duriantaco/vouch/blob/main/internal/vouch/render.go)
- Default release policy:
  [`DefaultReleasePolicy`](https://github.com/duriantaco/vouch/blob/main/internal/vouch/policy.go)

## Enforced Mode

After a shadow-mode pilot, remove `continue-on-error: true` from the gate step.
The CLI exits non-zero only when the final decision is `block`.
