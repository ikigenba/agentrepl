# Phase 6 — Renderer (decorated & raw) + usage/cost formatting

*Realizes design Decision 5 (turn execution, the Renderer, and color) — presentation half — and Decision 7 (usage & cost reporting) — format half. Depends on Phase 1.*

Build `internal/render`: the `Renderer` interface and its two implementations, `NewDecorated(out, color)` and `NewRaw(out)`, with output pinned by golden files under `testdata/` (with a `-update` flag). Decorated gives each kind a distinct treatment, streams `TextDelta`/`ReasoningDelta` incrementally, emits ANSI only when `color` is true, and renders the per-turn usage/cost line and cumulative summary in the exact bucket layout of Decision 7. Raw emits one undecorated JSON line per `Prompt`/`MessageDone`/`ToolUse`/`ToolResult` plus usage/summary as JSON, skips deltas, never emits ANSI. Depends only on agentkit's `Event`/`Usage`/`Cost` value types; tests feed synthesized events and usage/cost values directly.

This phase owns the **format/presentation** ids of D5 and D7. The driver-side ids — D5's R-LSKZ (driver calls `Error` not `Usage`) and D7's sourcing/trigger ids (R-OPZQ, R-OSFJ, R-OUVC) — are realized in Phases 7–8, where the turn driver supplies the numbers and the triggers fire.

**Done when:** R-LL9K-SKDQ, R-LMHH-6C4F, R-LNPD-K3V4, R-LOX9-XVLT, R-LRD2-PF37 (Decision 5, render side) and R-ONJY-6PJG, R-OORU-KHA5, R-OR7N-C0RJ, R-OW38-V3QB (Decision 7, render side) are covered by clearly-named tests (goldens where the format is pinned) and the suite is green.
