---
description: Trigger multi-agent code review, judge arbitration, and remediation via M3TAL Review Engine.
---

# M3TAL Multi-Agent Code Review Workflow

Invoke this workflow to run an independent, multi-agent code review on the current working tree or a branch/PR diff.

## Step 1: Initialize Session & Capture Diff

Run the review engine CLI to pin commits and extract `diff.patch`:
```bash
python3 scripts/review/engine.py --repo . --action init
```

## Step 2: Dispatch Review Orchestrator

Invoke the `review-orchestrator` subagent:
- Review Orchestrator launches the 6 specialist review agents concurrently (`review-syntax`, `review-slop`, `review-security`, `review-logic`, `review-architecture`, `review-tests`).
- Specialist reports are normalized and fingerprinted.
- The `review-judge` evaluates each candidate finding individually against `diff.patch`.

## Step 3: Handle Remediation Loop

- If blocking findings are confirmed by the Judge:
  1. The Orchestrator sends structured remediation instructions to the coding agent.
  2. The coding agent applies fixes.
  3. The Orchestrator captures the new diff, verifies it changed, and re-reviews.
  4. Bounded to a maximum of 3 iterations.

## Step 4: Review Decision & Summary

Check the review summary:
```bash
python3 scripts/review/engine.py --repo . --action summary-pr
```

The review output is preserved immutably in `.m3tal-review/history/` and summarized in `.m3tal-review/reports/latest.md`.
