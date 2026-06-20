# Phase 8 — Composition root, interrupt & log integrity

*Realizes design Decision 6 (REPL lifecycle: interrupt & log integrity, the signal half), Decision 11 (resilience, the signal/startup-fatal half), Decision 7 (success-only accounting under interrupt), and completes Decision 1's composition root. Depends on Phase 7b.*

Flesh out `cmd/agentrepl/main.go` into the real composition root and wire SIGINT through the same graceful path. End state: `main` resolves `IO`/`Getenv`/`Now`/`LogDir` (`~/.agentkit` via `os.UserHomeDir`), computes `color = IsTTY && NO_COLOR==""`, sets up `ctx, stop := signal.NotifyContext(ctx, os.Interrupt)`, calls `repl.Run(ctx, …)`, and `os.Exit(code)`. The signal handler never calls `os.Exit` — it cancels the context; the driver observes `ctx.Err()`, renders a brief interrupt notice (not an error), renders the cumulative summary, and exits through the same deferred cleanup as `/exit`. Exit-code taxonomy realized end-to-end: 0 clean, 130 on SIGINT, 1 on startup failure. The no-torn-line guarantee is proven: SIGINT mid-stream yields a log that parses as valid JSONL end-to-end ending in a well-formed `turn_end` then `summary`. Startup-fatal messages go to stderr; in-loop errors to stdout. No `recover` anywhere.

**Done when:** these are covered by clearly-named tests and the suite is green:
- Decision 6 — R-LXGK-M9SO, R-LYOH-01JD, R-M149-RL0R.
- Decision 11 — R-H9XM-D78S, R-HB5I-QYZH, R-HCDF-4QQ6.
- Decision 7 — R-OPZQ-Y90U (the *interrupted-turn* case, completing the id begun in Phase 7).
