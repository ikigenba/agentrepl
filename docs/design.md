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
  internal/config/         flat-key namespace: parse key=value, apply to Conversation, dump, validate
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
| **Waiter** | `interface { Start(model string); Stop() }` — the wait status line (Decision 13); live driver (goroutine+clock) bound only on a TTY, else `nopWaiter` | tests inject a recording fake or the `nopWaiter`; the line's text is a pure, table-tested formatter |

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
- **Provider-construction overrides ride on `Options`, not new catalog entries.** `Build` threads an `Options` into `New`; today only the `zai` entry consumes it, mapping a non-empty `Options.BaseURL` to agentkit's `zai.WithBaseURL(...)` option and otherwise leaving Z.ai's baked-in default root (`https://api.z.ai/api/paas/v4`). The other three entries ignore `Options`. This is the seam the `base_url` config key (Decision 3) drives — e.g. to point Z.ai at its coding-plan endpoint `https://api.z.ai/api/coding/paas/v4` — keeping the "new knob = new key, no bespoke flag" promise and avoiding a per-endpoint catalog entry.
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
    ReasoningRaw string // last raw reasoning value, for display only (ReasoningValue is opaque); "" = unset
    ReasoningKey string // which native reasoning key set it (effort|thinking_budget|thinking_level|thinking); "" = unset
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

Internally a `map[string]field`, where `field` carries `set(t, raw) error` and `get(t) string`. The keys are **flat, unprefixed names** — one per agentkit setting (no `gen.`/`retry.`/`zai.` namespace; the prefix never routed anything, since `Set` is a single `map[string]field` lookup). Reasoning is exposed as the **native term of each model** rather than one generic key:

| Key | Target field | Type |
|-----|--------------|------|
| `provider` | `Conv.Provider` (via catalog) | provider name |
| `model` | `Conv.Model` | model id |
| `system` | `Conv.System` | string |
| `temperature` | `Gen.Temperature` | `*float64` |
| `top_p` | `Gen.TopP` | `*float64` |
| `max_tokens` | `Gen.MaxTokens` | int |
| `effort` | `Gen.Reasoning` via `Level(raw)` (+ `ReasoningRaw`/`ReasoningKey`) | native level — Anthropic opus/sonnet, OpenAI gpt-5.x, GLM 5.x |
| `thinking_budget` | `Gen.Reasoning` via `Budget(int)` (+ `ReasoningRaw`/`ReasoningKey`) | native token budget — Anthropic haiku, Gemini 2.5 |
| `thinking_level` | `Gen.Reasoning` via `Level(raw)` (+ `ReasoningRaw`/`ReasoningKey`) | native level — Gemini 3.x |
| `thinking` | `Gen.Reasoning` via `on`→enabled / `off`→`DisableReasoning()` (+ `ReasoningRaw`/`ReasoningKey`) | native toggle — GLM 4.x |
| `max_attempts` | `Retry.MaxAttempts` | int |
| `base_delay` | `Retry.BaseDelay` | duration (e.g. `500ms`) |
| `max_delay` | `Retry.MaxDelay` | duration |
| `max_elapsed` | `Retry.MaxElapsed` | duration |
| `ignore_retry_after` | `Retry.IgnoreRetryAfter` | bool |
| `tool_loop_limit` | `Conv.MaxToolIterations` | int |
| `base_url` | `Target.ZaiBaseURL` → `catalog.Options.BaseURL` for the `zai` build | URL string |

- **The four reasoning keys are aliases over one field.** `effort`, `thinking_budget`, `thinking_level`, and `thinking` all read and write the single `Conv.Gen.Reasoning` value — a model has exactly one reasoning setting. Setting any one records the raw value on `Target.ReasoningRaw` **and** the key used on `Target.ReasoningKey`, so `Get`/`Dump` can render the value under the key the operator actually typed (the `Level` constructor is shared by `effort` and `thinking_level`, so the value's shape alone cannot recover which key set it — `ReasoningKey` disambiguates). Setting a second reasoning key overwrites both the value and the recorded key.

- **Unset sentinel.** Setting the literal value `default` resets *any* key to its zero/unset state (nil pointer, zero `ReasoningValue`, zero int/duration); `Dump`/`Get` render an unset key as `default`. One uniform rule, no per-key syntax.
- **provider / model coupling** (loose, to avoid ordering deadlocks):
  - `provider=<name>` → catalog `Lookup` + `Build(getenv)`; sets `Conv.Provider`; surfaces `ErrUnknownProvider` / `ErrMissingKey` through `Set`; does not touch model.
  - `model=<id>` → if a provider is set, pre-validate with `HasModel` (clear `ErrUnknownModel` with "choose from: …"); else accept the string. The (provider, model) pair is ultimately validated by agentkit at `Send` and surfaced cleanly — a transient post-switch mismatch is a clear send-time error, never a crash.
- **`base_url` is a provider-construction override, applied through the catalog `Options` seam (Decision 2), and order-independent with `provider`.** A base URL is baked into the provider handle at construction, not a `Conversation` field, so the value is stored on `Target.ZaiBaseURL` and the `zai` provider is (re)built to apply it. `provider=zai` builds with `Options{BaseURL: t.ZaiBaseURL}`; setting `base_url` while `zai` is already the active provider rebuilds it with the new root; setting it before any provider is selected just stores it for the eventual `zai` build — either order reaches the same state. `base_url=default` clears the override (back to Z.ai's baked-in root) and rebuilds if zai is active. For a non-zai active provider the key is stored but not applied (a no-op against the live conversation until zai is selected). (Z.ai is the only provider with a base-URL override today, so the flat key needs no provider qualifier; a second provider needing one would reintroduce a qualifier at that point.)
- **Reasoning is the carve-out: key-directed, never model-validated in agentrepl, never a hard error on non-native input.** The four reasoning keys are the ones whose *non-native* value does *not* return `ErrBadValue` (product success-criteria carve-out). The **key name** — not the value's shape — picks the `agentkit.ReasoningValue` constructor, so coercion is model-blind and works at launch before any model is selected and after a mid-conversation switch:
  - `effort` and `thinking_level` → `agentkit.Level(raw)` **verbatim** — `"high"`, `"xhigh"`, `"minimal"`, `"max"`, and notably `"none"` (a real native effort level on gpt-5.x, **not** a disable token), passed through untouched.
  - `thinking_budget` → `agentkit.Budget(n)`, where `raw` parses as a base-10 integer **including a leading `-`** (so sentinels like `-1`=dynamic and `0`=off pass). A non-integer is *structurally* unusable for this key and returns `ErrBadValue`.
  - `thinking` → `off`→`agentkit.DisableReasoning()`; `on`→the unset (zero) `ReasoningValue`, i.e. the model's native default, which for a toggle model is thinking-enabled (agentkit renders an unset toggle as `on`). `on` is recorded on `ReasoningRaw`/`ReasoningKey` for display even though the underlying value is the unset zero. Any token other than `on`/`off` is structurally unusable and returns `ErrBadValue`.

  The key name resolves the ambiguities the old single-key scheme had to guess at by shape: with `effort`/`thinking_level` the value is *always* a level (no integer-vs-level race), and `off` vs the native level `"none"` is decided by which key was used (`thinking=off` disables; `effort=none` is the level). The built value is assigned to `Conv.Gen.Reasoning`; the raw string is stored on `Target.ReasoningRaw` and the key on `Target.ReasoningKey` for display. agentrepl does **not** consult the active model's `ReasoningSpec` here and does **not** judge whether the value is native to the selected model — that judgment, and the warn-and-default, are agentkit's at request-build time (D6), which is the only place the *currently-selected* model is authoritative (a value valid for model A may be invalid for model B after a switch). A non-native value (e.g. `effort=high` set while a budget-only model is active) is therefore accepted by `Set` without error, stored, and later **warned + defaulted** by agentkit, with agentrepl relaying the `Warning` (Decision 5). The only `set`-time failures are structurally unusable inputs (an empty value for any reasoning key; a non-integer `thinking_budget`; a non-`on`/`off` `thinking`), which return `ErrBadValue`.
  - **Display.** `Get`/`Dump` render reasoning from `Target.ReasoningRaw` under `Target.ReasoningKey` (the native key + string the operator gave), and an unset value (`ReasoningRaw == ""`) as `default` for whichever reasoning key is queried — the same uniform unset rendering every other key uses. `Dump` emits the value once, under the key it was set with; the other three reasoning keys render as `default`. **Runtime `/get`/`/dump` show only the current value (and `default` for unset); they deliberately do *not* reprint the active model's accepted-values catalog** — that discovery view is `--help`-only (Decision 12), keeping the runtime surface thin and matching the product, which mandates the catalog for `--help` alone. The native default *value* shown in `--help` comes from `spec.Default`, not from this field.
  - **`default` resets it** to the unset `ReasoningValue` zero (model default) and clears `ReasoningRaw`/`ReasoningKey`, via the uniform unset rule above — set on *any* of the four reasoning keys it clears the one shared value; no reasoning-specific syntax.
  - The four reasoning keys are **excluded from the generic non-native-value error coverage**: the generic error-path id asserts `ErrBadValue` for every *other* key; reasoning's "no hard error on non-native input" (only on structurally unusable input) is asserted by its own ids below.
- Both control surfaces share this one entrypoint: the `-c` flag does `ParsePair` then `Set`; `/set <key> <value>` calls `Set` directly; `/dump` calls `Dump`. Adding a key automatically reaches both surfaces.

**Rejected.**
- **Reflection / struct-tag mapping** — clever and opaque, produces poor errors, fails the idiom bar.
- **A bespoke flag per setting** — contradicts the product's "new knob = new key, no new flag."
- **Strict pair validation at set time** — deadlocks: can't set the new provider while the old model is invalid, nor the new model while the old provider is current.

**Verification.**
- R-LYK7-Y7ZS — every known key coerces its value to the correct typed `Target` field (table-driven across the full key list, including pointer, int, duration, bool, enum, and string kinds).
- R-LZS4-BZQH — an unknown key returns a wrapped `ErrUnknownKey` naming the key and mutates nothing.
- R-M100-PRH6 — an unparseable value returns a wrapped `ErrBadValue` naming the key and reason, and mutates nothing (every key **except** the four reasoning keys `effort`/`thinking_level`/`thinking_budget`/`thinking`, whose non-native-value carve-out is asserted separately below).
- R-M27X-3J7V — the `default` value resets a pointer/enum key to unset, and `Dump`/`Get` then render it as `default`.
- R-M3FT-HAYK — `provider=` constructs via the catalog and surfaces `ErrUnknownProvider`/`ErrMissingKey` through `Set`; `model=` pre-validates against the current provider with `ErrUnknownModel`.
- R-M4NP-V2P9 — `Dump` returns all keys sorted as `key=value` lines reflecting current state.
- R-M5VM-8UFY — flag/runtime parity: `ParsePair`+`Set` and a direct `Set` reach identical state for the same key and value.
- R-SCS3-DV9R — setting `base_url` stores the override on `Target` and, when `zai` is (or becomes) the active provider, the constructed Z.ai provider is built with that base URL via the catalog `Options` seam — reached identically whether `base_url` is set before or after `provider=zai`; `base_url=default` clears it and rebuilds against the baked-in root.
- R-FZCE-VXJL — reasoning coercion is key-directed and model-blind: `effort`/`thinking_level` → `Level(verbatim)` (`high`, `xhigh`, `none`); `thinking_budget` → `Budget(n)` for a base-10 integer including negatives (`-1`, `0`, `8000`); `thinking=off` → `DisableReasoning()` and `thinking=on` → the unset zero `ReasoningValue`; in every case the result is assigned to `Conv.Gen.Reasoning` and the raw string + key stored on `Target.ReasoningRaw`/`ReasoningKey`.
- R-G0KB-9PAA — the reasoning carve-out: a non-native value on a reasoning key (e.g. `effort=high` on a budget-only model, or a made-up level) is accepted by `Set` **without** `ErrBadValue` and stored for agentkit to warn+default at turn time; only structurally unusable input errors (empty value on any reasoning key, non-integer `thinking_budget`, non-`on`/`off` `thinking`).
- R-G304-18RO — `Get`/`Dump` render reasoning from `Target.ReasoningRaw` under `Target.ReasoningKey` (the native key + string given), the other reasoning keys and an unset value as `default`; setting any reasoning key to `default` resets `Conv.Gen.Reasoning` to the zero `ReasoningValue` and clears `ReasoningRaw`/`ReasoningKey`.

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

// Renderer presents the input prompt, streamed events, outcome, and spend.
type Renderer interface {
    Prompt()                                                     // draw the input prompt before a read (decorated+TTY: "you › "; raw/non-TTY: no-op)
    Input(text string)                                           // the operator's entered turn message (raw records it; decorated does not echo it)
    Event(ev agentkit.Event)                                     // each streamed event, in order
    Usage(turn agentkit.Usage, turnCost, total agentkit.Cost)    // per-turn usage/cost line (raw only; decorated no-op)
    Summary(total agentkit.Usage, totalCost agentkit.Cost)       // cumulative usage+cost block (/usage, on exit)
    Warning(w agentkit.Warning)                                  // a setting agentkit could not honor as asked
    Error(err error)                                             // a failed turn or command
    Notice(line string)                                          // agentrepl info (e.g. /dump, hints)
}

func NewDecorated(out io.Writer, color, tty bool) Renderer
func NewRaw(out io.Writer) Renderer
```

`Prompt()` and `Input(text)` split what was one `Prompt(text)` method into the two jobs it conflated: drawing the input affordance (before a read, no text, terminal-only) and recording the entered message (after a read, raw's forensic stream). The decorated view never echoes the operator's input — the terminal already shows what was typed at the prompt; raw never draws an affordance — it has no interactive surface. The decorated renderer is constructed with both `color` and `tty`: `tty` gates whether the `you ›` prompt is drawn at all (an interactive terminal echoes typed input so the line reads `you › hi`), independent of `color`, which only gates ANSI (so `NO_COLOR=1` in a terminal still gets an uncolored prompt). `/render` reconstructs from `state.io.IsTTY`.

**Input affordance is loop-owned, not driver-owned.** The `Run` loop (Decision 9) calls `Renderer.Prompt()` *before* awaiting each input line — so the `you ›` prompt precedes **every** read uniformly: a turn message, a `/command`, or an empty line. The prompt is the renderer's, not the driver's; the driver no longer draws it.

**Turn driver** (`repl`): pre-check provider+model → `Input(text)` (records the message; raw-only effect) → `waiter.Start(conv.Model)` (decorated+TTY only — the wait status line, Decision 13; `nopWaiter` otherwise) → `stream := conv.Send(ctx, text)` → range `stream.Events()` calling `Event(ev)` for each, with `waiter.Stop()` on the first event and again via `defer` so every exit (success, error, empty, interrupt) erases the line → after draining, if `ctx.Err() != nil` render an interrupt notice and exit (Decision 6); otherwise **relay any settings warnings first** — `for _, w := range stream.Warnings() { Warning(w) }` — then, if `stream.Err() != nil` call `Error(err)`, else call `Usage(stream.Usage(), stream.Cost(), conv.TotalCost())`. The driver calls `Usage` uniformly; the decorated renderer no-ops it (per-turn spend is raw-only — Decision 7), so the difference lives in the renderer, not the driver. Warnings are rendered whether the turn then succeeds or errors, because they describe a setting that was not honored (most often reasoning: a non-native value carried in via the Decision 3 carve-out, which agentkit warned-and-defaulted), independent of turn outcome. The `ctx` passed to `Send` is the SIGINT-cancellable context from Decision 6; the loop survives any non-interrupt turn error.

**Warning relay is render-only.** agentrepl never mints, classifies, or suppresses a warning — it forwards each `agentkit.Warning` verbatim to the renderer. A `Warning` is **not** an `Error` (the turn was issued and, for the reasoning case, succeeds with the model's default); it gets its own kind so its treatment and placement (before the usage line) are distinct.

**decorated** (default): a distinct visual treatment per kind — `Prompt()` draws the `you ›` input prompt (**bold**, default foreground — the terminal's own white; ANSI emitted only when `color`; no trailing newline, so the terminal's own echo of typed input completes the line, and that echoed input keeps the default foreground too) **only when `tty`**, and draws nothing otherwise; `Input(text)` is a **no-op** (the entered text is never re-rendered — the terminal already showed it); `TextDelta` streamed inline as the reply, with the `assistant ›` label **bold + light blue** (ANSI bright-blue) and the streamed reply text the **same light blue but not bold** (the hue carries the whole reply; only the label is bold); `ReasoningDelta` dim/streamed; `ToolUse` and a **successful** `ToolResult` each rendered as a **subdued gray** line (ANSI bright-black), label and body alike; a `ToolResult` with `IsError` instead keeps the **red** error treatment so a failing tool stands out rather than blending into the gray; `Warning` in a distinct warn style (e.g. `⚠ <Setting>: <Detail>`, separate from the error style); `Usage(...)` is a **no-op** (per-turn spend is raw-only — Decision 7). `MessageDone` flushes the streamed line with a newline but prints **no separator rule** between messages — the prior `─` turn/message separator is removed. Exact glyphs/labels/ANSI codes are pinned by golden files under `testdata/`.

**Vertical spacing — one blank line between consecutive blocks, as a leading separator.** The decorated view separates every adjacent pair of transcript blocks (the `you ›` prompt, an `assistant ›` reply, `reasoning ›`, `tool call ›`, `tool result ›`, a notice, a warning, an error, the summary) with exactly **one blank line**, for a uniform vertical rhythm. The blank is emitted as a **leading** separator: a single newline written at the *start* of each block, suppressed until the first block of the session has been drawn — so the session opens with no leading blank and closes with no trailing blank after the final summary, while the blank the operator perceives "after the `you ›` line" is the leading separator of the block that follows it. The `you ›` prompt therefore never owns a trailing newline (the terminal's echo of the typed line completes it); whatever block comes next — a reply, a tool call, or a command's notice — supplies the separator, so turns, `/commands`, and notices space identically. A **bare empty line** at the prompt (ignored per Decision 9) produces **no** extra blank: the separator appears only between actual blocks, never between two consecutive prompts. To keep the single-blank rule exact regardless of tool output, the `tool call ›` / `tool result ›` emitters **trim a single trailing newline** from the rendered input/output, so a tool whose output ends in `\n` still yields exactly one blank, not two.

**raw**: emits **one undecorated JSON line per committed entry** — the prompt entry from `Input(text)` (`{"type":"prompt","text":…}`), `MessageDone`, `ToolUse`, `ToolResult` — plus the per-turn usage line as JSON and **one JSON line per `Warning`** (the `agentkit.Warning` struct marshaled verbatim, carrying `Setting`/`Code`/`Detail`); `Prompt()` is a **no-op** (no interactive affordance in raw), and it **skips `TextDelta`/`ReasoningDelta`** (streaming fragments, not entries — and `MessageDone.Message` already carries the fully assembled text/reasoning blocks). Raw is unchanged by this revision: it still records the prompt text and the per-turn usage. Never emits ANSI. Marshals with `encoding/json`, yielding block shapes consistent with agentkit's own log (agentkit does no custom marshaling).

**color**: the composition root computes `color = IO.IsTTY && Getenv("NO_COLOR") == ""` and passes it **plus `IO.IsTTY`** to `NewDecorated(out, color, tty)`. The prompt is drawn when `tty` (an interactive terminal); ANSI is emitted when `color`. Raw is always colorless and draws no prompt.

**Rejected.**
- **Raw = tee agentkit's `Conversation.Log` (LogRecord stream) to stdout** — byte-identical to the forensic file, but it makes `rawRenderer` a no-op while output sneaks in via a `MultiWriter`, turning the two-impl seam into a fiction and coupling raw render to the log. Marshaling events keeps both renderers honest and the log independent.
- **Raw marshals every event including deltas** — a flood of tiny delta objects; "one per entry / messages verbatim" argues for entry granularity.
- **A `Renderer` per event-kind / a visitor** — overwrought; a small sealed-union `switch` inside each impl is the idiomatic Go shape.

**Verification.**
- R-LL9K-SKDQ — decorated renders each transcript kind (reply text, reasoning, tool-call, tool-result) with a distinct treatment, and prints **no** `─` separator rule between messages (golden).
- R-Q52T-PXCR — decorated separates every adjacent pair of transcript blocks with exactly one blank line (leading separator): a multi-block transcript (`you ›` → reply → tool call → tool result → reply) shows one blank line between each pair, no leading blank before the first block, no trailing blank after the summary, a tool result whose output ends in `\n` still yields exactly one blank, and a bare empty line at the prompt adds no blank (golden).
- R-JFBW-TYU8 — the loop calls `Renderer.Prompt()` before every input read (turn, command, and empty line alike); in decorated mode `Prompt()` writes the `you ›` prompt (no trailing newline) when `tty` and writes nothing when not a `tty`, and `Input(text)` never echoes the entered text back as a transcript line (golden for the tty/non-tty decorated output; repl-level assertion that the prompt precedes each read).
- R-JGJT-7QKX — in decorated mode `Usage(...)` prints nothing (per-turn spend suppressed); the same driver call in raw mode emits the per-turn usage line.
- R-LMHH-6C4F — decorated streams `TextDelta`/`ReasoningDelta` incrementally: bytes are written as each delta arrives, not buffered to end of turn.
- R-LNPD-K3V4 — decorated emits ANSI color when `color` is true and none when it is false (golden for both).
- R-OBNM-N6XX — decorated applies the fixed palette (golden): the `you ›` prompt and its echoed input are bold/default-foreground (no hue); the `assistant ›` label is bold + light blue and the streamed reply text is the same light blue but not bold; `tool call ›` and a successful `tool result ›` line are subdued gray (label and body); a `tool result ›` with `IsError` is rendered in the red error treatment; `reasoning ›` is dim.
- R-LOX9-XVLT — raw emits exactly one undecorated JSON line for the prompt entry (from `Input`), and one per `MessageDone`/`ToolUse`/`ToolResult`, plus a per-turn usage line, skipping deltas and emitting nothing for `Prompt()`; the output is valid JSONL with no ANSI.
- R-LRD2-PF37 — a `ToolResult` with `IsError` gets the error treatment in decorated and preserves `IsError` in raw.
- R-LSKZ-36TW — on a non-interrupt failed turn the driver calls `Error` (not `Usage`) and the loop continues to the next input.
- R-G480-F0ID — after draining a turn's events, the driver calls `Renderer.Warning` once per entry in `stream.Warnings()`, before `Usage`/`Error`, and verbatim (the agentkit `Warning` is forwarded unmodified, never reclassified or suppressed); a turn with no warnings calls `Warning` zero times.
- R-G5FW-SS92 — decorated renders a `Warning` in its own warn treatment, distinct from the `Error` treatment (golden); raw emits exactly one JSON line carrying the `Warning`'s `Setting`/`Code`/`Detail`.
- R-G6NT-6JZR — a reasoning value the selected model does not natively understand (set via any reasoning key — the Decision 3 carve-out) produces an agentkit reasoning warning that the driver relays to the renderer, and the turn still completes with the model's default applied.

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

**Decision.** All spend numbers come straight from agentkit — agentrepl formats, never recomputes (the product: "surfaced from agentkit / drawn from agentkit's built-in pricing"). The cadence differs by renderer: the **decorated** view shows spend only as a **cumulative** summary (on `/usage` and at exit) — never a per-turn line; the **raw** view emits a per-turn usage line on every turn (its forensic, verbatim role). Because agentrepl exists to *verify* agentkit, both the per-turn line (raw) and the cumulative summary (both renderers) show the **full token-bucket breakdown**, not just a total.

Per-turn line (`Renderer.Usage(turn, turnCost, total)`), from `stream.Usage()`, `stream.Cost()`, `conv.TotalCost()` — **rendered by raw only**; the decorated renderer no-ops this call (Decision 5):

```
· tokens  in=123 cache(r=10 w=5) out=456 reasoning=78 total=657
· cost     $0.001234 turn   $0.005678 session
```

Cumulative summary (`Renderer.Summary(total, totalCost)`), from `conv.TotalUsage()`, `conv.TotalCost()` — the cumulative token breakdown plus session cost. This is the **only** spend the decorated view prints, and it carries the same bucket layout shown above.

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
- R-ONJY-6PJG — the raw per-turn line reports the turn's token buckets and total exactly as `stream.Usage()` reports them (no recomputation).
- R-OORU-KHA5 — the raw per-turn line reports turn cost from `stream.Cost()` and session cost from `conv.TotalCost()`, both in USD.
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

**Decision.** `internal/repl` owns the loop and a small **command table** (`map[string]command` with handler + help text, which also generates `/help`). The loop calls `Renderer.Prompt()` **before awaiting each input line**, so the `you ›` affordance precedes every read uniformly (decorated+TTY draws it; raw and non-TTY draw nothing — Decision 5). A line starting with `/` is a command; any other non-empty line is a turn message; an empty line is ignored. `/render` reconstructs the renderer from `state.io.Out`, `state.color`, and `state.io.IsTTY` so a swapped-in decorated renderer keeps the same prompt/color gating.

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
// reasoning shown as its own native config key (`effort`, `thinking_budget`,
// `thinking_level`, or `thinking`, derived from spec.Term) and accepted values
// in traditional CLI syntax (the native term kept as a trailing parenthetical).
// It reads only cat (no env, no constructed providers), so it is credential-blind.
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
    claude-opus-4-8     effort={low|medium|high|xhigh|max}         (effort; default high)
    claude-haiku-4-5    thinking_budget=<1024–max_tokens>          (thinking budget; default off)
  google
    gemini-2.5-flash    thinking_budget=<0–24576>                  (thinking budget; 0=off, -1=dynamic; default dynamic)
    gemini-3.5-flash    thinking_level={minimal|low|medium|high}   (thinking level; default medium)
  openai
    gpt-5.5             effort={none|low|medium|high|xhigh}        (effort; default medium)
  zai
    glm-5.2             effort={high|max}                          (effort (+ toggle); default max)
    glm-4.7             thinking={on|off}                          (thinking; default on)
```

- **The model's own native key leads each row.** Every reasoning line is labeled with the config key derived from that model's `spec.Term` — `effort`, `thinking_budget`, `thinking_level`, or `thinking` — followed by its accepted values in **traditional CLI syntax**: braces `{a|b|c}` for an enumerated choice, angle brackets `<…>` for a free numeric value. A reader copies the exact `-c <key>=<value>` terminology straight off the row, the same way the `providers:`/`models:` section labels telegraph the `provider`/`model` keys. The native term is no longer a separate-from-the-key parenthetical afterthought — it *is* the key — but the parenthetical is retained to carry the full term phrase (e.g. `effort (+ toggle)`), any sentinels, and the default.
- **The key comes from `spec.Term`, the values group from `spec.Kind`.** A small `termToKey` normalization maps the native term to one of the four registered config keys: lowercase, drop a trailing ` (+ toggle)`, replace the space in `thinking budget`/`thinking level` with `_`. This yields exactly `effort` / `thinking_budget` / `thinking_level` / `thinking` — the same four keys Decision 3 registers — so the help can never advertise a key the config layer does not accept (golden + a cross-check id below). One render routine then turns a `ReasoningSpec` into its clause — the per-model key prefix, the values group keyed on `spec.Kind`, then a trailing parenthetical carrying the native `Term`, any sentinels, and the default — no per-provider formatting:
  - `ReasoningEnum` → `<key>={<Levels joined "|">}  (<Term>; default <level>)`.
  - `ReasoningRange` → `<key>=<<Min>–<Max>>` then any `Sentinels` and the default in the parenthetical: `(<Term>; v=meaning, …; default <d>)`. The default is rendered from `spec.Default` in its native form (a budget int, a sentinel meaning, or `off` for a disabled default).
  - `ReasoningToggle` → `<key>={on|off}  (<Term>; default on|off)`.
- **Reasoning facts come only from `p.Reasoning`** (the Decision 2 inspector). A model whose `ReasoningSpec(id)` returns `false` is rendered with a plain `(no reasoning control)` clause rather than omitted.
- **Default value rendering reuses the same native formatter** the config display side uses (Decision 3), so `--help` and `/get` describe a model's default identically.
- **Provider order and model grouping follow `catalog.Default()`'s stable order** — the catalog owns display order; the renderer does not sort.
- **Strictly credential-blind:** `WriteHelp` takes the catalog and writes text; it never calls `Build`, `Getenv`, or any provider constructor. The env-key name in parens is the catalog's `EnvKey` *string*, printed as documentation, not a presence check — that live view is `/providers` (Decision 9).

**Rejected.**
- **The bare `flag.FlagSet` usage** (the prior decision) — lists no providers, models, or reasoning; leaves the product's self-describing-help promise unmet.
- **Sourcing reasoning from a constructed provider / `SupportedReasoning()` on a handle** — would need a key just to print static metadata; the credential-blind package-level inspector exists precisely to avoid this.
- **Per-provider format branches** — reintroduces the provider knowledge the thin-consumer principle forbids; a single `Kind`-keyed routine renders all three shapes.
- **Hardcoding the reasoning text in agentrepl** — drifts from agentkit the moment a model's vocabulary changes; reading `ReasoningSpec` at print time keeps display and acceptance from one source.
- **A single generic `gen.reasoning` key for every model** (the prior design) — a synthetic common interface over a setting that is natively per-model; removed in favor of passthrough native keys. The `gen.`/`retry.`/`zai.` prefixes went with it: they never routed anything (`Set` is one flat `map[string]field` lookup), so they were pure decoration.
- **A single header line naming the key once** (`models:  (set with -c <key>=<value>)`) — forces the reader to assemble the key from a distant header and the values from the row; each row should be independently copy-pasteable, carrying its own native key.
- **A `termToKey` that keys on `spec.Kind` instead of `spec.Term`** — `Kind=Enum` is shared by `effort` (Anthropic/OpenAI/Z.ai) and `thinking_level` (Gemini), so the kind alone cannot pick the key; the native term is the only unambiguous source.

**Verification.**
- R-FT8W-Z2U4 — `-h`/`-help` writes the catalog to `out` and exits `0` without starting the REPL loop, and does so with **no** environment variables set and **no** provider constructed (credential-blind).
- R-FUGT-CUKT — the catalog lists every provider from `catalog.Default()` and, grouped under each, every curated model id, in `Default()`'s order.
- R-FVOP-QMBI — each model's accepted-values group is rendered from its `ReasoningSpec` by `Kind`: an enum model shows its `Levels`, a range model shows `Min`–`Max` plus sentinel meanings, a toggle model shows `on`/`off`; each carries its native default and the native `Term` in the trailing parenthetical (golden-pinned across all three kinds).
- R-6DEO-9TXQ — every model row leads with that model's native reasoning key (`effort`/`thinking_budget`/`thinking_level`/`thinking`, derived from `spec.Term`) followed by its values in traditional CLI syntax — `{a|b|c}` for enum/toggle, `<…>` for a range — so the row is copy-pasteable as `-c <key>=<value>`; the native term phrase, sentinels, and default appear in the parenthetical.
- R-6DEO-KEYS — `termToKey(spec.Term)` for every supported model resolves to one of the four reasoning keys Decision 3 registers (`effort`/`thinking_budget`/`thinking_level`/`thinking`), so the catalog never prints a key the config layer would reject as unknown.
- R-FWWM-4E27 — a model whose inspector returns `ReasoningSpec(id) == (_, false)` renders a `(no reasoning control)` clause and is not dropped from the listing.
- R-FY4I-I5SW — `WriteHelp` performs no env read and constructs no provider (asserted by passing a catalog whose `New`/`Getenv` would record or panic if called), proving the help path cannot depend on credentials.

## Decision 13 — Wait status line (`waiting for <model>`)

**Decision.** While a turn is in flight and the model has not yet produced output, the decorated view paints a single ephemeral status line — `waiting for <model> (<elapsed>)` — repainted in place and erased the instant the turn produces anything. It is modeled on ralph's wait spinner, **without the spinner glyph**: only the elapsed seconds change. The animator is the **one impure seam** (goroutine + ticker + real wall clock + terminal erase writes); everything it draws comes from a pure, table-tested formatter.

```go
package repl

// Waiter shows an ephemeral "waiting for <model> (<elapsed>)" status while a
// turn is in flight, then erases it. Injected via Deps; the live driver is bound
// only when stdout is an interactive TTY, a nopWaiter otherwise (and in tests).
type Waiter interface {
    Start(model string) // begin the wait; first paint only after the pre-roll
    Stop()              // halt the painter and erase any line it drew; idempotent
}
```

```go
package render

// waitLine builds one paint of the status line: "waiting for <model> (<elapsed>)",
// dim-gray-wrapped when color, plain otherwise. No erase prefix, no trailing
// newline — the live driver prepends "\r\x1b[2K" per repaint. Pure → table test.
func waitLine(model string, elapsed time.Duration, color bool) string

// formatElapsed renders a duration compactly with h/m/s rollover, truncated to
// whole seconds, higher-order zero units omitted: 5s, 2m17s, 1h2m3s. Pure → table test.
func formatElapsed(d time.Duration) string
```

- **Pure core, impure driver.** `waitLine` + `formatElapsed` carry the text and the rollover and are table-tested with no clock or IO. The live driver (`liveWaiter`) owns the goroutine, the `time.Ticker`, the real `time.Now`, and the `\r\x1b[2K` erase writes — it is the documented exception to the Conventions "no `time.Now` outside the composition root" rule, constructed in the composition root and never reached by a golden test. A `nopWaiter` (both methods empty) is the default everywhere else.
- **Cadence & pre-roll.** A **2s pre-roll** of silence precedes the first paint, so a turn that finishes quickly never shows the line (and `Stop` then erases nothing). After the pre-roll the line repaints every **80ms** — fast enough that the elapsed counter never visibly jumps when its width changes (`9s`→`10s`, `59s`→`1m0s`) and the line never reads as choppy, even though only the seconds move.
- **Stream, color, and TTY gating.** The line is written to the decorated renderer's **stdout** (`IO.Out`), in **subdued gray** (ANSI bright-black) when `color`, matching the tool-line register (Decision 5). The live driver is bound **only when `IO.IsTTY`**; non-TTY and raw runs get `nopWaiter`, so machine-readable output never receives a status line. The model name is `conv.Model` at send time.
- **Driver wiring (Decision 5).** The turn driver calls `waiter.Start(conv.Model)` immediately before ranging the stream and `waiter.Stop()` both on the first event drawn and via `defer` (so an errored, empty, or interrupted turn still erases). The waiter runs **only while the decorated renderer is active** — a turn taken after `/render raw` uses the nop, mirroring how `Prompt()` is a no-op in raw.
- **Spacing interaction (Decision 5).** `Stop`'s erase is `\r\x1b[2K`, leaving the cursor at the start of a cleared line; the following block's *leading* separator then writes its single `\n`, so exactly one blank line stands between `you ›` and the reply — identical whether the status line painted or the turn finished inside the pre-roll (in which case nothing was drawn and nothing is erased). The status line never perturbs the one-blank rule.

**Rejected.**
- **Keep the spinner glyph** — the operator asked for a glyph-free line; the elapsed counter alone signals liveness.
- **`Start`/`Stop` as `Renderer` methods** — puts an impure goroutine and a real clock inside the golden-tested renderer; ralph deliberately keeps the animator out of the renderer, and so does this — the renderer stays pure-ish and golden-driven while the one untestable seam is isolated.
- **1s repaint cadence** — the seconds would tick correctly but the line redraws coarsely and its width-change (`9s`→`10s`) reads as a visible jump; 80ms keeps it smooth for negligible cost (it only paints on a TTY, after a 2s pre-roll, and stops on first output).
- **No pre-roll (paint immediately)** — flashes a status line on every fast turn; the 2s quiet period suppresses it for turns that never needed it.
- **Write to stderr** (ralph's choice) — agentrepl's decorated transcript lives on stdout and the line is TTY-only and erased, so co-locating it on stdout keeps it within the renderer's stream without leaking into any capture (no capture happens on a TTY).

**Verification.**
- R-6DZ8-F5IK — `waitLine`/`formatElapsed` are pure and table-tested: `waitLine` yields `waiting for <model> (<elapsed>)`, gray-wrapped when `color` and plain otherwise (no erase prefix, no trailing newline); `formatElapsed` renders `5s`, `2m17s`, `1h2m3s` with higher-order zero units omitted and lower units shown once a higher unit appears.
- R-6F74-SX99 — the turn driver calls `waiter.Start(conv.Model)` before draining the stream and `waiter.Stop()` on first output and via `defer` (covering success, error, empty, and interrupt); raw mode and non-TTY runs use `nopWaiter` so no status line appears in their output (asserted with a recording fake waiter at the repl level).
- R-6HMX-KGQN — pre-roll and erase: a turn that produces output before the 2s pre-roll elapses paints nothing and erases nothing; a turn that outlives the pre-roll paints the line and, on `Stop`, erases it with `\r\x1b[2K` so the following block's leading separator leaves exactly one blank line (the one-blank spacing rule of Decision 5).

## Status

Decided: Decisions 1–13 — package layout & seams; provider & model catalog; config-key namespace & coercion; CLI flags; turn execution & rendering; REPL lifecycle, interrupt & log integrity; usage & cost reporting; session log & session-id; slash-command dispatch & command set; built-in tools; error handling & REPL resilience; the self-describing `--help` catalog; the wait status line.

The seams, public interfaces, naming, struct/type definitions, data model, and the testing approach are fully decided. The construction order that realizes this design lives in the plan.
