# agentrepl — Plan

**Authority: construction order and history.** This document and the `project/plan/` directory it heads own the order agentrepl is built in and the record of what has been built. The plan is **append-only**: phases are added at the bottom of the manifest and marked done as they land; completed phases are never rewritten or deleted, so the plan doubles as the construction history. To extend the project later, update `project/product/README.md` and `project/design/` in place (they stay authoritative for the current state), then **append** a new `phase-NN.md` plus a new `STATUS.md` line — never edit a finished phase except to flip its status marker.

**Coverage invariant.** The phases collectively realize **every** *current* design Verification id in **exactly one** phase: no current id unassigned, none split across phases (except an explicit, stated slice), none duplicated. Coverage is **one-directional** — design (rewritten in place) is the denominator, and the plan must cover all of it; verify mechanically that the design-only difference is empty:

```sh
comm -23 <(grep -hoE 'R-[A-Z0-9]{4}-[A-Z0-9]{4}' project/design/*.md   | sort -u) \
         <(grep -hoE 'R-[A-Z0-9]{4}-[A-Z0-9]{4}' project/plan/phase-*.md | sort -u)
```

Empty output is the pass condition. The reverse is deliberately **not** checked: because finished phases are frozen and the plan is append-only, the plan may carry **retired ids** — behaviors built when their id was current, then dropped from design when it stopped applying. Such an id stays in its frozen `phase-NN.md` forever as the record of that work; never delete it to chase parity. A current id minted later can only be covered by a **newly appended** phase.

**One phase = one package = one build-turn context.** Each phase is a single coherent unit, almost always one `internal/` package (plus, for the last phase, the composition root), sized so the build loop can carry it in one fresh build-turn context and ideally finish it in a turn or two. A phase reads only the design Decision(s) it realizes and the *interfaces* (not the internals) of the packages it depends on: the small public surface listed in those packages' design Decisions. That is what keeps every phase the size of a small standalone tool no matter how large the project grows. Where a single package realizes several intertwined Decisions and will not fit one context (here: `internal/repl`), it is split across phases that each leave the build green, and each affected phase names the **slice** of that Decision's Verification ids it carries.

**Done bar.** A phase is **done** when every Verification id it realizes (or the slice of those ids assigned to it) is covered by a clearly-named, genuinely-asserting test and the suite is green. Every phase's acceptance bar is stated as **deterministic exit conditions** — never a subjective judgment, never a self-referential/unsatisfiable check. "The suite is green" is defined in design's *Conventions*: `go build ./...`, `go vet ./...`, and `go test ./...` all exit 0, and `gofmt -l .` prints nothing. Each Decision's Verification list in `project/design/` is the authority for what "covered" means; a purely structural phase gets a deterministic check instead (a clean build plus a named smoke or a `project/`-excluded grep).

## Layout

The plan is split for addressability so the build loop never loads the whole history to find its next unit of work:

- **`project/plan/STATUS.md`** — the manifest: one grep-able Markdown bullet per phase, carrying its status marker (`⬜`/`✅`) and the design Decision(s) it realizes. This is the **only** place a phase's status marker lives. The loop finds the next phase by grepping for the first `⬜`.
- **`project/plan/phase-NN.md`** — one file per phase (zero-padded; sub-phases keep their suffix, e.g. `phase-07a.md`). It holds that phase's body — objective, *Realizes / Depends on* line, observable end state, and *Done when* id list. The loop reads exactly one per turn. A phase body file carries **no** status marker of its own.
- **`project/plan/README.md`** (this file) — the invariant rules above. Static; it does not grow with the project.

**Append-only, restated for this layout:** never rewrite or delete a `phase-NN.md`; never delete a line in `STATUS.md`. The only mutation during a build is flipping one phase's `⬜ → ✅` in `STATUS.md`. New work = a new `phase-NN.md` plus a new `STATUS.md` line, both appended at the end.
