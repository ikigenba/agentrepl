# agentrepl — Plan Status

The manifest. One Markdown bullet per phase, in build order, each beginning with the literal `- Phase` — the **only** place a phase's status marker lives. Each phase line carries a done-marker (U+2705) or not-started-marker (U+2B1C). The build loop finds the next work with `grep -nE '^- Phase .* ⬜' project/plan/STATUS.md | head -1`, reads only that phase's `project/plan/phase-NN.md`, and on completion flips that phase's one marker here to done. Nothing else in this file or any phase file is edited at build time. Append a new bullet (and a new phase file) to extend. (This paragraph deliberately carries no bare status glyph, so the anchored grep matches only phase lines.)

- Phase 01  ✅  realizes D1        — Module bootstrap & package skeleton
- Phase 02  ✅  realizes D2        — Provider & model catalog
- Phase 03  ✅  realizes D8        — Session log & session-id
- Phase 04  ✅  realizes D10       — Built-in tools (bash / read / write / edit)
- Phase 05  ✅  realizes D3        — Config-key namespace & coercion
- Phase 06  ✅  realizes D5,D7     — Renderer (decorated & raw) + usage/cost formatting
- Phase 07a ✅  realizes D4,D9,D11 — REPL launch surface, loop & command dispatch (no live turn)
- Phase 07b ✅  realizes D5,D6,D7,D8,D9,D11 — REPL turn driver, usage triggers & graceful exit
- Phase 08  ✅  realizes D1,D6,D7,D11 — Composition root, interrupt & log integrity
- Phase 09  ✅  realizes —         — Makefile (build / fmt / test / install / clean)
- Phase 10  ✅  realizes D2,D3     — Configurable Z.ai base URL (`-c zai.base_url=…`)
- Phase 11  ✅  realizes D3        — agentkit native-reasoning pin & native `gen.reasoning` coercion
- Phase 12  ✅  realizes D2        — Catalog reasoning introspector field
- Phase 13  ✅  realizes D5,D11    — Settings-warning relay (`Renderer.Warning` + turn driver)
- Phase 14  ✅  realizes D4,D12    — Self-describing `--help` catalog
- Phase 15  ✅  realizes D12       — `--help` reasoning rows lead with the `gen.reasoning=` key
- Phase 16  ✅  realizes D3        — Flatten config keys & native reasoning keys
- Phase 17  ✅  realizes D12       — `--help` rows lead with each model's native reasoning key
- Phase 18  ✅  realizes D5,D7,D9  — Decorated input prompt & per-turn-report removal
- Phase 19  ✅  realizes D5        — Decorated palette & vertical spacing
- Phase 20  ✅  realizes D13       — Wait status line: formatters & live driver
- Phase 21  ✅  realizes D1,D5,D13 — Wait status line: seam wiring & composition root
- Phase 22  ✅  realizes D5        — Adopt agentkit message-granular delivery (drop delta rendering)
- Phase 23  ✅  realizes —         — Bump agentkit dependency pin to v0.3.0
- Phase 24  ✅  realizes —         — Adopt v0.3.0's new curated models (Claude 5, GPT-5.6)
- Phase 25  ✅  realizes D12       — `--help` marks each enum/toggle default inline with `*`
- Phase 26  ✅  realizes D12       — `--help` drops the redundant native term from range rows
- Phase 27  ✅  realizes D12       — `--help` stars a range default that matches a sentinel
- Phase 28  ✅  realizes D2        — agentkit v0.4.0 pin + provider table (`internal/catalog` rewrite)
- Phase 29  ✅  realizes D3        — Config resolution, auth keys, defaults & lazy construction
- Phase 30  ⬜  realizes D12       — `--help`: defaults, auth lines, routes & the two-tier footer
- Phase 31  ⬜  realizes D7,D9,D14 — `/login`, `/providers` auth status, lazy-failure directive & cost-unknown relay
