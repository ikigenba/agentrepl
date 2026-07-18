# Phase 29 — Config resolution, auth keys, defaults & lazy construction (`internal/config`)

*Realizes design Decision 3 (config-key namespace, resolution & coercion — rewritten). Depends on Phase 28.*

Rewrite `internal/config` to Decision 3's current shape. `Target` gains the selection/auth state (`ProviderName`/`ProviderExplicit`/`ModelName`/`Auth`/`AuthFile`/`BaseURL`) and the cached lazily-built provider; `NewTarget` seeds the launch defaults (`provider=openai` non-explicit, `model=gpt-5.6-sol` resolved with pricing installed, `auth` unset, `auth_file` at the supplied default path) and constructs nothing. `Set` gains its `(notice, err)` signature. New keys `auth` and `auth_file`; `base_url` generalized to the active provider. Model sets follow the two-tier rule (bare cataloged → derive provider, wire model, pricing; bare uncataloged → hard error; explicit provider + uncataloged → free-flow with notice). Reasoning keys keep their key-directed coercion and add catalog gating for cataloged models (`akcatalog.Check` → `ErrBadValue` naming accepted values), pass-through for free-flow. The `default` sentinel restores seeded state for `provider`/`model`/`auth`/`auth_file` and zero/unset for everything else. `internal/repl` is adapted at its call sites (`Run` builds the target via `NewTarget`; `/set` renders a non-empty notice via `Renderer.Notice`; the turn driver obtains the provider from `Target.Provider()`) without yet changing the command set or help — those are Phases 30/31.

**Done when:** the following are covered by clearly-named tests and the suite is green:
- R-4X80-4YSC — `NewTarget` seeds the defaults exactly, with pricing installed and no provider constructed.
- R-4YFW-IQJ1 — the `default` sentinel restores seeded state for `provider`/`model`/`auth`/`auth_file` and zero/unset (rendered `default`) for every other key.
- R-4ZNS-WI9Q — an unparseable value returns wrapped `ErrBadValue` naming key and reason, mutating nothing, across the key table.
- R-50VP-AA0F — bare cataloged model derives its home provider and installs wire model + pricing; bare uncataloged model errors with wrapped `ErrUnknownModel`, mutating nothing.
- R-523L-O1R4 — explicit-provider resolution: routed models get their wire slug and pricing; uncataloged models are accepted verbatim with nil pricing and a free-flow notice; display shows the operator-entered name.
- R-4TKA-ZNK9 — `auth` accepts exactly `key`/`sub`, rejects a method the current provider lacks at set time, and unset resolves to the provider default at build time.
- R-4W03-R71N — `auth_file` feeds the sub build; any of `provider`/`auth`/`auth_file`/`base_url` invalidates the cached provider (order-independent, `default` clears the override); an unchanged selection reuses the cache.
- R-54JE-FL8I — `Target.Provider()` builds lazily via the table with the current `Options`, assigns `Conv.Provider`, and returns construction errors unswallowed.
- R-53BI-1THT — reasoning gating: cataloged-model rejection with accepted values named; acceptance lands on `Conv.Gen.Reasoning`; free-flow passes the same value through unchecked.
