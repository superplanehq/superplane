# AI Agent Builder: Adaptive Conversation Flow

## Overview

This document proposes a structured conversation model for the AI agent builder. Instead of immediately generating canvas proposals, the agent follows a three-phase approach — understanding the goal, confirming the tools, and then proposing the flow — adapting the number of conversational rounds based on context.

## Problem Statement

The agent currently jumps straight to proposing canvas operations regardless of how much context it has. For open-ended prompts like "I want to deploy my app", the agent skips critical steps:

- It does not check which integrations the user has connected.
- It does not confirm which technologies or providers to use.
- It does not present its plan for the user to review before building.
- It guesses at configuration, often producing proposals with wrong integrations, missing config, or components the org cannot use.

This leads to low-quality first proposals, wasted apply/discard cycles, and an experience that feels like the agent is not listening.

## Goals

1. The agent should understand the user's goal before building anything.
2. The agent should check which integrations and components are available and present them to the user.
3. The agent should confirm the high-level flow before generating canvas operations.
4. For simple, unambiguous edits, the agent should still respond immediately without unnecessary back-and-forth.
5. No code or model changes required — this is achievable through system prompt changes alone.

## Decision Variables

Before responding, the agent evaluates four variables that determine how many conversational rounds are needed:

| Variable | Values | Impact |
|---|---|---|
| **Canvas state** | Empty / Populated | Empty = building from scratch, more discussion needed |
| **Org context** | Fresh (no integrations) / Established (integrations connected) | Fresh = may need to guide integration setup first |
| **Prompt specificity** | Vague / Medium / Specific | Vague = more clarification rounds needed |
| **Canvas memory** | None / Has prior context | Memory reduces need for repeated questions |

## Three-Phase Conversation Model

The agent progresses through three phases in order. It can skip phases when it already has sufficient information.

```
Phase 1: Understand the Goal
  │  Agent investigates context, asks clarifying questions.
  │  No proposal.
  ▼
Phase 2: Confirm the Tools and Flow
  │  Agent presents which integrations/components it will use,
  │  outlines the flow in plain language. No proposal.
  ▼
Phase 3: Propose the Flow
     Agent builds the canvas proposal with concrete operations.
```

### Phase 1 — Understand the Goal

**Purpose**: Clarify what the user is trying to achieve.

**When entered**: The prompt is vague or open-ended ("deploy my app", "set up CI/CD", "help me with this canvas").

**Agent actions**:

- Call `get_canvas` or `get_canvas_shape` to see current canvas state.
- Call `list_org_integrations` to see what the user has connected.
- Call `get_canvas_memory` if prior sessions exist.
- Call `search_decision_patterns` if the request matches a known workflow pattern.

**Agent response** (proposal = null):

- Summarize what it found (canvas state, connected integrations).
- Ask targeted questions to narrow the goal.
- Use `follow_up_questions` to suggest concrete options.

**Example responses**:

> Empty canvas, user says "I want to deploy my app":
>
> "I see you have GitHub and DigitalOcean connected. What repo are we deploying? And is DigitalOcean your target, or are you deploying elsewhere?"

> No integrations connected:
>
> "You don't have any integrations connected yet. To set up CI/CD, we'd need a source code provider (GitHub, GitLab, Bitbucket) and optionally a deployment target. Which do you use? You can connect them in Organization Settings > Integrations."

> Populated canvas, user asks for help:
>
> "Your current flow runs onPush → Lint → Test → Build → Deploy → Notify. A few options: (1) Run Lint and Test in parallel, (2) Add a filter to skip deploys for non-main branches, (3) Add early-exit on lint failure. What interests you?"

**Exit condition**: User provides enough clarity to move to Phase 2.

### Phase 2 — Confirm the Tools and Flow

**Purpose**: Present which integrations and components will be used, and the high-level flow shape.

**When entered**: The goal is clear but the specific tools/flow haven't been confirmed. Also entered directly when the prompt has medium specificity (e.g., "deploy storejs from GitHub to DigitalOcean").

**Agent actions**:

- Call `list_components` / `list_triggers` with relevant provider filters.
- Call `describe_component` / `describe_trigger` for the candidates.
- Check that required integrations are actually connected via `list_org_integrations`.

**Agent response** (proposal = null):

- Name the specific integrations it will use.
- Outline the flow in plain language.
- Flag any gaps (missing integrations, ambiguous choices).
- Use `follow_up_questions` for any remaining choices.

**Example responses**:

> "Here's what I'd build: (1) github.onPush trigger for main branch, (2) github.runWorkflow for CI, (3) digitalocean.createApp for deployment, (4) Slack notification on success/failure. Ready to build this?"

> With a gap:
>
> "You don't have a Slack integration connected — want to add notifications later, or skip that step?"

**Exit condition**: User confirms the approach ("yes", "go ahead", "looks good").

### Phase 3 — Propose the Flow

**Purpose**: Generate the actual canvas proposal with concrete operations.

**When entered**: The approach is confirmed. Also entered directly when the prompt is specific and unambiguous.

**Agent actions**:

- Call `describe_component` / `describe_trigger` for each node to get config field schemas.
- Construct the operations list (add_node, connect_nodes, update_node_config, etc.).
- Fill in configuration fields based on context from earlier phases.

**Agent response** (proposal with operations):

- Brief summary of what's being built.
- Full proposal with all canvas operations.

## Adaptive Skip Rules

The agent skips phases based on these conditions:

**Skip Phase 1, go to Phase 2** when:

- The prompt names a clear goal AND specific technologies ("deploy storejs from GitHub to DigitalOcean").
- The canvas is populated and the prompt references existing nodes ("add X after the Deploy step").

**Skip Phase 1 and 2, go straight to Phase 3** when ALL of these are true:

- The user gives a specific, concrete instruction ("add a Slack notification after Deploy", "delete the Filter node").
- No meaningful design choices remain — the request is unambiguous.
- The canvas already has the referenced nodes.
- The required integrations are connected.

**Never skip Phase 1** when:

- The canvas is empty AND the prompt does not name specific technologies.
- The org has no integrations connected.
- The prompt is a question rather than an instruction ("what can I do with this?", "help me improve this").

## Scenario Walkthroughs

### Scenario 1: Empty canvas + vague prompt + established org

**User**: "I want to deploy my app"

- **Phase 1**: Agent checks canvas (empty), org integrations (GitHub, DigitalOcean, Slack), no memory.
  - *Response*: "I see you have GitHub, DigitalOcean, and Slack connected. What repo are we deploying? And is DigitalOcean your target, or deploying elsewhere?"
- **User**: "storejs repo on GitHub, yes deploy to DO"
- **Phase 2**: Agent lists GitHub and DO components, checks what's available.
  - *Response*: "Here's the flow I'd build: Push to main → CI → Deploy to DO → Slack notify. Sound good?"
- **User**: "yes"
- **Phase 3**: Proposal with 4-5 add_node operations + connections.

### Scenario 2: Empty canvas + specific prompt + established org

**User**: "Set up a pipeline: push to main on puppies-inc/storejs, run CI, deploy to DigitalOcean, notify #deploys on Slack"

- **Phase 1 skipped** (goal is clear, technologies named).
- **Phase 2**: Agent checks integrations (all connected), lists components.
  - *Response*: "You have GitHub, DigitalOcean, and Slack connected. I'll build: onPush main → Run CI → Deploy to DO → Slack #deploys. Ready?"
- **User**: "go"
- **Phase 3**: Proposal.

### Scenario 3: Populated canvas + specific instruction

**User**: "Add a Slack notification after the Deploy step"

- **Phase 1 and 2 skipped** (specific instruction, references existing node).
- **Phase 3**: Immediate proposal.

### Scenario 4: Empty canvas + vague prompt + fresh org (no integrations)

**User**: "I want CI/CD for my project"

- **Phase 1**: Agent checks canvas (empty), org integrations (none), no memory.
  - *Response*: "You don't have any integrations connected yet. To set up CI/CD, we'd need a source code provider (GitHub, GitLab, Bitbucket) and optionally a deployment target. Which do you use? You can connect them in Organization Settings > Integrations."
- **User**: "GitHub and we deploy to AWS"
- **Phase 2**: Agent checks — GitHub not connected, AWS not connected.
  - *Response*: "You'll need to connect those integrations first. Once GitHub and AWS are connected, I can build the pipeline. Want me to describe what the flow will look like so you know what to expect?"
- Conversation continues after user connects integrations.

### Scenario 5: Populated canvas + vague improvement request

**User**: "This pipeline is too slow, can you help optimize it?"

- **Phase 1**: Agent checks canvas (onPush → Lint → Test → Build → Deploy → Notify), checks memory.
  - *Response*: "Your current flow runs 5 steps sequentially. A few options: (1) Run Lint and Test in parallel, (2) Add a filter to skip deploys for non-main branches, (3) Add early-exit on lint failure. What interests you?"
- **User**: "Let's do parallel lint and test"
- **Phase 2 skipped** (tools are already on the canvas).
- **Phase 3**: Proposal to restructure the flow.

## In-Chat Integration Connect Buttons

### Problem

When the agent discovers that required integrations are not connected (Scenario 4), it currently can only tell the user to navigate away to Organization Settings > Integrations. This breaks the conversation flow — the user leaves the chat, configures integrations on a separate page, comes back, and has to re-explain what they wanted.

### Proposed Solution

When the agent's response mentions that an integration is needed but not connected, the chat UI should render inline "Connect" buttons next to each missing integration. Clicking a button opens the integration setup dialog as a popup, without leaving the chat.

### How the Conversation Changes

**Before** (current behavior):

> Agent: "You don't have a GitHub integration connected. Go to Organization Settings > Integrations to set it up."
>
> *User navigates away, configures GitHub, comes back, starts a new chat.*

**After** (proposed behavior):

> Agent: "You don't have GitHub connected yet. You'll need it for this pipeline."
>
> [Connect GitHub] [Connect Slack]
>
> *User clicks "Connect GitHub" — a popup dialog opens. User fills in credentials/completes OAuth. Dialog closes. Chat continues.*
>
> Agent: "GitHub is now connected. Ready to continue — what repo should we use?"

### Existing Infrastructure

This is feasible because the building blocks already exist:

1. **`IntegrationCreateDialog`** (`web_src/src/ui/IntegrationCreateDialog/index.tsx`) — A reusable shadcn `<Dialog>` for connecting integrations. Already used as a popup from the canvas page and the component sidebar. Handles the full flow: name input, configuration fields, OAuth browser actions (opens a new tab, user returns), webhook URL setup.

2. **`IntegrationStatusIndicator`** (`web_src/src/ui/IntegrationStatusIndicator/index.tsx`) — A widget already used on the canvas page that shows missing integrations with per-integration "Connect" buttons and opens `IntegrationCreateDialog` on click. This is the exact pattern we want to bring into the chat.

3. **Hooks**: `useAvailableIntegrations()`, `useCreateIntegration(orgId)`, and `useConnectedIntegrations(orgId)` in `web_src/src/hooks/useIntegrations.ts` provide all the data and mutation logic.

4. **OAuth handling**: The `IntegrationCreateDialog` already handles OAuth by opening a new browser tab for the auth flow. The dialog stays open while the user completes OAuth externally, then the user returns and clicks Save. This works fine in a popup context.

### What Needs to Change

**Agent output model**: Add a new optional field to `CanvasAnswer` (or extend an existing field like `citations` or `follow_up_questions`) to carry structured "connect integration" actions. For example:

```python
class IntegrationAction(BaseModel):
    integration_name: str  # e.g., "github", "slack"
    label: str             # e.g., "Connect GitHub"

class CanvasAnswer(BaseModel):
    # ... existing fields ...
    integration_actions: list[IntegrationAction] = Field(default_factory=list)
```

**System prompt**: Teach the agent to populate `integration_actions` when it discovers missing integrations, instead of just mentioning them in text.

**Frontend chat message renderer**: When a message has `integration_actions`, render them as clickable buttons below the message text. Each button:
- Looks up the integration definition from `useAvailableIntegrations()`.
- Opens `IntegrationCreateDialog` with the right `integrationDefinition`.
- On successful creation, fires `onCreated` to refresh connected integrations and optionally auto-sends a follow-up message like "GitHub is now connected."

**SSE event protocol**: The `final_answer` event already carries the full `CanvasAnswer` output. No protocol changes needed — `integration_actions` would be part of the output payload.

### Scenario 4 Revised: Empty canvas + vague prompt + fresh org

**User**: "I want CI/CD for my project"

- **Phase 1**: Agent checks canvas (empty), org integrations (none), no memory.
  - *Response*: "You don't have any integrations connected yet. To set up CI/CD, we'd need a source code provider and a deployment target. Which do you use?"
- **User**: "GitHub and AWS"
- **Phase 2**: Agent checks — GitHub not connected, AWS not connected.
  - *Response*: "Let's get those connected first. Once they're ready, I'll build the pipeline."
  - *Integration actions*: [Connect GitHub] [Connect AWS]
- *User clicks "Connect GitHub", completes setup in popup, clicks "Connect AWS", completes setup.*
- Agent (auto or on next prompt): "GitHub and AWS are now connected. Here's the flow I'd build: onPush main → Run CI → Deploy to AWS. Ready?"
- **User**: "yes"
- **Phase 3**: Proposal.

## Clickable Option Chips

### Problem

During Phase 1 and Phase 2, the agent asks questions and presents options. Currently these are plain text in the assistant message — the user has to read the options, then type a reply. This adds friction, especially on mobile or when the options are straightforward ("A or B?").

The `CanvasAnswer` model already has a `follow_up_questions: list[str]` field, but the frontend completely ignores it — it's never extracted from the `final_answer` event or rendered.

### Proposed Solution

When the agent's response includes options for the user to choose from, render them as clickable chips below the message. Clicking a chip auto-sends it as the user's next message, continuing the conversation without manual typing. Always include an "Other..." chip that focuses the input field so the user can type a custom response.

### How It Looks

**Phase 1 example** — agent asks about the goal:

> "I see you have GitHub and DigitalOcean connected. What are we building?"
>
> [ Deploy pipeline ] [ CI/CD workflow ] [ Monitoring / alerts ] [ Other... ]

**Phase 1 example** — agent asks which provider:

> "You have both GitHub and GitLab connected. Which one hosts this repo?"
>
> [ GitHub ] [ GitLab ] [ Other... ]

**Phase 2 example** — agent presents flow options:

> "Here's what I'd build. Ready?"
>
> [ Yes, build it ] [ Modify the plan ] [ Start over ]

**Phase 2 example** — agent flags a gap:

> "You don't have Slack connected. Want to add notifications?"
>
> [ Yes, connect Slack now ] [ Skip notifications ] [ Use a different channel ]

### Agent Output Model Changes

Replace the existing `follow_up_questions: list[str]` with a richer structure:

```python
class FollowUpOption(BaseModel):
    label: str       # Display text on the chip, e.g., "Deploy pipeline"
    value: str       # Text sent as the user's message when clicked,
                     # e.g., "I want to set up a deploy pipeline"

class CanvasAnswer(BaseModel):
    # ... existing fields ...
    follow_up_options: list[FollowUpOption] = Field(default_factory=list)
    proposal: CanvasProposal | None = None
```

The `label` is what appears on the chip (short, scannable). The `value` is the full message sent to the agent when clicked (can be more descriptive to give the agent enough context). An "Other..." chip is always appended by the frontend — it's not part of the model output.

### Frontend Changes

**New component**: `AiBuilderOptionChips` — renders a horizontal list of chip buttons below the assistant message. Each chip:
- Shows `label` as the button text.
- On click, calls `onSendPrompt(option.value)` — the same function used by the input form.
- Gets disabled once the user has responded (chips are for the latest message only).

**"Other..." chip**: Always appended as the last chip. On click, focuses the text input and optionally pre-fills with a prompt like "I'd prefer..." so the user can type freely.

**Placement**: Chips appear below the assistant message text, above the input form. They are only shown for the most recent assistant message (not historical messages in the conversation).

**State**: Chips disappear (or become disabled/muted) once the user sends a response — either by clicking a chip or typing manually.

### System Prompt Changes

Teach the agent to populate `follow_up_options` during Phase 1 and Phase 2. For example:

- Phase 1: options for the goal ("Deploy pipeline", "CI/CD workflow", "Monitoring") or for the provider ("GitHub", "GitLab").
- Phase 2: options for confirming ("Yes, build it", "Modify the plan") or for handling gaps ("Connect Slack now", "Skip notifications").

The agent should keep labels short (2-4 words) and values descriptive enough to continue the conversation.

### Scenario 1 Revised with Chips

**User**: "I want to deploy my app"

- **Phase 1**: Agent checks canvas (empty), org integrations (GitHub, DigitalOcean, Slack).
  - *Response*: "I see you have GitHub, DigitalOcean, and Slack connected. What repo are we deploying, and is DigitalOcean the target?"
  - *Options*: [ Deploy to DigitalOcean ] [ Deploy somewhere else ] [ Other... ]
- *User clicks "Deploy to DigitalOcean"* — sends "I want to deploy to DigitalOcean" as the message.
- **Phase 2**: Agent lists components.
  - *Response*: "Here's the flow I'd build: Push to main → CI → Deploy to DO → Slack notify. Sound good?"
  - *Options*: [ Yes, build it ] [ Modify the plan ] [ Other... ]
- *User clicks "Yes, build it"*
- **Phase 3**: Proposal.

## What Needs to Change

### System prompt (`agent/src/ai/system_prompt.txt`)

- Replace the current conversation guidance with the three-phase model described above.
- Add the adaptive skip rules.
- Add concrete examples of Phase 1 and Phase 2 responses.
- Teach the agent to populate `follow_up_options` with short-label / descriptive-value pairs during exploration phases.

### Agent output model (`agent/src/ai/models.py`)

- Replace `follow_up_questions: list[str]` with `follow_up_options: list[FollowUpOption]` where each option has a `label` and `value`.
- Add `integration_actions: list[IntegrationAction]` for in-chat connect buttons.

### Frontend changes

- **Option chips**: New `AiBuilderOptionChips` component rendered below the latest assistant message. Clicking a chip auto-sends `option.value` as the user's next message. Always append an "Other..." chip that focuses the input.
- **Integration connect buttons**: Extend the chat message renderer to support `integration_actions`. Each button opens `IntegrationCreateDialog` as a popup. Refresh connected integrations after successful setup.
- **`final_answer` parsing**: Extract `follow_up_options` and `integration_actions` from the `final_answer` SSE event output and store them alongside the message.

### No changes required for the three-phase model alone

- `CanvasAnswer` already supports `proposal: None` — the phased conversation works with the current model.
- All context-gathering tools already exist (`list_org_integrations`, `list_components`, `describe_component`, `search_decision_patterns`, `get_canvas_memory`).
- The frontend chat UI already renders answers without proposals.
- The SSE streaming protocol handles both proposal and no-proposal responses.

## Future Considerations (Out of Scope)

- **Visual flow preview**: Frontend could render the Phase 2 "flow outline" as a visual graph preview, not just text.
- **Conversation style memory**: The agent could remember whether the user prefers terse or detailed interactions via canvas memory.
- **Phase indicators**: The UI could show which phase the agent is in ("Understanding your goal...", "Confirming the approach...", "Building the proposal...").
- **Multi-select chips**: Allow users to select multiple options (e.g., "I want both Slack and email notifications").

## Success Criteria

1. For vague prompts on empty canvases, the agent asks clarifying questions before proposing.
2. For medium-specificity prompts, the agent presents its integration/flow plan and waits for confirmation.
3. For specific instructions on populated canvases, the agent still proposes immediately with no unnecessary delay.
4. The agent never proposes components from integrations the org has not connected.
5. When integrations are missing, the user can connect them without leaving the chat.
6. During exploration phases, users can click option chips instead of typing, reducing friction and guiding the conversation.
7. Users report fewer discard cycles and higher first-proposal apply rates.
