# Phase 31 — `/login`, `/providers` auth status, lazy-failure directive & cost-unknown relay

*Realizes design Decisions 14 (subscription auth, `/login`, lazy credential failure), 9 (slice — the two new command ids), and 7 (slice — the cost-unknown relay id). Depends on Phase 30.*

Finish the REPL surface of the auth work. `repl.Deps` gains `AuthFile` (the composition root resolves `<home>/.agentrepl/auth.json`) and the injected `Login` func (bound to `subscription.Login` in `cmd/agentrepl/main.go`; faked in tests). The command table gains `/login` (runs the flow against the current `auth_file`, renders notice/error, invalidates the cached provider) and the rewritten `/providers` (per-method auth status, no model lists). The turn driver replaces the old provider/model pre-check with `target.Provider()` and renders Decision 14's directive message on construction failure — subscription failures naming the auth-file path, `/login`, and the `auth=key` alternative; key failures naming the env var — without calling `Send`. The `WarnCostUnknown` relay rides the existing warning path; its cost-side behavior is asserted here.

**Done when:** the following are covered by clearly-named tests and the suite is green:
- R-55RA-TCZ7 — `/providers` lists all five providers with per-method auth status (env var present/missing; openai's `sub` line with the current auth-file path present/missing) and prints no model lists.
- R-56Z7-74PW — `/login` dispatches from the command table, appears in `/help`, and fails non-fatally.
- R-5EAL-HR62 — `/login` invokes the injected seam with the current `auth_file` and REPL In/Out, renders the outcome, and invalidates the cached provider on success.
- R-5FIH-VIWR — a failed lazy build renders the method-keyed directive message, does not call `Send`, and the loop continues.
- R-5GQE-9ANG — with no credentials or auth file anywhere, bare startup reaches the prompt and slash-commands answer normally; nothing credential-touching runs before the first turn attempt.
- R-5HYA-N2E5 — after a (faked) login writes a valid auth file at the configured path, the next turn's lazy build succeeds in the same session, no restart.
- R-5J67-0U4U — a turn on an unpriced free-flow model relays `WarnCostUnknown` via `Renderer.Warning` with zero turn cost; a cataloged model's turn prices from the installed `Entry.Pricing` without that warning.
