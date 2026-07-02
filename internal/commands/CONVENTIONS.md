# Command conventions

Rules for adding or changing commands in this package. The spec-based
builders in `archetypes.go` encode most of them — reach for a builder first;
write a custom `RunE` only when the command's contract genuinely differs,
and say why in a comment.

## Builders

| Shape | Builder | Contract |
|---|---|---|
| Get by ID | `getCommand` | `--fields` + client-side projection |
| Paginated read (root/children/list) | `collectionCommand` | `--fields`/`--params`/`--skip`/`--take`/`--all` + triage, endpoint fallback |
| Search | `searchCommand` | `--query` + extras merge into `--params`; `--params` wins on collisions |
| Create | `createCommand` | required `--json`, optional `--print-template`, CLI-generated id, identity-echoing result |
| Update | `updateCommand` | exactly one of `--json` (replace) / `--merge-json` (fetch-and-merge) |
| Move/copy | `targetActionCommand` | `--to` shortcut or raw `--json`, method/route fallback |
| Hard delete | `deleteCommand` | gated by force/dry-run |
| References | `referencesCommand` / `areReferencedCommand` | shared document/media reference reads |

## Flags

- `--ids`: one comma-separated GUID list. When a command takes **two** ID
  lists, name both explicitly (`--user-ids`, `--group-ids`) — never leave one
  ambiguous `--ids` next to a qualified one.
- `--query`: server-side search input. `--filter`: substring filter parameter
  on filter-style endpoints (`/filter/...`).
- `--params`: raw JSON query parameters. Convenience flags fill missing keys;
  `--params` wins on collisions (see `mergeParams`).
- Pagination: always `addPaginationFlags` (`-1` sentinel = not sent) plus
  `addAutoPaginationFlag` where auto-paging is supported. Never a `0` default
  or a `Flags().Changed` check.
- Output shaping: `--fields` everywhere; document-shaped payloads add
  `--summary`/`--no-empty`/`--full` via `DocumentOutputTrim`.
- Every mutation takes `--dry-run` (`addDryRunFlag`).

## Destructive operations

Gate with `requireForceOrDryRun(cmd, "<consequence>", force, dryRun)` — the
canonical message is `<command> <consequence>; pass --force to confirm or
--dry-run to rehearse`. Hard deletes and bulk mutations are gated;
reversible recycle-bin moves (trash) intentionally are not.

## Output

- Always end with `printResult` (or `printMutationResult` for mutations, so
  an empty 204 success prints `{"<verb>": true}` instead of `null`).
- JSON output is the stable machine contract: only additive changes. Table
  layout is unstable and may reorder.

## File size

Keep command files under ~600 lines. When a group grows, split by concern
the way logs (`logs_query.go`/`logs_output.go`) and document
(`document_publish.go`/`document_bulk.go`/`document_lifecycle.go`) do —
registration and core CRUD stay in the group's main file.
