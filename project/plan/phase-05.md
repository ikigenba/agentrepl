# Phase 5 — Config-key namespace & coercion

*Realizes design Decision 3 (config-key namespace & coercion). Depends on Phase 2 (catalog interface) and Phase 1.*

Build `internal/config`: the `Target` struct, the explicit typed key table (`map[string]field`) covering every key in Decision 3's table across pointer/int/duration/bool/enum/string kinds, the `default` unset sentinel, the loose provider/model coupling (provider via catalog `Lookup`+`Build`; model pre-validated via `HasModel` when a provider is set), and `Set`/`Get`/`Dump`/`Keys`/`ParsePair` with `ErrUnknownKey`/`ErrBadValue`. Reads the *interface* of `catalog` only. Table-driven tests over the full key list; flag/runtime parity proven by `ParsePair`+`Set` vs direct `Set`.

**Done when:** R-LYK7-Y7ZS, R-LZS4-BZQH, R-M100-PRH6, R-M27X-3J7V, R-M3FT-HAYK, R-M4NP-V2P9, R-M5VM-8UFY are covered by clearly-named tests and the suite is green.
