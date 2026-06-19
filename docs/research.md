# agentrepl — Research

**Status: non-contractual.** This document informs the *author* of `docs/design.md`; nothing downstream (the autonomous build) reads it. It records options, prior art, constraints, and recommendations as of **2026-06-18**. Design remains the single authority for *how*; where this doc recommends a mechanism, design may adopt, refine, or reject it. Edit this doc in place as the product evolves — never append a log.

agentrepl is a **thin consumer** of AgentKit: an interactive Go CLI REPL that drives the library directly so a developer can exercise every AgentKit feature by hand (`docs/product.md` is the authority on *why*). Because it is thin, almost none of the hard cross-provider research lives here — it lives in `../agentkit/docs/research.md` (provider APIs, the canonical message model, usage/cost, MCP, and the reasoning vocabulary). **This doc covers only what is specific to agentrepl**: how the two control surfaces (`--help` at launch, `key=value` config + slash-commands at runtime) **consume** AgentKit's surface, with the focus area being the **native-first reasoning** consumption that the 2026-06-18 product change introduced.

The fixed target (`docs/product.md`): module `github.com/ikigenba/agentrepl`, built on `github.com/ikigenba/agentkit`; a text-only REPL; four providers (Anthropic, Google, OpenAI, Z.ai) from AgentKit's curated model set; flags set launch state, slash-commands inspect/mutate runtime state; a single generic `key=value` config passthrough mirroring AgentKit's config structure; four built-in tools (`bash`/`read`/`write`/`edit`); streaming with visible reasoning; per-turn + cumulative cost; decorated vs raw rendering; an always-on `~/.agentkit/<session-id>.jsonl` log; credentials only from `PROVIDER_API_KEY` env vars.

---

## 1. The central finding

**agentrepl's job on reasoning is render-and-relay, not decide.** The 2026-06-18 product change moved AgentKit from a single universal `ReasoningEffort` enum to **native-per-model** reasoning: each model's reasoning is expressed in that model's own native term and native values (a discrete level set, OR an integer token-budget range, OR a bare on/off), AgentKit exposes a **per-model introspection API** (term + accepted values/range + default + can-disable), and a non-native input is **warned-and-defaulted**, never silently misapplied. agentrepl must (a) **render** that introspection in `--help` and (b) **accept** reasoning as a native `key=value` whose validity is judged by AgentKit, **reimplementing none of the provider knowledge**. The entire correctness of agentrepl's reasoning surface therefore reduces to one dependency: **AgentKit must expose the introspection + validation API** (§4). Until it does, agentrepl is blocked — today's pinned `agentkit v0.1.0` exposes only the old universal enum (§3.4).

The other finding is smaller and mechanical: **`--help` is currently unimplemented** as a catalog (it is stdlib `flag`'s bare usage today), and the reasoning config key currently hardcodes the universal 7-word vocabulary and **hard-errors** on a bad value — both directly contradict the new product promises and must change (§3).

---

## 2. The two control surfaces (what the product promises)

| Surface | When | Reasoning behavior promised |
|---|---|---|
| **`--help`** | launch, no session | Static catalog: providers in one list; models grouped by provider in another; **for each model its native reasoning term + accepted values (discrete levels, or its valid token-budget range)**, as reported by AgentKit. **No credential/env checks** — a catalog of what can be *asked for*. |
| **`-c key=value` flag** | launch | Sets initial config, including reasoning, in the model's native vocabulary. Non-native reasoning input **warns + falls back to the model default** (carve-out from the general "bad key/value → error" rule). |
| **`/set key value` + `/get` + `/dump`** | runtime | Same dotted key namespace as the flag; mutate/inspect mid-session. Reasoning re-validated against the **currently selected** model (which can change between turns). |
| **`/providers`** | runtime | Shows live key *status* (present/absent per provider) — the runtime complement to `--help`'s static catalog. Unchanged by this work. |

**Key boundary:** `--help` is **static and credential-blind** (it must run with no keys set and reflect no environment); `/providers` is the **live** view. The reasoning catalog belongs to `--help`, sourced purely from AgentKit introspection.

---

## 3. Current codebase state and required changes

Grounded by reading the agentrepl tree (working dir `/mnt/projects/agent-repl`; mirror at `/home/mgreenly/projects/agent-repl`). File:line references are to the current source.

### 3.1 Catalog — `internal/catalog/catalog.go`
- `Provider{Name, EnvKey, Models []string, New ProviderFunc}` (≈ lines 20-25). Model lists are **hardcoded in agentrepl** in `Default()` (≈ 33-92), assembled from AgentKit's exported model-name **string constants** (e.g. `anthropic.ModelOpus48`, `openai.ModelGPT55Pro`). So agentrepl owns the provider→model grouping; AgentKit supplies only the id strings. **`Models` is `[]string` — it carries no reasoning metadata at all.**
- Helpers: `Lookup` (≈94), `HasModel` (≈103), `Build` (≈112, does the env-key/API-key presence check).
- **Required change:** `--help` must print each model's native reasoning term + accepted values/range. Options: (a) keep `Models []string` and have the help renderer query AgentKit introspection per model id at print time (preferred — keeps the catalog a thin list, zero reasoning knowledge in agentrepl); or (b) widen the catalog to carry a per-model reasoning descriptor sourced from AgentKit. Either way agentrepl reimplements **no** native-term/level knowledge.

### 3.2 Config — `internal/config/config.go`
- Reasoning lives under the single dotted key **`gen.reasoning`** (≈ line 111) — a one-size-fits-all field, **not per-model**.
- `set` for that key (≈ 112-119) calls `parseReasoning`; on failure it **returns `ErrBadValue`** (≈ 115). **Today invalid reasoning is a hard error — the exact opposite of the new "warn + fall back to default" carve-out.**
- `parseReasoning` (≈ 312-331) / `formatReasoning` (≈ 333-350) hardcode the **universal** vocabulary `default, off, minimal, low, medium, high, max`, mapping to `agentkit.EffortDefault … EffortMax`. `reset` (≈ 126-129) sets `agentkit.EffortDefault`. Reasoning is stored at `t.Conv.Gen.Reasoning` (type `agentkit.ReasoningEffort`); the selected model is `t.Conv.Model`.
- Generic field helpers (`floatField`/`intField`/…, ≈ 220-310) and the `fields` map drive `Set/Get/Dump/Keys` (≈ 165-210); `ParsePair` (≈212) splits `key=value`.
- **Required changes:**
  1. Replace the hardcoded universal `parseReasoning`/`formatReasoning` with logic that validates the raw value against the **active model's** native spec obtained from AgentKit introspection — accepting either a native level string or a native integer budget.
  2. Apply the **carve-out** in the reasoning `set` path: on non-native/invalid/out-of-range input, **relay AgentKit's warning and fall back to the model's reported default** instead of returning `ErrBadValue`. This is the explicit exception to the general "invalid key/value → error" rule that `Set` (≈165-183) otherwise enforces.
  3. Change the stored type from the `ReasoningEffort` enum to AgentKit's new native carrier (its tagged `ReasoningValue`, holding a native level / native budget / disabled / unset). `/get` and `/dump` must render the model's native current value and default.
  4. Decide the dotted key. Recommendation: **keep `gen.reasoning`** (it still mirrors AgentKit's config structure), but its accepted *values* become per-model native (a level, or an int in the valid budget range, or a disable token) rather than the fixed 7-word set. Mirroring AgentKit's own config key keeps the "keys mirror AgentKit" promise intact.

### 3.3 Help / args & commands — `internal/repl/args.go`, `internal/repl/commands.go`
- `ParseArgs` (≈9-31) registers only `-c` (repeatable `key=value`) and `-raw`. **`--help` is just stdlib `flag`'s default usage** — there is **no** static provider/model/reasoning catalog today. That product promise is **currently unmet**.
- `/providers` (≈103-116) iterates the catalog and does a **live env-key check** (present/absent) — the runtime live-status command. `/set` (≈36-46) → `config.Set`; `/get` (≈47-59) → `config.Get`; `/dump` (≈60-69) → `config.Dump`; in-REPL `/help` (≈117-128) lists commands + config keys.
- **Required change:** intercept `--help`/`-h` (in `args.go` and/or `cmd/agentrepl/main.go`) and emit a **custom catalog** — providers list, models-grouped-by-provider, and per-model native reasoning term + values/range from AgentKit introspection — with **no env/credential checks**. `/providers` keeps the live view. The runtime `/dump`/`/get` for reasoning should render the native value; optionally the per-model accepted values could also be surfaced at runtime, but the static catalog is the `--help` job.

### 3.4 The AgentKit dependency today — `go.mod`
- **`github.com/ikigenba/agentkit v0.1.0`** (module cache at `~/.local/go/pkg/mod/github.com/ikigenba/agentkit@v0.1.0`).
- Today's reasoning model (`gen.go`): the **universal ordinal enum** `ReasoningEffort int` with `EffortDefault/EffortOff/EffortMinimal/EffortLow/EffortMedium/EffortHigh/EffortMax`; `GenSettings.Reasoning` is that enum. A `Warning{Setting, Detail}` type already exists. Each provider maps the enum to native terms in **unexported** functions (e.g. `anthropic.applyReasoning`/`anthropicEffort`, `openai.openAIReasoningEffort`). Models are exposed only as **string constants** — there is **no exported `Models()`, no `ReasoningSpec`, no `ReasoningFor(model)`, no levels/range accessor**.
- **Consequence:** agentrepl literally cannot source native reasoning metadata from `v0.1.0`. **This work is blocked on an AgentKit release that exports the introspection + native-value API** (§4). The future shape is being designed in `/mnt/projects/agentkit` (the design target), but agentrepl builds against the published dependency, so the version bump is a hard prerequisite — see §5.

---

## 4. Required AgentKit surface (the consumer's contract on the library)

agentrepl needs AgentKit to expose, so agentrepl hardcodes **zero** provider knowledge. These are *requirements on AgentKit*, mapped to the shapes AgentKit's own research (`../agentkit/docs/research.md` §7.1) already recommends — so the two docs agree:

1. **Enumerate models (per provider, or model→spec map).** So `--help` groups models without agentrepl maintaining `Models []string` by hand. AgentKit's `SupportedReasoning() map[string]ReasoningSpec` plus the existing per-provider model-id constants cover this. (agentrepl may keep its own provider→model grouping for display order, but the *reasoning* facts must come from AgentKit.)
2. **Per-model reasoning spec** — AgentKit's `ReasoningSpec{ Term, Kind, Levels, Min, Max, Sentinels, Default, CanDisable }`:
   - `Term` — the native label to print and to use as the value's vocabulary ("effort" / "thinking level" / "thinking budget").
   - `Kind` — discrete **enum** (render `Levels`), integer **range** (render `[Min,Max]` + sentinel meanings like `0`=off, `-1`=dynamic), or **toggle** (on/off only, e.g. GLM 4.6/4.7).
   - `Default` — used by `/get`/`/dump` and shown in `--help`; also what the warn-fallback path applies.
   - `CanDisable` — whether "off"/disable is offered.
3. **A native value carrier** — AgentKit's tagged `ReasoningValue` (`Level(string)` / `Budget(int)` / `DisableReasoning()` / unset) that agentrepl builds directly from the raw `key=value` string per the model's `Kind`, and stores on `GenSettings.Reasoning`. No translation in agentrepl.
4. **Validation + warning relay.** Either (a) an advisory `spec.Validate(value) error` agentrepl can call at `/set` time to pre-judge native-ness, and/or (b) AgentKit's request-build-time validation that emits a `Warning` and falls back to `spec.Default`. Since the selected model can change between turns, **build-time validation in AgentKit is the authoritative enforcement** (a value valid for model A may be invalid for model B); agentrepl's `/set`-time check is a convenience that mirrors it. agentrepl **relays** AgentKit's `Warning{Setting,Detail,…}` to the operator rather than minting its own message.

**Single source of truth principle:** the same `ReasoningSpec` agentrepl renders in `--help` is the spec AgentKit validates against — display and accept come from one place, so they cannot drift.

---

## 5. Constraints, risks, and recommendations carried into design

1. **Hard prerequisite — AgentKit version bump.** agentrepl's reasoning surface cannot be built against `agentkit v0.1.0` (universal enum only). Design must target the AgentKit version that exports `ReasoningSpec`/`ReasoningInspector`/`ReasoningValue` and bump `go.mod` accordingly. This is a **breaking** dependency move (the `ReasoningEffort` enum and `EffortDefault…` constants agentrepl uses in `config.go` are removed). Flag it as the gating risk for plan-mode sequencing.
2. **Carve-out is real and narrow.** Reasoning is the **only** config key whose bad input warns-and-defaults instead of erroring. Design must keep the general `Set` error path intact for every other key and special-case only the reasoning key — and the carve-out is already documented in `docs/product.md` (success-criteria line and a What-we-promise bullet). Don't generalize it.
3. **Render all three value shapes.** `--help` and `/dump` must handle: a discrete level list (most models), an integer range with sentinels (Gemini 2.5 family: Flash `0–24576` with `0`=off, Pro `128–32768` no-disable), and on/off-only (GLM 4.6/4.7). A single rendering routine keyed on `spec.Kind` covers all three — don't hardcode per-provider formats. (The per-model specifics live in `../agentkit/docs/research.md` §7.1; agentrepl reads them at runtime, never bakes them in.)
4. **Native values are heterogeneous strings on the CLI.** A `key=value` flag carries text; agentrepl parses `"high"` → `Level("high")`, `"8000"` → `Budget(8000)`, `"off"`/`"disabled"` → `DisableReasoning()`, choosing the constructor by the model's `Kind`. Edge: when no model is selected yet at parse time, defer reasoning validation until the model is known (build-time validation in AgentKit is the backstop, so an unvalidatable launch value still can't break the turn — it warns + defaults).
5. **Keep agentrepl thin.** Every temptation to "help" by mapping or normalizing reasoning values is a regression toward the rejected universal-enum design. agentrepl's correctness comes from relaying AgentKit's spec, value carrier, validation, and warnings verbatim. The exemplary-Go-consumer bar (product §Purpose) is best served by the smallest possible reasoning code that simply renders introspection and forwards a native value.
6. **`--help` must stay credential-blind.** Sourcing the catalog from AgentKit introspection (pure model metadata) — not from constructed provider clients — keeps `--help` runnable with no keys and reflecting no environment, as promised. Don't reach for `catalog.Build`/API-key checks in the help path.

---

## 6. Open items / to verify in design

- **Exact AgentKit API names.** The shapes in §4 mirror AgentKit's research recommendation but are not yet a published API; design must pin the real exported names/signatures against the AgentKit design once it lands, and the agentrepl `go.mod` version that carries them.
- **Runtime display of accepted values.** Whether `/get gen.reasoning` (or a `/reasoning` helper) should also print the current model's accepted values/range at runtime, or whether that stays a `--help`-only concern. Lightweight either way; product only mandates it for `--help`.
- **`--help` ordering & formatting** (providers list, model grouping, how a range vs a level list is shown per model) is a design/altitude call, not researched here.
