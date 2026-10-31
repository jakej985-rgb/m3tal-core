# M3TAL Review Engine Standards & False-Positive Prevention Rules

This rule defines standards for multi-agent code reviews across M3TAL repositories (`m3tal-core`, `m3tal-plugins`, `M3tal-Hub`).

## 1. Zero Core Footprint Rule

- `m3tal-core` is strictly the host platform supervisor, daemon API, and dynamic loader.
- All plugin business logic, service widgets, and plugin-specific models MUST live in `m3tal-plugins`.
- Any PR introducing plugin-specific endpoints, hardcoded UI components, or custom dependencies into `m3tal-core` MUST be rejected with an `Architecture` finding.

## 2. Evidence-Based Defect Rule

- Every finding reported by an agent must have a **credible execution path** or failure scenario.
- Theoretical, non-reproducible, or impossible edge cases are invalid and will be dismissed by the Review Judge.
- If a security defect is claimed, the reviewer must explain how an attacker or untrusted input reaches the vulnerable call.

## 3. Diff Attribution Rule

- A defect is only reportable if it was **introduced or worsened by the reviewed diff**.
- Pre-existing bugs in unchanged surrounding code must not block the current review.
- If an agent identifies a pre-existing issue, it may only be noted as `Informational` (non-blocking).

## 4. Personal Style vs Engineering Flaws

- Do not report stylistic preferences, naming tastes, or unnecessary refactoring requests as defects.
- Code that is safe, idiomatic, and functioning as intended must receive a clean review.

## 5. Bounded Remediation Protocol

- Maximum remediation iterations: 3.
- If unresolved after 3 cycles, the review transitions to `EXHAUSTED` and blocks.
- If the coding agent reports fixes but the git diff remains identical, the review transitions to `BLOCKED`.
- Review history must remain immutable. Never overwrite previous iteration records.
