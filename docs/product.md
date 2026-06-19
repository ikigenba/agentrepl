# agentrepl — Product

**Authority: intent.** This document owns *why* agentrepl exists, *for whom*, what is in and out of scope, and the behavior we **promise** the user — stated once, in outcome terms. It does **not** state mechanism, type shapes, exact flag-parsing rules, key-coercion logic, wire/log formats, glyph code points, exit codes, or test assertions; those belong to `docs/design.md`. Where the two could overlap on behavior, this doc states the *promise* (what the operator observes) and design states the *exact, checkable proof* of that promise. That boundary is load-bearing: it keeps product, design, and plan from overlapping.

## Problem

[AgentKit](https://github.com/ikigenba/agentkit) is a Go library for holding tool-using, multi-turn conversations with an LLM across several providers. Its real-world consumers are deployed services where the agent loop is buried behind integrations with other systems — message queues, datastores, request handlers. That makes any single agentkit feature hard to exercise on its own: to watch streaming work, or confirm a tool call round-trips, or see what a provider switch does mid-conversation, a developer has to stand up a whole service and drive it indirectly. There is no fast, direct way to pick up agentkit, turn a knob, and *see* the result. agentkit ships a tiny example REPL to gesture at this, but it demonstrates one slice rather than exposing the library's full surface for hands-on testing.

## Purpose

agentrepl is an interactive command-line REPL that drives agentkit directly so a developer can exercise and verify every agentkit feature by hand. The single job it does: expose agentkit's entire conversation surface — provider/model, generation settings, custom tools, streaming, the message exchange, usage and cost — through a REPL where each feature is reachable, observable, and adjustable, so the library can be tested in isolation rather than only through the deployed services that embed it. It replaces, and supersedes, the example program agentkit currently ships (which is to be removed from agentkit). Because it stands in for that example, agentrepl is held to idiomatic, community-standard Go as a first-class requirement: its source is meant to read as an accurate, exemplary demonstration of consuming agentkit, not merely to function.

## Users

The agentkit developer — the person building and maintaining the library — and anyone evaluating it. They are not looking for a production chat client; they are trying to confirm that an agentkit feature behaves as intended, to reproduce a suspected bug, or to feel out how a knob affects a real conversation. Their measure of agentrepl is whether it makes agentkit's behavior **immediate and visible** with the least ceremony.

## Scope

agentrepl covers, for v1:

- An **interactive REPL** that holds a multi-turn, text-only conversation with an LLM through agentkit.
- **Two control surfaces that together expose every (v1) agentkit feature**: command-line flags that set initial state at launch, and `/slash-commands` that inspect and mutate that state at runtime. Anything not natural as a launch flag is reachable as a slash-command, and the launch-time settings have a runtime equivalent where changing them mid-conversation is meaningful.
- **A single generic config passthrough** for agentkit's conversation settings: a repeatable `key=value` flag whose keys are flat, native names — one per agentkit setting (provider, model, system prompt, generation settings, retry policy, tool-loop limit, and any future knob), with no namespace prefixes. The same key namespace is reachable at runtime through a slash-command and is dumpable on demand. New agentkit settings become new keys without new bespoke flags. The same vocabulary also exposes the per-provider construction overrides agentkit supports — notably the Z.ai API base URL — as keyed entries, so they too are set without a bespoke flag.
- **Four built-in local tools** registered with agentkit and offered to the model — `bash`, `read`, `write`, `edit` — all operating relative to agentrepl's current working directory. They are a fixed set, always present; they exist to prove that a consumer can supply tools and that agentkit drives the tool loop. (`bash` doubles as the original example's single-tool demonstration.)
- **All four agentkit providers** selectable — Anthropic, Google (Gemini), OpenAI, and Z.ai — choosing from agentkit's curated model set, at launch and between turns.
- **A self-describing `--help`.** Launch-time help enumerates, statically, the available providers in one list and the models grouped by provider in another, and — for each model — its native reasoning control: the term that model uses and the values it accepts (its discrete levels, or its valid token-budget range), as reported by agentkit rather than hardcoded by agentrepl. The help checks no credentials and reflects no live environment: it is a catalog of what can be *asked for*, not what is currently usable.
- **Provider/model switching mid-conversation**, with prior history carried over.
- **Streaming replies**, with the model's incremental text and reasoning visible as they arrive.
- **Full visibility of the exchange**: the user's prompts, the model's replies and reasoning, every tool-call request with its arguments, and every tool result fed back.
- **Token-usage and dollar-cost reporting** surfaced from agentkit, at a cadence that depends on the rendering: in the decorated view, cumulatively — on demand and automatically when the session ends; in the raw view, per turn.
- **Two rendering formats** for what the operator sees: a decorated human-readable transcript and a raw, undecorated stream of the underlying messages.
- **An always-on session log**: every run records its complete raw exchange to a per-session file for after-the-fact inspection.
- **Credentials sourced only from the environment**, one conventional variable per provider; agentrepl reads no credential files itself.

It deliberately does **nothing else.** In particular:

- **No MCP in v1.** Attaching remote MCP servers is a committed **phase 2** direction, deferred — not rejected. v1 exposes every *other* agentkit feature.
- **No conversation persistence or resume.** The conversation lives in memory for the life of the process; agentrepl never reads a prior session back to continue it. A runtime command clears in-session history to start fresh, but nothing is saved for reuse. The session log is write-only forensic output, not a resume format.
- **No credential handling beyond the environment.** No keys on the command line, no reading of secret files by agentrepl, no credential prompts or stores.
- **Not a production client.** agentrepl is a testing and verification harness, not a polished end-user chat application; it is not intended for ongoing real use.
- **No images, audio, batch processing, embeddings, or fine-tuning** — agentrepl exposes only what agentkit's v1 conversation surface offers, which is text only.

## Contractual constants

These fixed, promised values the design must use verbatim and never re-declare:

- **Module / repository path:** `github.com/ikigenba/agentrepl`
- **Dependency:** built on `github.com/ikigenba/agentkit`, including its per-model native reasoning introspection (each model's term, the values it accepts or its valid range, and its default) and its guarantee that a reasoning input a model does not understand is reported as a warning and falls back to that model's default. agentrepl requires both and reimplements neither — it renders what agentkit exposes.
- **Config separator:** agentkit config settings are passed as `key=value` (an equals sign), with flat, unprefixed keys named for each setting (e.g. `temperature`, `max_tokens`, `base_url`); reasoning uses the selected model's own native term as the key (`effort`, `thinking_budget`, `thinking_level`, or `thinking`).
- **Credential variables:** one environment variable per provider, of the form `PROVIDER_API_KEY` — `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, `ZAI_API_KEY`.
- **Session log location:** `~/.agentkit/<session-id>.jsonl`, one file per run.

## What we promise (user-facing behavior)

- **Launch into a conversation, immediately.** Running agentrepl with a provider and model selected drops the operator into an interactive prompt where they type a message and get a streamed reply. Initial settings come from flags; nothing else is required to start talking.

- **Type at a prompt; your input is not parroted back.** In an interactive terminal, agentrepl shows a `you ›` prompt before every input and waits there; the operator types their message or a slash-command at that prompt, and what they typed is not repeated back as a separate transcript line — the reply (or command output) follows directly beneath what they entered. When the output is not an interactive terminal, no prompt is drawn and the input is not echoed in the decorated view; the session log and the raw view remain the complete record of what was sent.

- **Every agentkit knob is reachable, from one config vocabulary.** agentkit's conversation settings are set at launch through a repeatable `key=value` flag and adjusted at runtime through the matching slash-command, using the same flat keys. For example, `-c temperature=0.7` at launch and a `/set temperature 0.7` at runtime mean the same thing, and the operator can dump the current configuration on demand. A bad key or unusable value is reported clearly rather than silently ignored.

- **Pick any provider whose key is present; switch between them mid-conversation.** The operator selects Anthropic, Google (Gemini), OpenAI, or Z.ai and a model from agentkit's curated set, and can change that selection between turns with the conversation continuing coherently. A provider whose environment key is absent cannot be selected, and saying so is a clear message, not a crash.

- **Z.ai is present but known-broken.** Z.ai is a selectable provider so its integration can be exercised, but it does not currently complete a conversation end-to-end; selecting it surfaces agentkit's failure cleanly rather than misbehaving. This is a known limitation, not a promise that Z.ai works in v1. Its API base URL is configurable through the config vocabulary (e.g. to target the GLM coding-plan endpoint), so alternate Z.ai endpoints can be exercised without rebuilding.

- **Discover providers, models, and reasoning options without starting a session.** Running agentrepl's help prints a static catalog: every selectable provider in one list, the models available under each provider in another, and — for each model — its native reasoning term and the values it accepts (its discrete levels, or its valid token-budget range), as reported by agentkit rather than a uniform list repeated everywhere. The help checks no credentials and inspects no environment, so it shows what is possible to *request*; the runtime `/providers` command shows what is currently *usable* given the keys present.

- **Set reasoning in the model's own terms; non-native input falls back, visibly.** The operator sets a model's reasoning using that model's native term *as the config key* and a value it accepts — the key and values `--help` lists for it (e.g. `effort=high`, `thinking_budget=8000`, `thinking_level=high`, `thinking=on`). A value the model understands is honored exactly. A term or value the model does not understand — including a setting left over after switching models mid-conversation — is not silently misapplied: agentrepl prints agentkit's warning and the model's default reasoning is used instead. There is no cross-model translation; the operator always sees when their reasoning input was not honored and what was used in its place.

- **Replies stream, and reasoning is visible.** The model's text appears incrementally as it is generated, and when a model emits reasoning, that reasoning is shown too — so the operator watches the turn unfold rather than waiting for a finished block.

- **The whole exchange is observable.** Beyond the final text, the operator sees the model's reasoning, each tool-call request the model makes (with its arguments), and each tool result fed back into the loop. Nothing about the turn is hidden.

- **Tools work, driven automatically.** The four built-in tools (`bash`, `read`, `write`, `edit`), rooted at the working directory, are offered to the model; when the model calls one, agentkit runs it and feeds the result back, looping until a final answer — with each step visible. This demonstrates, hands-on, that a consumer can supply tools and that the tool loop runs.

- **Spend on demand and at exit, not underfoot.** In the decorated view, agentrepl does not interrupt each turn with a usage/cost report; instead the operator asks for the session's cumulative token usage and dollar cost whenever they want it, and it is also shown automatically when the session ends — both drawn from agentkit's built-in pricing. (The raw view continues to emit per-turn usage for machine inspection.)

- **Stop cleanly, on command or on interrupt.** The operator can end the session with a runtime command, or press Ctrl-C to stop immediately — including mid-reply. Either way the stop is prompt, the session log is left intact (never truncated or corrupted by the interruption), and the cumulative spend is reported as the session closes.

- **Choose how the exchange is rendered.** A decorated, human-readable transcript (the default) presents the model's reply, reasoning, tool calls, and tool results — each with a distinct visual treatment per kind, using color only when writing to a terminal; it does not re-render the operator's input as a transcript line and does not print a per-turn usage/cost report. A raw mode instead emits the underlying messages verbatim, one per entry, undecorated — including the operator's prompt and the per-turn usage entries — for inspecting exactly what agentkit produces. The rendering choice does not change what the session log records.

- **Every session is recorded for forensics.** Each run writes its complete raw exchange to `~/.agentkit/<session-id>.jsonl` as it happens, independent of the chosen rendering. When something looks wrong during interactive use, the operator (or an agent acting for them) can read that file to see the exact conversation. agentrepl never reads it back to resume.

- **Errors are shown, not swallowed.** When agentkit returns an error — a transient failure exhausted by retries, an unreachable provider, an invalid configuration — agentrepl surfaces it visibly and the REPL stays usable. Automatic retrying of transient and rate-limit failures is agentkit's behavior; the operator simply sees the outcome.

## Success criteria (outcomes)

The verification gate runs the built agentrepl against exactly this list:

- Launching agentrepl with a provider and model selected yields an interactive prompt; typing a message returns a coherent, streamed reply.
- Every agentkit conversation setting can be set at launch via a `key=value` flag and adjusted at runtime via the matching slash-command using the same flat key; the current configuration can be dumped on demand; an invalid key or value produces a clear, non-fatal error — except reasoning, whose non-native input warns and falls back to the model's default rather than erroring (see below).
- The operator can select Anthropic, Google (Gemini), and OpenAI (each with its key present) and hold a working conversation against each.
- The operator can change provider/model between turns and the conversation continues coherently against the newly selected provider/model, with prior history intact.
- Selecting Z.ai is possible and surfaces agentkit's failure cleanly (Z.ai is not expected to complete a conversation in v1).
- Selecting a provider whose environment key is absent produces a clear message and does not crash the REPL.
- Running agentrepl's help prints, statically, the available providers in one list and the models grouped per provider in another, and for each model its native reasoning term and the values it accepts (its discrete levels, or its valid token-budget range); the help checks no credentials and does not reflect the current environment.
- The operator can set a model's reasoning using that model's native term as the config key and an accepted value, and it is honored exactly; a term or value the model does not understand — including one carried over after a model switch — produces a warning and the model's default reasoning is applied, with agentrepl showing what was used instead, and there is no cross-model translation.
- Replies are delivered incrementally, and when a model emits reasoning it is visible as the turn unfolds.
- For a turn that uses tools, the operator can observe the model's tool-call request(s) with arguments and the tool result(s) fed back, and the loop completes to a final answer.
- Each of the four built-in tools — `bash`, `read`, `write`, `edit` — can be invoked by the model and operates relative to agentrepl's current working directory.
- In the decorated view, no automatic per-turn usage/cost report appears; the session's cumulative token usage and dollar cost can be requested on demand and are shown automatically when the session ends. In the raw view, per-turn usage continues to be emitted.
- In an interactive terminal, a `you ›` prompt is shown before every input and what the operator typed is not echoed back as a separate transcript line; when the output is not an interactive terminal, neither the prompt nor the echoed input appears in the decorated view.
- The operator can end the session with a runtime command or with Ctrl-C; Ctrl-C stops promptly even mid-reply, the session log remains a complete, uncorrupted record of what occurred, and the cumulative spend is reported as the session closes.
- The default decorated rendering distinguishes reply, reasoning, tool calls, and tool results, with color only when output is a terminal, and it neither re-renders the operator's input nor prints a per-turn usage/cost report; the raw rendering emits the underlying messages verbatim, one per entry, undecorated, including the operator's prompt and the per-turn usage entries.
- Every run writes its complete raw exchange to `~/.agentkit/<session-id>.jsonl` regardless of the chosen rendering, and that file reflects the conversation that occurred.
- Clearing in-session history via the runtime command starts a fresh conversation within the same process; agentrepl never resumes a prior session from disk.
- When agentkit returns an error, agentrepl surfaces it visibly and the REPL remains usable for the next input.
