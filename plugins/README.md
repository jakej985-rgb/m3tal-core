# M3TAL Local Plugins Directory

This directory is an active search path for locally installed M3TAL plugins.
External plugins adhere to the specification detailed in [plans/plugin/standardization_spec.md](../plans/plugin/standardization_spec.md).

All plugin code (backend executable, frontend UI bundle, RPC handlers, manifest, and configuration schemas) lives strictly inside its own plugin directory.

To install a plugin locally, extract or clone its package into this directory (or into `~/.config/m3tal/plugins/` or `/var/lib/m3tal/plugins/`).
