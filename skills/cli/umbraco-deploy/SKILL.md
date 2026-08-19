---
name: umbraco-deploy
description: "Effect-based deployment observation (watch an environment, not a pipeline)"
metadata:
  version: 0.4.9
  requires:
    bins:
      - umbraco
    skills:
      - umbraco-shared
---

# deploy

> **PREREQUISITE:** Read `../umbraco-shared/SKILL.md` for auth, global flags, and security rules.

```bash
umbraco deploy <command> [flags]
```

## Read Commands

| Command | Description |
|---------|-------------|
| `deploy watch` | Watch an environment for the effects of a deployment and report phase transitions |

### watch

```bash
umbraco deploy watch
```

Observes the target environment for state deltas only a deployment can cause — no pipeline or portal API involved, so it works identically on Umbraco Cloud and on-prem, and it is strictly read-only.

Signals: the newest log entry's ProcessId/MachineName (an app recycle means the deploy landed), the management token endpoint probed unauthenticated (503/unreachable = down; 401 = app alive and rejecting the probe — the earliest all-clear, typically ~15s before public pages return), configured health paths on the public host, and Examine index health (deploys can trigger full index rebuilds, during which search is empty — "deploy succeeded" and "the site works" are different questions).

Phases: baseline → restarting → app-alive → serving → landed → verified | failed | timeout. Everything is baselined before arming — a signal already true on the target is not a signal. Transitions are emitted with timestamps as they are observed (fast recycles may skip phases); silence between transitions means "still in the current phase", and --heartbeat writes a periodic still-alive line to stderr so silence is never ambiguous. Success is never inferred from silence: reaching --timeout without verification exits 6 (status unknown), and sustained downtime or post-landing health failure beyond --escalation exits 5.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--escalation` | duration | 10m0s | Treat sustained downtime or post-landing health failure longer than this as failed (exit 5) |
| `--health-path` | stringArray | [] | Public path that must return 2xx for the serving/verified phases (repeatable; default /) |
| `--heartbeat` | duration | 1m0s | Interval for still-alive lines on stderr; 0 disables |
| `--interval` | duration | 5s | Poll interval |
| `--json` | bool | false | Emit phase transitions as NDJSON |
| `--public-url` | string | — | Public host for health paths when it differs from the management base URL |
| `--skip-index-verify` | bool | false | Do not require Examine indexes to be healthy for the verified phase |
| `--timeout` | duration | 30m0s | Give up after this long without verification (exit 6, status unknown) |

## Discovering Commands

```bash
# Browse subcommands
umbraco deploy --help

# Inspect a specific endpoint schema
umbraco schema deploy.<method>
```
