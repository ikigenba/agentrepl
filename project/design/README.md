# agentrepl — Design

**Authority: shape and its proof.** This document owns *how* agentrepl is built — its seams, public interfaces, naming, struct/type definitions, data model — and *how each behavior is proven*. The product (`project/product/README.md`) owns the *why*, the users, scope, and the user-facing promises; design states the **exact, checkable form** of those promises and never re-declares the why. Design *uses* the product's contractual constants (module path, config separator, credential variable names, session-log location) **by value** but does not own them. This is the single, current statement of the architecture: when a decision changes, this doc is rewritten in place to stay true — decisions are never stacked. The construction history lives in the plan.

## Requirement ids

- Each Decision ends with a **Verification** list: the concrete behaviors that decision requires.
- Every Verification item carries a **minted id** of the form `R-XXXX-XXXX` — a stable, unique handle for that one behavior.
- The ids live inline in these Verification lists and **nowhere else** — there is no separate requirements document.
- Design's responsibility for ids ends at **minting** them into this doc. How coverage is measured, what counts as a covered id, and when the work is "done" are **not** design's concern — downstream phases own that.

## Conventions

Shared facts every Decision leans on.

- **Language / version:** Go 1.26.
- **Module / repository path:** `github.com/ikigenba/agentrepl` (contractual; from product).
- **Dependency:** built on `github.com/ikigenba/agentkit`; agentrepl drives `*agentkit.Conversation` directly. It builds against an agentkit version that exports **native-per-model reasoning** (agentkit design D6 + D16); a version predating that surface (the universal `ReasoningEffort` enum) cannot satisfy this design. agentrepl consumes — and reimplements none of — this agentkit surface:
  - **Native value carrier:** `agentkit.ReasoningValue` with constructors `Level(string)`, `Budget(int)`, `DisableReasoning()`; the zero value = unset → model default. It is **opaque** (unexported fields — *no read-back*), which is why config keeps a separate display string (Decision 3). `agentkit.GenSettings.Reasoning` is now a `ReasoningValue` (replacing the removed `ReasoningEffort` enum and its `EffortDefault…` constants).
  - **Introspection (credential-blind):** `agentkit.ReasoningKind` (`ReasoningEnum` / `ReasoningRange` / `ReasoningToggle`), `agentkit.ReasoningSpec{Term, Kind, Levels, Min, Max, Sentinels, Default, CanDisable}`, `agentkit.Sentinel{Value, Meaning}`, and the interface `agentkit.ReasoningInspector{ ReasoningSpec(model) (ReasoningSpec, bool); SupportedReasoning() map[string]ReasoningSpec }`, implemented by per-sub-package package-level values `anthropic.Reasoning`, `google.Reasoning`, `openai.Reasoning`, `zai.Reasoning` (no `Provider` handle, no credentials, no network).
  - **Warning surface:** `agentkit.Warning{Setting, Code, Detail}`, `agentkit.WarningCode` (`WarnReasoningUnsupported`, `WarnReasoningCannotDisable`, `WarnToolChoiceForced`, `WarnToolSchemaLossy`), read after a turn via `(*agentkit.Stream).Warnings() []Warning`.
- **The exact agentkit version pin is a plan/build fact, not a design fact.** Design fixes the *named API* it consumes (above); the *exact version string* that carries it lives in the plan — each dependency-bump phase names its target version in that phase's Done-when — and is realized in `go.mod`. Design and the READMEs carry only the shape (that a pin exists and where it is realized); the number is a fact and lives with the phase that sets it. Advancing the pin is therefore a spec change — append a new plan phase naming the new target, never a bare `go.mod` edit.
- **Binary entry point:** `cmd/agentrepl/main.go`, package `main` — the composition root only (parse flags, wire dependencies, run). All logic lives in `internal/` packages.
- **Build / typecheck command:** `go build ./... && go vet ./...`.
- **Test command:** `go test ./...`.
- **"The suite is green" means:** `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing (no unformatted files).
- **Idiomatic Go is a requirement, mechanically gated.** The code must read as accurate, community-standard Go. The mechanical gate is `gofmt`-clean + `go vet ./...` clean. Beyond the gate: interfaces are defined at the consumer and only where runtime polymorphism is actually needed ("accept interfaces, return structs"); seams that need only substitution in tests are injected funcs, not interfaces; no speculative abstraction; errors are wrapped with `%w` and classified with sentinel/`errors.Is`/`errors.As`; no panics on expected conditions.
- **Time / IO sources:** a single injected clock `Now func() time.Time` and an injected `Getenv func(string) string`; no package-level calls to `time.Now`/`os.Getenv` outside the composition root.
- **Exit-code taxonomy:** `0` = clean exit (operator quit, or EOF on input); `1` = startup failure (bad flags, or a fatal precondition before the REPL loop begins). Once the interactive loop is running, per-turn and per-command errors are surfaced in-band and never exit the process.

## Layout

The design is split for addressability so the build loop never loads the whole
architecture to find the one Decision a phase realizes:

- **`project/design/INDEX.md`** — the manifest: each Decision mapped to its file and
  the Verification ids it owns, plus a sorted `R-id → Decision/file` reverse map.
  Id lookup is a grep against this file (or against the Decision files directly).
- **`project/design/DNN.md`** — one file per Decision (zero-padded; referenced in
  prose and the plan as `D<N>`). Each is self-contained: the Decision, its public
  interfaces/types, the rejected alternatives, and its **Verification** list of
  `R-XXXX-XXXX` ids. The build loop reads only the Decision(s) its phase realizes.
- **`project/design/README.md`** (this file) — the invariant spine above: Authority, the
  *Requirement ids* convention, and *Conventions*. Static cross-cutting facts; it
  does not carry per-Decision detail.

Design is **rewritten in place**, not append-only (the construction history lives
in the plan): when a Decision changes, its `DNN.md` is rewritten to stay true and
`INDEX.md` is regenerated. A new Decision adds a `DNN.md` and an INDEX entry.
