# agentrepl

An interactive command-line REPL that drives [agentkit](https://github.com/ikigenba/agentkit)
directly, exposing every feature — provider/model selection, generation and
reasoning settings, custom tools, streaming, the full message exchange, and
token/cost reporting — for hands-on inspection. It is a testing and verification
harness for agentkit, not a production chat client. Module path:
`github.com/ikigenba/agentrepl`.

## How changes are made

Changes go through the spec under `project/`, not direct edits — settle the spec,
then let the build loop realize it. Edit code directly only on explicit operator
instruction. The spec is authority-partitioned: `project/product/README.md` owns
the *why*, `project/design/` (spine `README.md` + `INDEX.md` + `DNN.md`) owns the
shape and its proof (minting `R-XXXX-XXXX` verification ids), `project/plan/`
(rules `README.md` + `STATUS.md` + `phase-NN.md`) owns the append-only construction
order. The unattended `gather → build → verify` loop (prompts in `project/loops/`)
drives one phase per turn; see the `$ralph` skill for that workflow.

## Layout

- `cmd/agentrepl/` — composition root: `main.go`.
- `internal/config/` — config keys, `-c key=value` parsing, defaults.
- `internal/catalog/` — providers and their curated models.
- `internal/session/` — conversation state and the `~/.agentkit/<id>.jsonl` log.
- `internal/tools/` — the built-in local tools (bash, read, write, edit).
- `internal/repl/` — the loop: args, commands, help, rendering seams.
- `internal/render/` — decorated vs. raw transcript rendering.
- `project/` — the spec (product/design/plan) the build loop works from.

## Tests

- Unit: `go test ./...` (or `make test`).
- Green bar (design's *Conventions*): `go build ./...`, `go vet ./...`, and
  `go test ./...` exit 0, and `gofmt -l .` prints nothing.

## Versioning

agentrepl is an unreleased internal harness — no version tags, no version
constant. What is pinned is its agentkit dependency: `go.mod` resolves
`github.com/ikigenba/agentkit` from the published module (no local `replace`).
The exact pinned version is a plan fact, not a doc fact — it lives in the
`go.mod` `require` line, set by a `project/plan/` dependency-bump phase. Bumping
it is a spec change: settle it under `project/` and append a phase naming the
new target; the build loop runs `go get` and updates `go.mod`. Don't hardcode
the version number here or in any README.
