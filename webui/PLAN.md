# m3tal-dashboard Plan (UI)

## Purpose
Visual interface for system.

---

## Goals
- Display system state
- Trigger actions via API
- No backend logic

---

## Data Sources

API only:

```
GET /status
GET /metrics
```

---

## Rules

### 1. No system logic

❌ No:
- Docker commands
- File system access
- Agents

---

### 2. API only communication

```
dashboard → API → core
```

---

### 3. UI responsibilities

- Show container status
- Show system health
- Show metrics
- Trigger actions

---

### 4. Overlay (optional later)

- Window overlay
- System tray integration

---

## Done When

- UI reflects real system state
- All actions go through API
- No direct system access
