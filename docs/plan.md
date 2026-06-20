# agentrepl — Plan

**Authority: construction order and history.** This document and the `docs/plan/` directory it heads own the order agentrepl is built in and the record of what has been built. The plan is **append-only**: phases are added at the bottom of the manifest and marked done as they land; completed phases are never rewritten or deleted, so the plan doubles as the construction history. To extend the project later, update `docs/product.md` and `docs/design.md` in place (they stay authoritative for the current state), then **append** a new phase — never edit a finished phase except to flip its status marker.

**One phase = one package = one accumulating context.** Each phase is a single coherent unit — almost always one `internal/` package (plus, for the last phase, the composition root) — built in one accumulating context against product and design. A phase reads only the design Decision(s) it realizes and the *interfaces* (not the internals) of the packages it depends on: the small public surface listed in those packages' design Decisions. That is what keeps every phase the size of a small standalone tool no matter how large the project grows. Where a single package realizes several intertwined Decisions and will not fit one context (here: `internal/repl`), it is split across phases that each leave the build green; the partial-Decision split is stated explicitly in the affected phases.

**Done bar.** A phase is **done** when every Verification item (the `R-XXXX-XXXX` ids) in the design Decisions it realizes — or the slice of those ids assigned to it — is covered by a clearly-named test and the suite is green. "The suite is green" is defined in design's *Conventions*: `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing. Each Decision's Verification list in `docs/design.md` is the authority for what "covered" means.

## Layout

The plan is split for addressability so the build loop never loads the whole history to find its next unit of work:

- **`docs/plan/STATUS.md`** — the manifest: one grep-able line per phase, carrying its status marker (`⬜`/`✅`) and the design Decision(s) it realizes. This is the **only** place a phase's status marker lives. The loop finds the next phase by grepping for the first `⬜`.
- **`docs/plan/phase-NN.md`** — one file per phase (zero-padded; sub-phases keep their suffix, e.g. `phase-07a.md`). It holds that phase's body — objective, *Realizes / Depends on* line, observable end state, and *Done when* id list. The loop reads exactly one per turn.
- **`docs/plan.md`** (this file) — the invariant rules above. Static; it does not grow with the project.

**Append-only, restated for this layout:** never rewrite or delete a `phase-NN.md`; never delete a line in `STATUS.md`. The only mutation during a build is flipping one phase's `⬜ → ✅` in `STATUS.md`. New work = a new `phase-NN.md` plus a new `STATUS.md` line, both appended at the end.
