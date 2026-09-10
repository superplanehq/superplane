# Live analysis spec (draft popup)

Date: 2026-09-10

## Problem

Backlog analysis writes a plan and a confidence score, then the run ends.
The reader cannot add context. Refine opens a second agent that edits the
request, not the analysis.

## Goal

The analyze run stays alive after the first write. The user chats in the
draft popup. The agent updates the spec and the confidence score. The
original request does not change.

## Non-goals

- Do not remove board **Create with an Agent**. That work is later.
- Do not migrate old `intent.md` artifacts.
- Do not add **End session**.
- Do not let the agent edit the left-side request.

## Product rules

- New analysis writes `spec.md`. The UI splits it into Summary and Plan.
- Confidence uses the existing 0–5 meter and check copy.
- Refine is gone on backlog drafts.
- Close the popup does not stop the run.
- After a send succeeds, the runner owns that turn. Close the popup. The
  agent continues in the background.
- The run stops when the job times out (default 1 hour) or when the user
  clicks **Start**.
- After timeout the right pane is static. Do not restart the session.
- Only a user who can update the task can send chat.

## Keep-alive

Use the Create with Agent wait loop on the **same** analyze job.

1. When analyze starts, SuperPlane opens a session on that run and mints
   the planning token.
2. The runner attaches `follow_up_loop.js` and the spec MCP tools.
3. The first prompt does today’s analysis work. The agent calls the publish
   tools, then stops.
4. The loop long-polls `GET /api/v1/runner/planning-sessions/wait`.
5. A user message resolves the wait. The loop runs the same `run.js` with
   `--continue`.
6. **Start** ends the session and cancels the run. Then dispatch as today.
7. The 1 hour job timeout ends the session.

The browser does not keep the agent alive. The runner does.

## Tools

Same MCP server as Create with Agent. New publish contract:

| Tool | Writes |
| --- | --- |
| `propose_spec` | Full `spec.md` markdown (title, executive summary, plan) |
| `propose_confidence` | Score and short check copy |
| `survey` | Questions above the composer (unchanged) |

The first turn calls `propose_spec` and `propose_confidence`. Later turns
call one or both. The left request is never a tool argument.

Canvas nodes `attach-intent` and `report-confidence` are a fallback when
the job exits. The live UI reads the tool writes.

## Right pane (draft Description tab)

Top to bottom:

1. Spec title and Summary / Plan toggle. Body updates on `propose_spec`.
2. Live agent stream. Same log style as Create with Agent.
3. Survey, when the agent asks. Same control as Create with Agent.
4. Chat box. The user sends the next prompt.
5. Confidence footer. Updates on `propose_confidence`.

The left column stays the original request. It can still collapse a long
message.

While the popup is open, the UI polls the session (about 1.5s). Close the
popup: polling stops. The runner keeps waiting or working.

If the run is dead, disable the composer.

## Errors

- Failed send: show an error. Do not lose the composer text.
- Failed follow-up turn: the loop waits for the next message.
- Failed tool write: keep the last good spec and score.

## Testing

- First analyze turn publishes spec and confidence before the job exits.
- A chat message updates spec and/or confidence on the next turn.
- Close the popup after send. The session stays running. Reopen shows
  the new spec, score, and stream.
- **Start** ends the session and cancels the run.
- After the 1 hour timeout, the composer is disabled.
- Refine is absent on a backlog draft.
- Board Create with Agent still opens.

## Open implementation notes

- Reuse the planning-session wait mailbox and heartbeat.
- Bind the session to the analyze canvas run and the draft work order.
- Existing `intent.md` cards stay as they are. New runs write `spec.md`.
