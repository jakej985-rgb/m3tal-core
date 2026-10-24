# Changelog

All notable changes to **M3TAL Core** are documented in this file following the [Keep a Changelog](https://keepachangelog.com/) format and [Semantic Versioning](https://semver.org/).

---

## [0.5.0] - 2026-10-24

### Overview & Release Highlights
Release v0.5.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(plugins): implement auto-sensing and auto-start for newly installed plugins** (`587a684`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.4.2] - 2026-10-24

### Overview & Release Highlights
Release v0.4.2 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **refactor(cli): decouple plugin catalog generation and purge legacy tui_widget capability** (`96e251d`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.4.1] - 2026-10-24

### Overview & Release Highlights
Release v0.4.1 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **fix(cli): add 'plugin catalog --export' command for CI release automation** (`f549cbe`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.4.0] - 2026-10-24

### Overview & Release Highlights
Release v0.4.0 delivers updates across 8 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **style(api): format Go files according to gofmt** (`4b2d3aad`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **feat(plugins): add plugin catalog & install endpoints and update plugin-author workflow** (`77a1c6d6`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **chore(ci): update plugin catalog repo reference to m3tal-plugins** (`df50ce09`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 🎨 Web GUI & Dashboard
- **feat(gui): add changelog modal, deb download, and sudo install confirmation to plugins page** (`898fcdee`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **docs(plugins): require CHANGELOG.md across plugin spec and agent suite** (`82523dcc`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **feat(plugins): enforce canonical process naming m3tal-<process> across supervisor and agent suite** (`641ff4bb`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **docs(plugins): align agent suite and specs with m3tal-plugins monorepo and m3tal-apt-key signing** (`629db631`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **feat(agents): add M3TAL plugin agent suite (director, auditor, author, tester, packager)** (`9fff7c6f`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.3.0] - 2026-10-23

### Overview & Release Highlights
Release v0.3.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(plugins)!: purge legacy plugin remnants and standardize on pkg/plugins.Manager** (`12e2850e`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.2.0] - 2026-10-23

### Overview & Release Highlights
Release v0.2.0 delivers updates across 2 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **chore(plugins): purge bundled plugins from core and document plugin catalog/standardization spec (v0.1.19)** (`d4993650`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 🎨 Web GUI & Dashboard
- **feat(dashboard): integrate expandable host info & subsystem health into top bar, purge legacy deploy plugins & catalog (v0.1.20)** (`68c3294b`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

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
