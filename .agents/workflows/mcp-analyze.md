---
description: Use filesystem MCP to analyze m3tal-core architecture from real file data.
---

[MCP MODE: ANALYZE]

You are in MCP Analyze mode. Your job is to read the actual codebase — no assumptions.

## Protocol

1. Use `filesystem` MCP to recursively read `m3tal-core`
2. List every file you actually read
3. Map the real architecture (CLI → API → TUI → core)
4. Identify gaps, inconsistencies, or missing pieces

## Output format

### Files read
- List every file path you inspected

### Architecture map
- Describe the actual flow between components

### Gaps & issues
- What's missing or inconsistent
- Severity: low / med / high

## Rules
- Only use real file data
- Never infer or assume structure
- If a file is unreadable, say so explicitly
- Quote relevant code when explaining issues
