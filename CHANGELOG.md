# Changelog

All notable changes to **M3TAL Core** are documented in this file following the [Keep a Changelog](https://keepachangelog.com/) format and [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

### Fixed
- Build the CLI and API for the AMD64 v1 baseline with cgo disabled, and gate Debian releases on package installation and AVX-less API startup checks.
- Validate plugin names and package/mode parameters against path traversal before filesystem and `dpkg` operations (`api/validate.go`); reject `..`, separators, and oversized names at every `{name}` endpoint.
- `findPluginDeb` rejects traversal names and no longer searches the process working directory (`./plugins`); `M3TAL_PLUGINS_DIR` is now the supported override for extra plugin directories.
- Plugin discovery no longer scans `./plugins`, so running the binary from a repository checkout cannot auto-start planted plugins.
- Plugin subprocesses inherit a filtered environment (allowlisted variables only) instead of `os.Environ()`.
- `NormalizePermissions` rejects empty or unknown scopes instead of silently granting `*` full access.
- `KillProcess` refuses PID 1, the API's own PID, and protected system processes.
- HTTP server now sets `ReadHeaderTimeout`, `ReadTimeout`, and `IdleTimeout`; WebSocket `CheckOrigin` validates same-origin and `M3TAL_WS_ALLOWED_ORIGINS` instead of accepting any origin.
- API token comparisons use constant-time comparison.

## [0.25.1] - 2026-11-10

### Overview & Release Highlights
Release v0.25.1 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### ⚠️ Breaking Changes & Migrations
- **chore: remove leftover Entire CLI hook template and checkpoint ref from changelog** (`6683b06`):
  - *What Changed*: Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.25.0] - 2026-11-10

### Overview & Release Highlights
Release v0.25.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(api): request validation, permissions tests, and plugin host updates** (`9ebc9b2`):
  - *What Changed*: Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.7] - 2026-11-06

### Overview & Release Highlights
Release v0.24.7 delivers updates across 2 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 📚 Documentation & Specifications
- **Merge pull request #147 from jakej985-rgb/docs/ecosystem-overhaul** (`3fb8f4d`):
  - *What Changed*: docs: ecosystem-wide documentation overhaul
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **docs: ecosystem-wide documentation overhaul** (`88af6cc`):
  - *What Changed*: Rewrite user/operator docs to match implemented behavior (API v2 on port 5050, m3tal-api.service, state.db, plugin env injection, SIGHUP discovery) and add OPERATIONS, SECURITY, DEVELOPMENT, PLUGIN_AUTHORING, and RELEASE guides. Sync plugin catalog generator, GUI fallback catalog, audit capabilities, and agent docs with the real plugin protocol. Disclaim unimplemented M3TAL_PLUGINS_DIR and Core-side m3tal.shutdown.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.6] - 2026-11-05

### Overview & Release Highlights
Release v0.24.6 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **⚡ Bolt: parallelize SystemMonitor poll requests (#146)** (`285a2de`):
  - *What Changed*: * ⚡ Bolt: parallelize SystemMonitor poll requests  fetchData awaited metrics, history, processes, and specs sequentially every 3.5s — wall-clock per refresh was the sum of all latencies. Promise.all collapses it to the slowest call; tab-conditional fetches resolve to null when skipped.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  * chore: remove dead EventBus publish-subscribe code  core/events.EventBus had zero non-test callers — the live websocket path uses ChannelEventBus exclusively. Removing ~150 lines of dead code including its test file.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  ---------  Co-authored-by: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.5] - 2026-11-05

### Overview & Release Highlights
Release v0.24.5 delivers updates across 2 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **⚡ Bolt: polling-path performance batch (#145)** (`0c20e9a`):
  - *What Changed*: * perf(api): dedupe API key lookup on authenticated requests  Authenticate() called LookupAPIKey() and then ValidateAPIKey(), which re-ran the same SELECT internally — two identical queries per request, plus the last_used_at UPDATE. Extract RecordAPIKeyUsage() so the auth middleware stamps usage on the record it already holds.  Queries per authenticated request drop from 2 SELECTs (+1 UPDATE when stale) to 1 SELECT (+1 UPDATE when stale); on the common throttled path (last use <60s ago, e.g. dashboard polling) it halves DB round-trips.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  * docs(journal): record auth double-lookup learning [skip ci]  * perf: batch of polling-path optimizations across API and GUI  Backend: - Cache plugin catalog responses for 30s (was: manifest discovery +   hundreds of .deb/CHANGELOG filesystem probes per 12s poll); invalidated   on install/uninstall/repo changes via InvalidateAptCache - Serve ListPlugins/buildPluginCatalog from the Manager's discovery   snapshot (refreshed by the 5s auto-discovery loop) instead of   rescanning plugin dirs per request - Cache apt-cache search results for 60s alongside the existing policy cache - Add chi Compress for JSON API responses (~5-10x smaller poll payloads) - Cache nvidia-smi results for 5s and inxi -m for process lifetime;   specs endpoint uses the cached GPU path too - Derive process memory percent from a single MemoryInfo read + one   VirtualMemory call instead of gopsutil's per-process double read - Log only errors and >200ms requests (polling emitted ~3 log lines/sec) - Immutable Cache-Control on fingerprinted /assets  Frontend: - Dashboard fires its 4 poll requests concurrently and fetches the   immutable version once instead of every 5s - fetchAPI gains a 30s AbortController timeout  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  * docs(journal): record polling-path optimization learnings [skip ci]  * fix(plugins): single stdout reader with ID demux for JSON-RPC calls  callLocked spawned a ReadBytes goroutine per call; on context timeout the goroutine survived and kept reading, leaking goroutines and racing the next call's reader — a late response could be consumed and discarded by the zombie, corrupting the following call.  A single readLoop goroutine now owns plugin stdout for the process lifetime and demuxes responses to pending callers by request ID. Calls are serialized by p.mu so at most one is in flight; ID-mismatched responses fall back to the sole pending call for compatibility with sloppy plugins. On stdout EOF/error all pending calls are failed immediately instead of hanging until context cancellation.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  * docs(journal): record JSON-RPC reader demux learning [skip ci]  ---------  Co-authored-by: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 🏗️ Architecture & Core System Primitives
- **⚡ Bolt: dedupe API key lookup on authenticated requests (#144)** (`d3b13b0`):
  - *What Changed*: * perf(api): dedupe API key lookup on authenticated requests  Authenticate() called LookupAPIKey() and then ValidateAPIKey(), which re-ran the same SELECT internally — two identical queries per request, plus the last_used_at UPDATE. Extract RecordAPIKeyUsage() so the auth middleware stamps usage on the record it already holds.  Queries per authenticated request drop from 2 SELECTs (+1 UPDATE when stale) to 1 SELECT (+1 UPDATE when stale); on the common throttled path (last use <60s ago, e.g. dashboard polling) it halves DB round-trips.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  * docs(journal): record auth double-lookup learning [skip ci]  ---------  Co-authored-by: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.4] - 2026-11-05

### Overview & Release Highlights
Release v0.24.4 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 📦 Packaging & Debian Distribution
- **fix(ci): drop stale xdotool Recommends assertion from DEB compare step** (`a7f0883`):
  - *What Changed*: The strip-core refactor removed recommends: xdotool from both packagers; the CI assertion was left behind and failed silently (test produces no output). Verified locally that nfpm and dpkg-deb packages are identical.  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.3] - 2026-11-05

### Overview & Release Highlights
Release v0.24.3 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **chore(plugins): rename samba-manager catalog entry to samba-nfs (#142)** (`72f88e9`):
  - *What Changed*: The Samba/NFS share plugin shipped as m3tal-samba-nfs (m3tal-plugins feat/samba-nfs-plugin). Update the built-in catalog, plugin alias map, GUI fallback list, catalog generator, roadmap doc, and agent docs to match the new name, package, process identity, and valid capabilities.  Generated with [Devin](https://devin.ai)  Co-authored-by: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.2] - 2026-11-05

### Overview & Release Highlights
Release v0.24.2 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **refactor: strip core to web dashboard, system monitor, and plugin host (#141)** (`4f21b7e`):
  - *What Changed*: * docs: add comprehensive Docker and container orchestration guide  * refactor: strip core to web dashboard, system monitor, and plugin host  M3TAL Core is now only the embedded Web GUI, host/GPU telemetry, system monitor, settings/API keys, and the external plugin supervisor. All workload features (Docker, Compose, stacks, Traefik, VPN, agents, reconcile, doctor, queue, CLI/TUI/tray) are removed from core and left to external plugins.  - Delete cli/, pkg/client|config|output|cmdutil|registry|compose, and   core workload packages (agents, docker, doctor, networking,   orchestrator, preflight, queue, stack, traefik) - Slim API to core routes: version, system stats/metrics/health/   processes/specs/network, plugins, settings, repositories, API keys,   and the events WebSocket - Remove TUI plugin protocol (tui_widget, m3tal.render) and stale models - Reduce state store to API-key persistence; drop route/stack/middleware - Clean GUI of Docker/stack/VPN/maintenance calls and dead widgets - Build/package only m3tal-api; systemd unit no longer depends on Docker - Update .env.example, scripts, and docs to match the new boundary  Generated with [Devin](https://devin.ai)  Co-Authored-By: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>  ---------  Co-authored-by: Devin <158243242+devin-ai-integration[bot]@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.1] - 2026-11-05

### Overview & Release Highlights
Release v0.24.1 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **⚡ Bolt: optimize disk partition label resolution in GetDiskPartitions (#140)** (`79ddc5b`):
  - *What Changed*: Batch scan /dev/disk/by-label once before partition inspection loop to eliminate repeated O(N*M) filesystem directory reads and os.Readlink syscalls on every metrics poll.  Co-authored-by: google-labs-jules[bot] <161369871+google-labs-jules[bot]@users.noreply.github.com> Co-authored-by: jakej985-rgb <224218482+jakej985-rgb@users.noreply.github.com>
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.24.0] - 2026-11-01

### Overview & Release Highlights
Release v0.24.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(auth): Jellyfin-style API key manager and plugin sync (#139)** (`e3bf1d5`):
  - *What Changed*: * feat(auth): add Jellyfin-style API key manager with server generation and plugin inheritance  * fix(review): remediate auth validation, atomic file sync, test isolation, and key resolution  * fix(review): deduplicate auth token validation and unify primary key sync  * fix(review): isolate migrations, return ErrAPIKeyNotFound, and clean temp files  * style(plugins): format pkg/plugins/host.go with gofmt
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.23.0] - 2026-10-31

### Overview & Release Highlights
Release v0.23.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **feat(agents): align push-gatekeeper for main releases and review-orchestrator for feature loop** (`7fe5586`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.22.0] - 2026-10-31

### Overview & Release Highlights
Release v0.22.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(review): integrate multi-agent review engine and push gatekeeper hooks** (`512fab0`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.21.0] - 2026-10-31

### Overview & Release Highlights
Release v0.21.0 delivers updates across 7 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **Merge pull request #138 from jakej985-rgb/feat/plugin-dpkg-triggers** (`d67a6cc`):
  - *What Changed*: feat(packaging): add dpkg trigger support for plugins
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **feat(plugins): handle SIGHUP for immediate plugin rediscovery on dpkg triggers** (`2174c7d`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 📦 Packaging & Debian Distribution
- **fix(plugins): register SIGHUP handler in RunServer instead of package init** (`0c59b2b`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(plugins): register SIGHUP early and target systemd unit safely** (`35f93e8`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(packaging): prevent trigger restart cgroup termination and clean packaging files** (`8f7fd82`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **feat(packaging): add dpkg trigger support for plugins** (`86604de`):
  - *What Changed*: Register interest-noawait on /usr/lib/m3tal/plugins in packaging/nfpm.yaml and packaging/deb/DEBIAN/triggers. Add trigger handler in postinst scripts to restart m3tal services on plugin changes.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 🐛 Bug Fixes & Stability
- **fix(plugins): initialize SIGHUP in init and reuse shared GlobalPluginManager** (`6ad17ec`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.7] - 2026-10-29

### Overview & Release Highlights
Release v0.20.7 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **fix(plugins): preserve custom action payload fields** (`8c8d1de`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.6] - 2026-10-28

### Overview & Release Highlights
Release v0.20.6 delivers updates across 4 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **Merge pull request #137 from jakej985-rgb/codex/mesh-pairing-download** (`d55fd6a`):
  - *What Changed*: fix(plugins): download the selected mode package
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 📦 Packaging & Debian Distribution
- **fix(plugins): prefer versioned package archives** (`7a79d58`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(plugins): find exact package archives** (`934a721`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(plugins): download the selected mode package** (`b26d1bc`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.5] - 2026-10-27

### Overview & Release Highlights
Release v0.20.5 delivers updates across 5 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **fix(build): include process lock on macOS** (`866e0e0`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(deb): build API for AMD64 v1 baseline** (`0da3fe4`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 📦 Packaging & Debian Distribution
- **Merge pull request #136 from jakej985-rgb/codex/deb-cpu-compatibility** (`dbf023f`):
  - *What Changed*: fix(deb): build API for AMD64 v1 baseline
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(release): reject reused Debian versions** (`648a4a4`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(deb): address compatibility trail findings** (`b443104`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.4] - 2026-10-27

### Overview & Release Highlights
Release v0.20.4 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 📦 Packaging & Debian Distribution
- **docs: add Debian package CPU compatibility fix plan** (`a8621d0`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.3] - 2026-10-27

### Overview & Release Highlights
Release v0.20.3 delivers updates across 3 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **Merge pull request #135 from jakej985-rgb/codex/fix-core-workflows** (`e54906f`):
  - *What Changed*: fix(ci): resolve release and catalog test failures
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.
- **fix(ci): resolve release and catalog test failures** (`62d8aa9`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 📚 Documentation & Specifications
- **Update standardization_spec.md** (`4e1e2cc`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.2] - 2026-10-26

### Overview & Release Highlights
Release v0.20.2 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **fix(plugins): add automatic dependency recovery on unmet dependencies** (`d9df72ee`):
  - *What Changed*: If low-level dpkg -i encounters unmet dependencies, automatically invoke sudo apt-get install -f -y to resolve, download, and configure missing dependencies directly from the APT keyring.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.20.1] - 2026-10-26

### Overview & Release Highlights
Release v0.20.1 resolves mode and package resolution for plugin installation across the Web GUI and daemon API, ensuring accurate targeting for multi-variant plugin packages like `m3tal-plugin-mesh`.

### 🐛 Bug Fixes & Improvements
- **fix(plugins): resolve mode and explicit package name targeting during installation**:
  - *What Changed*: Updated `gui/src/pages/Plugins.jsx` to pass `package` and `mode` from `getActiveMode(installTarget)` to `api.installPlugin`. Updated `api/plugins.go` `InstallPlugin` handler to parse `package` and `mode` payload fields and prioritize searching explicit package and mode target variations (`<package>`, `<mode>`, `<name>-<mode>`, `m3tal-<name>-<mode>`, `m3tal-plugin-<name>-<mode>`) when resolving the `.deb` archive.
  - *Why*: Multi-variant plugins (such as Mesh, Mesh Node, and Mesh Cluster) require targeted package identification rather than relying solely on the base plugin name during installation and mode switching.

---

## [0.20.0] - 2026-10-26

### Overview & Release Highlights
Release v0.20.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(plugins): implement dynamic APT candidate discovery and repository management** (`da0eface`):
  - *What Changed*: - Eliminate hardcoded plugin versions from Core source code (api/plugins.go, gui/src/pages/Plugins.jsx) - Implement dynamic APT candidate version discovery in api/apt.go using apt-cache policy with in-memory caching and cache invalidation - Implement Plugin Repositories management API in api/repositories.go (GET/POST/DELETE repositories, POST refresh) and register routes in api/server.go - Add Plugin Repositories section to gui/src/pages/Settings.jsx with keyring status, release channel toggle, live discovered APT candidate table, and custom repository modal - Add Check for Updates live APT polling button to gui/src/pages/Plugins.jsx
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.19.1] - 2026-10-26

### Overview & Release Highlights
Release v0.19.1 enhances the plugin management ecosystem and runtime lifecycle, introducing seamless 1-click plugin updates with graceful termination and automatic restarts, multi-mode plugin variant selection for `m3tal-mesh`, and catalog synchronization for `docker-monitor` at v0.6.0.

### 🚀 Enhancements & Packaging
- **Docker Monitor Catalog Sync**:
  - Synchronized default catalog entry and fallback for `docker-monitor` to v0.6.0 (matching the active APT repository package release) ensuring accurate "Up to Date" status reporting.
- **Mesh Multi-Mode Selection**:
  - Added explicit mode selection (`cluster` for Full Peer, `mesh` for Viewer, `node` for Headless Sender) across API catalog and GUI with custom sub-labels, descriptions, and dynamic install/download routing.

### 🐛 Bug Fixes & Stability
- **fix(plugins): implement clean plugin update lifecycle with automatic restart and zero host downtime** (`23f62a1b`):
  - *What Changed*: Gracefully stops plugin subprocess before dpkg extraction to prevent ETXTBSY binary lock and stale code execution; automatically restarts running plugins with the newly installed version after package installation and manifest re-discovery; ensures zero host downtime for host telemetry, WebSocket streams, and independent plugins during upgrades.
  - *Why*: Eliminates binary lock errors during live package upgrades and ensures uninterrupted service.

---

## [0.19.0] - 2026-10-26

### Overview & Release Highlights
Release v0.19.0 introduces the **Dynamic Core Version System** and the **Smart Versioning Engine** for M3TAL Core and the Plugin Manager. It eliminates all hardcoded version labels across the platform, exposes canonical version endpoints, powers intelligent SemVer evaluation for external plugins with capability diff inspection, and enables 1-click plugin upgrades directly from the GUI.

### 🚀 New Features & Capabilities
- **Dynamic Core Version System**:
  - Implemented `pkg/version` package providing `GetVersion()` and `GetInfo()` with multi-tiered resolution: `/etc/m3tal/VERSION`, working directory paths, executable directory lookup, and fallback default.
  - Added public `/version` and `/api/v2/version` REST endpoints returning structured version metadata (`version`, `display`, `api_version`, `codename`).
  - Dynamic Web GUI integration in `gui/src/App.jsx`, `TopBarWidget.jsx`, and `Dashboard.jsx`, displaying live versioning (`v0.19.0`) and eliminating obsolete `v2.0` labels.
  - CLI version resolution in `cli/main.go` hooked directly into `pkg/version`.
- **Smart Versioning Engine for Plugins**:
  - Added `CompareSemVer` in `api/plugins.go` and SemVer evaluation in `GetPluginCatalog` to calculate `update_status` (`up_to_date`, `update_available`, `dev_ahead`), `update_type` (`patch`, `minor`, `major`, `dev`), and `capabilities_diff`.
  - Enriched Plugin Manager GUI (`gui/src/pages/Plugins.jsx`) with dynamic status badges, transition indicators (`vInstalled → vCatalog`), and capability diff tags highlighting new capabilities introduced in updated releases.
  - Upgraded Plugin Install Modal to support 1-click plugin upgrades with version transition details and customized action styling.

### 🏗️ Architecture & Core System Primitives
- **Debian Packaging & Systemd Integration**:
  - Staged `/etc/m3tal/VERSION` and `/usr/share/m3tal/VERSION` across Debian package generation (`scripts/build-deb.sh` and `packaging/nfpm.yaml`).
  - Automated version bump synchronization across `VERSION`, `gui/package.json`, `pkg/version/version.go`, and `CHANGELOG.md`.

### 🛡️ Governance & Agent Systems
- Added `plugin-widget-builder` governance agent and updated `plugin-director` orchestration flow for dashboard widget micro-frontends.

### 🧪 Tests & Quality Assurance
- Added unit tests in `api/version_test.go` and `pkg/version/version_test.go`.
- Added SemVer comparison tests in `api/plugins_test.go` (`TestCompareSemVer`).

---

## [0.18.0] - 2026-10-26

### Overview & Release Highlights
Release v0.18.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(settings): implement dedicated settings page and dynamic plugin settings architecture** (`3e7cea8a`):
  - *What Changed*: Introduce dedicated Settings page with core host maintenance and preferences alongside dynamic registration and config dispatch for plugins declaring CapabilitySettings.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.17.0] - 2026-10-26

### Overview & Release Highlights
Release v0.17.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(gui): implement uniform modular block grid with strict multi-span multiples** (`0fe79f1c`):
  - *What Changed*: - Exact, uniform 1x1 base unit dimensions (H = 260px, W = 1 column in a 4-column responsive grid) - Pixel-identical 1x1 stat cards (CPU, RAM, GPU, Root Disk) with integrated VRAM badge and sparkline - Strict multi-span multiples: 1x1 (260px), 2x1 (260px), 4x2 (544px), and 4x3 (828px) - TopBarWidget and Edit Mode Banner hoisted above the modular grid as a full-width header - Grid container configured with auto-rows-[260px]
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.16.0] - 2026-10-26

### Overview & Release Highlights
Release v0.16.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(plugins): establish plugin widget ownership and decouple container fleet widget from core dashboard** (`5fdfa8a5`):
  - *What Changed*: - Decouple container fleet widget from DEFAULT_LAYOUT in core host dashboard - Dynamically mount docker-fleet widget via PluginWidgetSlot when provided by m3tal-docker - Standardize ui_widget capability and schema in plans/plugin/standardization_spec.md - Update plugin governance agents (.agents/agents/plugin-*) to enforce widget ownership, 4-column grid constraints, and optionality
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.15.0] - 2026-10-26

### Overview & Release Highlights
Release v0.15.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(gui): implement dashboard edit mode and grid layout persistence** (`bb5952b1`):
  - *What Changed*: - Add edit mode toggle (Edit Grid / Done Editing) in TopBarWidget - Implement reorder controls (move left/right) and hide action with amber toolbar across all widgets - Add drag-and-drop widget reordering and hidden widget management in Dashboard - Persist custom widget layout and visibility preferences in localStorage with reset option - Update WidgetCard, StatCardWidget, StorageDetailedWidget, FleetMiniOverview, NetworkSummary, EventLogWidget, and PluginWidgetSlot
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.14.0] - 2026-10-26

### Overview & Release Highlights
Release v0.14.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(dashboard): overhaul container fleet status overview and host network stats widget** (`c8ac2d19`):
  - *What Changed*: - Add color-coded status badges in FleetMiniOverview for healthy, action required, and crashed/version conflicts - Transform network summary into dedicated host network telemetry widget with interface list, IP assignments, and RX/TX bandwidth metrics - Introduce GET /api/v2/system/network endpoint and include network interfaces in DetailedStats
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.13.0] - 2026-10-26

### Overview & Release Highlights
Release v0.13.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(dashboard): modular 4-column widget grid and comprehensive storage management** (`4a7b9e44`):
  - *What Changed*: - introduce modular 4-column responsive grid layout on dashboard - add StatCardWidget, TopBarWidget, StorageDetailedWidget, and EventLogWidget - support dynamic plugin widgets with ui_widget capability and schema in protocol - implement safe USB drive unmount endpoint with protected system directory guards - filter mountpoint noise (docker, proc, sys, efi/boot) and group physical storage partitions
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.12.1] - 2026-10-26

### Overview & Release Highlights
Release v0.12.1 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **style: apply gofmt -s formatting to plugins_test.go and protocol.go** (`364ab6c8`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.12.0] - 2026-10-26

### Overview & Release Highlights
Release v0.12.0 delivers updates across 2 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🚀 New Features & Capabilities
- **feat(catalog): add m3tal-mesh cluster entry with segmented mode selector** (`87f686b`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 📦 Packaging & Debian Distribution
- **refactor(packaging): purge legacy stack templates, /docker symlink, and duplicate service** (`b793b9e`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.11.0] - 2026-10-24

### Overview & Release Highlights
Release v0.11.0 delivers updates across 2 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(gui): group Fleet and Compose plugins under Containers with top sub-tabs** (`ef20e9f`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

### 🐛 Bug Fixes & Stability
- **fix(plugins): map roadmap aliases to plugin folders and synthesize changelog specs** (`93891bd`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.10.0] - 2026-10-24

### Overview & Release Highlights
Release v0.10.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(plugins): enhance plugin catalog with roadmap specs, modern icons, and apt commands** (`69b283b`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.9.0] - 2026-10-24

### Overview & Release Highlights
Release v0.9.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(gui): implement dynamic plugin category grouping and top sub-tab switcher** (`e2d7de3`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.8.2] - 2026-10-24

### Overview & Release Highlights
Release v0.8.2 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 📚 Documentation & Specifications
- **docs(spec): document direct execution for canonical binaries and runtime symlinks** (`3fcaa5d`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.8.1] - 2026-10-24

### Overview & Release Highlights
Release v0.8.1 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **fix(plugins): execute canonical binaries directly and prefer runtime directory for symlinks** (`18f1e74`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.8.0] - 2026-10-24

### Overview & Release Highlights
Release v0.8.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🎨 Web GUI & Dashboard
- **feat(plugins): implement dynamic plugin UI file server and iframe mounting** (`64d5fcf`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.7.0] - 2026-10-24

### Overview & Release Highlights
Release v0.7.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🏗️ Architecture & Core System Primitives
- **feat(system): rename chrome processes to m3tal-chrome-* and differentiate renderers by client ID** (`ae6efe2`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

---

## [0.6.0] - 2026-10-24

### Overview & Release Highlights
Release v0.6.0 delivers updates across 1 commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.

### 🐛 Bug Fixes & Stability
- **feat(system): differentiate sandboxed zygote (SB) and fix crashpad process labeling** (`11e3f4d`):
  - *What Changed*: Implemented changes described in commit.
  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.

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
