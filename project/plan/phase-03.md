# Phase 3 — Session log & session-id

*Realizes design Decision 8 (session log & session-id). Depends on Phase 1.*

Build `internal/session`: `ID(t)` minting the sub-second timestamp id and `Open(dir, now)` doing `MkdirAll` then opening `<dir>/<id>.jsonl` unbuffered (`O_CREATE|O_WRONLY|O_TRUNC`, `0o644`) and returning the file + id. Stdlib only; tests use a temp dir and a fixed `time.Time`. (R-8IUX/R-8K2T, which assert the *content* a completed run writes, are proven later at the repl level where a real `Conversation` writes to this file — see Phases 7–8; this phase proves id determinism, path targeting, dir creation, and unbuffered open.)

**Done when:** R-8GF4-LRYU and R-8HN0-ZJPJ are covered by clearly-named tests and the suite is green.
