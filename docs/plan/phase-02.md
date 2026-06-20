# Phase 2 — Provider & model catalog

*Realizes design Decision 2 (provider & model catalog). Depends on Phase 1.*

Build `internal/catalog` as data plus an injected constructor (`ProviderFunc`), no interface: the `Provider` struct, `Default()` returning the four curated providers in stable order with their contractual env keys and agentkit model-constant lists, `Lookup`, `HasModel`, `Build(getenv)`, and the `ErrUnknownProvider`/`ErrUnknownModel`/`ErrMissingKey` sentinels. Includes the mechanical anti-drift test that every curated model is accepted by its constructed provider's `Pricing`. Tests use a fake `ProviderFunc`/fake `getenv`; no live keys.

**Done when:** R-OVEC-4AWS, R-OWM8-I2NH, R-OXU4-VUE6, R-OZ21-9M4V, R-P09X-NDVK, R-P1HU-15M9 are each covered by a clearly-named test and the suite is green.
