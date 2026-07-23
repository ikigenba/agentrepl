# agentrepl — Product

**Authority: intent.** This document owns *why* agentrepl exists, *for whom*, what is in and out of scope, and the behavior we **promise** the user — stated once, in outcome terms. It does **not** state mechanism, type shapes, exact flag-parsing rules, key-coercion logic, wire/log formats, glyph code points, exit codes, or test assertions; those belong to `project/design/`. Where the two could overlap on behavior, this doc states the *promise* (what the operator observes) and design states the *exact, checkable proof* of that promise. That boundary is load-bearing: it keeps product, design, and plan from overlapping.

## Problem

[AgentKit](https://github.com/ikigenba/agentkit) is a Go library for holding tool-using, multi-turn conversations with an LLM across several providers. Its real-world consumers are deployed services where the agent loop is buried behind integrations with other systems — message queues, datastores, request handlers. That makes any single agentkit feature hard to exercise on its own: to watch a multi-step turn unfold, or confirm a tool call round-trips, or see what a provider switch does mid-conversation, a developer has to stand up a whole service and drive it indirectly. There is no fast, direct way to pick up agentkit, turn a knob, and *see* the result. agentkit ships a tiny example REPL to gesture at this, but it demonstrates one slice rather than exposing the library's full surface for hands-on testing.

## Purpose

agentrepl is an interactive command-line REPL that drives agentkit directly so a developer can exercise and verify every agentkit feature by hand. The single job it does: expose agentkit's entire conversation surface — provider/model selection and auth, generation settings, tools, the message exchange, usage and cost — through a REPL where each feature is reachable, observable, and adjustable, so the library can be tested in isolation rather than only through the deployed services that embed it. It replaces, and supersedes, the example program agentkit currently ships (which is to be removed from agentkit). Because it stands in for that example, agentrepl is held to idiomatic, community-standard Go as a first-class requirement: its source is meant to read as an accurate, exemplary demonstration of consuming agentkit, not merely to function.

## Users

The agentkit developer — the person building and maintaining the library — and anyone evaluating it. They are not looking for a production chat client; they are trying to confirm that an agentkit feature behaves as intended, to reproduce a suspected bug, or to feel out how a knob affects a real conversation. Their measure of agentrepl is whether it makes agentkit's behavior **immediate and visible** with the least ceremony.

## Scope

agentrepl covers:

- An **interactive REPL** that holds a multi-turn, text-only conversation with an LLM through agentkit.
- **Two control surfaces that together expose every (v1) agentkit conversation feature**: command-line flags that set initial state at launch, and `/slash-commands` that inspect and mutate that state at runtime. Anything not natural as a launch flag is reachable as a slash-command, and the launch-time settings have a runtime equivalent where changing them mid-conversation is meaningful.
- **A single generic config passthrough** for agentkit's conversation settings: a repeatable `key=value` flag whose keys are flat, native names — one per agentkit setting (provider, model, auth method, system prompt, generation settings, retry policy, tool-loop limit, and any future knob), with no namespace prefixes. The same key namespace is reachable at runtime through a slash-command and is dumpable on demand. New agentkit settings become new keys without new bespoke flags. The same vocabulary also exposes the per-provider construction overrides agentkit supports — the API base URL of any provider — as keyed entries, so they too are set without a bespoke flag.
- **Usable defaults.** Launching with no flags starts a working selection — OpenAI's `gpt-5.6-sol` under subscription auth — so the shortest path from install to conversation is zero configuration (plus a one-time login done outside agentrepl).
- **agentkit's standard local toolkit** registered with the conversation and offered to the model — the six shipped tools `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep` — rooted at agentrepl's current working directory, with file access confined beneath it. They are a fixed set, always present; they exist to prove that agentkit's own toolkit wires into a consumer and that agentkit drives the tool loop. agentrepl ships no tools of its own.
- **All five agentkit providers** selectable — Anthropic, Google (Gemini), OpenAI, OpenRouter, and Z.ai — at launch and between turns. The models on offer are **agentkit's own advisory catalog** — agentrepl curates nothing and adds nothing; what agentkit's catalog lists is what agentrepl lists.
- **Free-flow models, deliberately gated.** A model name agentkit's catalog knows can be selected bare — its provider is derived for you. A name the catalog does not know is not silently guessed at: it is accepted only when the operator has *explicitly* named the destination provider, and it then passes through exactly as agentkit sends it, with agentrepl noting what that forfeits (no pricing, no reasoning vocabulary).
- **Two auth methods for OpenAI** — an API key, or a ChatGPT-subscription login held in a local auth file the operator creates outside agentrepl with the standalone `oauth-login` tool; agentrepl consumes the file and keeps its tokens fresh, but never creates it. Every other provider authenticates with its API key. API keys come only from the environment, one conventional variable per provider.
- **A self-describing `--help`.** Launch-time help enumerates, statically: the launch defaults; the available providers with their auth methods; the models grouped by provider — for each model its native reasoning control (the term that model uses and the values it accepts, as reported by agentkit's catalog rather than hardcoded by agentrepl), and, where a model reaches a provider under a different wire name, that routing; and the rule for sending a model the catalog does not list. The help checks no credentials and reflects no live environment: it is a catalog of what can be *asked for*, not what is currently usable.
- **Provider/model switching mid-conversation**, with prior history carried over.
- **Message-granular replies**, with each completed assistant message and its reasoning summary shown in order as the turn unfolds.
- **Full visibility of the exchange**: the user's prompts, the model's replies and reasoning, every tool-call request with its arguments, and every tool result fed back.
- **Token-usage and dollar-cost reporting** surfaced from agentkit, at a cadence that depends on the rendering: in the decorated view, cumulatively — on demand and automatically when the session ends; in the raw view, per turn.
- **Two rendering formats** for what the operator sees: a decorated human-readable transcript and a raw, undecorated stream of the underlying messages.
- **An always-on session log**: every run records its complete raw exchange to a per-session file for after-the-fact inspection.
- **A reported version and a one-line install.** agentrepl reports its own version on request; a build cut from a release tag reports that semver tag, an unstamped build reports a `dev` sentinel, and no version string is hand-maintained in the source. The binary is distributable as a prebuilt release: a released version is cut by pushing a `vMAJOR.MINOR.PATCH` tag, and the published binary is installable with a single shell command (building from source stays available for developers).

It deliberately does **nothing else.** In particular:

- **No MCP in v1.** Attaching remote MCP servers is a committed **phase 2** direction, deferred — not rejected.
- **No conversation persistence or resume.** The conversation lives in memory for the life of the process; agentrepl never reads a prior session back to continue it. A runtime command clears in-session history to start fresh, but nothing is saved for reuse. The session log is write-only forensic output, not a resume format.
- **No credential handling beyond the environment and the subscription auth file.** No keys on the command line, no credential prompts or stores, and no login flow of any kind — obtaining the auth file is the operator's job, done outside agentrepl; agentrepl never displays or edits the file's contents.
- **Not a production client.** agentrepl is a testing and verification harness, not a polished end-user chat application; it is not intended for ongoing real use.
- **No images, audio, batch processing, embeddings, or fine-tuning** — agentrepl exposes agentkit's text conversation surface only. agentkit's embeddings API is a known, deliberate exclusion.
- **No OpenRouter routing preferences** (upstream-vendor ordering, price caps) — not reachable through agentkit's conversation surface today.

## Contractual constants

These fixed, promised values the design must use verbatim and never re-declare:

- **Module / repository path:** `github.com/ikigenba/agentrepl`
- **Dependency:** built on `github.com/ikigenba/agentkit`, including its advisory model catalog (models, per-model native reasoning vocabulary, pricing, and cross-provider routing) and its credential-based provider construction. agentrepl requires both and reimplements neither — it renders and relays what agentkit exposes.
- **Launch defaults:** provider `openai`, model `gpt-5.6-sol`, subscription auth.
- **Config separator:** agentkit config settings are passed as `key=value` (an equals sign), with flat, unprefixed keys named for each setting (e.g. `temperature`, `max_tokens`, `auth`, `base_url`); reasoning uses the selected model's own native term as the key (`effort`, `thinking_budget`, `thinking_level`, or `thinking`).
- **Credential variables:** one environment variable per provider, of the form `PROVIDER_API_KEY` — `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, `ZAI_API_KEY`.
- **Subscription auth file:** defaults to `~/.agentrepl/auth.json`; its location is a config setting (`auth_file`), so a file created elsewhere by `oauth-login` can be pointed at instead.
- **Session log location:** `~/.agentrepl/logs/<session-id>.jsonl`, one file per run — agentrepl's on-disk footprint lives entirely under `~/.agentrepl/`.
- **Release version scheme:** released versions are named by semver git tags of the form `vMAJOR.MINOR.PATCH`; a build with no injected version reports the literal `dev`; no version number is maintained by hand anywhere in the sources — the git tag is the sole source of truth.

## What we promise (user-facing behavior)

- **Launch into a conversation, immediately.** Running `agentrepl` with no flags drops the operator into an interactive prompt talking to the default model under subscription auth; flags override the defaults. Nothing else is required to start talking once auth is in place.

- **Missing credentials direct, they don't crash.** Starting with no keys and no login always reaches the prompt; the first turn that needs the missing credential explains exactly what to do — including a ready-to-run, copy-paste `oauth-login` command that creates the auth file, the API-key alternative, and pointing at an existing login file — and the session stays usable throughout.

- **Log in beside the REPL, not inside it.** The subscription sign-in happens in the operator's own terminal via the standalone `oauth-login` tool; agentrepl's role is to hand over the exact command to run when credentials are missing, and to pick the resulting file up on the very next turn, no restart. Subscription-backed usage shows its cost figures like any other usage; those figures are notional rather than an additional charge.

- **Type at a prompt; your input is not parroted back.** In an interactive terminal, agentrepl shows a `you ›` prompt before every input and waits there; the operator types their message or a slash-command at that prompt, and what they typed is not repeated back as a separate transcript line — the reply (or command output) follows directly beneath what they entered. When the output is not an interactive terminal, no prompt is drawn and the input is not echoed in the decorated view; the session log and the raw view remain the complete record of what was sent.

- **Every agentkit knob is reachable, from one config vocabulary.** agentkit's conversation settings are set at launch through a repeatable `key=value` flag and adjusted at runtime through the matching slash-command, using the same flat keys. For example, `-c temperature=0.7` at launch and a `/set temperature 0.7` at runtime mean the same thing, and the operator can dump the current configuration on demand. A bad key or unusable value is reported clearly rather than silently ignored.

- **Name a cataloged model and the rest is derived; name anything else and be explicit.** Setting just a model that agentkit's catalog knows selects its provider automatically — and where that model reaches the chosen provider under a different wire name, the translation happens invisibly. A model name the catalog does not know is refused when bare, and accepted when the operator has explicitly set the provider — with agentrepl saying plainly what pass-through means: cost reports zero, and reasoning settings go unchecked to the wire.

- **Pick any provider; switch between them mid-conversation.** The operator selects Anthropic, Google (Gemini), OpenAI, OpenRouter, or Z.ai and can change that selection between turns with the conversation continuing coherently. A provider whose credentials are absent can be *selected* freely — the turn that needs it says clearly what is missing, and nothing crashes.

- **Z.ai is present but known-broken.** Z.ai is a selectable provider so its integration can be exercised, but it does not currently complete a conversation end-to-end; selecting it surfaces agentkit's failure cleanly rather than misbehaving. This is a known limitation, not a promise that Z.ai works.

- **Any provider's endpoint is overridable.** The API base URL of the active provider is configurable through the config vocabulary — pointing any provider at a proxy, mock, or alternate endpoint without rebuilding.

- **Discover defaults, providers, auth, models, and reasoning options without starting a session.** Running agentrepl's help prints a static catalog: the launch defaults; every selectable provider with its auth method(s); the models available under each provider with each model's native reasoning term and the values it accepts (its discrete levels, or its valid token-budget range), including cross-provider wire names where they apply; and the rule for sending an uncataloged model. The help checks no credentials and inspects no environment, so it shows what is possible to *request*; the runtime `/providers` command shows what is currently *usable* given the credentials present.

- **Set reasoning in the model's own terms; mistakes are caught where a vocabulary exists.** The operator sets a model's reasoning using that model's native term *as the config key* and a value it accepts — the key and values `--help` lists for it (e.g. `effort=high`, `thinking_budget=8000`, `thinking_level=high`, `thinking=on`). For a cataloged model, a value outside that model's vocabulary is refused on the spot with the accepted values shown. For a model outside the catalog there is no vocabulary to check against: the value passes through as given, and the provider's own verdict is what comes back — shown, not swallowed.

- **Replies and reasoning are shown as the turn unfolds.** Each completed assistant message, its reasoning summary, every tool call, and every tool result appears in order as the turn progresses — so the operator watches a multi-step turn unfold step by step rather than waiting for the whole exchange to finish. Delivery is message-granular: the smallest unit shown is a completed message, not a token; while a message is being produced, agentrepl shows that it is waiting.

- **The whole exchange is observable.** Beyond the final text, the operator sees the model's reasoning, each tool-call request the model makes (with its arguments), and each tool result fed back into the loop. Nothing about the turn is hidden.

- **Tools work, driven automatically.** agentkit's standard toolkit (`Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`), rooted at the working directory, is offered to the model; when the model calls a tool, agentkit runs it and feeds the result back, looping until a final answer — with each step visible. This demonstrates, hands-on, that agentkit's shipped toolkit wires into a consumer and that the tool loop runs.

- **Spend on demand and at exit, not underfoot.** In the decorated view, agentrepl does not interrupt each turn with a usage/cost report; instead the operator asks for the session's cumulative token usage and dollar cost whenever they want it, and it is also shown automatically when the session ends. Cost figures come from agentkit — the catalog's pricing, or the provider's own reported cost where one reports it; a model with no known price reports zero cost, with the unknown flagged visibly. (The raw view continues to emit per-turn usage for machine inspection.)

- **Stop cleanly, on command or on interrupt.** The operator can end the session with a runtime command, or press Ctrl-C to stop immediately — including mid-reply. Either way the stop is prompt, the session log is left intact (never truncated or corrupted by the interruption), and the cumulative spend is reported as the session closes.

- **Choose how the exchange is rendered.** A decorated, human-readable transcript (the default) presents the model's reply, reasoning, tool calls, and tool results — each with a distinct visual treatment per kind, using color only when writing to a terminal; it does not re-render the operator's input as a transcript line and does not print a per-turn usage/cost report. A raw mode instead emits the underlying messages verbatim, one per entry, undecorated — including the operator's prompt and the per-turn usage entries — for inspecting exactly what agentkit produces. The rendering choice does not change what the session log records.

- **Every session is recorded for forensics.** Each run writes its complete raw exchange to `~/.agentrepl/logs/<session-id>.jsonl` as it happens, independent of the chosen rendering. When something looks wrong during interactive use, the operator (or an agent acting for them) can read that file to see the exact conversation. agentrepl never reads it back to resume.

- **Errors are shown, not swallowed.** When agentkit returns an error — a transient failure exhausted by retries, an unreachable provider, an invalid configuration, a rejected pass-through setting — agentrepl surfaces it visibly and the REPL stays usable. Automatic retrying of transient and rate-limit failures is agentkit's behavior; the operator simply sees the outcome.

- **Ask for the version; get the build's identity.** `agentrepl -V` (or `--version`) prints the version and exits without starting a session. A build cut from a release tag prints that tag (e.g. `v0.1.0`); a build between tags or from modified sources says so; a build with no version information prints `dev`. The version is never hand-maintained in source — it identifies the exact commit the binary was built from.

- **Install with one command.** A prebuilt release binary can be fetched onto the PATH with a single shell command, no Go toolchain required; the latest release installs by default and a specific version can be pinned. Building from source stays available for developers who want it.

## Success criteria (outcomes)

The verification gate runs the built agentrepl against exactly this list:

- Launching agentrepl with no flags (and subscription auth in place) yields an interactive prompt; typing a message returns a coherent reply from the default model.
- Launching with no credentials at all still reaches the prompt; the first turn hands over a ready-to-run login command, the API-key alternative, and the existing-file option — and the session remains usable.
- After the operator runs the handed-over login command in another terminal, the next turn in the same session works without a restart.
- Every agentkit conversation setting can be set at launch via a `key=value` flag and adjusted at runtime via the matching slash-command using the same flat key; the current configuration can be dumped on demand; an invalid key or value produces a clear, non-fatal error.
- Setting a bare cataloged model selects its provider automatically; setting a bare uncataloged model is refused with a clear message; setting an uncataloged model with the provider explicitly chosen is accepted, with the pass-through consequences stated.
- The operator can select Anthropic, Google (Gemini), OpenAI (either auth method), and OpenRouter (with credentials present) and hold a working conversation against each.
- The operator can change provider/model between turns and the conversation continues coherently against the newly selected provider/model, with prior history intact.
- A cataloged model that reaches its provider under a different wire name (a routed model) completes a conversation through that route.
- Selecting Z.ai is possible and surfaces agentkit's failure cleanly (Z.ai is not expected to complete a conversation).
- Selecting a provider whose credentials are absent produces a clear directive message at the turn and does not crash the REPL.
- Running agentrepl's help prints, statically: the launch defaults; the providers with their auth methods; the models grouped per provider with each model's native reasoning term and accepted values, and wire-name routing where it applies; and the uncataloged-model rule — checking no credentials and reflecting no environment.
- Setting a cataloged model's reasoning to an accepted value is honored exactly; a value outside that model's vocabulary is refused on the spot with the accepted values shown; an uncataloged model's reasoning passes through and the provider's verdict is displayed.
- Each completed assistant message and its reasoning summary appear in order as the turn unfolds (message-granular, not token-by-token), and during a multi-step turn the operator sees each message, tool call, and tool result land as the loop runs.
- For a turn that uses tools, the operator can observe the model's tool-call request(s) with arguments and the tool result(s) fed back, and the loop completes to a final answer.
- Each of agentkit's six standard toolkit tools — `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep` — is offered to the model, can be invoked by it, and operates relative to agentrepl's current working directory.
- In the decorated view, no automatic per-turn usage/cost report appears; the session's cumulative token usage and dollar cost can be requested on demand and are shown automatically when the session ends. In the raw view, per-turn usage continues to be emitted. A turn on an unpriced model shows zero cost with the unknown flagged visibly.
- In an interactive terminal, a `you ›` prompt is shown before every input and what the operator typed is not echoed back as a separate transcript line; when the output is not an interactive terminal, neither the prompt nor the echoed input appears in the decorated view.
- The operator can end the session with a runtime command or with Ctrl-C; Ctrl-C stops promptly even mid-reply, the session log remains a complete, uncorrupted record of what occurred, and the cumulative spend is reported as the session closes.
- The default decorated rendering distinguishes reply, reasoning, tool calls, and tool results, with color only when output is a terminal, and it neither re-renders the operator's input nor prints a per-turn usage/cost report; the raw rendering emits the underlying messages verbatim, one per entry, undecorated, including the operator's prompt and the per-turn usage entries.
- Every run writes its complete raw exchange to `~/.agentrepl/logs/<session-id>.jsonl` regardless of the chosen rendering, and that file reflects the conversation that occurred.
- Clearing in-session history via the runtime command starts a fresh conversation within the same process; agentrepl never resumes a prior session from disk.
- When agentkit returns an error, agentrepl surfaces it visibly and the REPL remains usable for the next input.
- Running `agentrepl -V` or `agentrepl --version` prints the version and exits cleanly without starting a session: a release-tagged build prints its `vMAJOR.MINOR.PATCH` tag, and an unstamped build prints `dev`.
