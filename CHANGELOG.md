# Changelog

## v0.2.0 - 2026-03-13

- fixed datatype discovery commands to use compatibility-aware Management API routes
- added richer API endpoint error messages with resolved method/path hints
- added `datatype update --merge-json` for fetch-merge-write partial updates
- added `datatype extensions`, `datatype add-value`, `datatype remove-value`, `datatype add-extension`, and `datatype remove-extension`
- added layered config loading from env, project config, project `.env`, user config, and local `.NET` URL discovery
- updated docs and examples for the new datatype and config workflows
