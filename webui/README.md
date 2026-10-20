# 🖥️ M3TAL GoDash (Visual Interface)

**DocSmith Status:** *Modernization Applied | Go-Native Migration: Phase 1 (Python Legacy Wrapper)*

GoDash serves as the **Visualization & Control Layer** for the M3TAL ecosystem. It provides the "Mission Control" interface required for monitoring container health, system metrics, and AI node stability.

---

## 🏗️ Architectural Role
As a core component of the M3TAL ecosystem, GoDash functions as the dedicated frontend client for the `m3tal-goback` service. 

*   **System Integration:** GoDash acts as the presentation layer, relying entirely on the **`m3tal-goback`** backend API to perform state mutations and telemetry retrieval.
*   **Infrastructure Consistency:** GoDash adheres to M3TAL path standards, expecting volumes mounted at `/mnt/m3tal` for consistent log persistence and node state tracking across the stack.
*   **Decoupling:** As part of the Go-native migration, this interface is undergoing a transition to serve static assets directly, effectively decoupling the presentation layer from the current Python-based legacy controller.

---

## 📦 Deployment (Docker)

> [!IMPORTANT]
> GoDash is designed for the unified M3TAL mesh. Standalone deployment requires manual resolution of the `proxy` network for backend communication.

### 1. Configuration
Establish the handshake between the Dashboard and the Backend via `.env`:

```ini
BACKEND_URL=http://m3tal-goback:8080
DASHBOARD_SECRET=your_shared_secret
STATE_DIR=/mnt/m3tal/state
DEBUG=false
```

### 2. Launch
```bash
# Build the M3TAL image
docker build -t ghcr.io/jakej985-rgb/m3tal-godash:latest .

# Launch into the M3TAL mesh
docker run -d \
  --name m3tal-godash \
  --network proxy \
  -v /mnt/m3tal:/mnt/m3tal \
  -p 8080:8080 \
  --env-file .env \
  ghcr.io/jakej985-rgb/m3tal-godash:latest
```

---

## 🛠 Development
The current architecture utilizes a Flask-based wrapper. To contribute to the Go-native migration or update UI components:

```bash
# Environment Setup
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Launch Debug Mode
python server.py
```

*Note: For performance testing, ensure `BACKEND_URL` is configured to point to a local instance of the `m3tal-goback` development build.*

---

## 🔗 Related Projects
This repository is an integrated module within the M3TAL infrastructure. For a complete system deployment, refer to:

*   **[m3tal-media-server](https://github.com/jakej985-rgb/m3tal-media-server)**: The M3TAL Orchestrator/Core.
*   **[m3tal-goback](https://github.com/jakej985-rgb/m3tal-goback)**: The Go-native Backend API (The Ecosystem Brain).

---
*DocSmith: M3TAL Documentation Architecture | Protocol v2.0*