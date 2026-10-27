# Debian AMD64 CPU baseline and release checklist

The `m3tal` Debian package supports the original x86-64 baseline (Go `GOAMD64=v1`),
including the Intel Core i5-750. The CLI and API are built through `scripts/build-go.sh`
with `CGO_ENABLED=0`; this prevents C compiler defaults from selecting instructions
above the supported baseline. The module and CI use the Go version declared in `go.mod`.

Use `./build.sh` or `make build` for native development binaries. The Debian
package build uses the explicit `debian` profile in `scripts/build-go.sh` to pin
Linux AMD64 v1 with cgo disabled. The Debian package remains one
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

## Checks by stage

CI verifies dependencies, Go tests and formatting, installs GUI assets before Go
tests, builds packages through both `nfpm` and `dpkg-deb`, and compares their
contents and metadata. It also installs and upgrades the package in Debian and
checks API startup under QEMU's AVX-less Nehalem CPU model.

The release workflow repeats Go verification, builds and tests the `nfpm`
package, and checks that its filename, Debian version, tag, and SHA-256 agree
before creating the GitHub release. After publishing, it waits for the signed
APT index and downloaded package to match the tested package hash. Upgrading and
checking the affected Intel Core i5-750 server remains a manual post-publish
step.

Do not republish a used package version in place. Use a new version and keep the
package hash in the release assets and publication metadata.
