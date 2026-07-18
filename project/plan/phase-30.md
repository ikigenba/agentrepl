# Phase 30 — `--help`: defaults, auth lines, routes & the two-tier footer

*Realizes design Decision 12 (rewritten: the catalog-sourced help with its new sections; slice — the four new ids; the carried-forward rendering ids remain covered by their frozen phases). Depends on Phase 29.*

Extend `repl.WriteHelp` to Decision 12's full current shape. New over Phase 28's mechanical port: the `defaults:` section (`provider=openai model=gpt-5.6-sol auth=sub`); the per-method auth lines in `providers:` (each provider's `auth=key  (<ENV_VAR>)`, plus openai's `auth=sub` line naming the default auth-file path and `/login`); the routed-entry rendering under `openrouter` (`model -> wire-slug` ahead of the reasoning clause); and the closing two-tier footer (bare model must be cataloged; anything else needs an explicit provider and passes through unvalidated — no pricing, unchecked reasoning). The established reasoning-clause rendering (native key lead, starred defaults, sentinel handling, term dropping, en-dash ranges) is unchanged and stays pinned by the regenerated golden.

**Done when:** the following are covered by clearly-named tests (the regenerated `WriteHelp` golden plus focused assertions) and the suite is green:
- R-5873-KWGL — the models section renders exactly `catalog.Models(provider)` per provider in table order (sorted, embedding-free), with openrouter's routed entries as `model -> wire-slug` rows.
- R-5AMW-CFXZ — the `defaults:` section states `provider=openai`, `model=gpt-5.6-sol`, `auth=sub`.
- R-5BUS-Q7OO — the providers section lists all five with per-method auth lines, openai carrying the `auth=sub` line with the default auth-file path and `/login` pointer.
- R-5D2P-3ZFD — the catalog closes with the two-tier footer stating the bare-cataloged rule and the explicit-provider pass-through consequences.
