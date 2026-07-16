# Phase 9 — Makefile (build / fmt / test / install / clean)

*Realizes no design Decision — build tooling. Depends on Phase 8.*

Add a root `Makefile` that wraps design's canonical commands as convenience targets; it introduces no new build semantics — "the suite is green" stays exactly as design's *Conventions* define it (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0 and `gofmt -l .` empty). Modeled on the sibling `../ralph` Makefile, adapted for the `cmd/agentrepl` entry point. Targets:

- **`build`** (the default target) — compiles the binary to `bin/agentrepl` from `./cmd/agentrepl`.
- **`fmt`** — `go fmt ./...`.
- **`test`** — `go test ./...`.
- **`install`** — depends on `build`; installs the binary to `$(PREFIX)/bin` with `PREFIX ?= $(HOME)/.local`, so the default install path is `~/.local/bin/agentrepl` (`install -d $(PREFIX)/bin` then `install -m 0755`).
- **`clean`** — removes `bin/` and runs `go clean`.

Use `BINARY := agentrepl`, `BIN_DIR := bin`, `PREFIX ?= $(HOME)/.local`, and a `.PHONY` line for the non-file targets. This phase carries no `R-XXXX-XXXX` ids (there is no design Decision behind it); it is proven the way Phase 1 is — by the tooling working and the suite staying green.

**Done when:** the `Makefile` exists at the repo root; `make` (default) builds `bin/agentrepl`, `make fmt`/`make test` run gofmt/the tests, `make install` places the binary at `~/.local/bin/agentrepl` (via the default `PREFIX`), and `make clean` removes the build artifacts — and the suite (per design's *Conventions*) is green.
