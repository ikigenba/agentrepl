# Phase 23 — Bump the agentkit dependency pin to v0.3.0

*Realizes design Decision — (structural; no behavioral id). Depends on Phase 22.*

Advance the agentkit dependency from its current pin to the published **v0.3.0**
tag. `go.mod`'s `require github.com/ikigenba/agentkit` becomes exactly `v0.3.0`
and `go.sum` is updated to match (`go get github.com/ikigenba/agentkit@v0.3.0`).
No agentrepl source changes: v0.3.0 is API-compatible with the surface agentrepl
consumes — the native-reasoning introspection (`ReasoningSpec`/`ReasoningInspector`/
`ReasoningValue` and the per-package `Reasoning` values), `GenSettings.Reasoning`,
and the `Warning`/`Event`/`Usage`/`Cost`/`Conversation` types are all unchanged —
so the module resolves and the existing suite passes unmodified. This is the
atomic pin move that makes v0.3.0's newer surface — notably the additional
curated model constants Phase 24 adopts — resolvable.

This phase is the current instance of the version-pin rule in the design spine's
Conventions: the exact version is a plan fact, named here in the Done-when and
realized in `go.mod`.

**Done when:** `go.mod` pins the dependency at exactly `v0.3.0` —
`grep -qE '^require github\.com/ikigenba/agentkit v0\.3\.0$' go.mod` succeeds and
no other `github.com/ikigenba/agentkit v` line remains — `go mod verify` passes,
and the suite is green (`go build ./...`, `go vet ./...`, `go test ./...` exit 0,
and `gofmt -l .` prints nothing).
