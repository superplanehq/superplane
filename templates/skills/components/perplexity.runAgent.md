# Perplexity Run Agent Skill

Use this guidance when planning or configuring `perplexity.runAgent`.

## Purpose

`perplexity.runAgent` runs a Perplexity AI agent that can search the web and fetch URLs, returning a text response with citations.

## Required Configuration

- `modelSource` (required, default `"preset"`): choose between `"preset"` or `"model"`.
- `input` (required): the prompt or question for the agent. Supports expressions.
- `preset` (required when `modelSource` is `"preset"`, default `"pro-search"`): agent preset to use. Available presets:
  - `fast-search`: quick web search
  - `pro-search`: balanced search (recommended default)
  - `deep-research`: thorough multi-source research
  - `advanced-deep-research`: most comprehensive research
- `model` (required when `modelSource` is `"model"`): specific model to use.
- `instructions` (optional): system-level instructions to guide the agent's behavior.
- `webSearch` (optional, default `true`): enable the web_search tool.
- `fetchUrl` (optional, default `true`): enable the fetch_url tool.

## Output

Emits on the `default` channel with payload type `perplexity.agent.response`:

- `text`: the generated text response
- `citations`: array of source citations (`{ type, url }`)
- `model`: the model used
- `status`: completion status
- `usage`: token and cost usage information (`input_tokens`, `output_tokens`, `total_tokens`, `cost`)
- `response`: the full raw API response

## Planning Rules

When generating workflow operations that include `perplexity.runAgent`:

1. Always set `configuration.input` to a prompt string or expression.
2. Set `modelSource: "preset"` for most use cases (recommended).
3. Only set `modelSource: "model"` when the user explicitly requests a specific model.
4. Keep `webSearch` and `fetchUrl` enabled unless the user explicitly wants to disable them.
5. Use `instructions` to set agent persona or constrain output format when needed.
6. Access the response text downstream via `{{ $["Node Name"].text }}`.
7. Access citations via `{{ $["Node Name"].citations }}`.

## Expression Context

The `input` and `instructions` fields support expressions:

- `Summarize this incident: {{ $["Get Incident"].data.description }}`
- `Research the root cause of: {{ root().data.title }}`
- `{{ $["Slack Message"].data.text }}`

## Configuration Example

- `modelSource: "preset"`
- `preset: "pro-search"`
- `input: "Summarize the latest changes in {{ $[\"Get Release\"].data.tag_name }}"`
- `instructions: "Respond in bullet points. Focus on security implications."`
- `webSearch: true`
- `fetchUrl: true`

## Mistakes To Avoid

- Missing `input`.
- Setting `modelSource: "model"` without providing a `model` value.
- Disabling both `webSearch` and `fetchUrl` (the agent loses its key capabilities).
- Using `deep-research` or `advanced-deep-research` for simple lookups (use `fast-search` or `pro-search` instead).
- Referencing `$["Node Name"].data.text` instead of `$["Node Name"].text` (text is at the top level of the payload data).
