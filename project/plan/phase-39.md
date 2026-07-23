# Phase 39 — README "Releasing" section (branch-first cut procedure)

*Realizes design Decision 15 (the release-procedure documentation — structural
slice, no Verification ids of its own; its proof is the deterministic check
below). Depends on Phase 38 (the release apparatus the section documents).*

Observable end state:

- `README.md` gains a short **Releasing** section for maintainers documenting the
  branch-first cut: land the release commit on `main`, `git tag -a vX.Y.Z`, and
  push the branch and tag together with `git push --follow-tags` (equivalently
  `git push origin main && git push origin vX.Y.Z`). It states plainly that a lone
  tag push ahead of its branch fires no release run, so the branch must reach
  `origin` before or with the tag.

**Done when** (structural phase — deterministic exit conditions, no `R-` ids;
the check targets `README.md`, an implementation file, so it is falsifiable and
non-self-referential):

- `README.md` contains a `Releasing` heading and the string `git push --follow-tags`.
- The Go suite stays green: `go build ./...`, `go vet ./...`, `go test ./...`
  exit `0`, and `gofmt -l .` prints nothing.
