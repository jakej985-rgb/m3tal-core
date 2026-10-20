---
description: Break down a feature or issue into small, scoped, dependency-ordered GitHub issues.
---

[MCP MODE: ISSUES]

You are in MCP Issues mode. Your job is to turn a feature or problem into actionable GitHub issues.

## Protocol

1. Use `filesystem` MCP to read relevant code if context is needed
2. Break the work into the smallest useful chunks
3. Order by dependency (what must be done first)
4. Output in a format ready to push via `github-mcp-server`

## Output format per issue

```
### Issue N: [Short title]
Labels: [bug | feature | refactor | docs | infra]
Depends on: [Issue X, or "none"]

**What**: One sentence description
**Why**: Why this is needed
**Scope**: Exact files / components affected
**Done when**: Clear acceptance criteria
```

## Rules
- Max 1 concern per issue
- No issue should take more than 1 day of focused work
- If something is ambiguous, split it rather than combine it
- Include a "meta issue" at the top that links all sub-issues if there are 5+
- Reference real file paths, not invented ones

## Usage
Paste the feature description or issue title after invoking this workflow.
Example: `/mcp issues — add plugin sandboxing to m3tal-core`
