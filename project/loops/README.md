# agentrepl — Build loop

The installed loop is the three-prompt **`gather → build → verify`** cycle, driven
by `project/loops/run`. It is unattended: each prompt runs as one bounded iteration
in a **fresh context**, all state lives in the workspace (the `project/` spec, the
source tree, git history, and the ephemeral `brief.md`), and `ralph` re-invokes the
prompts in order, wrapping `verify → gather`. These prompts are **generated from the
finished spec**, not spec artifacts; regenerate them with the
`create-gather-build-verify-prompts` skill if the spec's shape changes.

## Running it

```sh
./project/loops/run
```

which is exactly:

```sh
exec ralph project/loops/gather.md project/loops/build.md project/loops/verify.md
```

`ralph` runs from the **service root** (its working directory), so every path the
prompts reference is service-root-relative (`project/…`). It reads only the **last**
message of each turn and advances on that.

## Status contract

Each turn's **final** message reports one of:

- **`NEXT`** — **terminal**: this turn is done; advance to the next prompt (`verify`
  wraps back to `gather`). `build` and `verify` **always** end on `NEXT`.
- **`DONE`** — **terminal**: the whole job is complete; the loop stops. **Only
  `gather` ever reports it**, and only when no `⬜` phase remains in `STATUS.md`.
- **`CONTINUE`** — **non-terminal**: the status a streaming model tags the progress
  messages it emits *before* its terminal message. `ralph` reads only the terminal
  message, so `CONTINUE` never advances or ends the loop.

The only exit is `gather → DONE` (zero `⬜` markers), or a `ralph` budget rail.

## Per-step reads / writes / commits / flips

| step | reads | writes | commits | flips marker | brief |
|---|---|---|---|---|---|
| **gather** | `STATUS.md`, one `phase-NN.md`, realized `DNN.md`, `INDEX.md`, dep interfaces | `brief.md` **contract region** (fresh phase only) | no | no | authors it once per phase; **no-ops** while in flight |
| **build** | `brief.md` only (contract + feedback) | source + tests | yes (code increment) | no | reads only; never writes it |
| **verify** | `brief.md`, the suite, the source tree | `brief.md` **feedback region** (on a gap) or `STATUS.md` (on a pass) | yes (the `⬜→✅` flip, on a pass) | **yes** (pass only) | deletes on pass / stall reset; overwrites feedback on a gap |

## Brief lifecycle

`project/loops/brief.md` is the ephemeral, single-phase, region-owned contract:

- **gather** authors the contract region **once** when a phase first becomes the
  active `⬜` phase, and **no-ops** (leaves it untouched) every cycle the phase stays
  in flight — preserving `verify`'s feedback across cycles.
- **build** consumes it (contract + any feedback), closing feedback gaps first.
- **verify** on a **pass** flips the marker and deletes the brief; on a **gap**
  leaves `⬜`, changes no source, and **overwrites** the feedback region with only the
  currently-open gaps — so the brief **persists across cycles** until the phase passes
  or a stall reset discards it.

It is **gitignored** and never committed, so it exists only from the `gather` that
authors it to the `verify` that clears it.

## Why it converges (human-free)

`verify` can neither halt the loop nor advance a phase on a gap — an incomplete
phase just stays `⬜` and is re-attacked next cycle, now with `verify`'s grounded,
command-anchored feedback in front of `build`, and without `gather` re-reading the
big docs (it no-ops on the in-flight brief). The persisted feedback also gives
`verify` cross-cycle memory: it distinguishes *slow convergence* (the open-gap id set
shrinking/changing) from a *true stall* (the **same** gap ids unsatisfied for ≥3
consecutive attempts with **no new build commit**). On a true stall it does a
**trajectory reset** — discards the brief and logs the stall, so the next `gather`
rebuilds the contract fresh from spec — all still inside "verify never halts, never
advances on a gap." The loop ends only when every phase is verified green (`gather →
DONE`) or a `ralph` budget rail trips.

## `project/loops/brief.md` schema

A single-phase file with two single-writer regions. The **contract region**
(everything above `## Verify feedback`) is gather-owned; the **feedback region** is
verify-owned.

```
# Brief — Phase NN

## Objective
<one line: the phase's cohesive objective>

## Realized Decisions
- D<N> — project/design/DNN.md
  <full Decision prose copied verbatim — Decision statement, shape/signatures,
   Rejected alternatives — WITHOUT that Decision's Verification list>

## Ids to cover
R-XXXX-XXXX — <full requirement text copied verbatim from the Decision's Verification list>
<one id per line, id at line-start; or the single line "(none — structural phase)">

## Files to touch
- <path> — <what lands there>
- <path>_test.go — id-tagged tests co-located with the code they exercise

## Dependency interfaces
<exported signatures of the packages this phase depends on, copied verbatim; or "(none)">

## Done bar
- every id covered by a genuinely-asserting `// R-XXXX-XXXX` test that runs under
  `go test ./...`, co-located with the code it exercises (never a per-phase/root-level
  test file); the suite green (`go build ./...`, `go vet ./...`, `go test ./...` exit 0
  and `gofmt -l .` prints nothing); a structural phase → clean build + named smoke.

## Verify feedback
(none yet)                          ← gather writes this empty; verify overwrites it
                                       on a gap with:
## Verify feedback — attempt N
build commit: <sha>   stall streak: <k>
- R-XXXX-XXXX — <exact failing command + observed output (+ file:line)>
```

`project/README.md` (the workspace map) points here for loop mechanics; those live
only in this file. `brief.md` is listed in `.gitignore`.
