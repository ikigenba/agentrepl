# Phase 35 — agentkit v0.6.0 pin + OpenRouter-native models in `--help`

*Realizes design Decision — (structural; extends Decision 2's catalog data and Decision 12's help rendering, no new behavioral id). Depends on Phase 34.*

Advance the agentkit dependency from its current pin to the published **v0.6.0**
tag (`go get github.com/ikigenba/agentkit@v0.6.0`; `go.mod`'s
`require github.com/ikigenba/agentkit` becomes exactly `v0.6.0`, `go.sum`
updated to match). v0.6.0's only change on the surface agentrepl consumes is
catalog data: nine new OpenRouter-native chat-model entries (`Provider:
"openrouter"` directly, no route from another provider) —
`deepseek-v4-flash`, `deepseek-v4-pro`, `grok-4.20`, `grok-4.20-multi-agent`,
`grok-4.3`, `grok-4.5`, `kimi-k2.6`, `kimi-k2.7-code`, `kimi-k3` — plus
`glm-5.1`'s `Reasoning` spec changing shape from an enum (`effort={high|*max}`)
to a toggle. The `Entry`, `ReasoningSpec`, and `catalog.Provider`/`Models`/
`Resolve` shapes are unchanged, so no source changes: `internal/catalog`
already passes `Models`/`Resolve` straight through to `agentkit/catalog`
(Decision 2), and D12's `R-5873-KWGL` ("the models section renders, for each
provider in table order, exactly `catalog.Models(provider)`") is already
generic over whatever that call returns — the new entries surface with no code
change.

The one artifact that must change is the `--help` golden,
`internal/repl/testdata/help_reasoning.golden`: its `openrouter` model section
is golden-pinned (R-5873-KWGL, R-FVOP-QMBI), so it goes stale the moment the
pin moves. Regenerate it so the `openrouter` section lists all thirteen
entries in `catalog.Models("openrouter")`'s sorted order — the four existing
zai-routed GLM rows (`model -> wire-slug reasoning-clause`, `glm-5.1`'s clause
now toggle-shaped) interleaved alphabetically with the nine new native rows
(no `->` arrow, since their home provider is already `openrouter` — same
render path as the `zai` section's native rows). Let the existing golden-test
tooling produce the exact bytes; do not hand-author the reasoning clauses.

This phase is the current instance of the version-pin rule in the design
spine's Conventions: the exact version is a plan fact, named here in the
Done-when and realized in `go.mod`.

**Done when:** `go.mod` pins the dependency at exactly `v0.6.0` —
`grep -qE '^require github\.com/ikigenba/agentkit v0\.6\.0$' go.mod` succeeds
and no other `github.com/ikigenba/agentkit v` line remains — `go mod verify`
passes; `internal/repl/testdata/help_reasoning.golden`'s `openrouter` section
contains all of `deepseek-v4-flash`, `deepseek-v4-pro`, `grok-4.20`,
`grok-4.20-multi-agent`, `grok-4.3`, `grok-4.5`, `kimi-k2.6`,
`kimi-k2.7-code`, `kimi-k3` (`grep -qE '<model>' internal/repl/testdata/help_reasoning.golden`
for each) alongside the unchanged four GLM rows; and the suite is green
(`go build ./...`, `go vet ./...`, `go test ./...` exit 0, and `gofmt -l .`
prints nothing).
