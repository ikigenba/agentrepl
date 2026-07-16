# Phase 19 — Decorated palette & vertical spacing

*Realizes design Decision 5 (amended: the decorated color palette and the leading-separator vertical spacing). Depends on Phase 18 (the current decorated renderer shape it amends) and Phase 6 (the render base).*

A pure presentation change confined to `internal/render`'s decorated impl — the `Renderer` interface and the raw impl are untouched. Two cohesive end states:

- **Palette.** The `you ›` prompt and its echoed input stay bold with the default foreground (no hue); the `assistant ›` label is bold + light blue (ANSI bright-blue) and the streamed reply text is the same light blue but **not** bold; `tool call ›` and a **successful** `tool result ›` line render in subdued gray (ANSI bright-black), label and body alike; a `tool result ›` with `IsError` keeps the red error treatment; `reasoning ›` stays dim. ANSI is still emitted only when `color`.
- **Spacing.** The decorated view emits exactly one blank line between every adjacent pair of transcript blocks, as a **leading** separator: a single newline written at the start of each block, suppressed until the first block of the session is drawn (no leading blank at start, no trailing blank after the summary) and suppressed between two consecutive prompts (a bare empty line adds no blank). The `tool call ›` / `tool result ›` emitters trim a single trailing newline from the rendered input/output so a tool whose output ends in `\n` still yields exactly one blank.

Because only the rendered bytes change, the decorated goldens under `testdata/` (transcript, color, and tty/non-tty prompt goldens) are regenerated to the new palette and spacing as part of staying green; the raw goldens are untouched.

**Done when:** R-OBNM-N6XX (the fixed palette, golden) and R-Q52T-PXCR (one blank line between blocks as a leading separator — no leading blank at start, no trailing blank after the summary, a `\n`-terminated tool output still one blank, a bare empty line none, golden) are covered by clearly-named tests (regenerated decorated goldens) and the suite is green.
