# Debian AMD64 CPU baseline and release checklist

The `m3tal` Debian package supports the original x86-64 baseline (Go `GOAMD64=v1`),
including the Intel Core i5-750. The CLI and API are built through `scripts/build-go.sh`
with `CGO_ENABLED=0`; this prevents C compiler defaults from selecting instructions
above the supported baseline. The module and CI use the Go version declared in `go.mod`.

Use `./build.sh`, `make build`, or `./scripts/build-deb.sh` for local Linux builds.
All of these use the shared binary build settings. The Debian package remains one
`m3tal` package containing the CLI, API, GUI assets, configuration templates, and
service files.

## Incident diagnosis

The released `0.20.3` API package was reproduced with `qemu-x86_64 -cpu Nehalem`;
it terminated with exit status 132 (`SIGILL`). Its Go build metadata records
`go1.25.0`, `GOAMD64=v1`, `CGO_ENABLED=1`, and empty `CGO_CFLAGS`, `CGO_CPPFLAGS`,
`CGO_CXXFLAGS`, and `CGO_LDFLAGS`. The function `x_cgo_getstackbound` at `0x401920`
contains AVX instructions, including `vmovq` (`c5 fa 7e`), `vpinsrq` (`c4 e3 f1 22`),
and `vmovdqu` (`c5 fa 7f`). This helper is part of Go's runtime/cgo C implementation:
the Go AMD64 v1 setting did not constrain the separate C compilation. The binary
reports Go 1.25.0 even though the old workflow requested Go 1.23; the workflows now
select the module's declared version directly with `go-version-file: go.mod`.

The released binary does not record the C compiler's configured defaults or all
compiler invocation details, so the specific setting that enabled AVX cannot be
recovered from the artifact. Disabling cgo for this application build removes that
compiler-dependent path. The package test runs the API under the AVX-less Nehalem
model and checks its authenticated health endpoint.

## Release checks

CI and the release workflow must pass these checks before publication:

- `go mod verify`, Go tests, formatting, and clean package build.
- Package metadata and contents for the `nfpm` and `dpkg-deb` paths.
- Fresh Debian installation, executable and GUI asset checks, and configuration
  preservation across a simulated package upgrade.
- API startup and `/api/health` response under QEMU's AVX-less Nehalem CPU model.
- Exact agreement between the package filename, Debian version, release tag, and
  the SHA-256 checksum of the tested `.deb`.
- Publication of the same tested `.deb`, followed by APT index, signature, and
  checksum verification.
- Upgrade and service verification on the affected Intel Core i5-750 server.

Do not republish a used package version in place. Use a new version and keep the
package hash in the release assets and publication metadata.
