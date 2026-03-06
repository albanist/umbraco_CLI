# Prompt: Add Dictionary Item Import to Umbraco CLI

## Context

We're building multi-language support for umbraco.com (Umbraco CMS v17). We have 171 dictionary items that need to be created in the Umbraco backoffice. The Umbraco Management API exposes full CRUD for dictionary items, but our CLI doesn't support it yet.

We have a reference file (`dictionary-items-reference.md`) with all 171 keys and their en-US values. We also have a JSON version at `dictionary-items.json` ready for import.

## What to Build

Add dictionary item management commands to the Umbraco CLI. The MVP is **import from JSON file**, but ideally we'd have full CRUD.

---

## Umbraco Management API — Dictionary Endpoints (from official OpenAPI spec)

Base URL: `{umbraco-base}/umbraco/management/api/v1`

All endpoints require `Backoffice-User` authentication (bearer token — same OAuth mechanism the CLI already uses).

### 1. List / Search

```
GET /dictionary?filter={name}&skip=0&take=100
```

Query params:
- `filter` (string, optional) — filter by name
- `skip` (int32, default: 0) — pagination offset
- `take` (int32, default: 100) — page size

Response (`PagedDictionaryOverviewResponseModel`):
```json
{
  "total": 171,
  "items": [
    {
      "id": "a1b2c3d4-...",
      "name": "Blog.TopStory",
      "parent": null,
      "translatedIsoCodes": ["en-US", "da-DK"]
    }
  ]
}
```

Status codes: `200 OK`, `401 Unauthorized`, `403 Forbidden`

### 2. Get by ID

```
GET /dictionary/{id}
```

Response (`DictionaryItemResponseModel`):
```json
{
  "name": "Blog.TopStory",
  "parent": null,
  "translations": [
    { "isoCode": "en-US", "translation": "Top story" },
    { "isoCode": "da-DK", "translation": "Tophistorie" }
  ],
  "id": "a1b2c3d4-..."
}
```

Status codes: `200 OK`, `404 Not Found`, `401`, `403`

### 3. Get Multiple by IDs

```
GET /item/dictionary?id={uuid1}&id={uuid2}
```

Response: Array of `DictionaryItemItemResponseModel`

Status codes: `200 OK`, `401`

### 4. Create

```
POST /dictionary
Content-Type: application/json
```

Request body (`CreateDictionaryItemRequestModel`):
```json
{
  "name": "Blog.TopStory",
  "parent": { "id": "parent-uuid" },
  "translations": [
    { "isoCode": "en-US", "translation": "Top story" }
  ],
  "id": "client-generated-uuid"
}
```

- `name` — the dictionary key (e.g. `Blog.TopStory`)
- `parent` — `null` for root items, or `{ "id": "uuid" }` for nested/child items
- `translations` — array of `{ isoCode, translation }` pairs
- `id` — client-generated UUID (use deterministic UUID from key name for idempotency, or random)

Response headers: `Umb-Generated-Resource` (new ID), `Location` (resource URI), `Umb-Notifications`

Status codes: `201 Created`, `400 Bad Request`, `404 Not Found`, `409 Conflict`, `401`, `403`

### 5. Update

```
PUT /dictionary/{id}
Content-Type: application/json
```

Request body (`UpdateDictionaryItemRequestModel`):
```json
{
  "name": "Blog.TopStory",
  "parent": null,
  "translations": [
    { "isoCode": "en-US", "translation": "Top story" },
    { "isoCode": "da-DK", "translation": "Tophistorie" }
  ]
}
```

Status codes: `200 OK`, `400 Bad Request`, `404 Not Found`, `401`, `403`

### 6. Delete

```
DELETE /dictionary/{id}
```

Status codes: `200 OK`, `400 Bad Request`, `404 Not Found`, `401`, `403`

### 7. Import (built-in)

```
POST /dictionary/import
```

Request body (`ImportDictionaryRequestModel`): file upload (UDT format)

Response headers: `Umb-Generated-Resource`, `Location`

Status codes: `201 Created`, `400`, `404`, `401`, `403`

### 8. Export

```
GET /dictionary/{id}/export?includeChildren=false
```

Query params:
- `includeChildren` (boolean, default: false)

Response: binary file download (UDT format)

Status codes: `200 OK`, `404`, `401`, `403`

### 9. Move

```
PUT /dictionary/{id}/move
```

Request body (`MoveDictionaryRequestModel`): target parent

Status codes: `200 OK`, `400`, `404`, `401`, `403`

### Tree Navigation Endpoints

```
GET /tree/dictionary/root?skip=0&take=100          — root-level items
GET /tree/dictionary/children?parentId={id}&skip=0&take=100  — children of a parent
GET /tree/dictionary/ancestors?descendantId={id}    — ancestors of an item
```

All return `NamedEntityTreeItemResponseModel` arrays.

---

## Commands to Implement

### 1. `dictionary import` (MVP — highest priority)

```bash
umbraco-cli dictionary import --file dictionary-items.json [--skip-existing] [--update-existing]
```

Behavior:
1. Read the JSON file (array of `{ key, translations: { isoCode: value } }`)
2. Fetch ALL existing dictionary items upfront via `GET /dictionary?take=9999` to build a lookup map (key → id). This is more efficient than querying per-item.
3. For each item in the JSON:
   - If it exists in the lookup map and `--skip-existing` (default): skip it, log "Skipped: {key} (already exists)"
   - If it exists and `--update-existing`: `GET /dictionary/{id}` to get current translations, merge new ones in, `PUT /dictionary/{id}`
   - If it doesn't exist: `POST /dictionary` to create it with a client-generated UUID
4. Support hierarchical keys: if a key is `Blog.TopStory`, optionally create a parent item `Blog` first (Umbraco dictionary supports parent-child nesting, but **flat keys are preferred for this project** — all items at root level)
5. Print summary: "Created: X, Updated: Y, Skipped: Z, Failed: W"

Flags:
- `--file <path>` (required) — path to JSON file
- `--skip-existing` (default behavior) — skip items that already exist
- `--update-existing` — update/merge translations on existing items
- `--dry-run` — show what would happen without making API calls
- `--batch-size <n>` — number of concurrent requests (default: 5, max: 10)

### 2. `dictionary list` (nice to have)

```bash
umbraco-cli dictionary list [--filter "Blog"] [--take 50]
```

### 3. `dictionary get` (nice to have)

```bash
umbraco-cli dictionary get --key "Blog.TopStory"
```

### 4. `dictionary create` (nice to have)

```bash
umbraco-cli dictionary create --key "Blog.TopStory" --en-US "Top story" --da-DK "Tophistorie"
```

### 5. `dictionary delete` (nice to have)

```bash
umbraco-cli dictionary delete --key "Blog.TopStory" [--force]
```

### 6. `dictionary export` (nice to have)

```bash
umbraco-cli dictionary export --output dictionary-items.json
```

Export all dictionary items to the same JSON format used by import. Useful for backup/migration between environments.

---

## JSON Import File Format

The import file is an array of objects:

```json
[
  {
    "key": "Blog.TopStory",
    "translations": {
      "en-US": "Top story"
    }
  },
  {
    "key": "Search.ResultsSummary",
    "translations": {
      "en-US": "Your search for <i>{0}</i> returned {1} results."
    }
  }
]
```

When additional translations are available:

```json
{
  "key": "Blog.TopStory",
  "translations": {
    "en-US": "Top story",
    "da-DK": "Tophistorie",
    "de-DE": "Top-Story",
    "es-ES": "Historia destacada"
  }
}
```

The full JSON file with all 171 items (en-US only for now) is at `docs/dictionary-items.json`.

---

## Edge Cases and Error Handling

1. **Duplicate keys**: The API returns `409 Conflict` or `400 Bad Request` if you try to create a dictionary item with a name that already exists. Always check first via the lookup map.
2. **Special characters in values**: Some values contain HTML (`<i>{0}</i>`), ampersands (`Awards & contributions`), and `{0}` format placeholders. These must be preserved exactly as-is — they are intentional.
3. **Rate limiting**: The Management API may rate-limit. Implement retry with exponential backoff on 429 responses.
4. **Authentication expiry**: Handle token refresh if long-running imports exceed token TTL.
5. **Partial failure**: If the import fails partway through, the summary should show what succeeded and what failed. Consider writing a failure log file.
6. **Empty translations**: A translation value of `""` (empty string) is valid — it means "use the fallback language value". Don't skip these.
7. **Large imports**: For 171+ items, batch the creates with controlled concurrency (default 5) to avoid overwhelming the API.

---

## Testing

1. **Unit tests**: Mock the API responses and test the import logic (skip/update/create decisions, translation merging)
2. **Integration test**: Run against a local Umbraco instance with a small subset (5-10 items)
3. **Idempotency test**: Run import twice — second run should skip all items with 0 creates
4. **Dry run test**: Verify `--dry-run` makes zero API calls but outputs correct plan
5. **Update test**: Create items, then import again with `--update-existing` and additional translations — verify merge

---

## Acceptance Criteria

- [ ] `dictionary import --file dictionary-items.json` creates all 171 items with en-US values
- [ ] Running import twice with `--skip-existing` results in 0 creates, 171 skips
- [ ] Running import with `--update-existing` merges new translations onto existing items without removing existing ones
- [ ] `--dry-run` outputs what would happen without making changes
- [ ] Clear progress output: "Creating Blog.TopStory... OK" or "Skipping Blog.TopStory (exists)"
- [ ] Summary line at end: "Created: 171, Skipped: 0, Failed: 0"
- [ ] Error handling for network failures, auth issues, and API errors
- [ ] Handles special characters (HTML, ampersands, format placeholders) in values correctly
