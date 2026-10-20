# 🤖 M3TAL Media Server — Production Architecture Vision (v1.7.0)

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Build](BUILD_CONFIGURATION.md) | [🌐 Networking](NETWORKING.md) | [🤖 Architecture](ARCHITECTURE_VISION.md) |
| :---: | :---: | :---: | :---: | :---: |

## 🎯 Objective

Transform the M3TAL Media Server into a **High-Performance, Go-Native Orchestration Plane**.

The system has transitioned from a collection of fragmented Python/Bash agents into a unified, statically-linked Go binary (`./m3tal`) that serves as the **Source of Truth** for infrastructure lifecycle, health monitoring, and system reconciliation.

---

## 🏛️ CORE ARCHITECTURE (GO-NATIVE)

### 🌏 Unified Orchestrator

* **Component**: `core/orchestrator`
* **Logic**: Manages the multi-compose stack (`deploy/stack/`) using native Docker API interactions and standardized execution paths.
* **Safety**: Replaces the legacy `reconcile.py` with type-safe execution and atomic state transitions.

### 💾 Atomic State & Configuration

* **Path**: `state/` and `.env`
* **Logic**: Uses a centralized environment configuration model. The Go binary enforces absolute path consistency (`/mnt` mapping) across all managed services.
* **Persistence**: All system state is stored in standardized JSON/YAML formats, accessible via the `core/system` layer.

### 👮 Health & Observability

* **Component**: `core/health`
* **Logic**: Implements sub-millisecond HTTP and Socket-based health polling.
* **Metrics**: Real-time telemetry is collected by the Go backend and served via the internal API for the dashboard.

---

## ⚙️ IMPLEMENTATION STATUS (v1.7)

### ✅ PHASE 1 — GO CORE (COMPLETE)

* **[COMPLETED]** `./m3tal` CLI: Unified command interface for `up`, `down`, `init`, and `config`.
* **[COMPLETED]** `core/orchestrator`: Multi-stack compose management logic.
* **[COMPLETED]** `build.sh`: Standardized compilation pipeline for Go 1.26+.

### ✅ PHASE 2 — INFRASTRUCTURE (COMPLETE)

* **[COMPLETED]** `deploy/stack/`: Standardized Docker manifests for Network, Routing, and Media services.
* **[COMPLETED]** Traefik Integration: Dynamic host-level routing for `api.localhost` and `m3tal.localhost`.

### 🚧 PHASE 3 — DASHBOARD MIGRATION (IN PROGRESS)

* **[IN PROGRESS]** `m3tal-godash`: Transitioning from Flask/HTML to a high-performance Go-native frontend.
* **[PLANNED]** WebSocket Integration: Real-time metric streaming from the Go orchestrator to the UI.

---

## 🛡️ SECURITY & RELIABILITY LOCKS

| Feature | Protection |
| :--- | :--- |
| **Type Safety** | Go's compiler ensures memory safety and prevents runtime nil-pointer crashes common in Python. |
| **Static Linking** | The orchestrator is a single binary with zero runtime dependencies (no `pip install` required). |
| **Path Enforcement** | Strict `/mnt` mapping ensures that media data is always where it's expected to be. |
| **Ingress Filter** | All external traffic is routed through Traefik with internal Docker networking isolation. |

---

## 🔮 FUTURE ROADMAP (PHASE 20+)

### 📡 Phase 20: Global Cluster Synchronization

* Implement a lightweight gossip protocol within the Go binary for multi-node state sharing without external databases.

### 🧠 Phase 21: Predictive Resource Allocation

* Integrate Go-native resource analysis to dynamically adjust container limits based on historical usage patterns.

### 🎨 Phase 22: Unified Command Center

* Finalize the `m3tal-godash` ecosystem, providing a single-pane-of-glass for both infrastructure management and media consumption.

---

## 🏁 DEFINITION OF DONE

The system is considered healthy if:

1. **Binary Integrity**: `./m3tal` compiles and executes on the target architecture (amd64/arm64).
2. **Stack Health**: All services in `deploy/stack/` report "Healthy" via `core/health`.
3. **Routing**: `m3tal.localhost` and `api.localhost` resolve and serve traffic correctly.
4. **Persistence**: Configuration changes via `./m3tal config` persist across service restarts.
5. **Recovery**: Services automatically recover from host reboots via Docker restart policies managed by the orchestrator.
