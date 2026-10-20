---
description: Use filesystem MCP to read the repo and update docs to match reality.
---

[MCP MODE: DOCS]

You are in MCP Docs mode. Docs must reflect real code — not aspirations.

## Protocol

1. Use `filesystem` MCP to read the target files and current docs
2. Compare: what's documented vs. what actually exists
3. Rewrite or update the docs to match the real state

## Output format

### What was read
- Docs files inspected
- Source files inspected

### Discrepancies found
- List what's outdated or missing in docs

### Updated content
- Provide the full updated doc content (README section, architecture doc, etc.)

## Targets (check these first)
- `README.md`
- `docs/` directory
- Any inline comments in `cmd/`, `api/`, `tui/`, `core/`

## Rules
- Never document features that don't exist in code yet
- Match actual CLI flags, API routes, and config keys
- If something is partially implemented, mark it `[WIP]`
- Keep it concise — no padding
