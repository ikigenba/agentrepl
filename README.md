# agentrepl

An interactive command-line REPL that drives [agentkit](https://github.com/ikigenba/agentkit)
directly, so every agentkit feature — provider/model selection, generation and
reasoning settings, custom tools, streaming, the full message exchange, and
token/cost reporting — is reachable, observable, and adjustable by hand. It is a
hands-on **testing and verification harness** for agentkit, not a production chat
client.

## Build & install

```sh
make build      # → bin/agentrepl
make install    # → ~/.local/bin/agentrepl (override with PREFIX=...)
make test       # go test ./...
```

Requires Go 1.26. The build resolves `agentkit` from its published module at
`github.com/ikigenba/agentkit` (no local `replace` directive), pinned to the
`v0.1.1` tag. Re-pin with `go get github.com/ikigenba/agentkit@v0.1.1`.

## Credentials

Provider keys are read **only** from the environment — one variable per provider,
never from the command line or a file:

| Provider | Env var |
|----------|---------|
| anthropic | `ANTHROPIC_API_KEY` |
| google    | `GEMINI_API_KEY` |
| openai    | `OPENAI_API_KEY` |
| zai       | `ZAI_API_KEY` |

A provider whose key is absent cannot be selected; you get a clear message, not a
crash.

## Quick start

```sh
agentrepl -c provider=anthropic -c model=claude-haiku-4-5
```

drops you straight into a prompt. Type a message and the reply streams back. End
with `/exit`, `/quit`, EOF (Ctrl-D), or Ctrl-C — the per-session spend is printed
on the way out and the session log is always left intact.

Run `agentrepl --help` for the full, credential-blind catalog of providers,
models, and each model's native reasoning key and accepted values.

## Usage examples

```sh
# Anthropic with an extended-thinking token budget
agentrepl -c provider=anthropic -c model=claude-haiku-4-5 -c thinking_budget=2048

# OpenAI with a reasoning effort level and a system prompt
agentrepl -c provider=openai -c model=gpt-5.5 -c effort=high -c system="You are terse."

# Gemini, raw (undecorated) rendering of the underlying messages
agentrepl -raw -c provider=google -c model=gemini-2.5-flash -c thinking_budget=-1

# Z.ai pointed at the GLM **coding** endpoint (instead of the default base URL)
agentrepl -c provider=zai -c model=glm-5.2 -c base_url=https://api.z.ai/api/coding/paas/v4
```

Everything set with `-c key=value` at launch has a runtime equivalent via
`/set key value`, using the same flat keys — so you can also switch the Z.ai
endpoint mid-session:

```
/set base_url https://api.z.ai/api/coding/paas/v4
/set base_url default        # back to Z.ai's baked-in root
```

`base_url` only applies to the **zai** provider; it is baked into the provider
handle when zai is built, so setting it before or after `provider=zai` reaches the
same state.

## Config keys (`-c key=value` / `/set key value`)

Flat, unprefixed names — one per agentkit setting. Set any key to the literal
`default` to reset it to its unset state.

| Key | Meaning |
|-----|---------|
| `provider` | `anthropic` \| `google` \| `openai` \| `zai` |
| `model` | a model id from the provider's curated set (see `--help`) |
| `system` | system prompt (may contain spaces) |
| `temperature`, `top_p` | sampling settings |
| `max_tokens` | output token cap |
| `effort` | native reasoning **level** — Anthropic opus/sonnet, OpenAI gpt-5.x, GLM 5.x |
| `thinking_budget` | native reasoning **token budget** — Anthropic haiku, Gemini 2.5 |
| `thinking_level` | native reasoning **level** — Gemini 3.x |
| `thinking` | native reasoning **toggle** (`on`/`off`) — GLM 4.x |
| `max_attempts`, `base_delay`, `max_delay`, `max_elapsed`, `ignore_retry_after` | retry policy |
| `tool_loop_limit` | max tool-call iterations per turn |
| `base_url` | provider API root override (zai only) |

**Reasoning is set in each model's own native term** (the key `--help` lists for
it) and a value it accepts. A value the model doesn't understand — including one
left over after a mid-conversation model switch — isn't an error: agentkit warns
and applies the model's default, and the warning is shown.

## Slash commands

| Command | Effect |
|---------|--------|
| `/set <key> <value>` | runtime equivalent of `-c` (errors are non-fatal) |
| `/get <key>` | show one key's current value |
| `/dump` | show every key's current value |
| `/usage` | cumulative token usage + cost so far |
| `/clear` | start a fresh conversation (cumulative spend persists) |
| `/render <decorated\|raw>` | switch rendering for subsequent turns |
| `/providers` | list providers (key present?) and their curated models |
| `/help` | list commands and config keys |
| `/exit`, `/quit` | graceful exit |

## Built-in tools

Four local tools are always registered and offered to the model, operating
relative to the current working directory: **bash**, **read**, **write**, **edit**.
When the model calls one, agentkit runs it and feeds the result back, looping to a
final answer — with every tool-call request and result visible.

## Rendering & session log

- **decorated** (default) — a human-readable transcript with a distinct treatment
  per kind (prompt, reply, reasoning, tool calls, tool results, usage/cost),
  colored only when writing to a terminal.
- **raw** (`-raw` or `/render raw`) — the underlying messages emitted verbatim,
  one JSON entry per line.

Independent of the rendering, every run writes its complete raw exchange to
`~/.agentkit/<session-id>.jsonl` for after-the-fact inspection. The log is
write-only forensic output — agentrepl never reads it back to resume.
