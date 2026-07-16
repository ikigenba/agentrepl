# Phase 1 — Module bootstrap & package skeleton

*Realizes design Decision 1 (package layout & seams). Depends on nothing.*

Stand up the buildable, testable skeleton from Decision 1's layout. The module `github.com/ikigenba/agentrepl` exists (`go.mod`, Go 1.26) with a `replace github.com/ikigenba/agentkit => ` directive pointing at the local agentkit checkout (the dependency is not published to the module cache; this is the build-level mechanism for resolving it). The directory tree exists with package declarations and the seam type definitions that have no behavior yet: `cmd/agentrepl/main.go` (package `main`, a composition-root stub that compiles and can be wired later), and empty-but-declared `internal/{repl,config,render,catalog,tools,session}` packages. The shared seam types from Decision 1 are declared where they belong (`IO` struct; the `Getenv`/`Now` func-type seams; the `Renderer` interface signature in `render`). No business logic. A single trivial test (e.g. a package-compiles/skeleton sanity test) exists so the suite is genuinely exercised and green.

**Done when:** `go build ./...`, `go vet ./...`, `go test ./...` exit 0 and `gofmt -l .` is empty. Decision 1 mints no requirement ids (it is a pure structural decision); this phase's bar is the green suite over the skeleton, and it is the substrate every later phase's ids are proven on.
