# Umbraco CLI (Agent-First)

A Go-based CLI for the Umbraco Management API, designed for **agents first, humans second**.

Core behavior:
- `--json` and `--params` are primary machine inputs
- `--fields` keeps responses small for context window discipline
- `--dry-run` validates mutating operations before execution
- `umbraco schema ...` provides runtime schema introspection
- JSON output is default when output is piped

## Requirements

- Go `1.26+`
- Node.js `20+` (only needed for skills verification scripts)
- Access to an Umbraco instance with Management API credentials

## Setup

1. Clone the repository and enter the project directory.
2. Configure environment variables (use `.env.example` as reference):

```bash
export UMBRACO_BASE_URL="https://localhost:44391"
export UMBRACO_CLIENT_ID="umbraco-back-office-api-user"
export UMBRACO_CLIENT_SECRET="your-secret"
# Optional local dev TLS bypass
export NODE_TLS_REJECT_UNAUTHORIZED=0
```

Notes:
- The Go CLI reads environment variables from the shell.
- `.env.example` is a template; it is not auto-loaded by the Go runtime.

3. Build and test:

```bash
go test ./...
go build ./...
```

4. Run directly:

```bash
go run ./cmd/umbraco --help
```

Optional binary build:

```bash
go build -o ./bin/umbraco ./cmd/umbraco
./bin/umbraco --help
```

## First Commands

Schema introspection:

```bash
go run ./cmd/umbraco schema --list
go run ./cmd/umbraco schema document.create
go run ./cmd/umbraco schema document
```

Safe read:

```bash
go run ./cmd/umbraco document get <id> --fields "id,name,updateDate"
```

Safe write pattern (always dry-run first):

```bash
go run ./cmd/umbraco document publish <id> --json '{"cultures":["en-US"]}' --dry-run --output json
# then run without --dry-run
```

## Skills Bundle

This repo includes 66 bundled Umbraco skills under `skills/`.

Verify bundle integrity:

```bash
npm run verify:skills
```

## Project Commands

- `go test ./...` - run tests
- `go build ./...` - build all packages
- `go run ./cmd/umbraco ...` - run CLI
- `npm run verify:skills` - verify skills count and structure

## Collections in MVP

- `document` (15)
- `dictionary` (6)
- `media` (10)
- `doctype` (10)
- `datatype` (8)
- `template` (6)
- `logs` (5)
- `server` (5)
- `health` (4)

Total: **69 commands**

## Agent Safety Rules

- Use `--dry-run` first for all mutating commands.
- Use `--fields` on reads to limit response size.
- Prefer `--json` payloads to avoid lossy argument mapping.
- Do not construct IDs manually; reuse IDs returned by API responses.
