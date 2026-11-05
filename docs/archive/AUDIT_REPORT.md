# M3TAL Repository Audit Report

**Date:** June 8, 2026  
**Auditor:** MCP Servers (`filesystem`, `docker`, `sequential-thinking`)  
**Target:** `playbook.md` (Refactoring Phases 1-7)

---

## 🔍 Audit Summary

An automated audit was conducted on the current repository (`m3tal-core`) to verify the implementation of the 7 phases outlined in `playbook.md`. The playbook erroneously marks all 7 phases as `[x]` (completed). However, the audit reveals several critical deviations from the intended architecture.

### 🔴 Phase 1: Core Architecture Lock — **FAILED**

- **Expected:** No direct Docker calls remain in handlers. HTTP handlers must delegate to `stackManager.Deploy()`.
- **Actual:** `api/handlers.go`, `api/stacks.go`, and `api/compose.go` still directly invoke the Docker provider (`docker.GetProvider()`, `docker.DeployStack()`, `docker.ValidateRoute()`). The handlers have not been stripped of their low-level implementation details.

### 🟡 Phase 2: Stack Engine — **PARTIAL**

- **Expected:** `StackManager` is the single source of truth controlling the deployment lifecycle.
- **Actual:** The `StackManager` model and lifecycle methods successfully exist under `core/stack/manager.go` and `core/stack/lifecycle.go`. However, they are completely disconnected from the API routing layer. The engine was built but never integrated.

### 🔴 Phase 3: Traefik Routing — **FAILED**

- **Expected:** Legacy `proxy.go` should be killed, and routing must happen exclusively via Traefik labels generated dynamically.
- **Actual:** `api/proxy.go` and `cli/proxy.go` are still present in the repository and actively parse proxy/route state instead of solely relying on the newly generated Traefik labels.

### 🟡 Phase 4: Reconciler — **PARTIAL**

- **Expected:** Stacks automatically self-heal and return to desired state.
- **Actual:** The reconciler agent loop exists in `core/agents/reconcile.go`. However, because the system doesn't rely entirely on `StackManager` for desired state deployment, the self-healing loop operates on fragmented logic.

### 🟡 Phase 5: Queue + AI — **PARTIAL**

- **Expected:** AI calls are async and processed via the worker queue.
- **Actual:** Queue interface and job structures exist (`core/queue/job.go`), but the API layer still lacks full integration.

### 🟢 Phase 6: Remove Python UI — **PASSED**

- **Expected:** Python runtime is completely removed (`webui/server.py` deleted).
- **Actual:** Legacy `server.py` and `auth.py` files have been successfully deleted from the `webui` directory. The codebase is successfully leaning on Go for serving statics.

### 🟡 Phase 7: Hardening — **INCOMPLETE**

- **Expected:** System is resilient, idempotent, and health endpoints are exposed.
- **Actual:** Health endpoints exist in `api/server.go`, but idempotency checks (e.g., preventing duplicate instance creations) during deployment still heavily rely on old Docker implementations rather than the manager's sync locks.

---

## 🛠️ Recommended Next Steps

1. **Uncheck Phases 1-5 and 7 in `playbook.md`** to accurately reflect the repository's current state.
2. **Execute API Layer Integration:** Connect `api/handlers.go` and `api/stacks.go` to the newly formed `StackManager`.
3. **Remove Dead Code:** Manually delete `api/proxy.go` and `cli/proxy.go` as intended by Phase 3, migrating all proxy validation logic over to the Traefik labels generation engine.
