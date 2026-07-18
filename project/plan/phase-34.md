# Phase 34 — Session logs move to `~/.agentrepl/logs/`

*Realizes design Decision 8 (session log — the log-home slice: R-GUN2-2ULO). Depends on Phase 33.*

The session log's contractual home becomes `~/.agentrepl/logs/<session-id>.jsonl`, consolidating agentrepl's on-disk footprint under `~/.agentrepl/` beside the auth file. `internal/session` gains `DefaultDir(home string) string` returning `<home>/.agentrepl/logs`; the composition root (`cmd/agentrepl/main.go`) resolves `LogDir` through it instead of joining `.agentkit` inline. Everything else about the log is untouched — id format, unbuffered open, `MkdirAll` (which already handles the nested path), always-on writing. No migration: existing `~/.agentkit/*.jsonl` files are left where they are and nothing references that directory afterward. The repo-root `CLAUDE.md` layout line describing `internal/session` is updated to name the new path.

**Done when:** R-GUN2-2ULO is covered by a clearly-named tagged test in `internal/session`; `grep -rn "agentkit\"" --include='*.go' cmd/ internal/session/` returns no matches and `grep -rn "\.agentkit" --include='*.go' .` returns no matches; `grep -n "\.agentkit" CLAUDE.md` returns no matches; and the green bar holds (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0, `gofmt -l .` prints nothing).
