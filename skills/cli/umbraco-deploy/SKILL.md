---
name: umbraco-deploy
description: "Effect-based deployment observation (watch an environment, not a pipeline)"
metadata:
  version: 0.4.10
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
| `deploy status` | Compare local .uda deploy artifacts against the environment, read-only |
| `deploy watch` | Watch an environment for the effects of a deployment and report phase transitions |

### status

```bash
umbraco deploy status
```

Reads the Umbraco Deploy artifacts in --uda-dir (the site repo's umbraco/Deploy/Revision) and compares each against the target environment's database via the Management API, reporting in-sync vs drifted per entity. Strictly read-only: a pre-flight check that turns "will this deploy blow up or carry surprises?" into an answerable question — in-sync artifacts are skipped by Deploy's schema pass and are therefore safe; drifted ones are processed.

Comparison is per entity kind (data types, document/media/member types, templates, containers, member groups, relation types) over the fields the artifact carries; environment-only additions like migration markers are ignored. Automate artifacts degrade to status "unknown" where the Automate API is unreachable (Cloud basic auth blocks package APIs on non-live environments) — never a false in-sync — but their step aliases are still read locally, and --flag-step-alias marks automations carrying aliases you know your Deploy version cannot validate (configuration, not encoded knowledge: those landmines change as bugs are fixed).

Exit 7 when drift or missing entities are found (suppress with --exit-zero); parse failures and unreachable comparisons are reported per artifact, never silently dropped. The report is stdout; the drift summary line is an error and goes to stderr, so -o json stdout stays parseable — do not merge the streams with 2>&1 if you parse the output.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--concurrency` | int | 8 | Maximum concurrent environment lookups |
| `--exit-zero` | bool | false | Exit 0 even when drift or missing entities are found |
| `--flag-step-alias` | stringArray | [] | Flag automations whose steps carry this action-alias substring (repeatable; e.g. a control-flow alias your Deploy version fails to validate) |
| `--kind` | stringArray | [] | Only compare these artifact kinds (Udi entity types, e.g. data-type, document-type; repeatable) |
| `--uda-dir` | string | umbraco/Deploy/Revision | Directory holding the .uda artifacts |

### watch

```bash
umbraco deploy watch
```

Observes the target environment for state deltas only a deployment can cause — no pipeline or portal API involved, so it works identically on Umbraco Cloud and on-prem, and it is strictly read-only.

Signals: the newest log entry's ProcessId/MachineName (an app recycle means the deploy landed), the management token endpoint probed unauthenticated (503/unreachable = down; 401 = app alive and rejecting the probe — the earliest all-clear, typically ~15s before public pages return), configured health paths on the public host, and Examine index health (deploys can trigger full index rebuilds, during which search is empty — "deploy succeeded" and "the site works" are different questions).

Phases: baseline → restarting → app-alive → serving → landed → settling → verified | failed | timeout. Everything is baselined before arming — a signal already true on the target is not a signal. Verified requires the environment to stay healthy for a full --settle window after everything first looks good: deployment pipelines can disturb the environment AFTER the app is already serving (observed in production: Umbraco Deploy wiped every Examine index 27 seconds after a single-sample check had passed, leaving search empty for 17 minutes), so a single passing sample is not verification. An interrupted settle (index rebuild, health flap) is emitted as settle-interrupted and the window restarts once the environment recovers. Transitions are emitted with timestamps as they are observed (fast recycles may skip phases); silence between transitions means "still in the current phase", and --heartbeat writes a periodic still-alive line to stderr so silence is never ambiguous. Success is never inferred from silence: reaching --timeout without verification exits 6 (status unknown), and sustained downtime or post-landing health failure beyond --escalation exits 5.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--escalation` | duration | 10m0s | Treat sustained downtime or post-landing health failure longer than this as failed (exit 5) |
| `--health-path` | stringArray | [] | Public path that must return 2xx for the serving/verified phases (repeatable; default /) |
| `--heartbeat` | duration | 1m0s | Interval for still-alive lines on stderr; 0 disables |
| `--interval` | duration | 5s | Poll interval |
| `--json` | bool | false | Emit phase transitions as NDJSON |
| `--public-url` | string | — | Public host for health paths when it differs from the management base URL |
| `--settle` | duration | 1m30s | How long the environment must stay healthy after everything first looks good before verified is emitted; 0 disables (single-sample verification) |
| `--skip-index-verify` | bool | false | Do not require Examine indexes to be healthy for the verified phase |
| `--timeout` | duration | 30m0s | Give up after this long without verification (exit 6, status unknown) |

## Discovering Commands

```bash
# Browse subcommands
umbraco deploy --help

# Inspect a specific endpoint schema
umbraco schema deploy.<method>
```
