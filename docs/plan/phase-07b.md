# Phase 7b — REPL turn driver, usage triggers & graceful exit

*Realizes design Decision 5 (turn driver, the driver half), Decision 7 (usage sourcing & triggers, the in-loop half), Decision 6 (graceful-exit cleanup, the non-signal half), Decision 9 (the turn-classification id), Decision 8 (session-log content), and Decision 11 (turn-error resilience, the in-loop half). Depends on Phase 7a.*

Fill in the turn-handler seam left by Phase 7a and complete the graceful-exit sequence, driving a **real `*agentkit.Conversation` with a fake `agentkit.Provider`** (per Decision 1's seam) in tests with scripted stdin. End state: a non-`/`, non-empty line drives a turn through the driver (pre-check → `Prompt(text)` → `conv.Send(ctx, text)` → range `stream.Events()` calling `Event` for each → after draining, on non-interrupt `stream.Err()` call `Error` and continue, else `Usage(stream.Usage(), stream.Cost(), conv.TotalCost())`); `/usage` renders the cumulative `Summary` from `conv.TotalUsage()`/`conv.TotalCost()`; and graceful `/exit`/`/quit`/EOF flow through deferred cleanup (`Summary` to the operator → `conv.Close()` writes agentkit's cumulative summary record → close log → exit 0). A completed run's records land in the session file, identical whether rendering was decorated or raw. In-loop turn errors render to stdout; the loop survives every turn error. The `ctx` is still a parameter (the cancellable one arrives in Phase 8).

**Done when:** these are covered by clearly-named tests (repl-level, real `Conversation` + fake `Provider`, captured stdout **and** JSONL log buffer) and the suite is green:
- Decision 9 (turn) — R-BJ8G-7A8M.
- Decision 5 (driver) — R-LSKZ-36TW.
- Decision 7 (in-loop) — R-OPZQ-Y90U (the *errored-turn* case), R-OSFJ-PSI8, R-OUVC-HBZM (the `/exit`/`/quit`/EOF cases).
- Decision 6 (graceful) — R-LW8O-8I1Z.
- Decision 8 (content) — R-8IUX-DBG8, R-8K2T-R36X.
- Decision 11 (turn) — R-H7HT-LNRE.
