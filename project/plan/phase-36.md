# Phase 36 — agentkit v0.7.0 pin + adopt the toolkit, delete `internal/tools`

*Realizes design Decision 10 (standard tools via agentkit's toolkit). Depends on Phase 35.*

Advance the agentkit dependency from its current pin to the published
**v0.7.0** tag (`go get github.com/ikigenba/agentkit@v0.7.0`; `go.mod`'s
`require github.com/ikigenba/agentkit` becomes exactly `v0.7.0`, `go.sum`
updated to match), then replace agentrepl's homegrown tools with agentkit's
new `toolkit` subpackage per the rewritten Decision 10:

- `cmd/agentrepl/main.go` resolves the process working directory once with
  `os.Getwd()` (a failure is a startup failure, exit 1) and passes it down.
- The REPL's conversation wiring (`internal/repl/repl.go`) sets
  `Tools: toolkit.All(cwd)` in place of the former `tools.All()`.
- The `internal/tools` package is **deleted entirely** (its `doc.go`,
  `tools.go`, `tools_test.go`); its six retired ids (R-NHBW-446N,
  R-NIJS-HVXC, R-NKZL-9FEQ, R-NM7H-N75F, R-NNFE-0YW4, R-NONA-EQMT) leave the
  suite with it.
- The repo `CLAUDE.md` Layout section's `internal/tools/` line is removed
  (the toolkit is agentkit's, not a package here).
- New wiring tests realize the two current ids — asserting the conversation's
  tool set is exactly toolkit's six, and that a wired file tool resolves
  relative paths against the process cwd (temp-dir working directory).

This phase is the current instance of the version-pin rule in the design
spine's Conventions: the exact version is a plan fact, named here in the
Done-when and realized in `go.mod`.

**Done when:** `go.mod` pins the dependency at exactly `v0.7.0` —
`grep -qE '^require github\.com/ikigenba/agentkit v0\.7\.0$' go.mod` succeeds
and no other `github.com/ikigenba/agentkit v` line remains — `go mod verify`
passes; the `internal/tools` directory does not exist (`test ! -e
internal/tools`); `grep -rn 'internal/tools' --include='*.go' .` prints
nothing; `grep -n 'internal/tools' CLAUDE.md` prints nothing; R-W3MV-WRJ7 and
R-W4US-AJ9W are each covered by a clearly-named test; and the suite is green
(`go build ./...`, `go vet ./...`, `go test ./...` exit 0, and `gofmt -l .`
prints nothing).
