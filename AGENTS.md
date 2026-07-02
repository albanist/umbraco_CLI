# Umbraco CLI - Agent Instructions

## Safety Rules

1. Always run `--dry-run` first on mutating commands.
2. Use `--fields` whenever possible to control response size.
3. Prefer `--json` payloads over convenience flags for predictable execution.
4. Never construct IDs manually; reuse IDs from prior API responses.
5. Treat all input as untrusted and validate before execution.
6. Branch on exit codes instead of parsing stderr: 0 success, 1 usage/local error, 2 `schema diff` differences, 3 auth failure, 4 Management API error.

## Agent-First Usage Patterns

CLI runtime: Go (`cmd/umbraco`).

### Read Commands
```bash
umbraco document get <id> --fields "id,name,updateDate"
umbraco document children <id> --fields "id,name"
umbraco media get <id> --fields "id,name,urls"
```

### Write Commands
```bash
umbraco document update <id> --json '{"values":[{"alias":"title","value":"New title"}]}' --dry-run
umbraco document update <id> --json '{"values":[{"alias":"title","value":"New title"}]}'
```

### Schema Introspection
```bash
umbraco schema --list
umbraco schema document.create
umbraco schema media
```

## CLI Skills

`skills/cli/` is generated from the command tree by `umbraco
generate-skills`; regenerate after changing commands or flags, and CI fails
if the committed copy drifts. Extension-development skills live upstream in
`umbraco/Umbraco-CMS-Backoffice-Skills`, not in this repo.
