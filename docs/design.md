# agentrepl — Design

**Authority: shape and its proof.** This document owns *how* agentrepl is built — its seams, public interfaces, naming, struct/type definitions, data model — and *how each behavior is proven*. The product (`docs/product.md`) owns the *why*, the users, scope, and the user-facing promises; design states the **exact, checkable form** of those promises and never re-declares the why. Design *uses* the product's contractual constants (module path, config separator, credential variable names, session-log location) **by value** but does not own them. This is the single, current statement of the architecture: when a decision changes, this doc is rewritten in place to stay true — decisions are never stacked. The construction history lives in the plan.

## Requirement ids

- Each Decision ends with a **Verification** list: the concrete behaviors that decision requires.
- Every Verification item carries a **minted id** of the form `R-XXXX-XXXX` — a stable, unique handle for that one behavior.
- The ids live inline in these Verification lists and **nowhere else** — there is no separate requirements document.
- Design's responsibility for ids ends at **minting** them into this doc. How coverage is measured, what counts as a covered id, and when the work is "done" are **not** design's concern — downstream phases own that.

## Conventions

Shared facts every Decision leans on.

- **Language / version:** Go 1.26.
- **Module / repository path:** `github.com/ikigenba/agentrepl` (contractual; from product).
- **Dependency:** built on `github.com/ikigenba/agentkit`; agentrepl drives `*agentkit.Conversation` directly. It builds against the agentkit version that exports **native-per-model reasoning** (agentkit design D6 + D16); the older `v0.1.0` (universal `ReasoningEffort` enum) cannot satisfy this design. agentrepl consumes — and reimplements none of — this agentkit surface:
  - **Native value carrier:** `agentkit.ReasoningValue` with constructors `Level(string)`, `Budget(int)`, `DisableReasoning()`; the zero value = unset → model default. It is **opaque** (unexported fields — *no read-back*), which is why config keeps a separate display string (Decision 3). `agentkit.GenSettings.Reasoning` is now a `ReasoningValue` (replacing the removed `ReasoningEffort` enum and its `EffortDefault…` constants).
  - **Introspection (credential-blind):** `agentkit.ReasoningKind` (`ReasoningEnum` / `ReasoningRange` / `ReasoningToggle`), `agentkit.ReasoningSpec{Term, Kind, Levels, Min, Max, Sentinels, Default, CanDisable}`, `agentkit.Sentinel{Value, Meaning}`, and the interface `agentkit.ReasoningInspector{ ReasoningSpec(model) (ReasoningSpec, bool); SupportedReasoning() map[string]ReasoningSpec }`, implemented by per-sub-package package-level values `anthropic.Reasoning`, `google.Reasoning`, `openai.Reasoning`, `zai.Reasoning` (no `Provider` handle, no credentials, no network).
  - **Warning surface:** `agentkit.Warning{Setting, Code, Detail}`, `agentkit.WarningCode` (`WarnReasoningUnsupported`, `WarnReasoningCannotDisable`, `WarnToolChoiceForced`, `WarnToolSchemaLossy`), read after a turn via `(*agentkit.Stream).Warnings() []Warning`.
- **agentkit version pin is a plan/build obligation.** This is a **breaking** dependency move; `go.mod`'s `require` advances from `v0.1.0` to the agentkit minor that ships the surface above. Design fixes the *named API* (settled in agentkit's design); the *exact version string* is pinned in plan/build when agentkit tags the release, since it is not yet tagged.
- **Binary entry point:** `cmd/agentrepl/main.go`, package `main` — the composition root only (parse flags, wire dependencies, run). All logic lives in `internal/` packages.
- **Build / typecheck command:** `go build ./... && go vet ./...`.
- **Test command:** `go test ./...`.
- **"The suite is green" means:** `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing (no unformatted files).
- **Idiomatic Go is a requirement, mechanically gated.** The code must read as accurate, community-standard Go. The mechanical gate is `gofmt`-clean + `go vet ./...` clean. Beyond the gate: interfaces are defined at the consumer and only where runtime polymorphism is actually needed ("accept interfaces, return structs"); seams that need only substitution in tests are injected funcs, not interfaces; no speculative abstraction; errors are wrapped with `%w` and classified with sentinel/`errors.Is`/`errors.As`; no panics on expected conditions.
- **Time / IO sources:** a single injected clock `Now func() time.Time` and an injected `Getenv func(string) string`; no package-level calls to `time.Now`/`os.Getenv` outside the composition root.
- **Exit-code taxonomy:** `0` = clean exit (operator quit, or EOF on input); `1` = startup failure (bad flags, or a fatal precondition before the REPL loop begins). Once the interactive loop is running, per-turn and per-command errors are surfaced in-band and never exit the process.

## Decision 1 — Package layout & seams (the testable skeleton)

**Decision.** agentrepl is a single binary built from `internal/` packages, with a thin composition root. The Provider boundary that agentkit already exposes is the primary test seam; agentrepl drives a real `*agentkit.Conversation` whose `Provider` is faked in tests, rather than wrapping the conversation behind a second mock layer.

Layout:

```
github.com/ikigenba/agentrepl
  cmd/agentrepl/main.go    package main — composition root ONLY (parse flags, wire deps, run)
  internal/repl/           orchestrator: REPL loop, slash-command dispatch, turn drive
  internal/config/         dotted-key namespace: parse key=value, apply to Conversation, dump, validate
  internal/render/         Renderer interface + decorated + raw implementations
  internal/catalog/        curated providers/models as data; env-key map; Provider construction
  internal/tools/          the four built-in tools (bash/read/write/edit)
  internal/session/        session-id generation + log-file creation
```

Seams — each a substitution point for tests:

| Seam | Shape | Substitution in tests |
|------|-------|-----------------------|
| **IO** | `type IO struct { In io.Reader; Out, Err io.Writer; IsTTY bool }` | script stdin; capture stdout/stderr; force color on/off |
| **Env** | `Getenv func(string) string` | inject fake credentials |
| **Clock** | `Now func() time.Time` | deterministic session-ids |
| **Provider construction** | `type ProviderFunc func(apiKey string) agentkit.Provider` (a field on each catalog entry) | inject a `ProviderFunc` returning a fake `agentkit.Provider` |
| **Engine** | the real `*agentkit.Conversation` with a fake `agentkit.Provider` | fake provider yields deterministic events — mirrors agentkit's own tests |
| **Renderer** | `interface` with `decorated` and `raw` impls, selected at runtime | the one place a runtime interface is warranted (polymorphic dispatch) |
| **Log** | an `io.Writer` assigned to `Conversation.Log` | tests pass a `bytes.Buffer` and assert JSONL |

Idiomatic-Go consequences baked into this skeleton:

- The catalog is **data plus an injected `ProviderFunc`**, not an interface — interfaces are defined at the consumer and only where runtime polymorphism is needed.
- `Renderer` *is* an interface, because decorated-vs-raw is genuine runtime polymorphism.
- Env and Clock are injected funcs (stdlib-style seams), not wrapper interfaces.

**Testing strategy (overall).** Per-package unit tests: `config` coercion as table-driven tests; `render` via golden files under `testdata/` (with a `-update` flag); `catalog` env-key/model-validation tests; `tools` exercised directly against a temp working directory. Above those, `repl`-level tests build a **real `Conversation` with a fake `Provider`** and script stdin, then assert on captured stdout **and** the JSONL log buffer in one pass. The gate is `go test ./...` plus `gofmt`/`go vet` clean.

**Rejected.**
- **Flat single `main` package** — weak test isolation, no naming seams, renderer/config logic hard to exercise alone; contradicts the product's hands-on-verifiability emphasis.
- **Root `main.go` (no `cmd/`)** — idiomatic for a trivial single binary, but `cmd/agentrepl/main.go` is the layout a Go reviewer expects for an application with `internal/` packages; keeps the root unambiguous.
- **Wrap `Conversation` behind an `Engine` interface** — `Send` returns a concrete `*agentkit.Stream` over an `iter.Seq`; wrapping it fights the grain and duplicates the Provider seam agentkit already provides. Faking at the Provider boundary is lower-friction and exercises real agentkit wiring.
- **Make `catalog` an interface** — speculative abstraction; only the Provider constructor needs swapping, which an injected func covers.

**Verification.** This is a pure structural/seam decision with no behavior of its own; it carries no requirement ids. Its proof is the behavioral ids of the decisions it enables (Decisions 2+).

## Decision 2 — Provider & model catalog

**Decision.** `internal/catalog` holds the four curated providers as **data plus an injected constructor**, with env-key resolution and model validation expressed as plain funcs — no interface.

```go
package catalog

// ProviderFunc constructs an agentkit.Provider from an API key plus optional
// per-construction overrides.
type ProviderFunc func(apiKey string, opts Options) agentkit.Provider

// Options carries per-construction overrides threaded from config (Decision 3).
// The zero value leaves every provider at its baked-in defaults.
type Options struct {
    BaseURL string // override the provider's API root; "" → provider default
}

// Provider is one curated agentkit provider agentrepl can drive.
type Provider struct {
    Name      string                      // "anthropic" | "google" | "openai" | "zai"
    EnvKey    string                      // "ANTHROPIC_API_KEY" | "GEMINI_API_KEY" | "OPENAI_API_KEY" | "ZAI_API_KEY"
    Models    []string                    // curated model ids, referencing agentkit's exported constants
    New       ProviderFunc                // e.g. func(k string, o Options) agentkit.Provider { return anthropic.New(k) }
    Reasoning agentkit.ReasoningInspector // the sub-package's credential-blind introspector: anthropic.Reasoning, …
}

// Default returns the built-in catalog of the four providers, in a stable order.
func Default() []Provider

// Lookup finds a provider by name.
func Lookup(cat []Provider, name string) (Provider, bool)

// HasModel reports whether model is in p's curated set.
func (p Provider) HasModel(model string) bool

// Build resolves p.EnvKey via getenv and constructs the provider with opts, or
// returns ErrMissingKey (wrapped, naming the env var) when the key is empty.
func (p Provider) Build(getenv func(string) string, opts Options) (agentkit.Provider, error)

var (
    ErrUnknownProvider = errors.New("unknown provider")
    ErrUnknownModel    = errors.New("unknown model for provider")
    ErrMissingKey      = errors.New("missing API key")
)
```

- `Default()`'s `Models` lists reference agentkit's exported model constants (`anthropic.ModelOpus48`, `google.ModelFlash25`, …), so the curated set is **enumerable** — for `/help`, model listings, and clear "choose from: …" errors — rather than hidden behind agentkit's unexported pricing registries.
- **`Reasoning` carries the sub-package's credential-blind introspector** (`anthropic.Reasoning`, `google.Reasoning`, `openai.Reasoning`, `zai.Reasoning`), set in `Default()` alongside `New`. It is **not** the constructed `Provider` and needs no key — it is the single source the `--help` catalog (Decision 12) reads each model's native reasoning term/values from, so agentrepl embeds **zero** provider reasoning knowledge. (Config coercion in Decision 3 is deliberately model-blind and does **not** consult it; runtime display, if added, would.) The curated `Models` list and the inspector are kept honest together by the anti-drift test below: every curated id must resolve to a `ReasoningSpec` just as it must resolve to a `Pricing`.
- The model is **pre-validated** against the mirrored `Models` list, giving an immediate, clear error before any turn. The list is kept honest by a mechanical **anti-drift test**: every curated model must be accepted by its constructed provider's `Pricing` — drift fails the suite rather than passing silently.
- **Z.ai is an ordinary catalog entry** — present and constructible when `ZAI_API_KEY` is set. Its known-broken-ness is a turn-time failure surfaced cleanly by the error-handling decision, not a catalog special case.
- **Provider-construction overrides ride on `Options`, not new catalog entries.** `Build` threads an `Options` into `New`; today only the `zai` entry consumes it, mapping a non-empty `Options.BaseURL` to agentkit's `zai.WithBaseURL(...)` option and otherwise leaving Z.ai's baked-in default root (`https://api.z.ai/api/paas/v4`). The other three entries ignore `Options`. This is the seam the `zai.base_url` config key (Decision 3) drives — e.g. to point Z.ai at its coding-plan endpoint `https://api.z.ai/api/coding/paas/v4` — keeping the "new knob = new key, no bespoke flag" promise and avoiding a per-endpoint catalog entry.
- `ErrMissingKey` is ordinary non-fatal data; the REPL renders it as a clear message and stays alive (resilience lives in the dispatch/error decisions). `EnvKey` is read via the injected `Getenv` — agentrepl never reads a credential file.

**Rejected.**
- **A `Catalog` interface** — speculative abstraction; only the constructor needs swapping, which `ProviderFunc` covers.
- **Model pass-through** (accept any string, let `conv.Send` reject unknowns) — the error is cryptic, arrives only at send time, and models can't be enumerated; weaker against the product's "reported clearly" promise.
- **Validate by constructing the provider and calling `Pricing`** — needs the API key present just to check a name, and can't enumerate the curated set.
- **A separate catalog entry per Z.ai endpoint** — multiplies the curated set for what is one construction option; an `Options.BaseURL` override is the smaller seam.

**Verification.**
- R-OVEC-4AWS — `Default()` returns exactly the four providers `anthropic`, `google`, `openai`, `zai`, each carrying its contractual env key (`ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, `ZAI_API_KEY`).
- R-S94E-8K1O — `Build` threads its `Options` into the provider constructor: a non-empty `Options.BaseURL` reaches the `zai` entry as `zai.WithBaseURL(...)` so the constructed Z.ai provider targets the override root, while an empty `Options.BaseURL` leaves the baked-in default and the other three entries ignore `Options`.
- R-OWM8-I2NH — anti-drift: for every provider in `Default()`, every id in its `Models` list is accepted by the constructed provider's `Pricing(model)` (returns ok).
- R-FQT4-7JCQ — `Default()` sets each provider's `Reasoning` to its sub-package introspector value (`anthropic.Reasoning`, `google.Reasoning`, `openai.Reasoning`, `zai.Reasoning`), non-nil and credential-blind (callable with no env key set).
- R-FS10-LB3F — reasoning anti-drift: for every provider in `Default()`, every id in its `Models` list resolves to a `ReasoningSpec` via `p.Reasoning.ReasoningSpec(id)` (returns ok), so `--help` never renders a curated model with no reasoning descriptor.
- R-OXU4-VUE6 — each provider's `Models` list is non-empty.
- R-OZ21-9M4V — `Build` returns an error wrapping `ErrMissingKey` and naming the env var when `getenv` yields empty for the provider's key, and constructs nothing.
- R-P09X-NDVK — `Build` returns a constructed `agentkit.Provider` (non-nil, no error) when the key is present.
- R-P1HU-15M9 — `Lookup` reports not-found for an unknown provider name; `HasModel` returns false for a model outside the curated list and true for one in it.

## Decision 3 — Config-key namespace & coercion

**Decision.** `internal/config` is the single vocabulary that both the launch flag and the runtime slash-command funnel through. It is an **explicit typed key table** — no reflection — so adding an agentkit knob is one table entry, never a new flag.

```go
package config

// Target is what config reads and mutates.
type Target struct {
    Conv         *agentkit.Conversation
    Catalog      []catalog.Provider
    Getenv       func(string) string
    ZaiBaseURL   string // pending zai base-URL override; applied when zai is (re)built
    ReasoningRaw string // last raw gen.reasoning value, for display only (ReasoningValue is opaque); "" = unset
}

// Set applies one key=value to t, or returns a clear, wrapped error.
func Set(t *Target, key, raw string) error

// Get returns the current display value for key.
func Get(t *Target, key string) (string, bool)

// Dump returns every key with its current value, sorted, as "key=value" lines.
func Dump(t *Target) []string

// Keys returns the known key names, sorted (for /help).
func Keys() []string

// ParsePair splits a launch-flag "key=value" on the first '='.
func ParsePair(s string) (key, value string, err error)

var (
    ErrUnknownKey = errors.New("unknown config key")
    ErrBadValue   = errors.New("invalid value for config key")
)
```

Internally a `map[string]field`, where `field` carries `set(t, raw) error` and `get(t) string`. The key namespace mirrors `agentkit.Conversation` + `GenSettings` + `RetryPolicy`:

| Key | Target field | Type |
|-----|--------------|------|
| `provider` | `Conv.Provider` (via catalog) | provider name |
| `model` | `Conv.Model` | model id |
| `system` | `Conv.System` | string |
| `gen.temperature` | `Gen.Temperature` | `*float64` |
| `gen.top_p` | `Gen.TopP` | `*float64` |
| `gen.max_tokens` | `Gen.MaxTokens` | int |
| `gen.reasoning` | `Gen.Reasoning` (`agentkit.ReasoningValue`) + `Target.ReasoningRaw` | native value, shape-directed — see "Reasoning is the carve-out" below |
| `retry.max_attempts` | `Retry.MaxAttempts` | int |
| `retry.base_delay` | `Retry.BaseDelay` | duration (e.g. `500ms`) |
| `retry.max_delay` | `Retry.MaxDelay` | duration |
| `retry.max_elapsed` | `Retry.MaxElapsed` | duration |
| `retry.ignore_retry_after` | `Retry.IgnoreRetryAfter` | bool |
| `tool_loop_limit` | `Conv.MaxToolIterations` | int |
| `zai.base_url` | `Target.ZaiBaseURL` → `catalog.Options.BaseURL` for the `zai` build | URL string |

- **Unset sentinel.** Setting the literal value `default` resets *any* key to its zero/unset state (nil pointer, zero `ReasoningValue`, zero int/duration); `Dump`/`Get` render an unset key as `default`. One uniform rule, no per-key syntax.
- **provider / model coupling** (loose, to avoid ordering deadlocks):
  - `provider=<name>` → catalog `Lookup` + `Build(getenv)`; sets `Conv.Provider`; surfaces `ErrUnknownProvider` / `ErrMissingKey` through `Set`; does not touch model.
  - `model=<id>` → if a provider is set, pre-validate with `HasModel` (clear `ErrUnknownModel` with "choose from: …"); else accept the string. The (provider, model) pair is ultimately validated by agentkit at `Send` and surfaced cleanly — a transient post-switch mismatch is a clear send-time error, never a crash.
- **`zai.base_url` is a provider-construction override, applied through the catalog `Options` seam (Decision 2), and order-independent with `provider`.** A base URL is baked into the provider handle at construction, not a `Conversation` field, so the value is stored on `Target.ZaiBaseURL` and the `zai` provider is (re)built to apply it. `provider=zai` builds with `Options{BaseURL: t.ZaiBaseURL}`; setting `zai.base_url` while `zai` is already the active provider rebuilds it with the new root; setting it before any provider is selected just stores it for the eventual `zai` build — either order reaches the same state. `zai.base_url=default` clears the override (back to Z.ai's baked-in root) and rebuilds if zai is active. For a non-zai active provider the key is stored but not applied (a no-op against the live conversation until zai is selected).
- **Reasoning is the carve-out: shape-directed, never model-validated in agentrepl, never a hard error.** `gen.reasoning` is the **one** key whose bad input does *not* return `ErrBadValue` (product success-criteria carve-out). Its `set` builds an `agentkit.ReasoningValue` from the raw string by **shape alone** — model-blind, so it works at launch before any model is selected and after a mid-conversation switch:
  - raw (trim+lowercase) ∈ {`off`, `disable`, `disabled`} → `agentkit.DisableReasoning()`;
  - raw parses as a base-10 integer, **including a leading `-`** (so sentinels like `-1`=dynamic and `0`=off pass) → `agentkit.Budget(n)`;
  - otherwise → `agentkit.Level(raw)` **verbatim** — `"high"`, `"xhigh"`, `"minimal"`, and notably `"none"` (a real native effort level on gpt-5.x, **not** a disable token), passed through untouched.

  The built value is assigned to `Conv.Gen.Reasoning` and the raw string is stored on `Target.ReasoningRaw` for display. agentrepl does **not** consult the active model's `ReasoningSpec` here and does **not** judge whether the value is native — that judgment, and the warn-and-default, are agentkit's at request-build time (D6), which is the only place the *currently-selected* model is authoritative (a value valid for model A may be invalid for model B after a switch). A non-native value is therefore accepted by `Set` without error, stored, and later **warned + defaulted** by agentkit, with agentrepl relaying the `Warning` (Decision 5). The sole `set`-time failure is a structurally unusable input (an empty value), which returns `ErrBadValue` like any key.
  - **Display.** `Get`/`Dump` render `gen.reasoning` from `Target.ReasoningRaw` (the native string the operator gave), and an unset value (`ReasoningRaw == ""`) as `default` — the same uniform unset rendering every other key uses. **Runtime `/get`/`/dump` show only the current value (and `default` for unset); they deliberately do *not* reprint the active model's accepted-values catalog** — that discovery view is `--help`-only (Decision 12), keeping the runtime surface thin and matching the product, which mandates the catalog for `--help` alone. The native default *value* shown in `--help` comes from `spec.Default`, not from this field.
  - **`default` resets it** to the unset `ReasoningValue` zero (model default) and clears `ReasoningRaw`, via the uniform unset rule above — no reasoning-specific syntax.
  - Reasoning is **excluded from the generic invalid-value error coverage**: the generic error-path id asserts that path for every *other* key; reasoning's "no hard error on non-native input" is asserted by its own id below.
- Both control surfaces share this one entrypoint: the `-c` flag does `ParsePair` then `Set`; `/set <key> <value>` calls `Set` directly; `/dump` calls `Dump`. Adding a key automatically reaches both surfaces.

**Rejected.**
- **Reflection / struct-tag mapping** — clever and opaque, produces poor errors, fails the idiom bar.
- **A bespoke flag per setting** — contradicts the product's "new knob = new key, no new flag."
- **Strict pair validation at set time** — deadlocks: can't set the new provider while the old model is invalid, nor the new model while the old provider is current.

**Verification.**
- R-LYK7-Y7ZS — every known key coerces its value to the correct typed `Target` field (table-driven across the full key list, including pointer, int, duration, bool, enum, and string kinds).
- R-LZS4-BZQH — an unknown key returns a wrapped `ErrUnknownKey` naming the key and mutates nothing.
- R-M100-PRH6 — an unparseable value returns a wrapped `ErrBadValue` naming the key and reason, and mutates nothing (every key **except** `gen.reasoning`, whose non-erroring carve-out is asserted separately below).
- R-M27X-3J7V — the `default` value resets a pointer/enum key to unset, and `Dump`/`Get` then render it as `default`.
- R-M3FT-HAYK — `provider=` constructs via the catalog and surfaces `ErrUnknownProvider`/`ErrMissingKey` through `Set`; `model=` pre-validates against the current provider with `ErrUnknownModel`.
- R-M4NP-V2P9 — `Dump` returns all keys sorted as `key=value` lines reflecting current state.
- R-M5VM-8UFY — flag/runtime parity: `ParsePair`+`Set` and a direct `Set` reach identical state for the same key and value.
- R-SCS3-DV9R — setting `zai.base_url` stores the override on `Target` and, when `zai` is (or becomes) the active provider, the constructed Z.ai provider is built with that base URL via the catalog `Options` seam — reached identically whether `zai.base_url` is set before or after `provider=zai`; `zai.base_url=default` clears it and rebuilds against the baked-in root.
- R-FZCE-VXJL — `gen.reasoning` builds the `ReasoningValue` shape-directed and model-blind: `off`/`disable`/`disabled` → `DisableReasoning()`; a base-10 integer including negatives (`-1`, `0`, `8000`) → `Budget(n)`; any other string (`high`, `xhigh`, `none`) → `Level(verbatim)`; the result is assigned to `Conv.Gen.Reasoning` and the raw string stored on `Target.ReasoningRaw`.
- R-G0KB-9PAA — the reasoning carve-out: a non-native or unrecognized-vocabulary `gen.reasoning` value (e.g. a level on a budget model, or a made-up level) is accepted by `Set` **without** `ErrBadValue` and stored for agentkit to warn+default at turn time; only a structurally empty value errors.
- R-G304-18RO — `Get`/`Dump` render `gen.reasoning` from `Target.ReasoningRaw` (the native string given), and as `default` when unset; setting `gen.reasoning=default` resets `Conv.Gen.Reasoning` to the zero `ReasoningValue` and clears `ReasoningRaw`.

## Decision 4 — CLI flags

**Decision.** The launch surface is deliberately tiny — the config passthrough carries every agentkit setting, so there are no bespoke per-setting flags. Parsing lives in a testable `internal/repl` function over a local `flag.FlagSet` (never the global `flag`), keeping `cmd/agentrepl/main.go` a thin composition root.

```go
package repl

// Options is the parsed launch surface.
type Options struct {
    Config []string // raw "key=value" args, in encounter order
    Raw    bool      // select the raw renderer (default: decorated)
}

// ParseArgs parses argv (excluding program name) into Options.
func ParseArgs(name string, args []string, out io.Writer) (Options, error)
```

| Flag | Meaning |
|------|---------|
| `-c key=value` | config passthrough; **repeatable** (a `flag.Value` slice), applied in encounter order at startup |
| `-raw` | use the raw renderer; default is the decorated transcript |
| `-h` / `-help` | the **self-describing catalog** (Decision 12), not the bare `FlagSet` usage |

Provider and model are **not** their own flags — they are `-c provider=… -c model=…`, keeping the "one config vocabulary, no bespoke flags" promise honest.

- **`-c` is collected raw, validated at apply.** `ParseArgs` only gathers the strings; `Run` builds the `config.Target` and applies each via `config.ParsePair`+`config.Set` — one validation path shared with runtime `/set`.
- **`-h`/`-help` is intercepted into the self-describing catalog.** `ParseArgs` overrides the `FlagSet`'s usage so that `-h`/`-help` writes the Decision 12 catalog to `out` and returns the sentinel `flag.ErrHelp`; the composition root treats that as a clean exit `0` and never starts the loop. The catalog is **credential-blind** — `ParseArgs` builds no `config.Target` and reads no env, so help runs with no keys set.
- **A bad `-c` at launch is fatal** (clear stderr message, exit 1): an impossible initial state should stop, not start surprising. This is the deliberate asymmetry with runtime `/set`, which is non-fatal — and matches the exit taxonomy (startup vs in-loop).
- **provider/model are optional at launch.** Starting bare is allowed; sending before they are set yields a clear "set a provider and model first" hint (the REPL pre-checks before `Send`), never a crash. Valid `-c provider/model` drops the operator straight into a usable conversation.

**Rejected.**
- **`-render decorated|raw`** instead of `-raw` — two modes, decorated default; the bool is simpler (YAGNI).
- **Bespoke `-provider`/`-model`/`-system`/`-temperature` flags** — contradict the single-vocabulary promise.
- **Requiring provider+model at launch** — needlessly rigid; runtime equivalents plus a pre-send hint cover the bare start.
- **A color flag** — color is auto by terminal detection (Decision 5), not a launch knob.

**Verification.**
- R-EU69-75V4 — `ParseArgs` collects repeated `-c` into `Options.Config` in encounter order.
- R-EWM1-YPCI — `ParseArgs` sets `Raw` from `-raw` (default false); an unknown flag returns an error and writes usage to `out`.
- R-EXTY-CH37 — at startup, `-c` pairs apply in order to the live `Target`, with a later pair overriding an earlier one for the same key.
- R-EZ1U-Q8TW — a launch `-c` with a bad key/value/format is fatal: clear stderr message, exit code 1, and the REPL loop never starts.

## Decision 5 — Turn execution, the Renderer, and color

**Decision.** The turn driver (in `repl`) owns the loop; the `Renderer` (in `render`) owns presentation — a genuine two-impl interface, the one place runtime polymorphism is warranted.

```go
package render

// Renderer presents one turn's prompt, streamed events, outcome, and spend.
type Renderer interface {
    Prompt(text string)                                          // the user's message
    Event(ev agentkit.Event)                                     // each streamed event, in order
    Usage(turn agentkit.Usage, turnCost, total agentkit.Cost)    // per-turn usage/cost line
    Summary(total agentkit.Usage, totalCost agentkit.Cost)       // cumulative usage+cost block (/usage, on exit)
    Warning(w agentkit.Warning)                                  // a setting agentkit could not honor as asked
    Error(err error)                                             // a failed turn or command
    Notice(line string)                                          // agentrepl info (e.g. /dump, hints)
}

func NewDecorated(out io.Writer, color bool) Renderer
func NewRaw(out io.Writer) Renderer
```

**Turn driver** (`repl`): pre-check provider+model → `Prompt(text)` → `stream := conv.Send(ctx, text)` → range `stream.Events()` calling `Event(ev)` for each → after draining, if `ctx.Err() != nil` render an interrupt notice and exit (Decision 6); otherwise **relay any settings warnings first** — `for _, w := range stream.Warnings() { Warning(w) }` — then, if `stream.Err() != nil` call `Error(err)`, else call `Usage(stream.Usage(), stream.Cost(), conv.TotalCost())`. Warnings are rendered whether the turn then succeeds or errors, because they describe a setting that was not honored (most often reasoning: a non-native value carried in via the Decision 3 carve-out, which agentkit warned-and-defaulted), independent of turn outcome. The `ctx` passed to `Send` is the SIGINT-cancellable context from Decision 6; the loop survives any non-interrupt turn error.

**Warning relay is render-only.** agentrepl never mints, classifies, or suppresses a warning — it forwards each `agentkit.Warning` verbatim to the renderer. A `Warning` is **not** an `Error` (the turn was issued and, for the reasoning case, succeeds with the model's default); it gets its own kind so its treatment and placement (before the usage line) are distinct.

**decorated** (default): a distinct visual treatment per kind — `Prompt` ("you ›"), `TextDelta` streamed inline as the reply, `ReasoningDelta` dim/streamed, `ToolUse` labeled with name + arguments, `ToolResult` (error treatment when `IsError`), `Warning` in a distinct warn style (e.g. `⚠ <Setting>: <Detail>`, separate from the error style), and a usage/cost line; `MessageDone` flushes a separator between messages in a tool loop. Exact glyphs/labels/ANSI codes are pinned by golden files under `testdata/`.

**raw**: emits **one undecorated JSON line per committed entry** — `Prompt`, `MessageDone`, `ToolUse`, `ToolResult` — plus the usage line as JSON and **one JSON line per `Warning`** (the `agentkit.Warning` struct marshaled verbatim, carrying `Setting`/`Code`/`Detail`); **skips `TextDelta`/`ReasoningDelta`** (streaming fragments, not entries — and `MessageDone.Message` already carries the fully assembled text/reasoning blocks). Never emits ANSI. Marshals with `encoding/json`, yielding block shapes consistent with agentkit's own log (agentkit does no custom marshaling).

**color**: the composition root computes `color = IO.IsTTY && Getenv("NO_COLOR") == ""` and passes it to `NewDecorated`. Raw is always colorless.

**Rejected.**
- **Raw = tee agentkit's `Conversation.Log` (LogRecord stream) to stdout** — byte-identical to the forensic file, but it makes `rawRenderer` a no-op while output sneaks in via a `MultiWriter`, turning the two-impl seam into a fiction and coupling raw render to the log. Marshaling events keeps both renderers honest and the log independent.
- **Raw marshals every event including deltas** — a flood of tiny delta objects; "one per entry / messages verbatim" argues for entry granularity.
- **A `Renderer` per event-kind / a visitor** — overwrought; a small sealed-union `switch` inside each impl is the idiomatic Go shape.

**Verification.**
- R-LL9K-SKDQ — decorated renders each kind (prompt, reply text, reasoning, tool-call, tool-result, usage line) with a distinct treatment (golden).
- R-LMHH-6C4F — decorated streams `TextDelta`/`ReasoningDelta` incrementally: bytes are written as each delta arrives, not buffered to end of turn.
- R-LNPD-K3V4 — decorated emits ANSI color when `color` is true and none when it is false (golden for both).
- R-LOX9-XVLT — raw emits exactly one undecorated JSON line per `Prompt`/`MessageDone`/`ToolUse`/`ToolResult` plus a usage line, skipping deltas; the output is valid JSONL with no ANSI.
- R-LRD2-PF37 — a `ToolResult` with `IsError` gets the error treatment in decorated and preserves `IsError` in raw.
- R-LSKZ-36TW — on a non-interrupt failed turn the driver calls `Error` (not `Usage`) and the loop continues to the next input.
- R-G480-F0ID — after draining a turn's events, the driver calls `Renderer.Warning` once per entry in `stream.Warnings()`, before `Usage`/`Error`, and verbatim (the agentkit `Warning` is forwarded unmodified, never reclassified or suppressed); a turn with no warnings calls `Warning` zero times.
- R-G5FW-SS92 — decorated renders a `Warning` in its own warn treatment, distinct from the `Error` treatment (golden); raw emits exactly one JSON line carrying the `Warning`'s `Setting`/`Code`/`Detail`.
- R-G6NT-6JZR — a `gen.reasoning` value the selected model does not natively understand (the Decision 3 carve-out) produces an agentkit reasoning warning that the driver relays to the renderer, and the turn still completes with the model's default applied.

## Decision 6 — REPL lifecycle: exit, interrupt, and log integrity

**Decision.** Exit is a single graceful path; SIGINT is wired through that same path so it can never tear the log.

- **`/exit`, `/quit`, and EOF on stdin** end the loop and fall into one sequence: render the cumulative usage/cost summary to the operator (`Renderer.Summary`, Decision 7), then clean up — stop the current stream, `conv.Close()` (writes agentkit's cumulative **summary** record), close the log file. Exit `0`.
- **Ctrl-C (SIGINT)** is wired via `signal.NotifyContext` in the composition root. The handler **never calls `os.Exit`** — it cancels the root context; the main goroutine observes cancellation, unwinds, and runs the *same* deferred cleanup. Exit `130` (conventional 128+SIGINT).

```go
// cmd/agentrepl/main.go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()
code := repl.Run(ctx, io, getenv, now, opts) // Run's deferred cleanup always executes
os.Exit(code)
```

Why this cannot corrupt the log:
1. SIGINT only cancels the context — it does not kill the goroutine mid-write, so any in-flight `Encode` completes and the last record is always a whole, valid JSON line.
2. The session log is a **direct, unbuffered `*os.File`**: agentkit emits one record per `Encode`/`Write`, so there is no agentrepl-side buffer to lose and no partial line on exit.
3. **Every exit path flows through deferred cleanup**, so `conv.Close()` always writes the summary and the file is always closed.

"Immediate" follows from cancellation propagating straight into the live `Send` (agentkit checks `ctx.Err()` between round-trips and its retry `Sleep` returns on `ctx.Done()`); an in-flight turn stops promptly. The driver detects `ctx.Err() != nil`, renders a brief interrupt notice (not an error), then renders the cumulative summary and exits — the interrupt path runs the same summary-then-cleanup sequence as `/exit`.

Ctrl-C semantic: **immediate graceful exit** (a turn in flight is aborted and the program exits cleanly), not abort-turn-keep-alive.

**Rejected.**
- **`os.Exit` from inside the signal handler** — races a half-written log line, the exact corruption to avoid.
- **A `bufio`-buffered log** — a crash or interrupt could drop or tear the buffer.
- **Swallowing SIGINT to force `/exit`-only** — fights the terminal convention.
- **Abort-turn-keep-alive** — needs a per-turn signal context that resets each turn; more moving parts, and contradicts "immediate."

**Verification.**
- R-LW8O-8I1Z — `/exit`, `/quit`, and EOF each end the loop, write the summary record, close the file, and exit `0`.
- R-LXGK-M9SO — SIGINT at an idle prompt exits cleanly with the summary record present and exit `130`.
- R-LYOH-01JD — SIGINT during a streaming turn stops it promptly; the resulting log is valid JSONL end-to-end (every line parses) and ends with a well-formed `turn_end` then `summary` — no torn line.
- R-M149-RL0R — the log file is opened unbuffered and closed on every exit path, including the interrupt path.

## Decision 7 — Usage & cost reporting

**Decision.** All spend numbers come straight from agentkit — agentrepl formats, never recomputes (the product: "surfaced from agentkit / drawn from agentkit's built-in pricing"). Because agentrepl exists to *verify* agentkit, the decorated per-turn line shows the **full token-bucket breakdown**, not just a total.

Per-turn line (`Renderer.Usage(turn, turnCost, total)`), from `stream.Usage()`, `stream.Cost()`, `conv.TotalCost()`:

```
· tokens  in=123 cache(r=10 w=5) out=456 reasoning=78 total=657
· cost     $0.001234 turn   $0.005678 session
```

Cumulative summary (`Renderer.Summary(total, totalCost)`), from `conv.TotalUsage()`, `conv.TotalCost()` — the cumulative token breakdown plus session cost.

- Buckets shown: `InputUncached`, `CacheReadInput`, `CacheWriteInput`, `Output`, `ReasoningOutput`, `Total` — exactly the `Usage` fields, verbatim.
- Cost via `Cost.USD()`, formatted to micro-dollar resolution (6 decimals); per-turn costs are small.
- **Session cumulative is `conv.TotalCost()` / `conv.TotalUsage()`** — agentkit advances these only on successfully completed turns, so an errored or interrupted turn never inflates the running total. agentrepl displays them as-is; no agentrepl-side accumulation.
- **raw** mode emits both the per-turn usage and the cumulative summary as JSON objects carrying the `Usage` buckets plus the relevant cost(s).
- The exact byte layout is pinned by the render golden files (Decision 5); this decision fixes *which numbers* appear and *where they come from*.

**Two triggers for the cumulative summary:**
1. **`/usage`** — runtime command (routed by the dispatch decision) renders `Summary` on demand.
2. **Every graceful exit** — `/exit`, `/quit`, EOF, and SIGINT render `Summary` as the **final stdout output**, then run cleanup (Decision 6). The operator-facing `Summary` (stdout) and agentkit's forensic `summary` `LogRecord` (file, via `Close`) are the same numbers to two destinations.

**Rejected.**
- **Compact total-only line** — hides the cache/reasoning splits a verification harness specifically wants to see.
- **agentrepl recomputes cost from `Usage` + a local rate table** — duplicates agentkit's pricing, invites drift, contradicts "drawn from agentkit's built-in pricing."
- **agentrepl maintains its own cumulative sum** — `conv.TotalUsage()/TotalCost()` already exist and correctly exclude failed turns; a parallel sum would risk disagreeing.

**Verification.**
- R-ONJY-6PJG — the per-turn line reports the turn's token buckets and total exactly as `stream.Usage()` reports them (no recomputation).
- R-OORU-KHA5 — the per-turn line reports turn cost from `stream.Cost()` and session cost from `conv.TotalCost()`, both in USD.
- R-OPZQ-Y90U — after a turn that errored or was interrupted, the displayed session cumulative is unchanged from before it (success-only accounting).
- R-OR7N-C0RJ — raw mode emits the per-turn usage as a JSON object carrying the `Usage` buckets and the turn/session costs.
- R-OSFJ-PSI8 — `/usage` renders the cumulative summary (`TotalUsage` buckets + `TotalCost`), sourced from agentkit.
- R-OUVC-HBZM — every graceful exit (`/exit`, `/quit`, EOF, SIGINT) renders the cumulative summary as the final output before cleanup runs.
- R-OW38-V3QB — raw mode renders the cumulative summary as a JSON object carrying the cumulative `Usage` buckets and session cost.

## Decision 8 — Session log & session-id

**Decision.** `internal/session` owns the file at the contractual `~/.agentkit/<session-id>.jsonl`. It is deliberately tiny — agentkit does the writing (its `LogRecord` protocol via `Conversation.Log`); session just mints the id and opens the file.

```go
package session

// ID returns a session id derived from t, stable for a given t.
func ID(t time.Time) string

// Open ensures dir exists and opens dir/<ID(now)>.jsonl unbuffered for writing,
// returning the file and the id.
func Open(dir string, now time.Time) (*os.File, string, error)
```

- **session-id = a sub-second timestamp** from the injected `Now`, e.g. `20060102T150405.000000` — sortable, human-readable, deterministic in tests, and collision-free for manual use without randomness or pid (which would break determinism).
- The composition root resolves `dir` (`~/.agentkit`, from `os.UserHomeDir()`) and passes it in; tests pass a temp dir. `Open` does `os.MkdirAll(dir, 0o755)`, then opens `O_CREATE|O_WRONLY|O_TRUNC`, mode `0o644`, **unbuffered** (Decision 6's no-torn-line guarantee).
- The returned file is assigned to `Conversation.Log` and is **always on, independent of render mode** — decorated and raw runs write the identical log.

**Rejected.**
- **A random/UUID id** — non-deterministic; fights golden paths in tests.
- **pid in the id** — non-deterministic.
- **A buffered file writer** — Decision 6 forbids it.
- **agentrepl formatting its own log records** — agentkit already owns the `LogRecord` protocol; reusing it via `Conversation.Log` is the point.

**Verification.**
- R-8GF4-LRYU — `ID(t)` is deterministic for a given `t`, and `Open` targets `<dir>/<id>.jsonl`.
- R-8HN0-ZJPJ — `Open` creates `dir` when missing and opens the file unbuffered for writing.
- R-8IUX-DBG8 — a completed run writes the conversation's records to the file (`turn_start` … `message`/`tool_use`/`tool_result`/`usage` … `turn_end`) and a `summary` on `Close`.
- R-8K2T-R36X — the log file content is identical whether the run used decorated or raw rendering.

## Decision 9 — Slash-command dispatch & the command set

**Decision.** `internal/repl` owns the loop and a small **command table** (`map[string]command` with handler + help text, which also generates `/help`). A line starting with `/` is a command; any other non-empty line is a turn message; an empty line is ignored.

```go
type command struct {
    summary string
    usage   string
    run     func(s *state, args string) error
}

// state is the live REPL state threaded to handlers.
type state struct {
    conv   *agentkit.Conversation
    target *config.Target
    cat    []catalog.Provider
    io     IO
    rend   render.Renderer // mutable: /render swaps it
    color  bool
    getenv func(string) string
    quit   bool            // set by /exit, /quit
}

// Run opens the session log, builds the Conversation + Target, applies opts.Config
// (fatal on error), then drives the loop. Returns the process exit code.
func Run(ctx context.Context, d Deps, opts Options) int

type Deps struct {
    IO     IO
    Getenv func(string) string
    Now    func() time.Time
    LogDir string // ~/.agentkit, resolved by the composition root
}
```

The command set:

| Command | Effect |
|---------|--------|
| `/set <key> <value>` | `config.Set` — runtime equivalent of `-c`; errors are **non-fatal** (rendered, loop continues). Value may contain spaces (e.g. `/set system You are helpful`): the handler splits `args` into key + remainder. |
| `/get <key>` | `config.Get` for one key |
| `/dump` | `config.Dump` — every `key=value` |
| `/usage` | cumulative summary (`Renderer.Summary`, Decision 7) |
| `/clear` | empties `conv.History` (fresh conversation); **cumulative spend persists** — it is real session spend, and agentkit's cumulative cannot be reset without a new `Conversation` |
| `/render <decorated\|raw>` | swaps the active renderer for subsequent turns |
| `/providers` | lists providers, each with env-key-present? and its curated models (drives "pick a provider whose key is present") |
| `/help` | lists commands and config keys (`config.Keys()`) |
| `/exit`, `/quit` | graceful exit (Decision 6) |

**Turn pre-check:** a non-command line, when `Provider`/`Model` are not both set, renders a clear hint ("set a provider and model first — e.g. `/set provider anthropic` / `/set model …`") instead of calling `Send` — the friendly form of Decision 4's promise.

**Rejected.**
- **A giant `switch`** — the map gives `/help` for free and is the idiomatic registry shape.
- **Fatal `/set` errors** — contradicts Decision 4's launch/runtime asymmetry.
- **`/clear` also zeroing spend** — agentkit's cumulative can't be reset without a new `Conversation`, and the spend is genuinely real.

**Verification.**
- R-BI0J-TIHX — a `/`-prefixed line dispatches to its command; an unknown `/command` is a clear non-fatal error and the loop continues.
- R-BJ8G-7A8M — a non-`/`, non-empty line is treated as a turn; an empty line is ignored.
- R-BKGC-L1ZB — `/set`/`/get`/`/dump` reach `config` with runtime (non-fatal) error handling.
- R-BLO8-YTQ0 — `/clear` empties `conv.History` so the next turn carries no prior context, while the cumulative spend is unchanged.
- R-BMW5-CLGP — `/render decorated|raw` switches the active renderer for subsequent turns.
- R-BO41-QD7E — `/help` lists the commands and the config keys.
- R-BPBY-44Y3 — `/providers` lists each provider with whether its env key is present and its curated models.
- R-BQJU-HWOS — a turn attempted before provider+model are both set renders a clear hint and does not call `Send`.

## Decision 10 — Built-in tools (bash / read / write / edit)

**Decision.** `internal/tools` builds the four tools with `agentkit.NewTool[In]` (typed input struct → derived JSON schema — the idiomatic agentkit path). Public surface is just:

```go
package tools

// All returns the four built-in tools, operating relative to the process
// working directory.
func All() []agentkit.Tool
```

Typed inputs (with `jsonschema_description` tags):

```go
type bashInput  struct { Command string `json:"command"` }
type readInput  struct { Path string `json:"path"` }
type writeInput struct { Path, Content string }
type editInput  struct { Path, Old, New string }
```

Behavior:
- **bash** — `bash -lc <command>`, returns **combined stdout+stderr**; on non-zero exit it appends `[exit status N]` and returns a **nil** error. Deliberate: agentkit's `runTool` discards a tool's output string when the tool returns a non-nil error (it surfaces only `err.Error()`), so returning the output as the value preserves what is actually needed. (The shipped example loses it; agentrepl fixes that.)
- **read** — `os.ReadFile(path)`; a missing/unreadable file returns a normal (non-terminal) error, which agentkit feeds back as an `IsError` tool result so the model can react and the loop continues.
- **write** — `os.WriteFile(path, content, 0o644)`, create/truncate; returns a short confirmation.
- **edit** — read, **replace all** occurrences of `Old` with `New`, write back; result reports the count; `Old` absent → non-terminal error (`IsError` result).
- **Paths are relative to the process cwd; no sandbox/confinement** — these are real local tools on the developer's own machine (bash already implies full local access). "Rooted at cwd" means relative-path resolution, not a jail.

**Rejected.**
- **`RawTool` with hand-written schemas** — more boilerplate; `NewTool` reflection is the idiomatic path.
- **Returning bash failures as Go errors** — loses the command output.
- **Sandboxing paths to cwd** — not promised, and bash defeats it anyway.
- **require-unique `edit`** — adds failure modes that aren't the point of a demonstration tool.

**Verification.**
- R-NHBW-446N — `All()` returns exactly four tools named `bash`, `read`, `write`, `edit`, each with a valid JSON schema.
- R-NIJS-HVXC — `bash` returns combined stdout+stderr, preserving output even on non-zero exit (with the exit status noted).
- R-NKZL-9FEQ — `read` returns a file's contents; a missing file yields a non-terminal `IsError` result.
- R-NM7H-N75F — `write` creates/overwrites a file with the given content.
- R-NNFE-0YW4 — `edit` replaces all occurrences of `Old` with `New`; absent `Old` yields a non-terminal `IsError` result.
- R-NONA-EQMT — all four tools resolve paths relative to the process working directory.

## Decision 11 — Error handling & REPL resilience

**Decision.** One resilience invariant governs the loop: **the only things that end it are `/exit`, `/quit`, EOF, and SIGINT.** No turn error, command error, config error, missing key, or non-terminal tool error ever stops the REPL. Expected failures are typed and *surfaced*; only genuine bugs (panics) and startup-fatal conditions stop the process.

- **Turn errors.** After draining `stream.Events()`, if `ctx.Err() != nil` → interrupt path (Decision 6); else relay `stream.Warnings()` (Decision 5 — a warning is not an error and never ends or alters the loop), then if `stream.Err() != nil` → `Renderer.Error(err)` and continue. agentkit's errors are already descriptive (`*agentkit.Error.Error()` carries provider/status/type); the decorated treatment shows the message in the error style, raw emits it as a JSON line. The session log independently captures agentkit's own `error` record, so failures are forensically preserved regardless of render mode.
- **Command / config / selection errors** (runtime) are rendered via `Renderer.Error`; the loop continues. This is where "a provider whose env key is absent produces a clear message and does not crash" lands: `catalog.ErrMissingKey` flows `/set provider …` → `config.Set` → `Renderer.Error`.
- **Z.ai known-broken** needs no special case — selecting it succeeds (key present), and the send-time agentkit failure flows the ordinary turn-error path, surfaced cleanly, REPL stays usable.
- **No panic recovery.** Per the project's principles ("fail loudly; crash over silent corruption"), agentrepl does not `recover`. Every *expected* condition — bad input, missing file, provider/network error, missing key, invalid config — is a typed error that gets surfaced; a panic means a real bug and should crash, not be masked.
- **Error stream split.** In-loop errors go to the renderer's `out` (**stdout**), so the interactive session — and piped raw output — reads as one coherent, in-order stream. **stderr is reserved for startup-fatal messages only** (bad flags, bad `-c`; exit 1).

**Rejected.**
- **Catch-all `recover` around the loop** — masks bugs, contradicts fail-loudly.
- **agentrepl re-classifying agentkit errors into its own taxonomy** — needless; agentkit's typed errors and messages are already the contract, and agentrepl just surfaces them.
- **Ending the loop on any error** — contradicts the always-usable promise.

**Verification.**
- R-H7HT-LNRE — a non-interrupt turn error is rendered via `Renderer.Error` and the loop accepts the next input.
- R-H8PP-ZFI3 — a runtime command/config error (e.g. missing key on `/set provider`) is rendered clearly and the loop continues.
- R-H9XM-D78S — the loop exits only on `/exit`, `/quit`, EOF, or SIGINT — no error condition ends it.
- R-HB5I-QYZH — in-loop errors are written to the renderer's `out` (stdout); startup-fatal errors go to stderr.
- R-HCDF-4QQ6 — expected failure conditions (bad input, missing file, provider error, missing key, invalid config) are surfaced as rendered errors, never as a panic or process exit.

## Decision 12 — The self-describing `--help` catalog

**Decision.** `-h`/`-help` prints a **static, credential-blind catalog** sourced entirely from the curated catalog (Decision 2) and agentkit's per-package introspectors — never from constructed clients or the environment. It is the launch-time answer to the product's "discover providers, models, and reasoning options without starting a session." Rendering lives in `internal/repl` as a pure function over the catalog and an `io.Writer`; `ParseArgs` invokes it on the help flag (Decision 4) and the composition root exits `0`.

```go
package repl

// WriteHelp renders the static catalog: a one-line usage, the launch flags, the
// providers list, and the models-grouped-by-provider list with each model's
// native reasoning term and accepted values. It reads only cat (no env, no
// constructed providers), so it is credential-blind.
func WriteHelp(out io.Writer, name string, cat []catalog.Provider)
```

Catalog shape (illustrative — exact bytes pinned by a golden file under `testdata/`):

```
usage: agentrepl [-c key=value ...] [-raw] [-h]

flags:
  -c key=value   set an agentkit config value (repeatable); see config keys via /help at runtime
  -raw           emit the raw, undecorated message stream
  -h, -help      show this catalog and exit

providers:
  anthropic   (ANTHROPIC_API_KEY)
  google      (GEMINI_API_KEY)
  openai      (OPENAI_API_KEY)
  zai         (ZAI_API_KEY)

models:
  anthropic
    claude-opus-4-8     effort: low, medium, high, xhigh, max  (default high)
    claude-haiku-4-5    thinking budget: 1024–<max_tokens>  (default off)
  google
    gemini-2.5-flash    thinking budget: 0–24576  (0=off, -1=dynamic; default dynamic)
    gemini-3.5-flash    thinking level: minimal, low, medium, high  (default medium)
  openai
    gpt-5.5             effort: none, low, medium, high, xhigh  (default medium)
  zai
    glm-5.2             effort: high, max  (default max)
    glm-4.7             thinking: on/off  (default on)
```

- **One render routine keyed on `spec.Kind`** turns a `ReasoningSpec` into its values clause — no per-provider formatting:
  - `ReasoningEnum` → `<Term>: <Levels joined ", ">  (default <level>)`.
  - `ReasoningRange` → `<Term>: <Min>–<Max>` then any `Sentinels` as `(v=meaning, …; default <d>)`. The default is rendered from `spec.Default` in its native form (a budget int, a sentinel meaning, or `off` for a disabled default).
  - `ReasoningToggle` → `<Term>: on/off  (default on|off)`.
- **Reasoning facts come only from `p.Reasoning`** (the Decision 2 inspector). A model whose `ReasoningSpec(id)` returns `false` is rendered with a plain `(no reasoning control)` clause rather than omitted.
- **Default value rendering reuses the same native formatter** the config display side uses (Decision 3), so `--help` and `/get` describe a model's default identically.
- **Provider order and model grouping follow `catalog.Default()`'s stable order** — the catalog owns display order; the renderer does not sort.
- **Strictly credential-blind:** `WriteHelp` takes the catalog and writes text; it never calls `Build`, `Getenv`, or any provider constructor. The env-key name in parens is the catalog's `EnvKey` *string*, printed as documentation, not a presence check — that live view is `/providers` (Decision 9).

**Rejected.**
- **The bare `flag.FlagSet` usage** (the prior decision) — lists no providers, models, or reasoning; leaves the product's self-describing-help promise unmet.
- **Sourcing reasoning from a constructed provider / `SupportedReasoning()` on a handle** — would need a key just to print static metadata; the credential-blind package-level inspector exists precisely to avoid this.
- **Per-provider format branches** — reintroduces the provider knowledge the thin-consumer principle forbids; a single `Kind`-keyed routine renders all three shapes.
- **Hardcoding the reasoning text in agentrepl** — drifts from agentkit the moment a model's vocabulary changes; reading `ReasoningSpec` at print time keeps display and acceptance from one source.

**Verification.**
- R-FT8W-Z2U4 — `-h`/`-help` writes the catalog to `out` and exits `0` without starting the REPL loop, and does so with **no** environment variables set and **no** provider constructed (credential-blind).
- R-FUGT-CUKT — the catalog lists every provider from `catalog.Default()` and, grouped under each, every curated model id, in `Default()`'s order.
- R-FVOP-QMBI — each model's reasoning clause is rendered from its `ReasoningSpec` by `Kind`: an enum model shows `Term` + its `Levels` + native default; a range model shows `Term` + `Min`–`Max` + sentinel meanings + native default; a toggle model shows `Term` + `on/off` + default (golden-pinned across all three kinds).
- R-FWWM-4E27 — a model whose inspector returns `ReasoningSpec(id) == (_, false)` renders a `(no reasoning control)` clause and is not dropped from the listing.
- R-FY4I-I5SW — `WriteHelp` performs no env read and constructs no provider (asserted by passing a catalog whose `New`/`Getenv` would record or panic if called), proving the help path cannot depend on credentials.

## Status

Decided: Decisions 1–12 — package layout & seams; provider & model catalog; config-key namespace & coercion; CLI flags; turn execution & rendering; REPL lifecycle, interrupt & log integrity; usage & cost reporting; session log & session-id; slash-command dispatch & command set; built-in tools; error handling & REPL resilience; the self-describing `--help` catalog.

The seams, public interfaces, naming, struct/type definitions, data model, and the testing approach are fully decided. The construction order that realizes this design lives in the plan.
