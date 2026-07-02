---
name: umbraco-media
description: "Media asset operations"
metadata:
  version: 0.4.6
  requires:
    bins:
      - umbraco
    skills:
      - umbraco-shared
---

# media

> **PREREQUISITE:** Read `../umbraco-shared/SKILL.md` for auth, global flags, and security rules.

```bash
umbraco media <command> [flags]
```

## Read Commands

| Command | Description |
|---------|-------------|
| `media are-referenced` | Bulk check: which of these media IDs are referenced by something |
| `media bin children <id>` | List children of a trashed media item |
| `media bin list` | List media items at the recycle bin root |
| `media bin original-parent <id>` | Get the original parent of a trashed media item (the default restore target) |
| `media children <id>` | Get child media items (paginated; --skip/--take/--all) |
| `media get <id>` | Get media by ID |
| `media referenced-descendants <id>` | List items that reference this media item or any of its descendants |
| `media references <id>` | List items that reference this media item (paginated; --skip/--take/--all) |
| `media root` | Get root media items (paginated; --skip/--take/--all) |
| `media search` | Search media items |
| `media urls <id>` | Get media URLs |

### are-referenced

```bash
umbraco media are-referenced
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--ids` | string | — | Comma-separated media GUIDs to check (required) |

### bin children

```bash
umbraco media bin children <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### bin list

```bash
umbraco media bin list
```

GET /recycle-bin/media/root. Paginated; use 'bin children <id>' to descend into trashed subtrees.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### bin original-parent

```bash
umbraco media bin original-parent <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |

### children

```bash
umbraco media children <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### get

```bash
umbraco media get <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |

### referenced-descendants

```bash
umbraco media referenced-descendants <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### references

```bash
umbraco media references <id>
```

Wraps GET /media/{id}/referenced-by. Same content-audit role as 'document references' for media assets.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### root

```bash
umbraco media root
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Walk every page until exhausted (auto-paginates with --take as the page size, default 500; combine with --skip to start partway through). Bounded by an internal 100k-item ceiling. |
| `--fields` | string | — | Limit response fields (comma-separated top-level keys) |
| `--first-n` | int | 0 | Return only the first N items from item collections |
| `--ids-only` | bool | false | Return only item IDs for item collections |
| `--params` | string | — | Query parameters as JSON |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--summarize` | bool | false | Return only id/name/alias fields for item collections |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### search

```bash
umbraco media search
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--params` | string | — | Search parameters as JSON; convenience flags fill in missing keys, --params wins on collisions |
| `--query` | string | — | Search query |
| `--skip` | int | -1 | Skip count (passes through as ?skip=N; lets you walk past the server page size on large children/root collections) |
| `--take` | int | -1 | Take count (passes through as ?take=N; combine with --skip to page) |

### urls

```bash
umbraco media urls <id>
```

## Mutation Commands

> **Safety:** Always use `--dry-run` first. Remove the flag only after verifying the dry-run output.

| Command | Description |
|---------|-------------|
| `media bin delete <id>` | Permanently delete one media item from the recycle bin |
| `media bin empty` | Permanently delete everything in the media recycle bin |
| `media create` | Create media from JSON payload |
| `media create-folder [name]` | Create media folder |
| `media move <id>` | Move media item |
| `media trash <id>` | Move media item to recycle bin |
| `media update <id>` | Update media item |
| `media upload <file>` | Upload a file and create a media item |

### bin delete

```bash
umbraco media bin delete <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--force` | bool | false | Confirm permanent deletion |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media bin delete <id> --dry-run

# 2. Execute
umbraco media bin delete <id> --force
```

### bin empty

```bash
umbraco media bin empty
```

DELETE /recycle-bin/media. Destroys every trashed media item; there is no undo.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--force` | bool | false | Confirm emptying the recycle bin |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media bin empty --dry-run

# 2. Execute
umbraco media bin empty --force
```

### create

```bash
umbraco media create
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--json` | string | — | Create payload as JSON |
| `--print-template` | bool | false | Print an annotated JSON skeleton; substitute placeholders before passing to --json |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media create --dry-run

# 2. Execute
umbraco media create
```

### create-folder

```bash
umbraco media create-folder [name]
```

Folders are regular media items of the built-in Folder type, so this resolves the Folder media type and POSTs /media with a variants envelope. --json passes a full media create payload through verbatim.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--json` | string | — | Full media create payload as JSON (bypasses Folder-type resolution) |
| `--parent` | string | — | Target parent media ID (omit for a root-level folder) |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media create-folder [name] --dry-run

# 2. Execute
umbraco media create-folder [name]
```

### move

```bash
umbraco media move <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--json` | string | — | Action payload as JSON |
| `--to` | string | — | Target parent ID shortcut for {"target":{"id":...}} |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media move <id> --dry-run

# 2. Execute
umbraco media move <id>
```

### trash

```bash
umbraco media trash <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media trash <id> --dry-run

# 2. Execute
umbraco media trash <id>
```

### update

```bash
umbraco media update <id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Print the planned request without executing |
| `--json` | string | — | Full replacement payload as JSON (fields not mentioned are reset by the server) |
| `--merge-json` | string | — | Partial JSON deep-merged into the current resource before update (fields not mentioned are preserved) |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media update <id> --dry-run

# 2. Execute
umbraco media update <id>
```

### upload

```bash
umbraco media upload <file>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--culture` | string | — | Culture code for culture-varying media types |
| `--dry-run` | bool | false | Print the planned request without executing |
| `--name` | string | — | Media item name (defaults to file name without extension) |
| `--parent` | string | — | Target parent media ID |
| `--property` | string | umbracoFile | File property alias |
| `--type` | string | — | Media type id, alias, or name |

**Safe pattern:**

```bash
# 1. Dry run first
umbraco media upload <file> --dry-run

# 2. Execute
umbraco media upload <file>
```

## Discovering Commands

```bash
# Browse subcommands
umbraco media --help

# Inspect a specific endpoint schema
umbraco schema media.<method>
```
