# agentrepl — Project workspace

Everything needed to design, plan, and build agentrepl lives under `project/`, at
the root of the codebase it governs (`github.com/ikigenba/agentrepl`). This README
is a **map, not a manual**: it says where each artifact lives and who writes it.
The shapes themselves are owned by the `$ikispec` contract; the loop mechanics are
owned by `project/loops/README.md`.

Every artifact has exactly one writer:

| folder | what's in it | written by |
|---|---|---|
| `product/` | `README.md` — the *why*: problem, users, scope, promises, success criteria | `$seal-spec` (rewritten in place) |
| `research/` | `research.md` — collected external ground truth design references (non-contractual) | `$seal-spec` (rewritten in place) |
| `design/` | `README.md` (spine) + `INDEX.md` (manifest) + `DNN.md` (one per Decision) | `$seal-spec` (rewritten in place) |
| `plan/` | `README.md` (rules) + `STATUS.md` (manifest + `⬜`/`✅` markers) + `phase-NN.md` (one per phase) | `$seal-spec` (append-only) |
| `loops/` | the generated `gather → build → verify` build-loop prompts + `README.md` | a prompt-generator workflow |
| `README.md` | this workspace map | `$seal-spec` |

**Authority partition** (each fact lives in exactly one place, and none restate
each other): product owns *why/promise*; research owns *external evidence*; design
owns *shape and its checkable proof* (minting `R-XXXX-XXXX` requirement ids); plan
owns *construction order and history* (append-only).

The codebase this governs is the Go module rooted here: `cmd/agentrepl/` (the
composition root) and the `internal/` packages. For how the installed build loop
turns this spec into code, see `project/loops/README.md`.
