# Umbraco CLI - MVP Specification

## Overview

A CLI that wraps the Umbraco Management API. **Built agent-first.**

> "Human DX optimizes for discoverability and forgiveness. Agent DX optimizes for predictability and defense-in-depth." - Justin Poehnelt

**Design Principles:**
1. **Raw JSON payloads over bespoke flags** - Agents prefer `--json` over 10 custom flags
2. **Schema introspection at runtime** - Agents self-serve, no docs in context
3. **Context window discipline** - `--fields` limits response size
4. **Input hardening** - Agents hallucinate, validate everything
5. **Dry-run by default thinking** - `--dry-run` before mutating

**Bundled with:**
- 63 commands across 8 collections
- Schema introspection (`umbraco schema <endpoint>`)
- 66 Umbraco backoffice skills (from Umbraco-CMS-Backoffice-Skills)
- AGENTS.md with safety rules

---

## Architecture

```
umbraco <collection> <command> [options]
     │
     ▼
  Input Validation (path traversal, control chars, etc.)
     │
     ▼
  Umbraco Management API
     │
     ▼
  Umbraco Backoffice
```

**Tech Stack:** Node.js, TypeScript, Commander.js

---

## Project Structure

```
umbraco-cli/
├── src/
│   ├── index.ts
│   ├── config.ts
│   ├── auth.ts
│   ├── api.ts
│   ├── output.ts
│   ├── validate.ts              # Input validation
│   └── commands/
│       ├── document.ts
│       ├── media.ts
│       ├── doctype.ts
│       ├── datatype.ts
│       ├── template.ts
│       ├── logs.ts
│       ├── server.ts
│       └── health.ts
├── skills/                       # Bundled Umbraco backoffice skills
│   ├── foundation/
│   │   ├── umbraco-context-api/SKILL.md
│   │   ├── umbraco-repository-pattern/SKILL.md
│   │   ├── umbraco-extension-registry/SKILL.md
│   │   ├── umbraco-conditions/SKILL.md
│   │   ├── umbraco-state-management/SKILL.md
│   │   ├── umbraco-localization/SKILL.md
│   │   ├── umbraco-routing/SKILL.md
│   │   ├── umbraco-notifications/SKILL.md
│   │   ├── umbraco-umbraco-element/SKILL.md
│   │   └── umbraco-controllers/SKILL.md
│   ├── extensions/
│   │   ├── umbraco-sections/SKILL.md
│   │   ├── umbraco-menu/SKILL.md
│   │   ├── umbraco-dashboard/SKILL.md
│   │   ├── umbraco-workspace/SKILL.md
│   │   ├── umbraco-tree/SKILL.md
│   │   ├── umbraco-collection/SKILL.md
│   │   ├── umbraco-entity-actions/SKILL.md
│   │   ├── umbraco-modals/SKILL.md
│   │   ├── umbraco-icons/SKILL.md
│   │   ├── umbraco-search-provider/SKILL.md
│   │   └── ... (30 total)
│   ├── property-editors/
│   │   ├── umbraco-property-editor-ui/SKILL.md
│   │   ├── umbraco-property-editor-schema/SKILL.md
│   │   └── ... (6 total)
│   ├── rich-text/
│   │   ├── umbraco-tiptap-extension/SKILL.md
│   │   └── ... (4 total)
│   ├── backend/
│   │   ├── umbraco-openapi-client/SKILL.md
│   │   ├── umbraco-auth-provider/SKILL.md
│   │   └── ... (4 total)
│   └── testing/
│       ├── umbraco-testing/SKILL.md
│       ├── umbraco-unit-testing/SKILL.md
│       ├── umbraco-e2e-testing/SKILL.md
│       └── ... (8 total)
├── AGENTS.md                     # Agent safety rules + skill index
├── CONTEXT.md                    # CLI context for agents
├── package.json
├── tsconfig.json
└── .env.example
```

---

## Configuration

```env
UMBRACO_BASE_URL=https://localhost:44391
UMBRACO_CLIENT_ID=umbraco-back-office-api-user
UMBRACO_CLIENT_SECRET=your-secret-here
NODE_TLS_REJECT_UNAUTHORIZED=0
```

---

## Global CLI Options

```
umbraco <collection> <command> [options]

Primary (Agent-First):
  --json <payload>         Raw JSON payload (maps directly to API)
  --params <json>          Query parameters as JSON
  --fields <fields>        Limit response fields (context window discipline)
  --output json            Machine-readable output (default when piped)
  --dry-run                Validate without executing

Secondary (Human Convenience):
  -o, --output <format>    json | table | plain (default: plain for TTY)
  -h, --help               Show help
  --version                Show version
```

**Agent-first means:**
- `--json` is the primary input method, convenience flags are secondary
- Output is JSON by default when stdout is not a TTY
- Schema is queryable at runtime via `umbraco schema`

---

## Schema Introspection

Agents can't google docs. The CLI is the documentation.

```bash
# Get schema for any endpoint
umbraco schema document.create
umbraco schema document.publish
umbraco schema media.upload

# List all available endpoints
umbraco schema --list

# Get schema for a collection
umbraco schema document
```

**Output:**
```json
{
  "endpoint": "document.create",
  "method": "POST",
  "path": "/umbraco/management/api/v1/document",
  "requestBody": {
    "type": "object",
    "required": ["documentType", "parent"],
    "properties": {
      "documentType": { "type": "object", "properties": { "id": { "type": "string" } } },
      "parent": { "type": "object", "properties": { "id": { "type": "string" } } },
      "values": { "type": "array", "items": { ... } },
      "variants": { "type": "array", "items": { ... } }
    }
  },
  "response": { ... }
}
```

This replaces static documentation. The agent introspects what the API accepts *right now*.

---

## Agent-First Patterns

### 1. Raw JSON Payloads > Bespoke Flags

Humans hate writing JSON in terminal. Agents prefer it.

```bash
# Human-first (convenience flags, lossy, can't express nested structures)
umbraco document set-property abc-123 --alias title --value "New Title" --culture en-US

# Agent-first (raw payload, maps directly to API, zero translation loss)
umbraco document update abc-123 --json '{
  "values": [{"alias": "title", "value": "New Title", "culture": "en-US"}]
}'
```

The `--json` flag accepts the full API payload. No custom argument layers between agent and API.

**Design tension:** Human ergonomics vs agent ergonomics. Solution: support both paths. Raw JSON is primary, convenience flags are secondary.

### 2. Schema Introspection Replaces Documentation

Static docs in system prompt = expensive tokens + goes stale.

```bash
# Agent self-serves schema at runtime
umbraco schema document.create
umbraco schema document.update
```

The CLI becomes the canonical source of truth for what the API accepts *right now*.

### 3. Context Window Discipline

APIs return massive blobs. Agents pay per token.

```bash
# Without field mask - returns everything (kills context window)
umbraco document get abc-123

# With field mask - returns only what you need
umbraco document get abc-123 --fields "id,name,updateDate"
```

### 4. Input Hardening Against Hallucinations

Humans typo. Agents hallucinate. Different failure modes.

```typescript
// src/validate.ts

// Agents hallucinate path traversals
export function validatePath(path: string): void {
  const normalized = path.replace(/\\/g, '/');
  if (normalized.includes('..') || normalized.startsWith('/')) {
    throw new Error(`Invalid path: ${path}`);
  }
}

// Agents generate invisible control characters
export function validateString(str: string): void {
  if (/[\x00-\x1F\x7F]/.test(str)) {
    throw new Error('Input contains control characters');
  }
}

// Agents embed query params in IDs (fileId?fields=name)
export function validateResourceId(id: string): void {
  if (/[?#%]/.test(id)) {
    throw new Error(`Invalid resource ID: ${id}`);
  }
}

// Agents pre-encode strings that get double-encoded
export function validateNoPreEncoding(str: string): void {
  if (/%[0-9A-Fa-f]{2}/.test(str)) {
    throw new Error('Input appears to be pre-encoded');
  }
}

export function validateInput(input: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(input)) {
    if (typeof value === 'string') {
      validateString(value);
      validateNoPreEncoding(value);
      if (key.toLowerCase().includes('id')) {
        validateResourceId(value);
      }
      if (key.toLowerCase().includes('path')) {
        validatePath(value);
      }
    }
  }
}
```

**The agent is not a trusted operator.** Validate everything.

### 5. Dry-Run for Mutating Operations

Agents can "think out loud" before acting.

```bash
# Validate without executing
umbraco document publish abc-123 --dry-run

# Output shows what would happen
{
  "dryRun": true,
  "valid": true,
  "method": "POST",
  "path": "/document/abc-123/publish",
  "body": {}
}

# Then execute for real
umbraco document publish abc-123
```

### 6. JSON Output by Default (for pipes)

```bash
# TTY - human-readable
umbraco document root

# Piped - JSON automatically
umbraco document root | jq '.items[0]'
```

```typescript
// Detect if stdout is TTY
const defaultFormat = process.stdout.isTTY ? 'plain' : 'json';
```

---

## AGENTS.md

This file ships with the CLI and tells agents how to use it safely.

```markdown
# Umbraco CLI - Agent Instructions

## Safety Rules

1. **Always use --dry-run first** for any write operation.
   - Run with --dry-run to validate
   - Review the output
   - Only then run without --dry-run

2. **Always use --fields** to limit response size.
   - Large responses consume your context window
   - Request only the fields you need

3. **Never construct IDs from user input** without validation.
   - IDs should come from previous API responses
   - Do not concatenate or modify IDs

4. **Agent-specific restrictions** are enforced in the agent harness, not the CLI.
   - If your harness restricts you to read-only, do not attempt write commands
   - Check your SOUL.md or AGENTS.md for your specific boundaries

## Command Patterns

### Reading Content
```bash
# Get document with minimal fields
umbraco document get <id> --fields "id,name,properties"

# List children
umbraco document children <id> --fields "id,name"
```

### Writing Content
```bash
# Always dry-run first
umbraco document set-property <id> --alias title --value "New Title" --dry-run

# Then execute
umbraco document set-property <id> --alias title --value "New Title"
```

### Debugging
```bash
# Check server status
umbraco server status

# View recent logs
umbraco logs list --level Error --take 50

# Run health checks
umbraco health run "Security"
```

## Skills Reference

This CLI bundles 66 Umbraco backoffice skills for extension development.

### Foundation (10 skills)
- umbraco-context-api - Provider-consumer pattern
- umbraco-repository-pattern - Data access layer
- umbraco-extension-registry - Dynamic registration
- umbraco-conditions - Control where extensions appear
- umbraco-state-management - Reactive UI
- umbraco-localization - Multi-language
- umbraco-routing - URL structure
- umbraco-notifications - Toast messages
- umbraco-umbraco-element - Base class
- umbraco-controllers - C# API endpoints

### Extension Types (30 skills)
Navigation: umbraco-sections, umbraco-menu, umbraco-header-apps
Content: umbraco-dashboard, umbraco-workspace, umbraco-tree, umbraco-collection
Actions: umbraco-entity-actions, umbraco-entity-bulk-actions
UI: umbraco-modals, umbraco-icons, umbraco-theme
Search: umbraco-search-provider, umbraco-search-result-item

### Property Editors (6 skills)
- umbraco-property-editor-ui
- umbraco-property-editor-schema
- umbraco-property-action
- umbraco-property-value-preset
- umbraco-file-upload-preview
- umbraco-block-editor-custom-view

### Rich Text (4 skills)
- umbraco-tiptap-extension
- umbraco-tiptap-toolbar-extension
- umbraco-tiptap-statusbar-extension
- umbraco-monaco-markdown-editor-action

### Backend (4 skills)
- umbraco-openapi-client
- umbraco-auth-provider
- umbraco-mfa-login-provider
- umbraco-granular-user-permissions

### Testing (8 skills)
- umbraco-testing - Router skill
- umbraco-unit-testing
- umbraco-mocked-backoffice
- umbraco-e2e-testing
- umbraco-playwright-testhelpers
- umbraco-test-builders
- umbraco-msw-testing
- umbraco-example-generator

## When to Use Skills vs CLI

- **CLI**: Execute operations against existing Umbraco instance
- **Skills**: Build new backoffice extensions (dashboards, trees, property editors)

Use CLI for: content management, schema queries, debugging, auditing
Use Skills for: building custom UI, extending the backoffice, creating property editors
```

---

## CONTEXT.md

Minimal context for agents to understand the CLI.

```markdown
# Umbraco CLI Context

## What This Is
CLI for Umbraco CMS Management API. Execute backoffice operations from terminal.

## Current Mode
Check with: `umbraco server status`
- Readonly: Can only read data
- Full: Can read and write

## Quick Reference

### Content
umbraco document get <id>
umbraco document root
umbraco document children <id>
umbraco document search <query>

### Media
umbraco media get <id>
umbraco media root
umbraco media children <id>

### Schema
umbraco doctype get <id>
umbraco doctype list
umbraco datatype get <id>

### Debugging
umbraco server status
umbraco logs list --level Error
umbraco health groups

## Important
- Use --fields to limit response size
- Use --dry-run before write operations
- Use --output json for machine-readable output
```

---

## MVP Collections (8)

| Collection | Purpose | Commands |
|------------|---------|----------|
| document | Content management | 15 |
| media | Asset management | 10 |
| doctype | Content schema | 10 |
| datatype | Property editors | 8 |
| template | Razor templates | 6 |
| logs | Debugging | 5 |
| server | Status & info | 5 |
| health | Health checks | 4 |
| **Total** | | **63** |

---

## 1. Document (`document`)

**Agent-first:** Use `--json` for full control. Convenience flags available for simple cases.

```bash
# ============================================
# READ
# ============================================

umbraco document get <id> [--fields <f>]
umbraco document root [--fields <f>] [--params <json>]
umbraco document children <id> [--fields <f>]
umbraco document ancestors <id>
umbraco document search --params '{"query": "...", "skip": 0, "take": 20}'

# ============================================
# WRITE (Agent-first: --json is primary)
# ============================================

# Create - full payload
umbraco document create --json '{
  "documentType": {"id": "..."},
  "parent": {"id": "..."},
  "values": [{"alias": "title", "value": "Hello"}],
  "variants": [{"culture": "en-US", "name": "My Page"}]
}' [--dry-run]

# Update - full payload
umbraco document update <id> --json '{
  "values": [{"alias": "title", "value": "Updated"}],
  "variants": [{"culture": "en-US", "name": "Updated Page"}]
}' [--dry-run]

# Update properties - partial (agent-friendly shortcut)
umbraco document update-properties <id> --json '{
  "properties": [
    {"alias": "title", "value": "New Title", "culture": "en-US"},
    {"alias": "body", "value": "<p>Content</p>"}
  ]
}' [--dry-run]

# Publish
umbraco document publish <id> [--json '{"cultures": ["en-US"]}'] [--dry-run]

# Unpublish
umbraco document unpublish <id> [--json '{"cultures": ["en-US"]}'] [--dry-run]

# Copy
umbraco document copy <id> --json '{"target": {"id": "parent-id"}}' [--dry-run]

# Move
umbraco document move <id> --json '{"target": {"id": "parent-id"}}' [--dry-run]

# Delete / Trash / Restore
umbraco document delete <id> [--dry-run]
umbraco document trash <id> [--dry-run]
umbraco document restore <id> [--dry-run]

# ============================================
# CONVENIENCE FLAGS (Human-friendly shortcuts)
# ============================================

# These map to --json internally
umbraco document publish <id> --culture en-US [--dry-run]
umbraco document copy <id> --to <parent-id> [--dry-run]
umbraco document move <id> --to <parent-id> [--dry-run]
```

---

## 2. Media (`media`)

```bash
# ============================================
# READ
# ============================================

umbraco media get <id> [--fields <f>]
umbraco media root [--fields <f>]
umbraco media children <id> [--fields <f>]
umbraco media urls <id>

# ============================================
# WRITE (Agent-first: --json is primary)
# ============================================

# Create from file path
umbraco media create --json '{
  "mediaType": {"id": "..."},
  "parent": {"id": "..."},
  "name": "my-image.jpg",
  "source": {"type": "filePath", "path": "/path/to/file.jpg"}
}' [--dry-run]

# Create from URL
umbraco media create --json '{
  "mediaType": {"id": "..."},
  "parent": {"id": "..."},
  "name": "downloaded.jpg",
  "source": {"type": "url", "url": "https://example.com/image.jpg"}
}' [--dry-run]

# Create folder
umbraco media create-folder --json '{
  "name": "My Folder",
  "parent": {"id": "..."}
}' [--dry-run]

# Update
umbraco media update <id> --json '{...}' [--dry-run]

# Move
umbraco media move <id> --json '{"target": {"id": "parent-id"}}' [--dry-run]

# Delete / Trash
umbraco media delete <id> [--dry-run]
umbraco media trash <id> [--dry-run]

# ============================================
# CONVENIENCE FLAGS
# ============================================

umbraco media upload <file> --folder <parent-id> [--name <n>] [--dry-run]
umbraco media create-folder <name> --parent <parent-id> [--dry-run]
umbraco media move <id> --to <parent-id> [--dry-run]
```

---

## 3. Document Type (`doctype`)

```bash
# ============================================
# READ
# ============================================

umbraco doctype get <id> [--fields <f>]
umbraco doctype list [--fields <f>]
umbraco doctype root
umbraco doctype children <id>
umbraco doctype search --params '{"query": "..."}'

# ============================================
# WRITE (Agent-first: --json is primary)
# ============================================

# Create - use schema introspection first
umbraco schema doctype.create  # Get the schema
umbraco doctype create --json '{...full payload...}' [--dry-run]

# Update
umbraco doctype update <id> --json '{...}' [--dry-run]

# Copy
umbraco doctype copy <id> --json '{"target": {"id": "parent-id"}}' [--dry-run]

# Move
umbraco doctype move <id> --json '{"target": {"id": "parent-id"}}' [--dry-run]

# Delete
umbraco doctype delete <id> [--dry-run]

# ============================================
# CONVENIENCE FLAGS
# ============================================

umbraco doctype copy <id> --to <parent-id> [--dry-run]
umbraco doctype move <id> --to <parent-id> [--dry-run]
```

---

## 4. Data Type (`datatype`)

```bash
# ============================================
# READ
# ============================================

umbraco datatype get <id> [--fields <f>]
umbraco datatype list [--fields <f>]
umbraco datatype root
umbraco datatype search --params '{"query": "..."}'
umbraco datatype is-used <id>

# ============================================
# WRITE (Agent-first: --json is primary)
# ============================================

umbraco schema datatype.create  # Get the schema first
umbraco datatype create --json '{...}' [--dry-run]
umbraco datatype update <id> --json '{...}' [--dry-run]
umbraco datatype delete <id> [--dry-run]
```

---

## 5. Template (`template`)

```bash
# ============================================
# READ
# ============================================

umbraco template get <id> [--fields <f>]
umbraco template root
umbraco template search --params '{"query": "..."}'

# ============================================
# WRITE (Agent-first: --json is primary)
# ============================================

umbraco schema template.create  # Get the schema
umbraco template create --json '{
  "name": "My Template",
  "alias": "myTemplate",
  "content": "@inherits Umbraco.Cms.Web.Common.Views.UmbracoViewPage\n..."
}' [--dry-run]

umbraco template update <id> --json '{...}' [--dry-run]
umbraco template delete <id> [--dry-run]
```

---

## 6. Log Viewer (`logs`)

```bash
# Agent-first: use --params for filtering
umbraco logs list --params '{
  "level": "Error",
  "startDate": "2024-01-01",
  "endDate": "2024-01-31",
  "skip": 0,
  "take": 50
}'

umbraco logs levels
umbraco logs level-count --params '{"startDate": "...", "endDate": "..."}'
umbraco logs templates
umbraco logs search --params '{"filterExpression": "..."}'

# Convenience flags
umbraco logs list --level Error --from 2024-01-01 --to 2024-01-31 --take 50
```

---

## 7. Server (`server`)

```bash
# Read only
umbraco server status
umbraco server info
umbraco server config
umbraco server troubleshoot
umbraco server upgrade-check
```

---

## 8. Health (`health`)

```bash
# Read only
umbraco health groups
umbraco health group <name>
umbraco health run <group-name>
umbraco health action <action-id>
```

---

## Core Implementation

### config.ts

```typescript
import 'dotenv/config';

export const config = {
  baseUrl: process.env.UMBRACO_BASE_URL || 'https://localhost:44391',
  clientId: process.env.UMBRACO_CLIENT_ID || '',
  clientSecret: process.env.UMBRACO_CLIENT_SECRET || '',
  outputFormat: (process.env.UMBRACO_OUTPUT_FORMAT || 'plain') as 'json' | 'table' | 'plain',
};

if (!config.clientId || !config.clientSecret) {
  console.error('Missing UMBRACO_CLIENT_ID or UMBRACO_CLIENT_SECRET');
  process.exit(1);
}
```

### auth.ts

```typescript
import { config } from './config.js';

let cachedToken: { token: string; expiresAt: number } | null = null;

export async function getAccessToken(): Promise<string> {
  if (cachedToken && cachedToken.expiresAt > Date.now()) {
    return cachedToken.token;
  }

  const response = await fetch(
    `${config.baseUrl}/umbraco/management/api/v1/security/back-office/token`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        grant_type: 'client_credentials',
        client_id: config.clientId,
        client_secret: config.clientSecret,
      }),
    }
  );

  if (!response.ok) {
    throw new Error(`Auth failed: ${response.status} ${await response.text()}`);
  }

  const data = await response.json();

  cachedToken = {
    token: data.access_token,
    expiresAt: Date.now() + data.expires_in * 1000 - 60000,
  };

  return cachedToken.token;
}
```

### api.ts

```typescript
import { getAccessToken } from './auth.js';
import { config } from './config.js';
import { validateInput } from './validate.js';

interface ApiOptions {
  fields?: string;
  params?: Record<string, unknown>;
  dryRun?: boolean;
}

export async function api<T>(
  method: string,
  path: string,
  body?: unknown,
  options?: ApiOptions
): Promise<T> {
  // Validate all inputs (agents hallucinate)
  if (body && typeof body === 'object') {
    validateInput(body as Record<string, unknown>);
  }
  if (options?.params && typeof options.params === 'object') {
    validateInput(options.params as Record<string, unknown>);
  }

  // Build URL with query params
  let url = `${config.baseUrl}/umbraco/management/api/v1${path}`;
  const queryParts: string[] = [];
  
  if (options?.fields) {
    queryParts.push(`fields=${encodeURIComponent(options.fields)}`);
  }
  if (options?.params) {
    for (const [key, value] of Object.entries(options.params)) {
      queryParts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
    }
  }
  if (queryParts.length > 0) {
    url += `?${queryParts.join('&')}`;
  }

  // Dry run - validate and show what would happen
  if (options?.dryRun) {
    return {
      dryRun: true,
      valid: true,
      method,
      path: url.replace(config.baseUrl, ''),
      body: body || null,
    } as T;
  }

  const token = await getAccessToken();

  const response = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`API ${response.status}: ${error}`);
  }

  const text = await response.text();
  return text ? JSON.parse(text) : null;
}

api.get = <T>(path: string, options?: ApiOptions) => 
  api<T>('GET', path, undefined, options);
api.post = <T>(path: string, body?: unknown, options?: ApiOptions) => 
  api<T>('POST', path, body, options);
api.put = <T>(path: string, body?: unknown, options?: ApiOptions) => 
  api<T>('PUT', path, body, options);
api.delete = <T>(path: string, options?: ApiOptions) => 
  api<T>('DELETE', path, undefined, options);
```

### validate.ts

```typescript
// Reject path traversal attempts
export function validatePath(path: string): void {
  const normalized = path.replace(/\\/g, '/');
  if (normalized.includes('..') || normalized.startsWith('/')) {
    throw new Error(`Invalid path: ${path}`);
  }
}

// Reject control characters
export function validateString(str: string): void {
  if (/[\x00-\x1F\x7F]/.test(str)) {
    throw new Error('Input contains control characters');
  }
}

// Reject embedded query params in IDs
export function validateResourceId(id: string): void {
  if (/[?#%]/.test(id)) {
    throw new Error(`Invalid resource ID: ${id}`);
  }
}

// Validate all inputs
export function validateInput(input: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(input)) {
    if (typeof value === 'string') {
      validateString(value);
      if (key.toLowerCase().includes('id')) {
        validateResourceId(value);
      }
      if (key.toLowerCase().includes('path')) {
        validatePath(value);
      }
    }
  }
}
```

### output.ts

```typescript
import { config } from './config.js';

export function output(data: unknown, format?: 'json' | 'table' | 'plain') {
  // Agent-first: JSON by default when piped
  const defaultFormat = process.stdout.isTTY ? 'plain' : 'json';
  const fmt = format || config.outputFormat || defaultFormat;

  if (data === null || data === undefined) {
    if (fmt === 'json') {
      console.log(JSON.stringify({ success: true }));
    } else {
      console.log('Done');
    }
    return;
  }

  switch (fmt) {
    case 'json':
      console.log(JSON.stringify(data, null, 2));
      break;
    case 'table':
      if (Array.isArray(data)) {
        console.table(data);
      } else {
        console.log(data);
      }
      break;
    default:
      // Plain - still JSON but could be prettier in future
      console.log(JSON.stringify(data, null, 2));
  }
}
```

### index.ts

```typescript
#!/usr/bin/env node
import { Command } from 'commander';
import { registerDocumentCommands } from './commands/document.js';
import { registerMediaCommands } from './commands/media.js';
import { registerDoctypeCommands } from './commands/doctype.js';
import { registerDatatypeCommands } from './commands/datatype.js';
import { registerTemplateCommands } from './commands/template.js';
import { registerLogsCommands } from './commands/logs.js';
import { registerServerCommands } from './commands/server.js';
import { registerHealthCommands } from './commands/health.js';

const program = new Command();

program
  .name('umbraco')
  .description('Umbraco CLI - Management API')
  .version('1.0.0')
  .option('-o, --output <format>', 'Output format: json, table, plain', 'plain');

registerDocumentCommands(program);
registerMediaCommands(program);
registerDoctypeCommands(program);
registerDatatypeCommands(program);
registerTemplateCommands(program);
registerLogsCommands(program);
registerServerCommands(program);
registerHealthCommands(program);

program.parse();
```

---

## Example Command: document.ts

```typescript
import { Command } from 'commander';
import { api } from '../api.js';
import { output } from '../output.js';

export function registerDocumentCommands(program: Command) {
  const doc = program.command('document').alias('doc').description('Content management');

  // ============================================
  // READ COMMANDS
  // ============================================

  doc
    .command('get <id>')
    .description('Get document by ID')
    .option('--fields <fields>', 'Limit response fields')
    .action(async (id, opts) => {
      const result = await api.get(`/document/${id}`, { fields: opts.fields });
      output(result);
    });

  doc
    .command('root')
    .description('Get root documents')
    .option('--fields <fields>', 'Limit response fields')
    .option('--params <json>', 'Query parameters as JSON')
    .action(async (opts) => {
      const params = opts.params ? JSON.parse(opts.params) : undefined;
      const result = await api.get('/document/root', { fields: opts.fields, params });
      output(result);
    });

  doc
    .command('children <id>')
    .description('Get child documents')
    .option('--fields <fields>', 'Limit response fields')
    .action(async (id, opts) => {
      const result = await api.get(`/document/${id}/children`, { fields: opts.fields });
      output(result);
    });

  doc
    .command('search')
    .description('Search documents')
    .option('--params <json>', 'Search parameters as JSON')
    .option('--query <q>', 'Search query (convenience)')
    .action(async (opts) => {
      const params = opts.params ? JSON.parse(opts.params) : { query: opts.query };
      const result = await api.get('/document/search', { params });
      output(result);
    });

  // ============================================
  // WRITE COMMANDS (Agent-first: --json primary)
  // ============================================

  doc
    .command('create')
    .description('Create document')
    .requiredOption('--json <payload>', 'Full document payload as JSON')
    .option('--dry-run', 'Validate without executing')
    .action(async (opts) => {
      const body = JSON.parse(opts.json);
      const result = await api.post('/document', body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('update <id>')
    .description('Update document')
    .requiredOption('--json <payload>', 'Update payload as JSON')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const body = JSON.parse(opts.json);
      const result = await api.put(`/document/${id}`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('update-properties <id>')
    .description('Update document properties (partial)')
    .requiredOption('--json <payload>', 'Properties payload as JSON')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const body = JSON.parse(opts.json);
      const result = await api.put(`/document/${id}/properties`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('publish <id>')
    .description('Publish document')
    .option('--json <payload>', 'Publish options as JSON')
    .option('--culture <culture>', 'Culture to publish (convenience)')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      // Agent-first: --json takes precedence
      const body = opts.json 
        ? JSON.parse(opts.json)
        : opts.culture 
          ? { cultures: [opts.culture] }
          : {};
      const result = await api.post(`/document/${id}/publish`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('unpublish <id>')
    .description('Unpublish document')
    .option('--json <payload>', 'Unpublish options as JSON')
    .option('--culture <culture>', 'Culture to unpublish (convenience)')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const body = opts.json 
        ? JSON.parse(opts.json)
        : opts.culture 
          ? { cultures: [opts.culture] }
          : {};
      const result = await api.post(`/document/${id}/unpublish`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('copy <id>')
    .description('Copy document')
    .option('--json <payload>', 'Copy options as JSON')
    .option('--to <parent-id>', 'Target parent ID (convenience)')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const body = opts.json 
        ? JSON.parse(opts.json)
        : { target: { id: opts.to } };
      const result = await api.post(`/document/${id}/copy`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('move <id>')
    .description('Move document')
    .option('--json <payload>', 'Move options as JSON')
    .option('--to <parent-id>', 'Target parent ID (convenience)')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const body = opts.json 
        ? JSON.parse(opts.json)
        : { target: { id: opts.to } };
      const result = await api.post(`/document/${id}/move`, body, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('delete <id>')
    .description('Delete document')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      await api.delete(`/document/${id}`, { dryRun: opts.dryRun });
      output(null);
    });

  doc
    .command('trash <id>')
    .description('Move document to recycle bin')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const result = await api.post(`/document/${id}/move-to-recycle-bin`, undefined, { dryRun: opts.dryRun });
      output(result);
    });

  doc
    .command('restore <id>')
    .description('Restore document from recycle bin')
    .option('--dry-run', 'Validate without executing')
    .action(async (id, opts) => {
      const result = await api.post(`/document/${id}/restore`, undefined, { dryRun: opts.dryRun });
      output(result);
    });
}
```

---

## Schema Command: schema.ts

The CLI is the documentation. Agents introspect schemas at runtime.

```typescript
import { Command } from 'commander';
import { output } from '../output.js';
import { schemas } from '../schemas/index.js';

export function registerSchemaCommands(program: Command) {
  const schema = program.command('schema').description('Introspect API schemas');

  schema
    .command('list')
    .description('List all available endpoints')
    .action(() => {
      const endpoints = Object.keys(schemas).sort();
      output({ endpoints });
    });

  schema
    .argument('[endpoint]', 'Endpoint name (e.g., document.create)')
    .description('Get schema for an endpoint')
    .action((endpoint) => {
      if (!endpoint) {
        // List all
        const endpoints = Object.keys(schemas).sort();
        output({ endpoints });
        return;
      }

      const endpointSchema = schemas[endpoint];
      if (!endpointSchema) {
        console.error(`Unknown endpoint: ${endpoint}`);
        console.error(`Run 'umbraco schema --list' to see available endpoints`);
        process.exit(1);
      }

      output(endpointSchema);
    });
}
```

### schemas/index.ts

Pre-built from Umbraco's OpenAPI spec (or fetched at runtime).

```typescript
// Generated from Umbraco Management API OpenAPI spec
export const schemas: Record<string, Schema> = {
  'document.get': {
    endpoint: 'document.get',
    method: 'GET',
    path: '/document/{id}',
    pathParams: {
      id: { type: 'string', format: 'uuid', required: true }
    },
    queryParams: {
      fields: { type: 'string', description: 'Comma-separated field names' }
    },
    response: {
      type: 'object',
      properties: {
        id: { type: 'string' },
        documentType: { type: 'object' },
        values: { type: 'array' },
        variants: { type: 'array' },
        // ... full schema
      }
    }
  },

  'document.create': {
    endpoint: 'document.create',
    method: 'POST',
    path: '/document',
    requestBody: {
      type: 'object',
      required: ['documentType', 'parent'],
      properties: {
        documentType: {
          type: 'object',
          properties: { id: { type: 'string', format: 'uuid' } }
        },
        parent: {
          type: 'object',
          properties: { id: { type: 'string', format: 'uuid' } }
        },
        values: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              alias: { type: 'string' },
              value: { type: 'any' },
              culture: { type: 'string', nullable: true },
              segment: { type: 'string', nullable: true }
            }
          }
        },
        variants: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              culture: { type: 'string', nullable: true },
              segment: { type: 'string', nullable: true },
              name: { type: 'string' }
            }
          }
        }
      }
    },
    response: {
      type: 'string',
      description: 'Created document ID'
    }
  },

  'document.publish': {
    endpoint: 'document.publish',
    method: 'POST',
    path: '/document/{id}/publish',
    pathParams: {
      id: { type: 'string', format: 'uuid', required: true }
    },
    requestBody: {
      type: 'object',
      properties: {
        cultures: {
          type: 'array',
          items: { type: 'string' },
          description: 'Cultures to publish (empty = all)'
        }
      }
    }
  },

  // ... more endpoints
  // Full list generated from OpenAPI spec
};

interface Schema {
  endpoint: string;
  method: string;
  path: string;
  pathParams?: Record<string, ParamSchema>;
  queryParams?: Record<string, ParamSchema>;
  requestBody?: ObjectSchema;
  response?: ObjectSchema;
}

interface ParamSchema {
  type: string;
  format?: string;
  required?: boolean;
  description?: string;
}

interface ObjectSchema {
  type: string;
  required?: string[];
  properties?: Record<string, unknown>;
  items?: unknown;
  description?: string;
}
```

---

## package.json

```json
{
  "name": "umbraco-cli",
  "version": "1.0.0",
  "description": "CLI for Umbraco Management API - Agent First",
  "type": "module",
  "bin": {
    "umbraco": "./dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "dev": "tsx src/index.ts",
    "start": "node dist/index.js",
    "link": "npm run build && npm link"
  },
  "dependencies": {
    "commander": "^12.0.0",
    "dotenv": "^16.3.1"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "tsx": "^4.6.0",
    "typescript": "^5.3.0"
  }
}
```

---

## tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true
  },
  "include": ["src/**/*"]
}
```

---

## .env.example

```env
UMBRACO_BASE_URL=https://localhost:44391
UMBRACO_CLIENT_ID=umbraco-back-office-api-user
UMBRACO_CLIENT_SECRET=your-secret-here
NODE_TLS_REJECT_UNAUTHORIZED=0
```

---

## Build Order

### Phase 1: Core (6 files)
1. `config.ts`
2. `auth.ts`
3. `validate.ts`
4. `api.ts`
5. `output.ts`
6. `index.ts`

### Phase 2: Commands (8 files)
7. `commands/document.ts`
8. `commands/media.ts`
9. `commands/doctype.ts`
10. `commands/datatype.ts`
11. `commands/template.ts`
12. `commands/logs.ts`
13. `commands/server.ts`
14. `commands/health.ts`

### Phase 3: Agent Files (2 files)
15. `AGENTS.md`
16. `CONTEXT.md`

### Phase 4: Skills (66 files)
17. Clone from https://github.com/umbraco/Umbraco-CMS-Backoffice-Skills
18. Copy `plugins/umbraco-backoffice-skills/skills/*` to `skills/`
19. Copy `plugins/umbraco-testing-skills/skills/*` to `skills/testing/`

---

## Testing

```bash
# Build and link globally
npm run link

# Test commands
umbraco server status
umbraco document root
umbraco logs list --take 10

# Test dry-run
umbraco document publish abc-123 --dry-run
```

---

## Future Expansion

See `SPECIFICATION-FULL.md` for all 300+ commands when needed.

Collections to add later:
- member, member-group, member-type
- user, user-group
- webhook
- dictionary
- partial-view, script, stylesheet
- redirect
- indexer, searcher