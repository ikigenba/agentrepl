# Phase 24 — Adopt v0.3.0's new curated models

*Realizes design Decision — (structural; extends the Decision 2 catalog data, no new behavioral id). Depends on Phase 23.*

Add the model-id constants v0.3.0 introduced to `internal/catalog`'s `Default()`
curated lists, alongside the existing entries (Decision 2's `Models []string`
referencing agentkit's exported constants): the `anthropic` entry gains
`anthropic.ModelFable5` (`claude-fable-5`) and `anthropic.ModelSonnet5`
(`claude-sonnet-5`); the `openai` entry gains `openai.ModelGPT56Luna`
(`gpt-5.6-luna`), `openai.ModelGPT56Sol` (`gpt-5.6-sol`), and
`openai.ModelGPT56Terra` (`gpt-5.6-terra`). Fronting each list with the newest
generation matches the existing order (newest first). The `google` and `zai`
entries are unchanged — v0.3.0 added no models there.

No design change: the five ids flow through Decision 2's existing generic
machinery. The catalog stays a thin list referencing agentkit constants; the
D2 anti-drift tests (every curated id resolves to a `Pricing` — R-OWM8-I2NH —
and to a `ReasoningSpec` — R-FS10-LB3F) now exercise the five new ids, and the
D12 `--help` catalog (R-FUGT-CUKT lists every curated model in `Default()` order)
picks them up, so the `WriteHelp` golden under `testdata/` is regenerated to
include the new rows. agentrepl embeds no per-model reasoning knowledge; each new
model's reasoning descriptor is read from its provider's introspector as before.

**Done when:** `internal/catalog`'s `Default()` references all five constants —
`grep -REq 'ModelFable5|ModelSonnet5' internal/catalog/` and
`grep -REq 'ModelGPT56Luna|ModelGPT56Sol|ModelGPT56Terra' internal/catalog/`
both succeed — the regenerated `--help` golden under `internal/repl`'s `testdata/`
contains the model ids `claude-fable-5`, `claude-sonnet-5`, `gpt-5.6-luna`,
`gpt-5.6-sol`, and `gpt-5.6-terra`, and the suite is green (the D2 pricing and
reasoning anti-drift tests pass over the enlarged lists, all of
`go build ./...`, `go vet ./...`, `go test ./...` exit 0, and `gofmt -l .` prints
nothing).
