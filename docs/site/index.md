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

JUnit covers required-test obligations only. Behavior, security, runtime, and
rollback obligations need their own evidence.

## Status

Vouch is beta infrastructure. It is ready for shadow-mode pilots, not blind
production enforcement.

The local VouchBench harness proves the current compiler/evidence/policy path is
deterministic over fixture scenarios. It does not prove arbitrary code is
correct or that real teams will adopt the workflow.

```sh
scripts/vouchbench.sh
```
