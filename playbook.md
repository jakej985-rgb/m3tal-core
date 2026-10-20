# M3TAL Refactoring Playbook

This playbook outlines the exact step-by-step commands, files touched, validation checks, and rollbacks for each of the 7 phases of the M3TAL system refactoring.

---

## Table of Contents

1. [Phase 1 — Core Architecture Lock](#-phase 1-core-architecture-lock)
2. [Phase 2 — Stack Engine](#-phase-2-stack-engine)
3. [Phase 3 — Traefik](#-phase-3-traefik)
4. [Phase 4 — Reconciler](#-phase-4-reconciler)
5. [Phase 5 — Queue + AI](#-phase-5-queue-ai)
6. [Phase 6 — Remove Python UI](#-phase-6-remove-python-ui)
7. [Phase 7 — Hardening](#-phase-7-hardening)
8. [Final Execution Checklist](#-final-execution-checklist)

---

## 🧱 Phase 1 — Core Architecture Lock

### ⚙️ Servers

- Primary: `filesystem`
- Required: `gopls`
- Planning: `sequential-thinking`

### 🔹 Step 1 — Snapshot Current Repo

Run this first to ensure you can roll back easily:

```bash
git checkout -b refactor/core-architecture
```

*If anything goes wrong, execute a hard reset.*

### 🔹 Step 2 — Create Canonical Structure

Create the following folders under `core/`:

```bash
mkdir -p core/stack
mkdir -p core/docker
mkdir -p core/traefik
mkdir -p core/ai
mkdir -p core/queue
mkdir -p core/state
```

### 🔹 Step 3 — Move Files into Domains

1. Scan the repository to locate files with:
   - Docker usage (move to `core/docker`)
   - HTTP routing (leave in `/api`)
   - Traefik logic (move to `core/traefik`)
2. Move files using `filesystem.move_file`.
3. Use `gopls.rename` or import refactoring tools to fix imports across the project.

### 🔹 Step 4 — Strip Handlers

Refactor HTTP handlers to delegate logic to services instead of interacting with low-level details directly.

**Before:**

```go
func DeployHandler(w http.ResponseWriter, r *http.Request) {
    docker.Run(...)
}
```

**After:**

```go
func DeployHandler(w http.ResponseWriter, r *http.Request) {
    stackManager.Deploy(...)
}
```

### 🔍 Validation

Run `gopls.diagnostics` or:

```bash
go build ./...
```

Must return `0 errors`.

### 🔁 Rollback

```bash
git reset --hard HEAD
```

### ✅ Done When

- No direct Docker calls remain in handlers.
- Core domain folders are populated.
- Build passes successfully.

---

## 🧱 Phase 2 — Stack Engine

### ⚙️ Servers

- Primary: `gopls`
- Supporting: `filesystem`, `sequential-thinking`

### 🔹 Step 1 — Create Models

File: `core/stack/model.go`

```go
package stack

type Stack struct {
    ID          string
    Name        string
    ComposePath string
    Status      string
    Services    []Service
}

type Service struct {
    Name string
    Port int
}
```

### 🔹 Step 2 — Create Manager

File: `core/stack/manager.go`

```go
package stack

type StackManager struct {
    docker  DockerClient
    traefik TraefikClient
    state   StateStore
}
```

### 🔹 Step 3 — Wire Dependencies

1. Locate the Docker client, Traefik client, and State store implementations.
2. Inject these instances into the `StackManager`.

### 🔹 Step 4 — Implement Deploy()

File: `core/stack/lifecycle.go`

```go
func (m *StackManager) Deploy(stack Stack) error {
    if err := m.state.SaveDesired(stack); err != nil {
        return err
    }

    if err := m.docker.ComposeUp(stack); err != nil {
        return err
    }

    if err := m.traefik.Apply(stack); err != nil {
        return err
    }

    return m.state.SetStatus(stack.ID, "running")
}
```

### 🔹 Step 5 — Add Locking

Add concurrent map locks to prevent race conditions during deployments:

```go
var locks sync.Map
```

Wrap deploy calls with lock and unlock commands:

```go
lock(stack.ID)
defer unlock(stack.ID)
```

### 🔍 Validation

- Verify `gopls.references(Stack)` shows Stack model usage across stack orchestrations.
- Ensure the build succeeds.

### 🚨 Failure Mode

If you encounter:

```
import cycle not allowed
```

Move shared interfaces out to `core/interfaces` to break the dependency cycle.

### ✅ Done When

- `StackManager` is the single source of truth controlling the deployment lifecycle.
- Scattered deploy logic is completely cleaned up.

---

## 🌐 Phase 3 — Traefik

### ⚙️ Servers

- Primary: `filesystem`
- Runtime: `docker`

### 🔹 Step 1 — Label Generator

File: `core/traefik/config.go`

```go
func GenerateLabels(s Service) map[string]string {
    return map[string]string{
        "traefik.enable": "true",
        "traefik.http.routers." + s.Name + ".rule":
            "Host(`" + s.Name + ".local`)",
    }
}
```

### 🔹 Step 2 — Inject into Docker

Modify `ComposeUp` in the Docker client to dynamically generate and attach these Traefik labels to the services.

### 🔹 Step 3 — Verify via Docker

Inspect a running container:

```bash
docker inspect <container>
```

Verify that the output contains the correct Traefik enabling labels:

```json
"Labels": {
  "traefik.enable": "true"
}
```

### 🔥 Step 4 — Kill Proxy

Locate the legacy `proxy.go` file and delete it if it is no longer used by any routing logic.

### 🔍 Validation

- Access the services at `http://<service-name>.local`.
- Verify there are no 502/routing errors.

### 🚨 Failure Mode

If routing fails:

- Ensure Traefik is actively reading container labels.
- Verify container network configuration (ensure they are on the proxy network).

### ✅ Done When

- Service routing works via Traefik labels ONLY.

---

## 🔁 Phase 4 — Reconciler

### ⚙️ Servers

- Primary: `docker`
- State: `sqlite`

### 🔹 Step 1 — Create Loop

Implement a periodic background ticker:

```go
func Start() {
    for {
        reconcileAll()
        time.Sleep(10 * time.Second)
    }
}
```

### 🔹 Step 2 — Compare State

Retrieve desired state from SQLite and actual container states from Docker:

- `sqlite.get(stack)`
- `docker.ps`

### 🔹 Step 3 — Detect Drift

Reconciler must scan for and detect:

| Issue | Fix |
| :--- | :--- |
| Missing Container | Recreate container |
| Stopped | Restart container |
| Wrong Port | Rebuild/redeploy container |

### 🔹 Step 4 — Auto-fix

Apply the corrective deployment action:

```go
docker.ComposeUp(stack)
```

### 🔍 Validation

Manually stop a managed container:

```bash
docker stop <service-name>
```

The reconciler must automatically restart/recreate it within the interval.

### 🚨 Failure Mode

If nothing happens:

- Verify that the reconciler loop is active.
- Inspect DB to ensure correct desired state is saved.

### ✅ Done When

- Stacks automatically self-heal and return to desired state.

---

## 🧠 Phase 5 — Queue + AI

### ⚙️ Servers

- Primary: `sequential-thinking`

### 🔹 Step 1 — Define Job

File: `core/queue/job.go`

```go
type Job interface {
    Execute() error
}
```

### 🔹 Step 2 — Queue Engine

Create queue submission mechanism:

```go
func Submit(job Job)
```

### 🔹 Step 3 — Move AI Calls

Refactor AI generation to run asynchronously via the queue.

**Before:**

```go
ai.Generate()
```

**After:**

```go
queue.Submit(AIJob{})
```

### 🔍 Validation

- Verify that AI model requests no longer block the main API daemon thread.

### ✅ Done When

- Async jobs are processing successfully via the worker queue.

---

## 🖥️ Phase 6 — Remove Python

### ⚙️ Servers

- Primary: `filesystem`

### 🔹 Step 1 — Delete

Delete the legacy Python/Flask files:

```bash
rm webui/server.py
```

### 🔹 Step 2 — Replace

Serve static files directly from Go binary using embedded FS:

```go
http.Handle("/", http.FileServer(embedFS))
```

### 🔍 Validation

- Load the GUI dashboard.
- Verify API requests execute correctly.

### ✅ Done When

- Python runtime is completely removed from the deployment stack.

---

## 🔒 Phase 7 — Hardening

### ⚙️ Servers

- Primary: `docker`, `sqlite`

### 🔹 Step 1 — Idempotency

Ensure duplicate deployment requests return without recreating running instances:

```go
if state == running {
    return nil
}
```

### 🔹 Step 2 — Rollback

Roll back to previous working container configurations if new changes fail:

```go
if traefik fails {
    docker.Down(stack)
}
```

### 🔹 Step 3 — Health Endpoints

Expose standard endpoints for monitoring:

```bash
GET /health
GET /stack/:id
```

### 🔍 Validation

- Terminate Traefik container and verify the system automatically recovers.
- Trigger deployment of a stack twice and verify no duplicate resources are created.

### ✅ Done When

- The system is resilient to runtime failures and operates with standard production safeguards.

---

## 🚀 Final Execution Checklist

Run and mark these checkmarks in order:

- [ ] Phase 1 complete (build passes)
- [ ] Phase 2 manager works
- [ ] Phase 3 routing verified
- [ ] Phase 4 self-healing works
- [ ] Phase 5 queue async
- [x] Phase 6 python removed
- [ ] Phase 7 hardened

---

## 🛠️ Audit Fixes Required

Following an audit, the following fixes are needed to complete the architectural refactoring correctly:

- [ ] **Execute API Layer Integration:** Connect `api/handlers.go` and `api/stacks.go` to the newly formed `StackManager` from Phase 2 instead of using direct docker calls.
- [ ] **Remove Dead Code:** Manually delete `api/proxy.go` and `cli/proxy.go` as intended by Phase 3. Migrate all proxy validation logic over to the Traefik labels generation engine.
