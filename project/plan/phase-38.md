# Phase 38 — Release distribution: goreleaser, the release workflow & the `curl | sh` installer

*Realizes design Decision 15 (the release apparatus — structural slice, no
Verification ids of its own; its proof is the deterministic checks below).
Depends on Phase 37 (the `main.version` linker seam the release builds stamp).*

Observable end state (all files at the repo root, inside the module this
`project/` governs):

- `.goreleaser.yaml` (goreleaser v2): `project_name: agentrepl`; one build
  `main: ./cmd/agentrepl`, `binary: agentrepl`, `CGO_ENABLED=0`,
  `goos: [linux, darwin]`, `goarch: [amd64, arm64]`,
  `ldflags: -s -w -X main.version={{ .Tag }}`; `tar.gz` archives with the
  versionless name template `{{ .Binary }}_{{ .Os }}_{{ .Arch }}`;
  `checksums.txt`; `release.github` owner `ikigenba`, name `agentrepl`;
  `changelog.use: github`.
- `.github/workflows/release.yml`: triggers `on: push: tags: ['v*']`, grants
  `contents: write`, checks out with `fetch-depth: 0`, sets up Go 1.26, and runs
  `goreleaser/goreleaser-action@v6` with `args: release --clean` and
  `GITHUB_TOKEN`.
- `install.sh`: POSIX `#!/bin/sh` installer for `REPO=ikigenba/agentrepl`,
  `BINARY=agentrepl`; honors `BINDIR`/`PREFIX` and `AGENTREPL_VERSION` (default
  `latest`); resolves `linux|darwin` × `amd64|arm64`, fetches the versionless
  `releases/{latest/download,download/<ver>}/agentrepl_<os>_<arch>.tar.gz`,
  untars, `install -m 0755` into `BINDIR`, warns if `BINDIR` is off `PATH`.
- `README.md`'s install section carries the one-line
  `curl -fsSL https://raw.githubusercontent.com/ikigenba/agentrepl/main/install.sh | sh`
  path (latest default, `AGENTREPL_VERSION=v0.1.0` to pin, `BINDIR`/`PREFIX` to
  relocate) alongside the existing `make build`/`make install` from-source path.
  No `v0.1.0` literal appears anywhere except as this illustrative pin example.

**Done when** (structural phase — deterministic exit conditions, no
`R-` ids; every check targets an implementation file, never a `project/` doc, so
each is falsifiable and non-self-referential):

- `sh -n install.sh` exits `0` (valid POSIX syntax); `install.sh` begins with
  `#!/bin/sh` and contains both `ikigenba/agentrepl` and `AGENTREPL_VERSION`.
- `.goreleaser.yaml` exists and contains `project_name: agentrepl`,
  `./cmd/agentrepl`, `main.version={{ .Tag }}`, and each of `linux`, `darwin`,
  `amd64`, `arm64`.
- `.github/workflows/release.yml` exists, triggers on a `v*` tag push, and
  invokes `goreleaser`.
- `README.md` contains the `curl -fsSL … install.sh | sh` one-liner.
- The Go suite stays green: `go build ./...`, `go vet ./...`, `go test ./...`
  exit `0`, and `gofmt -l .` prints nothing.
- If `goreleaser` is on `PATH`, `goreleaser check` exits `0`.
