---
name: umbraco-api
description: "Call an authenticated raw Umbraco Management API endpoint"
metadata:
  version: 0.4.12
  requires:
    bins:
      - umbraco
    skills:
      - umbraco-shared
---

# api

> **PREREQUISITE:** Read `../umbraco-shared/SKILL.md` for auth, global flags, and security rules.

```bash
umbraco api <method> <path> [flags]
```

## Command

### api

```bash
umbraco api <method> <path>
```

Call a core Umbraco Management API endpoint that does not have a curated CLI command yet.

Pass paths relative to /umbraco/management/api/v1, for example /item/document/ancestors?id=a&id=b.
Full Management API paths are also accepted and normalized to the core API root.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--body` | string | — | JSON request body, or @path to read JSON from a file |
| `--dry-run` | bool | false | Print the planned request without executing |

**Safe pattern:**

```bash
# 1. Rehearse with the exact flags you will execute with
umbraco api <method> <path> [flags] --dry-run

# 2. Execute with the same flags
umbraco api <method> <path> [flags]
```

## Discovering Commands

```bash
umbraco api --help
```
