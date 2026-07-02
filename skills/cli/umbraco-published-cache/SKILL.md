---
name: umbraco-published-cache
description: "Published content cache operations"
metadata:
  version: 0.4.6
  requires:
    bins:
      - umbraco
    skills:
      - umbraco-shared
---

# published-cache

> **PREREQUISITE:** Read `../umbraco-shared/SKILL.md` for auth, global flags, and security rules.

```bash
umbraco published-cache <command> [flags]
```

## Read Commands

| Command | Description |
|---------|-------------|
| `published-cache status` | Get published cache rebuild status |

### status

```bash
umbraco published-cache status
```

## Mutation Commands

> **Safety:** Always use `--dry-run` first. Remove the flag only after verifying the dry-run output.

| Command | Description |
|---------|-------------|
| `published-cache rebuild` | Rebuild the published content cache from the database |
| `published-cache reload` | Reload the in-memory published cache |

### rebuild

```bash
umbraco published-cache rebuild
```

POST /published-cache/rebuild. Rebuilds the published content cache from the database — the standard fix for stale published content. Expensive on large sites; poll 'published-cache status' to see when the rebuild finishes.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--force` | bool | false | Confirm the rebuild |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco published-cache rebuild --dry-run

# 2. Execute
umbraco published-cache rebuild
```

### reload

```bash
umbraco published-cache reload
```

POST /published-cache/reload. Reloads the in-memory published cache from the cache store without a database rebuild; much cheaper than 'published-cache rebuild'.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco published-cache reload --dry-run

# 2. Execute
umbraco published-cache reload
```

## Discovering Commands

```bash
# Browse subcommands
umbraco published-cache --help

# Inspect a specific endpoint schema
umbraco schema published-cache.<method>
```
