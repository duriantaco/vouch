<p align="center">
  <img src="assets/vouch.png" alt="Vouch logo" width="220">
</p>

# Vouch

Vouch is a compiler for release contracts.

The release gate is the first runtime target, not the whole project.

```text
human-owned intent YAML
  -> typed AST
  -> spec JSON
  -> obligation IR
  -> verification plan and runner artifacts
  -> evidence and policy input
  -> block | human_escalation | canary | auto_merge
```

Use Vouch when a risky AI-authored change needs more than "CI passed." A
contract says what a repo area owns; the compiler gives those requirements
stable obligation IDs; the gate checks whether evidence covers those IDs.

## Quick Start

Preview a repo without writing files:

```sh
vouch try --repo /path/to/repo
```

Write the draft `.vouch/` layout when the preview is useful:

```sh
vouch try --repo /path/to/repo --write
```

Compile contracts and run the current gate:

```sh
cd /path/to/repo
vouch compile
pytest --junitxml .vouch/artifacts/pytest.xml
vouch evidence import junit .vouch/artifacts/pytest.xml
vouch gate
```

JUnit covers required-test obligations only. `security_check` artifacts can use
SARIF 2.1.0 when scanner rules or results reference exact obligation IDs.
Behavior, runtime, and rollback obligations need their own evidence.

## Install

```sh
go install github.com/duriantaco/vouch/cmd/vouch@latest
```

From this checkout:

```sh
go install ./cmd/vouch
```

## Docs

Published docs:

https://duriantaco.github.io/vouch/

The site is deployed from `.github/workflows/pages.yml` after changes land on
`main`. The source lives in `docs/site/` and is built with MkDocs.

- [Compiler Architecture](docs/COMPILER.md)
- [GitHub Actions](docs/GITHUB_ACTIONS.md)
- [Benchmarks](docs/BENCHMARKS.md)
- [Comparison](COMPARISON.md)
- [Roadmap](ROADMAP.md)
- [Agent/product operating brief](skills.md)
- [Contributing](CONTRIBUTING.md)

## Status

Vouch is beta infrastructure. It is ready for shadow-mode pilots, not blind
production enforcement.

The local VouchBench harness proves the current compiler/evidence/policy path is
deterministic over fixture scenarios. It does not prove arbitrary code is correct
or that real teams will adopt the workflow.

```sh
scripts/vouchbench.sh
```

## Development

```sh
go test ./...
```

## License

Apache-2.0. See [LICENSE](LICENSE).
