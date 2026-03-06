# Umbraco CLI Context

## What This Is

`umbraco` is an agent-first command-line wrapper around the Umbraco Management API.
Implementation runtime is Go (`cmd/umbraco`).

## Core Principles

- Primary input path is raw JSON (`--json`) and structured query JSON (`--params`).
- Schema is introspectable at runtime (`umbraco schema ...`).
- Response size should be constrained (`--fields`) to protect context window budget.
- Mutations must be rehearsed first (`--dry-run`).

## Quick Command Reference

### Content
- `umbraco document get <id>`
- `umbraco document root`
- `umbraco document children <id>`
- `umbraco document search --params '{"query":"home"}'`

### Media
- `umbraco media get <id>`
- `umbraco media root`
- `umbraco media children <id>`

### Schema
- `umbraco doctype get <id>`
- `umbraco datatype list`
- `umbraco schema document.update`

### Diagnostics
- `umbraco server status`
- `umbraco logs list --level Error --take 50`
- `umbraco health groups`

## Local Dev Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Run: `go run ./cmd/umbraco --help`
