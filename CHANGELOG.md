# Changelog

All notable changes to **M3TAL Core** are documented in this file following the [Keep a Changelog](https://keepachangelog.com/) format and [Semantic Versioning](https://semver.org/).

---

## [0.1.1] - 2026-10-21

### Overview & Release Highlights
Release v0.1.1 establishes the automated release pipeline linking M3tal Core to M3tal-Hub and the central APT keyring repository.

### 📦 Packaging & Debian Distribution
- **feat(ci): auto-sync changelog and version to M3tal-Hub and publish deb to central APT repository** (`0e00b749`):
  - *What Changed*: Automated Debian package publishing via repository dispatch to `jakej985-rgb/m3tal-apt-key` and synchronized version/changelog updates to `M3tal-Hub/apps/manifest.yml`.
  - *Why*: Eliminates manual repo management and ensures seamless distribution across Debian/Ubuntu nodes.

### 🛠️ Developer Toolchains & CI/CD
- **chore: auto-bump version to v0.1.1 [skip ci]** (`cd15a049`):
  - *What Changed*: Incremented patch version following initial release tag setup.
  - *Why*: Synchronizes tags with GitHub release assets and package registry indices.

---

## [0.1.0] - 2026-10-20

### Overview & Release Highlights
Initial repository reset establishing clean v0.1.0 baseline for M3TAL Core — DEB-first infrastructure orchestration platform.

### 🚀 New Features & Capabilities
- **Core CLI & Orchestration Engine**:
  - *What Changed*: Native Go binary `m3tal` providing daemon management, plugin catalog integration, and container orchestration commands.
  - *Why*: Foundation CLI for homelab and edge infrastructure.
- **Embedded Web GUI**:
  - *What Changed*: Vite + React dashboard embedded directly into the control-plane binary and Debian distribution.
  - *Why*: Single-binary deployment with modern visual management console.
- **Debian Distribution & Systemd Integration**:
  - *What Changed*: Standardized packaging via `scripts/build-deb.sh` with systemd unit files (`m3tal.service`, `m3tal-api.service`).
  - *Why*: Production-grade Linux service management out of the box.
