# AI Canvas Builder Sidebar Tab

## Overview

This PRD defines a new AI-first canvas configuration flow in SuperPlane. Users can describe what they
want in natural language, and SuperPlane proposes structured canvas changes (add components, configure
fields, connect channels, and update existing nodes).

The experience is delivered as a new tab in the Components sidebar so users can switch between manual
component browsing and AI-assisted building without leaving the canvas.

## Problem Statement

Building a working canvas currently requires users to know which components to pick, how to configure
them, and how to wire data channels correctly. New users and occasional users often struggle with:

- Discovering the right components for their goal.
- Understanding required configuration fields and defaults.
- Mapping output fields from one node into the next node input.
- Iterating quickly when requirements change.

This increases time-to-value and can block users before they execute their first successful workflow.

## Goals

1. Let users configure a canvas by chatting with AI in natural language.
2. Add this flow as a first-class tab in the Components sidebar.
3. Generate safe, transparent, and reversible proposed canvas edits.
4. Reduce time from empty canvas to first successful run.
5. Keep users in control with explicit review and apply actions.

## Non-Goals

- Fully autonomous execution without user review.
- Auto-publishing, auto-enabling, or auto-running workflows by default.
- Replacing manual canvas editing and component browsing.
- Building a generic chat assistant unrelated to canvas authoring.
- Backend persistence for chat sessions, messages, or proposal history in v1.

## Primary Users

- **Workflow Builders**: Users creating new canvases quickly from intent.
- **Power Users**: Users accelerating iterative edits through natural language.
- **New Users**: Users who need guidance on component selection and wiring.

## User Stories

1. As a workflow builder, I can ask AI to create an end-to-end flow from a plain-English goal.
2. As a user, I can ask AI to modify an existing canvas without rebuilding it manually.
3. As a user, I can review exactly what AI wants to change before applying it.
4. As a user, I can apply only approved changes and undo if needed.
5. As a user, I can ask follow-up questions and refine the canvas in a continuous conversation.

## Functional Requirements

### Components Sidebar Integration

- Add a new tab in the Components sidebar: **AI Builder**.
- The existing Components tab remains unchanged.
- The AI Builder tab is visible only when Agent Mode is enabled in Organization Settings.
- When Agent Mode is disabled, the AI Builder tab is hidden from the sidebar.
- If prerequisites are missing (for example API key not configured), show a setup CTA and clear message.

### AI Builder Chat Experience

- Show a chat thread with user prompts and assistant responses.
- Provide a composer with submit, loading, and cancel states.
- Support follow-up prompts that reference prior chat context and current canvas state.
- Include starter prompts for common workflows (for example webhook to Slack, schedule to email).
- Keep chat state in-memory on the client only for v1 (cleared on refresh/navigation).

### Canvas-Aware Generation

- AI receives structured context from the active canvas:
  - Existing nodes and node types.
  - Current node configurations (with sensitive values redacted).
  - Existing channel mappings and graph layout metadata.
- AI proposes a structured change set, not direct uncontrolled mutations.

### Proposed Change Set and Review

- Each AI response that changes the canvas includes:
  - A plain-language explanation.
  - A list of proposed operations (add node, update node, delete node, connect nodes, disconnect nodes).
- Render proposed operations in a review panel with per-operation details.
- Users can apply all changes at once in v1.
- Users can dismiss a proposal without changing the canvas.

### Apply and Undo

- Applying a proposal creates one atomic canvas edit transaction.
- If any operation fails validation, no partial canvas mutations are persisted.
- Applied proposals are added to undo history as a single grouped action.
- Undo restores the prior canvas state.

### Validation and Guardrails

- Validate component availability, required fields, and channel compatibility before apply.
- Prevent destructive operations (node deletion, disconnect) unless explicitly asked or confirmed.
- Block invalid or unsafe operations and return actionable guidance in chat.
- Never expose secret values in assistant responses, logs, or prompts.

### Error and Fallback States

- Handle model timeout/unavailable errors with retry guidance.
- If AI cannot complete a request, suggest manual next steps or narrower prompts.
- Preserve chat and proposal state across transient failures.
- In v1, state persistence is limited to the current browser session and is not stored server-side.

## UX Requirements

- AI Builder tab uses the same sidebar visual language as existing tabs.
- Empty state includes:
  - What AI Builder can do.
  - 3 to 5 example prompts.
- Proposal state includes clear primary actions:
  - **Apply changes**
  - **Discard**
- Chat messages should clearly distinguish:
  - Informational replies.
  - Replies with executable canvas proposals.
- Show a compact "changes pending" indicator when a proposal is waiting for approval.

## Implementation Scope (v1)

### UI-Only Scope

- v1 is intentionally UI-only, with no new backend endpoints and no backend storage.
- Chat messages, generated proposals, and pending-change state exist only in client memory.
- Refreshing the page or reopening the canvas clears AI Builder conversation history.
- Applying changes mutates the current canvas in the existing editor flow only.

### Authorization and Access

- Reuse existing canvas page access and edit permissions in the UI.
- Users without edit permissions can view AI responses but cannot apply proposal changes.

### Privacy and Data Handling

- Do not persist AI Builder chat/proposal payloads to database tables in v1.
- Do not include secret values from component configuration in UI-visible prompts or responses.
- Keep any optional client analytics sanitized and free of secrets.

## Future Backend Work (Out of Scope for v1)

- Introduce server-backed AI sessions and message history.
- Add persisted proposal records and apply/discard audit trails.
- Add dedicated API contracts for session management and proposal lifecycle.
- Define long-term retention and organization-level governance controls.

## Acceptance Criteria

1. If Agent Mode is enabled in Organization Settings, users can see and open the **AI Builder** tab.
2. If Agent Mode is disabled in Organization Settings, users do not see the **AI Builder** tab.
3. Users can submit a natural-language prompt and receive a response tied to current canvas context.
4. When applicable, the system returns a structured proposal preview of canvas operations.
5. Users can apply a valid proposal and observe expected canvas updates.
6. Invalid proposals are blocked with clear error feedback and no partial mutations.
7. Users can discard a proposal without altering the canvas.
8. Users can undo an applied proposal as one grouped action.
9. No AI Builder chat/proposal data is persisted server-side in v1.
10. Sensitive data is redacted from prompts, responses, and any optional client telemetry.

## Success Metrics

- Median time from new canvas to first valid workflow.
- Percentage of canvases created or edited with AI Builder.
- Proposal apply rate and proposal discard rate.
- First-attempt apply success rate after validation.
- Reduction in user-reported configuration and wiring friction.

## Risks and Mitigations

- **Risk:** Incorrect wiring or invalid configs from AI.  
  **Mitigation:** Strict pre-apply validation and explicit user approval.

- **Risk:** Users over-trust AI suggestions.  
  **Mitigation:** Always show concrete operation diff and require apply confirmation.

- **Risk:** Sensitive data leakage in prompts/logs.  
  **Mitigation:** Redaction pipeline, secret filters, and storage policy limits.

- **Risk:** Latency makes chat feel slow.  
  **Mitigation:** Streaming responses, progress states, and partial reasoning summaries.

## Open Questions

1. Should v1 allow partial apply (select specific operations) or only apply-all?
2. Should read-only users be allowed to chat if they cannot apply?
3. When backend persistence is introduced, what retention period should we use for message history?
4. Should proposals be shareable across collaborators once backend storage exists?
5. Which model providers should be supported after the UI-only v1 release?
