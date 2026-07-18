# Phase 28 — agentkit v0.4.0 pin + provider table (`internal/catalog` rewrite)

*Realizes design Decision 2 (provider table & agentkit-catalog sourcing). Depends on Phase 24.*

Advance the agentkit pin to the published **v0.4.0** tag (`go get github.com/ikigenba/agentkit@v0.4.0`) — the restructured surface: credential constructors, the advisory `agentkit/catalog` package, consumer-owned cost resolution, and the removal of the legacy model constants and reasoning inspectors. Because the removal breaks the old `internal/catalog` outright, this phase rewrites that package to Decision 2's shape in the same move: the five-provider table (`anthropic`, `google`, `openai`, `openrouter`, `zai` with their env keys and `Methods`), credential-constructor `New` funcs (including the `openai` subscription path via `subscription.Load` and `ErrAuthUnsupported`), the universal `Options.BaseURL`, and the `Models`/`Resolve` adapters over `agentkit/catalog` (embedding entries filtered; routes surfaced).

The other packages are adapted **mechanically, preserving their current observable behavior**: `internal/config` calls the new table shape (still constructing eagerly, passing `Options{BaseURL: …, Auth: AuthKey}` so existing env-key behavior is unchanged — laziness, auth keys, and resolution rules land in Phase 29), and `internal/repl`'s `WriteHelp` reads reasoning specs from `catalog.Models(...)` entries instead of the removed inspectors (goldens regenerated: model rows now in the agentkit catalog's sorted order, an `openrouter` section appears; the new help sections land in Phase 30). Tests asserting behaviors whose design ids were retired with this rewrite (the curated-list anti-drift, inspector, and warn-and-default carve-out tests) are removed with them.

This phase is the current instance of the version-pin rule in the design spine's Conventions: the exact version is a plan fact, named here in the Done-when and realized in `go.mod`.

**Done when:** `go.mod` pins the dependency at exactly `v0.4.0` — `grep -qE '^require github\.com/ikigenba/agentkit v0\.4\.0$' go.mod` succeeds and no other `github.com/ikigenba/agentkit v` line remains — `go mod verify` passes, the following are covered by clearly-named tests, and the suite is green:
- R-4IL7-JPW0 — `Default()` returns exactly the five providers with their contractual env keys; `Lookup` reports not-found for an unknown name.
- R-4JT3-XHMP — `Methods` is `[sub, key]` for `openai` and `[key]` for the other four.
- R-4L10-B9DE — `AuthKey` construction succeeds with a present key and returns wrapped `ErrMissingKey` naming the env var when empty.
- R-4M8W-P143 — `openai`+`AuthSub` builds from `subscription.Load(opts.AuthFile)`; a missing/malformed file errors naming the path; an unsupported `opts.Auth` returns wrapped `ErrAuthUnsupported` naming provider and method.
- R-4NGT-2SUS — `Models(name)` equals agentkit's `ListByProvider(name)` minus embedding entries, in sorted order, including openrouter's routed GLM entries.
- R-4OOP-GKLH — `Resolve` derives home providers, yields routed wire slugs, and reports `ok=false` for uncataloged selections without error.
- R-4PWL-UCC6 — cataloged chat models expose the `Entry.Pricing` the config layer installs; an uncataloged model yields no entry.
- R-4R4I-842V — a non-empty `Options.BaseURL` reaches the constructed provider of any of the five entries as its `WithBaseURL` option; empty leaves the default.
