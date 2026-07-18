# agentrepl — Research

**Status: non-contractual.** This document informs the *author* of `project/design/`; nothing downstream (the autonomous build) reads it. It records external ground truth as of **2026-07-17** — the actual public surface of the agentkit release the design consumes — so design references facts instead of re-deriving them. Design remains the single authority for *how*. Edit this doc in place as the product evolves — never append a log.

agentrepl is a **thin consumer** of agentkit (`project/product/README.md` is the authority on *why*). This doc covers only the agentkit facts the current design leans on: the restructured provider/auth surface, the advisory catalog, the pass-through configuration model, and the subscription-auth file contract.

---

## 1. The agentkit restructure (released as v0.4.0)

The restructure landed as a run of agentkit phases (54–64) and is tagged **v0.4.0** at the agentkit HEAD this spec targets. (agentkit's CHANGELOG heading for the entry says "v0.3.0" — a mislabel in agentkit; the `v0.3.0` *tag* points at a pre-restructure commit, which is why agentrepl's prior pin still compiled. The pin target is the tag, `v0.4.0`.) The headline changes, verified by reading the source:

1. **Credential constructors.** Every provider package exposes a sealed `Credential` type; clients are built `pkg.New(cred, opts...)`. The library reads **no** environment variables — the caller supplies key material.
2. **Advisory catalog.** A new `catalog` package carries the model tables (models, providers, routes, pricing, reasoning specs, context sizes). It is explicitly advisory: "catalog coverage never controls whether a model can be sent to a provider" — unknown model names remain runnable (**free-flow**).
3. **OpenRouter provider.** A fifth provider, plus per-model `Routes` in the catalog mapping a home-provider model to its wire slug on another provider.
4. **Subscription auth for OpenAI.** ChatGPT-subscription-backed access via an auth file holding the raw OAuth token-endpoint response (created externally by the standalone `oauth-login` CLI); usage under it reports **notional** cost (what it would have cost), not an extra charge.
5. **Consumer-owned cost resolution.** Per-turn cost resolves: provider-**reported** cost (OpenRouter only) → `Conversation.Pricing` (caller-supplied) → `0` + `WarnCostUnknown`. agentkit no longer prices turns from internal registries.
6. **Removed surfaces.** The legacy model-ID constants (`anthropic.ModelOpus48`, …), the per-package reasoning inspectors (`anthropic.Reasoning`, `ReasoningInspector`, `SupportedReasoning`), and the reasoning warning codes `WarnReasoningUnsupported`/`WarnReasoningCannotDisable` are **gone**. Reasoning values now pass through to the provider; a value a provider cannot encode is a provider-side error (or, for OpenAI + budget, `ErrInvalidConfig`), not a warn-and-default.

What is **unchanged**: `ReasoningValue` (`Level`/`Budget`/`DisableReasoning`, zero = unset) on `GenSettings.Reasoning`; `ReasoningSpec{Term, Kind, Levels, Min, Max, Sentinels, Default, CanDisable}` and its `Accepts(v)`; the `Conversation`/`Send`/`Stream` driving surface; `Usage`/`Cost`/`TotalUsage`/`TotalCost`; the JSONL `Log`; `Warning{Setting, Code, Detail}` via `(*Stream).Warnings()` (with the reduced code set `WarnToolChoiceForced`/`WarnToolSchemaLossy`/`WarnCostUnknown`).

---

## 2. The `catalog` package (`github.com/ikigenba/agentkit/catalog`)

```go
type Entry struct {
    Model, Provider string            // model id and home provider
    Routes          map[string]string // other provider → wire model (e.g. "openrouter" → "z-ai/glm-5.2")
    Pricing         *agentkit.Pricing // nano-USD/token rate tiers; may be nil
    Reasoning       *ReasoningSpec    // the model's native reasoning descriptor
    Context         int64
    Embedding       *EmbeddingInfo    // non-nil marks an embedding model
    Options         json.RawMessage
}

func Lookup(model string) (Entry, bool)
func Resolve(provider, model string) (routeProvider, wireModel string, entry Entry, ok bool)
func ListByProvider(provider string) []Entry // home entries + routed entries, sorted by model
func Check(model string, v agentkit.ReasoningValue) (accepted bool, spec *ReasoningSpec, ok bool)
```

- `Resolve("", model)` finds the entry's home provider; `Resolve(p, model)` consults `Routes` and yields the wire model for that provider. It never rejects a (provider, model) pair — `ok` is simply false when uncataloged.
- `ListByProvider` **includes routed entries** (the four GLM models appear under both `zai` and `openrouter`) and includes embedding entries (`Embedding != nil`), which a chat-model listing must filter out.
- `catalog.ReasoningKind`/`Sentinel`/`ReasoningSpec` are aliases of the root agentkit types.
- Chat-model contents (2026-07-17): anthropic ×5 (claude-fable-5, claude-haiku-4-5, claude-opus-4-8, claude-sonnet-4-6, claude-sonnet-5), google ×5 (gemini-2.5-flash, gemini-2.5-pro, gemini-3.1-flash-lite, gemini-3.1-pro-preview, gemini-3.5-flash), openai ×8 (gpt-5.4, gpt-5.4-mini, gpt-5.4-nano, gpt-5.5, gpt-5.5-pro, gpt-5.6-luna, gpt-5.6-sol, gpt-5.6-terra), zai ×4 (glm-4.6, glm-4.7, glm-5.1, glm-5.2; each with an `openrouter` route to `z-ai/<model>`). Embedding entries: text-embedding-3-small/-large (openai), gemini-embedding-001 (google). The reasoning specs carry the same terms/levels/ranges/sentinels/defaults the pre-restructure inspectors reported.
- `catalog/openrouterx` builds OpenRouter routing-preference JSON for `Request.ProviderOptions` — **unreachable through `Conversation`** (no field carries it; only the SPI `Request` does), so it is not a consumable feature for a `Conversation`-driving harness and is out of agentrepl's scope.

---

## 3. Provider construction & auth

Uniform per package: `type APIKey string` implementing a sealed `Credential`; `New(cred Credential, opts ...Option) *Provider`; options `WithBaseURL(string)` and `WithHTTPClient(*http.Client)` on **all five** providers.

| Package | Construction | Base API | Notes |
|---|---|---|---|
| `anthropic` | `anthropic.New(anthropic.APIKey(k))` | api.anthropic.com | |
| `google` | `google.New(google.APIKey(k))` | generativelanguage.googleapis.com | |
| `openai` | `openai.New(openai.APIKey(k))` **or** `openai.New(openai.Subscription(ts))` | api.openai.com / chatgpt.com | see §4 |
| `openrouter` | `openrouter.New(openrouter.APIKey(k))` | openrouter.ai/api/v1 | model names are OpenRouter slugs; the **only** provider that reports actual cost (`usage.cost` → `RoundTrip.ReportedCost`), which wins over `Conversation.Pricing` |
| `zai` | `zai.New(zai.APIKey(k))` | api.z.ai/api/paas/v4 | |

Reasoning encoding is per-provider translation of whatever `ReasoningValue` arrives; notable: OpenAI cannot encode a `Budget` (returns `ErrInvalidConfig`); `effort=none` is a real OpenAI level, distinct from disable.

---

## 4. OpenAI subscription auth (`openai/subscription`, agentkit ≥ v0.5.0)

As of 2026-07-18, agentkit v0.5.0 ships **no login flow** — the package is `Load` + `Token` only, over an auth file that is the **raw RFC 6749 token-endpoint response, verbatim** (top-level `access_token`, `refresh_token`, `id_token`; no wrapper, no `account_id` field — the account id is derived inside agentkit from the `chatgpt_account_id` value under the `https://api.openai.com/auth` JWT claim). The prior Codex-CLI wrapper shape is no longer read.

```go
func Load(path string) (*Store, error)                       // raw token-response shape; derives account id from JWT claims; loud error otherwise
func (s *Store) Token(ctx) (bearer, accountID string, error) // auto-refreshes near JWT expiry; atomic rewrite (temp+rename, 0600) in the same raw shape
```

Wire-up: `openai.New(openai.Subscription(store))`. Requests go to `chatgpt.com/backend-api/codex/responses` with the account-id header; credential name `openai.subscription`.

- **The file is produced by the standalone `oauth-login` CLI** (`github.com/ikigenba/oauth-login`) — a generic OAuth authorization-code+PKCE tool that serves its own loopback callback, prints the token response verbatim to stdout, and writes all human-facing output to stderr. Its documented OpenAI invocation (spec-pinned in that repo's design, D06) is, as one line:
  `oauth-login --auth-url https://auth.openai.com/oauth/authorize --token-url https://auth.openai.com/oauth/token --client-id app_EMoamEEZ73f0CkXaXp7hrann --scope "openid profile email offline_access" --port 1455 --callback-path /auth/callback > <auth_file>`
  (`--port 1455 --callback-path /auth/callback` match OpenAI's registered redirect URI; no `--callback-host` needed — the default `localhost` is the registered form.)
- **Contention warning (agentkit docs):** the Store rewrites the file on token refresh, rotating a per-login refresh-token lineage — the file agentrepl refreshes should be one from the operator's own `oauth-login` run, not a copy shared with another live tool.
- Cost under subscription auth is **notional** — computed like any priced usage, but not an additional charge.

---

## 5. Facts that drove design calls

- **Pass-through is the library's philosophy**: agentkit validates no provider/model/effort combination; the catalog is advice. Gating (or not) is the consumer's decision — which is why agentrepl's resolution rules (bare model must be cataloged; explicit provider unlocks free-flow) live in agentrepl's design, not agentkit's.
- **Pricing is now the consumer's to supply**: without `Conversation.Pricing` (from `Entry.Pricing`), every non-OpenRouter turn costs 0 and warns `WarnCostUnknown`.
- **Reasoning warn-and-default is gone**: the old agentrepl carve-out (accept any reasoning value, let agentkit warn and default) has no library backing anymore; validation against `catalog.Check` at set time is the only pre-wire check available, and an unchecked value reaching the wire surfaces as a provider error.
- **`OPENROUTER_API_KEY`** is the conventional env var for OpenRouter keys (matches the provider's own docs and the project's `PROVIDER_API_KEY` pattern).
