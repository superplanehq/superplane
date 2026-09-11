# Analysis instruction layers

Date: 2026-09-11

## Problem

Backlog analysis mixes five kinds of instruction in one canvas prompt and
repeats the protocol in three runner CLIs. Factory editors see tool and chat
rules. The stream can show that same prompt. The agent also hears a 0-100
score, a 0-5 meter, and leftover file paths.

## Goal

The reader gets a clear score, a readable Summary and Plan, and a short chat.
Protocol stays off the canvas and off the stream.

## Non-goals

- Do not change Create with an Agent. That product goes away later.
- Do not auto-replace a factory-edited Analyze prompt.
- Do not add a rematerialize button.

## Layers

| Layer | Who sees it | Owns |
| --- | --- | --- |
| Analyze canvas prompt | Factory editor, and the stream step title | Scoring, spec craft, the task |
| System pack | Agent only, on the CLI system channel | Session kind, chat voice, wait, publish vs talk |
| MCP tools | Agent only, as tool schema | `propose_spec`, `propose_confidence`, `survey` |
| Follow-up wrap | Agent only, on later turns | The user text only |

## Config prompt

The Analyze step keeps scoring and spec craft.

- Score from 0 through 5. Match the meter. Do not use 0-100.
- Keep the current Summary and Plan headings and length rules.
- Keep the task payload `{{ root().data.workOrder }}`.

The Analyze step drops tool names, wait rules, chat voice, file paths, and
the UI-split sentence.

## System pack

One module, `pkg/components/runner/analysis_protocol.js`.

Claude uses `--append-system-prompt`. Codex uses `developer_instructions`.
OpenCode uses the `instructions` config field. No CLI concatenates the pack
onto the user prompt.

The pack tells the agent to:

- Write short plain text to the user.
- Publish the spec and the score with tools. Do not paste them in chat.
- Leave the original request unchanged.
- Call `survey` when the task is unclear or two valid readings exist.
- Explore the repository. Do not edit it.

Do not print the pack in live logs. Do not write it into the cloned repo.

## Follow-up

The wait wrapper sends only the user message. The pack already says how to
treat later turns.

## Exit graph

`propose_spec` and `propose_confidence` still publish to SuperPlane. They also
write `/tmp/spec.md` and `/tmp/intake-analysis.json` so the existing exit
nodes keep working. The JSON score is `score * 20` so old graphs that divide
by 20 still map to 0-5. The agent is not told about those files.

## Stream

Compact session log hides:

- Runner setup lines (already hidden)
- The Analyze prompt step (same as Create with an Agent hide)
- `propose_spec` and `propose_confidence` payloads (collapsed tools)

The left chat shows the original request, short agent talk, and collapsed
tool rows.

## Errors

A failed tool write keeps the last good spec and score. The wait loop still
waits for the next message.

## Testing

- The Analyze prompt has no tool names and no `/tmp` paths.
- The follow-up wait text is the user message only.
- Claude, Codex, and OpenRouter put the pack on a system channel.
- Compact log hides the Analyze step and publish-tool JSON.
