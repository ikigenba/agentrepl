# Phase 22 — Adopt agentkit message-granular delivery

*Realizes design Decision 5 (Turn execution, the Renderer, and color). Depends on
Phase 06 (renderer) and Phase 19 (decorated palette & spacing).*

agentkit's message-granular delivery (its Phase 24) removed the consumer-facing
`TextDelta`/`ReasoningDelta` events and dropped the leading events argument from
`agentkit.NewRoundTrip`; the stream now yields only `MessageDone`, `ToolUse`, and
`ToolResult`, and `MessageDone.Message` carries the assembled reasoning/text/
tool-use blocks. agentrepl does not currently build against this surface.

Bring agentrepl onto the new surface — the only package with behavior to change
is `internal/render`:

- The decorated renderer renders a `MessageDone` by walking `Message.Blocks` in
  order — a non-empty `ReasoningBlock.Summary` as a dim `reasoning ›` block, a
  non-empty `TextBlock` as an `assistant ›` block (bold light-blue label,
  light-blue body), and a `ToolUseBlock` as a gray `tool call ›` line —
  interleaved exactly as emitted, skipping empty blocks; the standalone `ToolUse`
  event becomes a no-op (the call is already rendered from the block, never
  doubled); the `ToolResult` event still renders the tool-result line; the
  `TextDelta`/`ReasoningDelta` cases and the `streaming`/`finishStream`
  machinery are removed.
- The raw renderer keeps its behavior (already message-granular) and still emits
  the `ToolUse` event as its own JSON entry.
- The decorated goldens under `internal/render/testdata` are regenerated to the
  new whole-block output; render tests that asserted incremental delta streaming
  are replaced by tests for the new behavior.
- Mechanical: every `agentkit.NewRoundTrip(nil, …)` test-fake call site (in
  `internal/render`, `internal/tools`, `internal/repl`) drops the leading `nil`
  events argument so the module compiles and `go vet ./...` is clean.

The agentkit dependency is consumed via the `replace => ../agentkit` directive,
so no `go.mod` version bump is required.

**Done when:** R-CCHP-S6AJ, R-CDPM-5Y18, R-LL9K-SKDQ, R-OBNM-N6XX, and
R-LOX9-XVLT (Decision 5, render side) are covered by clearly-named tests
(goldens where the format is pinned), any other Decision-5 goldens that shift
with the new output (e.g. R-Q52T-PXCR vertical spacing) are regenerated and
still asserted, and the suite is green (`go build ./...`, `go vet ./...`,
`go test ./...` exit 0 and `gofmt -l .` prints nothing).
