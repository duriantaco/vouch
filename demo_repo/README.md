# Demo Repo

This directory is a tiny fixture for the Vouch compiler MVP.

It contains one high-risk feature contract and three agent-change manifests:

- `.vouch/intents/auth.password_reset.yaml`
- `.vouch/specs/auth.password_reset.json`
- `.vouch/manifests/blocked.json`
- `.vouch/manifests/pass.json`
- `.vouch/manifests/traceability-blocked.json`

Run from the parent directory:

```sh
vouch intent parse --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.ast.json
vouch intent compile --intent demo_repo/.vouch/intents/auth.password_reset.yaml --out /tmp/auth.password_reset.json
vouch ir build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/auth.password_reset.ir.json
vouch plan build --spec demo_repo/.vouch/specs/auth.password_reset.json --manifest demo_repo/.vouch/manifests/pass.json --out /tmp/auth.password_reset.plan.json
vouch artifacts build --spec demo_repo/.vouch/specs/auth.password_reset.json --out /tmp/vouch-artifacts
vouch --repo demo_repo spec lint
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/blocked.json evidence
vouch --repo demo_repo --manifest demo_repo/.vouch/manifests/pass.json evidence
```

Expected gate results:

| Manifest | Decision |
| --- | --- |
| `blocked.json` | `block` |
| `pass.json` | `canary` |
