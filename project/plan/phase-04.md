# Phase 4 — Built-in tools (bash / read / write / edit)

*Realizes design Decision 10 (built-in tools). Depends on Phase 1.*

Build `internal/tools`: `All()` returning the four `agentkit.NewTool[In]` tools with their typed input structs, exercised directly against a temp working directory. Behaviors per Decision 10: `bash` returns combined stdout+stderr (preserving output and noting `[exit status N]` on non-zero, nil Go error); `read` non-terminal `IsError` on missing file; `write` create/truncate; `edit` replace-all with count, non-terminal `IsError` when `Old` absent; all paths relative to cwd.

**Done when:** R-NHBW-446N, R-NIJS-HVXC, R-NKZL-9FEQ, R-NM7H-N75F, R-NNFE-0YW4, R-NONA-EQMT are covered by clearly-named tests and the suite is green.
