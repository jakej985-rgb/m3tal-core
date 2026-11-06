# M3TAL Plugin Authoring Guide

M3TAL plugins are external executables supervised by `m3tal-api`. Core discovers manifests, launches processes, performs a JSON-RPC 2.0 handshake, collects metrics/tables on demand, executes actions, persists settings, and mounts plugin UI assets.

For the ecosystem-wide catalog, see the [M3tal Plugins repository](https://github.com/jakej985-rgb/m3tal-plugins). For the formal target specification, see `plans/plugin/standardization_spec.md`; that specification contains aspirational fields that are not all consumed by the current Core parser.

## Directory layout

Recommended layout:

```text
m3tal-<plugin>/
├── plugin.yaml
├── README.md
├── CHANGELOG.md
├── Makefile
├── bin/
│   └── m3tal-<process>
├── src/
├── ui/
│   └── dist/
│       └── index.html
├── config/
│   └── config.schema.json
└── .agents/
    └── agents/
        └── plugin-maintainer/
            └── agent.md
```

Core currently recognizes `plugin.yaml` or `plugin.yml`. The manifest directory becomes the plugin root and the subprocess working directory.

## Manifest fields consumed by Core

```yaml
apiVersion: m3tal.io/v1alpha1
kind: Plugin
metadata:
  name: docker-monitor
  version: "0.8.1"
  title: "Containers"
  description: "Docker fleet monitor"
  author: "jakej985-rgb"
  homepage: "https://example.invalid"
  icon: "IconContainers"
spec:
  command: "./bin/m3tal-docker"
  args: []
  process_name: "m3tal-docker"
  capabilities:
    - collector
    - actions
    - ui_view
    - ui_widget
    - settings
  ui:
    entry: "ui/dist/index.html"
    route: "/containers"
    category: "containers"
    category_title: "Containers"
    tab_id: "docker_containers"
    title: "Fleet"
    sidebar: true
    order: 20
    roles: ["admin", "viewer"]
  widgets:
    - id: "docker-fleet"
      title: "Containers"
      size: [2, 1]
      type: "card"
  settings:
    - id: "docker_config"
      title: "Docker Host Engine"
      category: "containers"
      fields:
        - key: "docker_host"
          label: "Docker Host Endpoint"
          type: "text"
          default: "unix:///var/run/docker.sock"
```

Recognized manifest fields:

- `metadata.name` — plugin/catalog name. If omitted, Core uses the directory name.
- `metadata.version`, `description`, `author`, `homepage`, `icon`.
- `spec.command` — executable path relative to the manifest directory.
- `spec.args` — command arguments.
- `spec.process_name` — canonical process identity such as `m3tal-docker`.
- `spec.capabilities` — declared capabilities.
- `spec.ui` — static UI mount metadata.
- `spec.widgets` — fallback dashboard widget declarations.
- `spec.settings` — fallback settings sections.
- `config` — arbitrary configuration passed to `m3tal.init`.

Fields such as `metadata.title`, `metadata.license`, `spec.rpc`, `spec.requirements`, and `spec.poll_interval` may appear in plugin manifests for future use or packaging metadata, but the current Core Go structs do not consume them. Do not rely on them for Core behavior.

## Capability vocabulary

Core recognizes these capability constants:

| Capability | Meaning |
|---|---|
| `collector` | Plugin can return metrics/tables through `m3tal.collect`. |
| `actions` | Plugin can execute commands through `m3tal.action`. |
| `ui_view` | Plugin provides a routed UI view. |
| `ui_widget` | Plugin contributes dashboard widgets. |
| `settings` | Plugin exposes configurable settings sections. |
| `exporter` | Reserved for telemetry export integrations. |

Catalog entries may use descriptive free-form capability names, but Core route behavior is driven by the implemented RPC methods, UI metadata, widgets, and settings.

## Process naming

Use canonical process names:

```text
m3tal-<process>
```

Examples:

- `m3tal-docker`
- `m3tal-compose`
- `m3tal-mesh`
- `m3tal-mesh-node`
- `m3tal-samba-nfs`

If the executable name is not canonical, Core may create a runtime symlink under the user/system runtime directory so process monitoring sees the expected name. First-party plugins should still compile directly to the canonical name.

## JSON-RPC 2.0 protocol

Transport is newline-delimited JSON-RPC 2.0 over stdin/stdout. Do not write logs to stdout; use stderr for diagnostics.

### `m3tal.init`

Core sends this immediately after process startup and waits for the result before marking the plugin running.

Request params:

```json
{
  "core_version": "0.24.6",
  "config": {}
}
```

Expected result:

```json
{
  "name": "docker-monitor",
  "version": "0.8.1",
  "capabilities": ["collector", "actions", "ui_view", "settings"],
  "tabs": [
    {
      "id": "docker_containers",
      "title": "Fleet",
      "plugin": "docker-monitor",
      "url": "/plugins/docker-monitor/",
      "category": "containers",
      "category_title": "Containers",
      "order": 20,
      "actions": []
    }
  ],
  "actions": [
    {
      "id": "restart_container",
      "label": "Restart",
      "description": "Restart a container",
      "confirm": true
    }
  ],
  "widgets": [],
  "settings": []
}
```

If `tabs`, `widgets`, or `settings` are empty, Core falls back to the corresponding `plugin.yaml` declarations where available.

### `m3tal.collect`

Request params:

```json
{
  "timestamp": 1760000000
}
```

Expected result:

```json
{
  "status": "ok",
  "metrics": [
    {
      "name": "containers_running",
      "value": 4,
      "unit": "count",
      "category": "docker",
      "status": "ok"
    }
  ],
  "tables": [
    {
      "id": "containers",
      "title": "Containers",
      "headers": ["Name", "State"],
      "rows": [["web", "running"]]
    }
  ],
  "message": ""
}
```

`status` may be `ok`, `degraded`, or `error`. Collection is invoked on demand through Core; there is no Core-side scheduled collector loop for plugin `poll_interval` today.

### `m3tal.action`

Request params:

```json
{
  "action_id": "restart_container",
  "target": "web",
  "args": {}
}
```

Expected result:

```json
{
  "success": true,
  "message": "Container restarted",
  "error": "",
  "data": {},
  "payload": {}
}
```

Plugin-specific fields may be returned at the top level; Core preserves unknown result fields in `payload`.

## Environment variables injected by Core

| Variable | Purpose |
|---|---|
| `M3TAL_CORE_VERSION` | Core version string. |
| `M3TAL_PLUGIN_DIR` | Plugin manifest directory. |
| `M3TAL_PROCESS_NAME` | Canonical process name. |
| `M3TAL_API_KEY` | Credential for Core API calls. |
| `M3TAL_API_TOKEN` | Compatibility alias for the same credential. |

Core resolves the credential in this order:

1. `M3TAL_API_KEY`
2. `M3TAL_API_TOKEN`
3. `API_TOKEN`
4. `~/.config/m3tal/api_keys.json` primary token
5. `/etc/m3tal/api_keys.json` primary token fallback

Do not log API keys or write them to stdout.

## UI contract

If `spec.ui` is configured, Core mounts the plugin UI at `/plugins/<metadata.name>/`.

- `spec.ui.entry` should point to the plugin's HTML entry, commonly `ui/dist/index.html`.
- `spec.ui.route` is the dashboard route, for example `/containers`.
- `sidebar: true` adds the view to navigation.
- `roles` controls which dashboard roles can see the tab.
- The UI should use relative asset paths so it works under `/plugins/<name>/`.

## Settings contract

Plugin settings can be declared statically in `plugin.yaml` and/or returned by `m3tal.init`.

Core stores plugin config values at:

```text
~/.config/m3tal/plugins/<plugin>/config.json
```

When a user saves settings through Core, Core also dispatches a `set_settings` action to a running plugin. Plugins that support live reload should implement that action.

Supported field types in the current GUI/settings model include `text`, `password`, `number`, `boolean`, and `select`.

## Packaging

First-party plugins are packaged as Debian packages named:

```text
m3tal-plugin-<name>
```

Install with:

```bash
sudo apt update
sudo apt install m3tal-plugin-<name>
```

After installation, Core discovers the plugin through its normal search paths and SIGHUP refresh.

## Testing checklist

Before release:

1. `plugin.yaml` parses and declares a unique `metadata.name`.
2. Executable path is relative to the plugin directory and uses a canonical `process_name`.
3. Process responds to `m3tal.init`, `m3tal.collect`, and any advertised `m3tal.action` calls.
4. No stdout logging contaminates JSON-RPC.
5. UI assets exist at the configured entry point.
6. Settings persist through `~/.config/m3tal/plugins/<name>/config.json`.
7. Plugin starts, stops, restarts, and survives Core restart.
8. Debian package installs into the expected plugin directory and triggers Core discovery.

## Common mistakes

- Writing human-readable logs to stdout instead of stderr.
- Returning plugin actions that are not declared in `m3tal.init`.
- Using absolute UI asset paths that break under `/plugins/<name>/`.
- Expecting `m3tal.shutdown`; Core does not call it today.
- Treating manifest-only `capabilities` as proof that RPC methods exist; Core uses runtime init results and actual behavior.
