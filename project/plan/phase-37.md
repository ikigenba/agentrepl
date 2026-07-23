# Phase 37 — `-V`/`--version` version reporting, the linker seam & the Makefile stamp

*Realizes design Decision 15 (versioning & version reporting — the flag,
seam, stamp, and help listing). Depends on Phase 21 (composition root wiring in
`cmd/agentrepl/main.go`) and Phase 30 (the `--help` catalog in `internal/repl`).*

Observable end state:

- `cmd/agentrepl/main.go` declares the linker seam `var version = "dev"` and a
  `hasVersionFlag(args []string) bool` that is true for `-V`, `-version`, and
  `--version`. `run` prescans argv (parallel to the existing `hasHelpFlag`) and,
  when a version flag is present, writes `version` + newline to `out`, returns
  `0`, and never enters `repl.Run` — ahead of any `-c`/config handling.
- `internal/repl`'s `WriteHelp` gains a `-V, --version   show the version and
  exit` line in its `flags:` block, and the usage line lists the flag.
- The `Makefile` `build` target stamps the seam from git state:
  `go build -ldflags "-X main.version=$(shell git describe --tags --always --dirty)" -o $(BIN_DIR)/$(BINARY) ./cmd/agentrepl` (so `install`, which depends on
  `build`, inherits the stamp).

**Done when:** each id below is covered by a clearly-named, genuinely-asserting
test and the suite is green (`go build ./...`, `go vet ./...`, `go test ./...`
exit 0; `gofmt -l .` prints nothing).

- R-S45L-UT0N — an unstamped build (the `go test` binary) invoking `run` with
  `-V` writes exactly `dev` + newline to stdout, returns exit code `0`, and the
  REPL loop never starts.
- R-S5DI-8KRC — building the real `cmd/agentrepl` package with
  `go build -ldflags "-X main.version=<sentinel>"` and executing that binary with
  `-V` writes exactly `<sentinel>` + newline to stdout (exercises the real linker
  seam, not the in-process default).
- R-S6LE-MCI1 — invoking `run` with `--version` (and with `-version`) behaves
  identically to `-V`: exactly `dev` + newline on the unstamped binary, exit
  code `0`.
- R-S7TB-048Q — the `--help` catalog's `flags:` block names both `-V` and
  `--version`.
