# agentrepl — Plan

**Authority: construction order and history.** This document owns the order agentrepl is built in and the record of what has been built. It is **append-only**: phases are added at the bottom and marked done as they land; completed phases are never rewritten or deleted, so the plan doubles as the construction history. To extend the project later, update `docs/product.md` and `docs/design.md` in place (they stay authoritative for the current state), then **append** a new phase here — never edit a finished phase except to flip its status marker.

**One phase = one package = one accumulating context.** Each phase is a single coherent unit — almost always one `internal/` package (plus, for the last phase, the composition root) — built in one accumulating context against product and design. A phase reads only the design Decision(s) it realizes and the *interfaces* (not the internals) of the packages it depends on: the small public surface listed in those packages' design Decisions. That is what keeps every phase the size of a small standalone tool no matter how large the project grows. Where a single package realizes several intertwined Decisions and will not fit one context (here: `internal/repl`), it is split across phases that each leave the build green; the partial-Decision split is stated explicitly in the affected phases.

**Done bar.** A phase is **done** when every Verification item (the `R-XXXX-XXXX` ids) in the design Decisions it realizes — or the slice of those ids assigned to it below — is covered by a clearly-named test and the suite is green. "The suite is green" is defined in design's *Conventions*: `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing. Each Decision's Verification list in `docs/design.md` is the authority for what "covered" means.

## Status

Not started. The workspace holds product, design, and this plan; no code, no `go.mod`, no module yet.

## Phases

### Phase 1 — Module bootstrap & package skeleton · ✅ done

*Realizes design Decision 1 (package layout & seams). Depends on nothing.*

Stand up the buildable, testable skeleton from Decision 1's layout. The module `github.com/ikigenba/agentrepl` exists (`go.mod`, Go 1.26) with a `replace github.com/ikigenba/agentkit => ` directive pointing at the local agentkit checkout (the dependency is not published to the module cache; this is the build-level mechanism for resolving it). The directory tree exists with package declarations and the seam type definitions that have no behavior yet: `cmd/agentrepl/main.go` (package `main`, a composition-root stub that compiles and can be wired later), and empty-but-declared `internal/{repl,config,render,catalog,tools,session}` packages. The shared seam types from Decision 1 are declared where they belong (`IO` struct; the `Getenv`/`Now` func-type seams; the `Renderer` interface signature in `render`). No business logic. A single trivial test (e.g. a package-compiles/skeleton sanity test) exists so the suite is genuinely exercised and green.

**Done when:** `go build ./...`, `go vet ./...`, `go test ./...` exit 0 and `gofmt -l .` is empty. Decision 1 mints no requirement ids (it is a pure structural decision); this phase's bar is the green suite over the skeleton, and it is the substrate every later phase's ids are proven on.

### Phase 2 — Provider & model catalog · ✅ done

*Realizes design Decision 2 (provider & model catalog). Depends on Phase 1.*

Build `internal/catalog` as data plus an injected constructor (`ProviderFunc`), no interface: the `Provider` struct, `Default()` returning the four curated providers in stable order with their contractual env keys and agentkit model-constant lists, `Lookup`, `HasModel`, `Build(getenv)`, and the `ErrUnknownProvider`/`ErrUnknownModel`/`ErrMissingKey` sentinels. Includes the mechanical anti-drift test that every curated model is accepted by its constructed provider's `Pricing`. Tests use a fake `ProviderFunc`/fake `getenv`; no live keys.

**Done when:** R-OVEC-4AWS, R-OWM8-I2NH, R-OXU4-VUE6, R-OZ21-9M4V, R-P09X-NDVK, R-P1HU-15M9 are each covered by a clearly-named test and the suite is green.

### Phase 3 — Session log & session-id · ✅ done

*Realizes design Decision 8 (session log & session-id). Depends on Phase 1.*

Build `internal/session`: `ID(t)` minting the sub-second timestamp id and `Open(dir, now)` doing `MkdirAll` then opening `<dir>/<id>.jsonl` unbuffered (`O_CREATE|O_WRONLY|O_TRUNC`, `0o644`) and returning the file + id. Stdlib only; tests use a temp dir and a fixed `time.Time`. (R-8IUX/R-8K2T, which assert the *content* a completed run writes, are proven later at the repl level where a real `Conversation` writes to this file — see Phases 7–8; this phase proves id determinism, path targeting, dir creation, and unbuffered open.)

**Done when:** R-8GF4-LRYU and R-8HN0-ZJPJ are covered by clearly-named tests and the suite is green.

### Phase 4 — Built-in tools (bash / read / write / edit) · ✅ done

*Realizes design Decision 10 (built-in tools). Depends on Phase 1.*

Build `internal/tools`: `All()` returning the four `agentkit.NewTool[In]` tools with their typed input structs, exercised directly against a temp working directory. Behaviors per Decision 10: `bash` returns combined stdout+stderr (preserving output and noting `[exit status N]` on non-zero, nil Go error); `read` non-terminal `IsError` on missing file; `write` create/truncate; `edit` replace-all with count, non-terminal `IsError` when `Old` absent; all paths relative to cwd.

**Done when:** R-NHBW-446N, R-NIJS-HVXC, R-NKZL-9FEQ, R-NM7H-N75F, R-NNFE-0YW4, R-NONA-EQMT are covered by clearly-named tests and the suite is green.

### Phase 5 — Config-key namespace & coercion · ✅ done

*Realizes design Decision 3 (config-key namespace & coercion). Depends on Phase 2 (catalog interface) and Phase 1.*

Build `internal/config`: the `Target` struct, the explicit typed key table (`map[string]field`) covering every key in Decision 3's table across pointer/int/duration/bool/enum/string kinds, the `default` unset sentinel, the loose provider/model coupling (provider via catalog `Lookup`+`Build`; model pre-validated via `HasModel` when a provider is set), and `Set`/`Get`/`Dump`/`Keys`/`ParsePair` with `ErrUnknownKey`/`ErrBadValue`. Reads the *interface* of `catalog` only. Table-driven tests over the full key list; flag/runtime parity proven by `ParsePair`+`Set` vs direct `Set`.

**Done when:** R-LYK7-Y7ZS, R-LZS4-BZQH, R-M100-PRH6, R-M27X-3J7V, R-M3FT-HAYK, R-M4NP-V2P9, R-M5VM-8UFY are covered by clearly-named tests and the suite is green.

### Phase 6 — Renderer (decorated & raw) + usage/cost formatting · ✅ done

*Realizes design Decision 5 (turn execution, the Renderer, and color) — presentation half — and Decision 7 (usage & cost reporting) — format half. Depends on Phase 1.*

Build `internal/render`: the `Renderer` interface and its two implementations, `NewDecorated(out, color)` and `NewRaw(out)`, with output pinned by golden files under `testdata/` (with a `-update` flag). Decorated gives each kind a distinct treatment, streams `TextDelta`/`ReasoningDelta` incrementally, emits ANSI only when `color` is true, and renders the per-turn usage/cost line and cumulative summary in the exact bucket layout of Decision 7. Raw emits one undecorated JSON line per `Prompt`/`MessageDone`/`ToolUse`/`ToolResult` plus usage/summary as JSON, skips deltas, never emits ANSI. Depends only on agentkit's `Event`/`Usage`/`Cost` value types; tests feed synthesized events and usage/cost values directly.

This phase owns the **format/presentation** ids of D5 and D7. The driver-side ids — D5's R-LSKZ (driver calls `Error` not `Usage`) and D7's sourcing/trigger ids (R-OPZQ, R-OSFJ, R-OUVC) — are realized in Phases 7–8, where the turn driver supplies the numbers and the triggers fire.

**Done when:** R-LL9K-SKDQ, R-LMHH-6C4F, R-LNPD-K3V4, R-LOX9-XVLT, R-LRD2-PF37 (Decision 5, render side) and R-ONJY-6PJG, R-OORU-KHA5, R-OR7N-C0RJ, R-OW38-V3QB (Decision 7, render side) are covered by clearly-named tests (goldens where the format is pinned) and the suite is green.

### Phase 7a — REPL launch surface, loop & command dispatch (no live turn) · ✅ done

*Realizes design Decision 4 (CLI flags), Decision 9 (slash-command dispatch & command set, the non-turn half), and Decision 11 (resilience, the command/config in-loop half). Depends on Phases 2, 3, 4, 5, 6.*

Build the orchestrator's launch surface and read-dispatch loop in `internal/repl`, consuming only the public interfaces of catalog, config, render, session, and tools. This phase carries **no live turn execution** — a plain-message line routes to a turn-handler seam that Phase 7b fills in; here it is an inert stub. End state: `ParseArgs`/`Options` (the launch surface over a local `flag.FlagSet`); `Run(ctx, Deps, opts)` that opens the session log (deferred close for resource hygiene), builds the `Conversation`+`Target`, applies `opts.Config` in order (a bad pair is fatal — returns exit code 1, loop never starts), registers the four tools, then drives the loop; the `command` table + `state`; line classification (`/`-command dispatched, empty line ignored, other line handed to the turn seam); the command set that needs no live turn — `/set`, `/get`, `/dump`, `/clear`, `/render`, `/providers`, `/help`, `/exit`, `/quit` — with runtime errors non-fatal and rendered to stdout; and the turn pre-check hint when provider+model aren't both set (renders the hint, does **not** call `Send`). `/exit`/`/quit`/EOF terminate the loop and `Run` returns exit 0; the full graceful summary-then-`conv.Close()` cleanup sequence (R-LW8O) is completed in Phase 7b. `/usage`'s real summary trigger also lands in 7b. The `ctx` is taken as a parameter (a plain `context.Background()` in this phase's tests and in main until Phase 8 supplies the cancellable one). The loop survives every command/config/selection error.

**Done when:** these are covered by clearly-named tests (repl-level, captured stdout) and the suite is green:
- Decision 4 — R-EU69-75V4, R-EWM1-YPCI, R-EXTY-CH37, R-EZ1U-Q8TW.
- Decision 9 (non-turn) — R-BI0J-TIHX, R-BKGC-L1ZB, R-BLO8-YTQ0, R-BMW5-CLGP, R-BO41-QD7E, R-BPBY-44Y3, R-BQJU-HWOS.
- Decision 11 (command/config) — R-H8PP-ZFI3.

### Phase 7b — REPL turn driver, usage triggers & graceful exit · ✅ done

*Realizes design Decision 5 (turn driver, the driver half), Decision 7 (usage sourcing & triggers, the in-loop half), Decision 6 (graceful-exit cleanup, the non-signal half), Decision 9 (the turn-classification id), Decision 8 (session-log content), and Decision 11 (turn-error resilience, the in-loop half). Depends on Phase 7a.*

Fill in the turn-handler seam left by Phase 7a and complete the graceful-exit sequence, driving a **real `*agentkit.Conversation` with a fake `agentkit.Provider`** (per Decision 1's seam) in tests with scripted stdin. End state: a non-`/`, non-empty line drives a turn through the driver (pre-check → `Prompt(text)` → `conv.Send(ctx, text)` → range `stream.Events()` calling `Event` for each → after draining, on non-interrupt `stream.Err()` call `Error` and continue, else `Usage(stream.Usage(), stream.Cost(), conv.TotalCost())`); `/usage` renders the cumulative `Summary` from `conv.TotalUsage()`/`conv.TotalCost()`; and graceful `/exit`/`/quit`/EOF flow through deferred cleanup (`Summary` to the operator → `conv.Close()` writes agentkit's cumulative summary record → close log → exit 0). A completed run's records land in the session file, identical whether rendering was decorated or raw. In-loop turn errors render to stdout; the loop survives every turn error. The `ctx` is still a parameter (the cancellable one arrives in Phase 8).

**Done when:** these are covered by clearly-named tests (repl-level, real `Conversation` + fake `Provider`, captured stdout **and** JSONL log buffer) and the suite is green:
- Decision 9 (turn) — R-BJ8G-7A8M.
- Decision 5 (driver) — R-LSKZ-36TW.
- Decision 7 (in-loop) — R-OPZQ-Y90U (the *errored-turn* case), R-OSFJ-PSI8, R-OUVC-HBZM (the `/exit`/`/quit`/EOF cases).
- Decision 6 (graceful) — R-LW8O-8I1Z.
- Decision 8 (content) — R-8IUX-DBG8, R-8K2T-R36X.
- Decision 11 (turn) — R-H7HT-LNRE.

### Phase 8 — Composition root, interrupt & log integrity · ⬜ not started

*Realizes design Decision 6 (REPL lifecycle: interrupt & log integrity, the signal half), Decision 11 (resilience, the signal/startup-fatal half), Decision 7 (success-only accounting under interrupt), and completes Decision 1's composition root. Depends on Phase 7b.*

Flesh out `cmd/agentrepl/main.go` into the real composition root and wire SIGINT through the same graceful path. End state: `main` resolves `IO`/`Getenv`/`Now`/`LogDir` (`~/.agentkit` via `os.UserHomeDir`), computes `color = IsTTY && NO_COLOR==""`, sets up `ctx, stop := signal.NotifyContext(ctx, os.Interrupt)`, calls `repl.Run(ctx, …)`, and `os.Exit(code)`. The signal handler never calls `os.Exit` — it cancels the context; the driver observes `ctx.Err()`, renders a brief interrupt notice (not an error), renders the cumulative summary, and exits through the same deferred cleanup as `/exit`. Exit-code taxonomy realized end-to-end: 0 clean, 130 on SIGINT, 1 on startup failure. The no-torn-line guarantee is proven: SIGINT mid-stream yields a log that parses as valid JSONL end-to-end ending in a well-formed `turn_end` then `summary`. Startup-fatal messages go to stderr; in-loop errors to stdout. No `recover` anywhere.

**Done when:** these are covered by clearly-named tests and the suite is green:
- Decision 6 — R-LXGK-M9SO, R-LYOH-01JD, R-M149-RL0R.
- Decision 11 — R-H9XM-D78S, R-HB5I-QYZH, R-HCDF-4QQ6.
- Decision 7 — R-OPZQ-Y90U (the *interrupted-turn* case, completing the id begun in Phase 7).

### Phase 9 — Makefile (build / fmt / test / install / clean) · ⬜ not started

*Realizes no design Decision — build tooling. Depends on Phase 8.*

Add a root `Makefile` that wraps design's canonical commands as convenience targets; it introduces no new build semantics — "the suite is green" stays exactly as design's *Conventions* define it (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0 and `gofmt -l .` empty). Modeled on the sibling `../ralph` Makefile, adapted for the `cmd/agentrepl` entry point. Targets:

- **`build`** (the default target) — compiles the binary to `bin/agentrepl` from `./cmd/agentrepl`.
- **`fmt`** — `go fmt ./...`.
- **`test`** — `go test ./...`.
- **`install`** — depends on `build`; installs the binary to `$(PREFIX)/bin` with `PREFIX ?= $(HOME)/.local`, so the default install path is `~/.local/bin/agentrepl` (`install -d $(PREFIX)/bin` then `install -m 0755`).
- **`clean`** — removes `bin/` and runs `go clean`.

Use `BINARY := agentrepl`, `BIN_DIR := bin`, `PREFIX ?= $(HOME)/.local`, and a `.PHONY` line for the non-file targets. This phase carries no `R-XXXX-XXXX` ids (there is no design Decision behind it); it is proven the way Phase 1 is — by the tooling working and the suite staying green.

**Done when:** the `Makefile` exists at the repo root; `make` (default) builds `bin/agentrepl`, `make fmt`/`make test` run gofmt/the tests, `make install` places the binary at `~/.local/bin/agentrepl` (via the default `PREFIX`), and `make clean` removes the build artifacts — and the suite (per design's *Conventions*) is green.
