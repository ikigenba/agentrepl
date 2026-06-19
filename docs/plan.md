# agentrepl — Plan

**Authority: construction order and history.** This document owns the order agentrepl is built in and the record of what has been built. It is **append-only**: phases are added at the bottom and marked done as they land; completed phases are never rewritten or deleted, so the plan doubles as the construction history. To extend the project later, update `docs/product.md` and `docs/design.md` in place (they stay authoritative for the current state), then **append** a new phase here — never edit a finished phase except to flip its status marker.

**One phase = one package = one accumulating context.** Each phase is a single coherent unit — almost always one `internal/` package (plus, for the last phase, the composition root) — built in one accumulating context against product and design. A phase reads only the design Decision(s) it realizes and the *interfaces* (not the internals) of the packages it depends on: the small public surface listed in those packages' design Decisions. That is what keeps every phase the size of a small standalone tool no matter how large the project grows. Where a single package realizes several intertwined Decisions and will not fit one context (here: `internal/repl`), it is split across phases that each leave the build green; the partial-Decision split is stated explicitly in the affected phases.

**Done bar.** A phase is **done** when every Verification item (the `R-XXXX-XXXX` ids) in the design Decisions it realizes — or the slice of those ids assigned to it below — is covered by a clearly-named test and the suite is green. "The suite is green" is defined in design's *Conventions*: `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing. Each Decision's Verification list in `docs/design.md` is the authority for what "covered" means.

## Status

Phases 1–10 done (the full REPL plus the configurable Z.ai base URL), built against agentkit `v0.1.0`'s universal `ReasoningEffort` enum. Phases 11–14 done for the **native-first reasoning** consumer change (design D2/D3/D5/D12 rewritten in place, plus the breaking agentkit pin bump). Phase 15 is appended for the `--help` reasoning-row rework (design Decision 12 amended so each row leads with the literal `gen.reasoning=` key): ✅ done. Phases 16–17 are appended for the **flatten-to-passthrough** change (design D3/D12 amended in place: the `gen.`/`retry.`/`zai.` prefixes dropped and the single `gen.reasoning` key replaced by four native keys — `effort`/`thinking_budget`/`thinking_level`/`thinking`): ⬜ not started.

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

### Phase 8 — Composition root, interrupt & log integrity · ✅ done

*Realizes design Decision 6 (REPL lifecycle: interrupt & log integrity, the signal half), Decision 11 (resilience, the signal/startup-fatal half), Decision 7 (success-only accounting under interrupt), and completes Decision 1's composition root. Depends on Phase 7b.*

Flesh out `cmd/agentrepl/main.go` into the real composition root and wire SIGINT through the same graceful path. End state: `main` resolves `IO`/`Getenv`/`Now`/`LogDir` (`~/.agentkit` via `os.UserHomeDir`), computes `color = IsTTY && NO_COLOR==""`, sets up `ctx, stop := signal.NotifyContext(ctx, os.Interrupt)`, calls `repl.Run(ctx, …)`, and `os.Exit(code)`. The signal handler never calls `os.Exit` — it cancels the context; the driver observes `ctx.Err()`, renders a brief interrupt notice (not an error), renders the cumulative summary, and exits through the same deferred cleanup as `/exit`. Exit-code taxonomy realized end-to-end: 0 clean, 130 on SIGINT, 1 on startup failure. The no-torn-line guarantee is proven: SIGINT mid-stream yields a log that parses as valid JSONL end-to-end ending in a well-formed `turn_end` then `summary`. Startup-fatal messages go to stderr; in-loop errors to stdout. No `recover` anywhere.

**Done when:** these are covered by clearly-named tests and the suite is green:
- Decision 6 — R-LXGK-M9SO, R-LYOH-01JD, R-M149-RL0R.
- Decision 11 — R-H9XM-D78S, R-HB5I-QYZH, R-HCDF-4QQ6.
- Decision 7 — R-OPZQ-Y90U (the *interrupted-turn* case, completing the id begun in Phase 7).

### Phase 9 — Makefile (build / fmt / test / install / clean) · ✅ done

*Realizes no design Decision — build tooling. Depends on Phase 8.*

Add a root `Makefile` that wraps design's canonical commands as convenience targets; it introduces no new build semantics — "the suite is green" stays exactly as design's *Conventions* define it (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0 and `gofmt -l .` empty). Modeled on the sibling `../ralph` Makefile, adapted for the `cmd/agentrepl` entry point. Targets:

- **`build`** (the default target) — compiles the binary to `bin/agentrepl` from `./cmd/agentrepl`.
- **`fmt`** — `go fmt ./...`.
- **`test`** — `go test ./...`.
- **`install`** — depends on `build`; installs the binary to `$(PREFIX)/bin` with `PREFIX ?= $(HOME)/.local`, so the default install path is `~/.local/bin/agentrepl` (`install -d $(PREFIX)/bin` then `install -m 0755`).
- **`clean`** — removes `bin/` and runs `go clean`.

Use `BINARY := agentrepl`, `BIN_DIR := bin`, `PREFIX ?= $(HOME)/.local`, and a `.PHONY` line for the non-file targets. This phase carries no `R-XXXX-XXXX` ids (there is no design Decision behind it); it is proven the way Phase 1 is — by the tooling working and the suite staying green.

**Done when:** the `Makefile` exists at the repo root; `make` (default) builds `bin/agentrepl`, `make fmt`/`make test` run gofmt/the tests, `make install` places the binary at `~/.local/bin/agentrepl` (via the default `PREFIX`), and `make clean` removes the build artifacts — and the suite (per design's *Conventions*) is green.

### Phase 10 — Configurable Z.ai base URL (`-c zai.base_url=…`) · ✅ done

*Realizes design Decision 2 (the `Options` construction-override seam) and Decision 3 (the `zai.base_url` key and its order-independent provider-build coupling). Depends on Phase 2 (catalog) and Phase 5 (config).*

Thread a per-construction override from config down to provider construction so an operator can point Z.ai at an alternate endpoint (e.g. the GLM coding-plan root `https://api.z.ai/api/coding/paas/v4`) via `-c zai.base_url=…` at launch or `/set zai.base_url …` at runtime — no bespoke flag, no per-endpoint catalog entry. In `internal/catalog`: add the `Options` struct (`BaseURL string`), change `ProviderFunc` to `func(apiKey string, opts Options) agentkit.Provider` and `Build` to `Build(getenv, opts)`; the `zai` entry maps a non-empty `Options.BaseURL` to `zai.WithBaseURL(...)` and the other three entries ignore `Options` (so Z.ai's baked-in `https://api.z.ai/api/paas/v4` is the default). In `internal/config`: add `Target.ZaiBaseURL`, build the provider with `Options{BaseURL: t.ZaiBaseURL}` when the selected provider is `zai`, and add the `zai.base_url` key whose setter stores the value and rebuilds the provider when `zai` is the active provider — order-independent with `provider` (set before or after `provider=zai` reaches the same state), with `zai.base_url=default` clearing the override and rebuilding against the baked-in root. Because the `ProviderFunc`/`Build` signatures change, the existing catalog and config tests (the `newTarget` fake catalog entry, the `Build`/`New` call sites) are updated to the new signatures; the full-key-list ids **R-LYK7-Y7ZS** (coercion) and **R-M4NP-V2P9** (`Dump` all keys) gain a `zai.base_url` case as part of staying green.

**Done when:** R-S94E-8K1O (catalog `Options`/`Build` threads `BaseURL`, zai applies `WithBaseURL`, others ignore) and R-SCS3-DV9R (config `zai.base_url` stores-and-rebuilds, order-independent with `provider`, `default` clears) are covered by clearly-named tests, the existing full-key-list ids still pass with the new key, and the suite is green.

### Phase 11 — agentkit native-reasoning pin & native `gen.reasoning` coercion · ✅ done

*Realizes design Decision 3 (the `gen.reasoning` native-value carve-out and `Target.ReasoningRaw`) and the Conventions' agentkit-version obligation. Depends on Phase 5 (config) and on the **external** prerequisite that the agentkit checkout the `replace` directive targets has built its native-reasoning surface — agentkit plan Phases 21–22: `ReasoningValue`/`Level`/`Budget`/`DisableReasoning`, the removed `ReasoningEffort`, and `Warning.Code`.*

This is the **atomic dependency move**, the agentrepl mirror of agentkit's Phase 22. The instant agentrepl resolves against the native-reasoning agentkit, `agentkit.ReasoningEffort`/`EffortDefault`/`Effort*` vanish and `agentkit.GenSettings.Reasoning` becomes a `ReasoningValue`, so `internal/config/config.go`'s `gen.reasoning` handling (`parseReasoning`/`formatReasoning` and the `EffortDefault` reset) stops compiling — the pin advance and the config rework therefore land together and the build is green only at phase end (no transitional shim). `go.mod`'s `require` advances from `agentkit v0.1.0` to the agentkit minor that exports D6+D16 (the exact version string pinned here when agentkit tags; the local `replace` already resolves to that tree, so the hard prerequisite is that Phases 21–22 are *built* there). `internal/config` replaces the old ordinal coercion with shape-directed, **model-blind** `ReasoningValue` construction: raw (trim+lowercase) ∈ {`off`,`disable`,`disabled`} → `DisableReasoning()`; a base-10 integer including a leading `-` (`-1`,`0`,`8000`) → `Budget(n)`; any other string (`high`,`xhigh`,`none`) → `Level(verbatim)`. The built value is assigned to `Conv.Gen.Reasoning` and the raw string stored on the new `Target.ReasoningRaw`. `gen.reasoning` is the carve-out that does **not** return `ErrBadValue` on a non-native value (only a structurally empty value errors), leaving the warn-and-default to agentkit at request-build time; `Get`/`Dump` render from `ReasoningRaw` (unset → `default`), and `gen.reasoning=default` resets the value to the zero `ReasoningValue` and clears `ReasoningRaw`. The previously-green full-key-list ids that touch reasoning — R-LYK7-Y7ZS (coercion), R-M100-PRH6 (generic invalid-value, whose design now **excludes** `gen.reasoning`), R-M27X-3J7V (`default` reset), R-M4NP-V2P9 (`Dump`) — are updated to the native shape as part of staying green, exactly as Phase 10 updated them for `zai.base_url`.

**Done when:** R-FZCE-VXJL (shape-directed, model-blind coercion assigning `Conv.Gen.Reasoning` and storing `ReasoningRaw`), R-G0KB-9PAA (the carve-out: a non-native value is accepted without `ErrBadValue` and stored, only a structurally empty value errors), and R-G304-18RO (`Get`/`Dump` render from `ReasoningRaw`, `default` resets the value and clears the raw) are covered by clearly-named tests; the previously-green full-key-list ids still pass under the native shape (R-M100-PRH6 now excluding `gen.reasoning`); and the suite is green.

### Phase 12 — Catalog reasoning introspector field · ✅ done

*Realizes design Decision 2 (the `Reasoning agentkit.ReasoningInspector` field and its anti-drift). Depends on Phase 2 (catalog) and Phase 11 (the agentkit pin that exports `ReasoningInspector` and the per-sub-package `Reasoning` values).*

Additive — the existing `Default()`/`Lookup`/`HasModel`/`Build` surface keeps its shape. `catalog.Provider` gains `Reasoning agentkit.ReasoningInspector`, and `Default()` sets each entry's `Reasoning` to its sub-package's credential-blind introspector value (`anthropic.Reasoning`, `google.Reasoning`, `openai.Reasoning`, `zai.Reasoning`) alongside `New` — it is not the constructed provider and needs no key. This is the single source the `--help` catalog (Phase 14) reads each model's native reasoning vocabulary from, so agentrepl embeds zero provider reasoning knowledge; config coercion (Phase 11) deliberately does **not** consult it. A reasoning anti-drift test mirrors the existing pricing anti-drift (R-OWM8-I2NH): every curated `Models` id must resolve to a `ReasoningSpec` via `p.Reasoning.ReasoningSpec(id)`, so `--help` never renders a curated model with no descriptor.

**Done when:** R-FQT4-7JCQ (`Default()` sets each `Reasoning` to its sub-package introspector, non-nil and credential-blind) and R-FS10-LB3F (reasoning anti-drift: every curated id resolves to a `ReasoningSpec`) are covered by clearly-named tests and the suite is green.

### Phase 13 — Settings-warning relay (`Renderer.Warning` + turn driver) · ✅ done

*Realizes design Decision 5 (the `Renderer.Warning` method and the driver's verbatim warning relay) and amends Decision 11 (a warning is not an error and never ends the loop). Depends on Phase 6 (render), Phase 7b (turn driver), and Phase 11 (the agentkit pin that adds `Warning.Code`).*

The `Renderer` interface gains `Warning(agentkit.Warning)`, implemented by both impls — `decorated` in a distinct warn treatment (e.g. `⚠ <Setting>: <Detail>`, separate from the error style, golden-pinned) and `raw` as one JSON line carrying the `agentkit.Warning`'s `Setting`/`Code`/`Detail`. The turn driver, after draining `stream.Events()` and before `Usage`/`Error` (but after the interrupt check), relays `for _, w := range stream.Warnings() { rend.Warning(w) }` verbatim — never minting, reclassifying, or suppressing — whether the turn then succeeds or errors; a turn with no warnings calls `Warning` zero times. Adding the interface method updates both render impls (and any repl-level renderer use) together, so the build stays green.

**Done when:** R-G5FW-SS92 (decorated renders a `Warning` in its own treatment, distinct from `Error` (golden); raw emits exactly one JSON line carrying `Setting`/`Code`/`Detail`), R-G480-F0ID (the driver calls `Warning` once per entry in `stream.Warnings()`, before `Usage`/`Error`, verbatim; zero calls when there are none), and R-G6NT-6JZR (a non-native `gen.reasoning` value from the Phase 11 carve-out yields an agentkit reasoning warning the driver relays, the turn completing with the model's default — exercised through the real `Conversation` + fake `Provider` seam emitting a `Warning`) are covered by clearly-named tests (repl-level captured stdout + render goldens) and the suite is green.

### Phase 14 — Self-describing `--help` catalog · ✅ done

*Realizes design Decision 12 (the credential-blind `--help` catalog) and the Decision 4 `-h`/`-help` interception row. Depends on Phase 12 (the catalog `Reasoning` field it reads) and Phase 7a (`ParseArgs` and the launch surface it hooks).*

A new `repl.WriteHelp(out io.Writer, name string, cat []catalog.Provider)` renders a static, credential-blind catalog: a one-line usage, the launch flags, the providers list (each with its `EnvKey` *string* as documentation, not a presence check), and models grouped by provider in `Default()` order, each model's reasoning clause produced by **one render routine keyed on `spec.Kind`** — `ReasoningEnum` → `<Term>: <Levels joined>  (default <level>)`; `ReasoningRange` → `<Term>: <Min>–<Max>` then sentinels and native default; `ReasoningToggle` → `<Term>: on/off  (default …)` — and a model whose `ReasoningSpec(id)` returns `false` rendered as `(no reasoning control)` rather than dropped. It reads only `cat` (no env, no constructed providers) and reuses the Decision 3 native default formatter so `--help` and `/get` describe a default identically. `ParseArgs` overrides the `FlagSet` usage so `-h`/`-help` writes this catalog to `out` and returns the sentinel `flag.ErrHelp`; the composition root treats that as a clean exit `0` and never starts the loop, building no `config.Target` and reading no env.

**Done when:** R-FT8W-Z2U4 (`-h`/`-help` writes the catalog and exits `0` without starting the loop, with no env set and no provider constructed), R-FUGT-CUKT (every provider and its curated models, in `Default()` order), R-FVOP-QMBI (each reasoning clause rendered from its `ReasoningSpec` by `Kind`, golden across enum/range/toggle), R-FWWM-4E27 (a `false`-spec model renders `(no reasoning control)` and is not dropped), and R-FY4I-I5SW (no env read, no provider constructed — asserted via a catalog whose `New`/`Getenv` would record or panic if called) are covered by clearly-named tests and the suite is green.

### Phase 15 — `--help` reasoning rows lead with the `gen.reasoning=` key · ✅ done

*Realizes design Decision 12 (the reworked reasoning-row format: the literal config key as the row label, native term demoted to a parenthetical). Depends on Phase 14 (`WriteHelp` and its golden).*

Rework the `spec.Kind` render routine inside `repl.WriteHelp` so each model's reasoning line no longer leads with the native term but with the **literal config key `gen.reasoning=`** — byte-identical across every model and provider — followed by its accepted values in **traditional CLI syntax**: braces `{a|b|c}` for an enumerated choice (`ReasoningEnum` and `ReasoningToggle`) and angle brackets `<min–max>` for a free numeric value (`ReasoningRange`). The native term (`effort`, `thinking budget`, `thinking level`, `thinking`), any sentinels, and the native default move into a trailing parenthetical, e.g. `gen.reasoning=<0–24576>  (thinking budget; 0=off, -1=dynamic; default dynamic)` and `gen.reasoning={low|medium|high|xhigh|max}  (effort; default high)`. The routine stays a single `Kind`-keyed function with no per-provider branches; the `gen.reasoning=` prefix is a constant string; a model whose `ReasoningSpec(id)` returns `false` still renders `(no reasoning control)` and is not dropped; the credential-blind contract and `Default()`-order grouping are unchanged. The change is observable only in the rendered bytes, so the `WriteHelp` golden under `testdata/` is regenerated to the new row shape and the credential-blindness/order assertions ride along unchanged. This closes the usability gap where a reader saw `thinking budget:` and typed `-c thinking=12000`, hitting `unknown config key`.

**Done when:** R-6DEO-9TXQ (every model row leads with the literal `gen.reasoning=` key, byte-identical across models/providers, values in traditional CLI syntax — `{a|b|c}` for enum/toggle, `<…>` for range — copy-pasteable as `-c gen.reasoning=<value>`, native term only in the parenthetical) and the reworked R-FVOP-QMBI (the accepted-values group rendered from its `ReasoningSpec` by `Kind` with the native `Term`, sentinels, and default in the trailing parenthetical, golden across enum/range/toggle) are covered by clearly-named tests (the regenerated `WriteHelp` golden) and the suite is green.

### Phase 16 — Flatten config keys & native reasoning keys · ✅ done

*Realizes design Decision 3 (amended: flat unprefixed keys; the single `gen.reasoning` key replaced by four native keys `effort`/`thinking_budget`/`thinking_level`/`thinking` with key-directed coercion; new `Target.ReasoningKey`). Depends on Phase 5 (the original config package) and Phase 11 (native `ReasoningValue` coercion).*

Flatten `internal/config` so every key is a flat, unprefixed name: drop the `gen.`/`retry.`/`zai.` prefixes (`temperature`, `top_p`, `max_tokens`, `max_attempts`, `base_delay`, `max_delay`, `max_elapsed`, `ignore_retry_after`, `base_url`; `provider`/`model`/`system`/`tool_loop_limit` already flat). The prefix was always pure decoration — `Set` is a single `map[string]field` lookup — so this is a rename of the table keys, not a routing change. Replace the single `gen.reasoning` entry with **four native reasoning keys** aliasing the one `Conv.Gen.Reasoning` field: `effort` and `thinking_level` → `agentkit.Level(raw)`; `thinking_budget` → `agentkit.Budget(int)`; `thinking` → `off`→`agentkit.DisableReasoning()` and `on`→the unset zero `ReasoningValue`. Each records the raw string on `Target.ReasoningRaw` **and** the key on the new `Target.ReasoningKey` field. Coercion is **key-directed** (the key name picks the constructor) and model-blind — a non-native value is accepted without error and left for agentkit to warn+default at turn time; only structurally unusable input errors (empty value on any reasoning key, a non-integer `thinking_budget`, a non-`on`/`off` `thinking`). `Get`/`Dump` render the value under `ReasoningKey`; the other three reasoning keys and an unset value render `default`; `default` set on any reasoning key clears the shared value and key. The observable end state: `-c effort=high`, `-c thinking_budget=8000`, `-c thinking_level=high`, `-c thinking=on`, `-c temperature=0.7`, `-c base_url=…` all work; the old `gen.reasoning`/`gen.*`/`retry.*`/`zai.*` keys are gone.

**Done when:** the Decision 3 ids are re-covered against the flat keys — R-LYK7-Y7ZS, R-M100-PRH6, R-M27X-3J7V, R-M3FT-HAYK, R-M4NP-V2P9, R-M5VM-8UFY, R-LZS4-BZQH, R-SCS3-DV9R (now `base_url`), R-FZCE-VXJL (key-directed coercion across all four keys), R-G0KB-9PAA (carve-out: non-native accepted, only structurally unusable errors), and R-G304-18RO (`Get`/`Dump` under `ReasoningKey`; `default` clears value+key) — are each covered by a clearly-named test and the suite is green.

### Phase 17 — `--help` rows lead with each model's native reasoning key · ✅ done

*Realizes design Decision 12 (amended: each row leads with the model's own native key derived from `spec.Term` via `termToKey`, replacing the constant `gen.reasoning=`). Depends on Phase 16 (the four registered keys) and Phase 15 (`WriteHelp` and its golden).*

Rework the render routine inside `repl.WriteHelp` so each model's reasoning line leads with **that model's own native config key** instead of the constant `gen.reasoning=`. A small `termToKey` normalization maps the model's `spec.Term` to one of the four registered keys — lowercase, drop a trailing ` (+ toggle)`, replace the space in `thinking budget`/`thinking level` with `_` — yielding exactly `effort` / `thinking_budget` / `thinking_level` / `thinking`. The accepted-values group is still keyed on `spec.Kind` (braces for enum/toggle, angle brackets for range) and the native term phrase, any sentinels, and the native default stay in the trailing parenthetical, e.g. `thinking_budget=<0–24576>  (thinking budget; 0=off, -1=dynamic; default dynamic)` and `effort={low|medium|high|xhigh|max}  (effort; default high)`. The routine stays a single function with no per-provider branches; a model whose `ReasoningSpec(id)` returns `false` still renders `(no reasoning control)` and is not dropped; the credential-blind contract and `Default()`-order grouping are unchanged. The change is observable only in the rendered bytes, so the `WriteHelp` golden under `testdata/` is regenerated to the new per-model-key row shape.

**Done when:** R-6DEO-9TXQ (reworded — every model row leads with that model's native reasoning key derived from `spec.Term`, values in traditional CLI syntax, copy-pasteable as `-c <key>=<value>`, term/sentinels/default in the parenthetical) and the new R-6DEO-KEYS (`termToKey(spec.Term)` resolves to one of the four keys Decision 3 registers for every supported model, so the catalog never advertises an unknown key) are covered by clearly-named tests (the regenerated `WriteHelp` golden plus the cross-check) and the suite is green.
