# Canvas Control Tab

## Overview

This PRD defines a third tab on the canvas page in SuperPlane: **Control**.

The **Control** tab lets users build a simple operator-facing interface directly on top of a canvas. In v1, users can render markdown text, tables, and buttons. Button clicks trigger manual canvas runs with a payload, and the UI can read/write canvas memory so values update dynamically between interactions.

## Problem Statement

Canvas execution today is optimized for workflow construction and run inspection, but there is no lightweight in-product interface for:

- Presenting operational context to users in a structured layout.
- Triggering common manual runs with one click.
- Displaying and updating state over time without leaving the canvas page.

Users currently need external tooling or manual parameter entry for these tasks, which increases friction and slows repeated operational workflows.

## Goals

1. Add a third canvas page tab named **Control**.
2. Let users compose a basic UI with markdown, tables, and buttons.
3. Allow button clicks to invoke manual runs on the current canvas.
4. Integrate canvas memory so UI values can be dynamic and stateful.
5. Keep v1 implementation straightforward, predictable, and safe.

## Non-Goals

- Building a full no-code app builder with arbitrary layouts/widgets.
- Replacing the existing canvas graph editor.
- Supporting custom JavaScript execution in the browser.
- Cross-canvas shared state in v1.
- Public unauthenticated UI pages in v1.

## Primary Users

- **Canvas Operators:** Trigger known workflows quickly through a focused UI.
- **Workflow Builders:** Expose a simplified interface for teammates.
- **Internal Teams:** Run recurring operational actions without editing the canvas graph.

## User Stories

1. As a user, I can open the Control tab and see a simple interface tied to the current canvas.
2. As a builder, I can add markdown and tables to explain actions and show context.
3. As a user, I can click a button to run the canvas manually with predefined/dynamic inputs.
4. As a builder, I can read from memory to show the latest state in UI elements.
5. As a user, I can trigger actions that update memory and immediately see refreshed values.

## Functional Requirements

### Canvas Tab Integration

- Add a third tab on the canvas page: **Control**.
- Preserve existing tabs and behavior.
- Reuse existing canvas page authorization:
  - Users with edit permission can edit Control tab config.
  - Users with run permission can click action buttons that trigger manual runs.

### Control Configuration Model

- Control is configured as an ordered list of blocks.
- v1 supported block types:
  - `markdown`
  - `table`
  - `button`
- Configuration is stored as part of canvas metadata (exact storage contract TBD by implementation).

### Markdown Block

- Supports standard markdown rendering (headings, paragraphs, lists, links, inline code, code blocks).
- Supports templated placeholders that resolve from a runtime context (`run input`, `last run output summary`, `memory values`).
- If a placeholder cannot be resolved, render an explicit fallback (for example `-` or empty string by config).

### Table Block

- Supports static rows and dynamic rows.
- Dynamic rows can bind to:
  - A list in memory (`namespace` + optional lookup/filter).
  - A structured value from the latest manual run output.
- Column configuration includes:
  - `key` (source field)
  - `label` (display name)
  - optional formatting mode (`text`, `json`, `timestamp`).

### Button Block

- A button has:
  - `label`
  - optional `style` variant (`primary`, `secondary`, `danger`)
  - optional confirmation text
  - manual run payload template
  - optional post-run memory write mapping
- On click:
  1. Resolve payload template with current runtime context.
  2. Trigger manual run for the current canvas.
  3. Show run status (`pending`, `running`, `succeeded`, `failed`).
  4. If configured and run succeeds, write mapped fields into memory.
  5. Refresh UI-bound memory values.

### Memory Integration

- Control can read from existing canvas memory namespaces.
- Control can write to memory only through configured post-run mappings in v1.
- Memory operations stay scoped to the current canvas.
- Any memory read/write failures should be surfaced in the UI with actionable error text.

### Runtime Context and Templating

- Template sources in v1:
  - `memory.<namespace>...`
  - `run.input...`
  - `run.output...` (last button-triggered run in this session)
- Template engine must not allow arbitrary code execution.
- Missing fields resolve safely (no hard crash of the Control tab).

### Execution and Concurrency Behavior

- Default: a button is disabled while its own run is in progress.
- Optional setting (future flag): allow concurrent clicks for specific buttons.
- Run requests include the identity of the initiating user for audit consistency.

### Error Handling

- If manual run trigger fails, show inline error and keep UI state intact.
- If run fails, show failure status and any available summarized error message.
- If memory refresh fails after a run, show partial-success state ("run succeeded, refresh failed").

## UX Requirements

- Control tab follows existing canvas page visual system.
- Empty state for unconfigured Control includes:
  - Brief explanation of what Control does.
  - CTA for users with edit access to configure first blocks.
- During run:
  - Show loading state on the clicked button.
  - Show non-blocking status indicator for the latest run.
- Buttons with confirmation enabled require explicit confirm before triggering run.

## Security and Authorization

- Reuse existing canvas authorization checks for viewing, editing, and running.
- Do not expose secret configuration values in rendered markdown/table/button templates.
- Apply output sanitization for markdown rendering to prevent script injection.
- Log button-triggered manual runs with user and button identifier metadata for auditability.

## Implementation Scope (v1)

### In Scope

- Third canvas tab: **Control**.
- Block renderer for markdown, table, and button.
- Manual run invocation from button clicks.
- Memory-backed dynamic values for rendering and post-run updates.
- Basic templating and fallback behavior.

### Out of Scope

- Drag-and-drop layout editor.
- Rich widget ecosystem beyond markdown/table/button.
- Complex form inputs (text fields, pickers, file uploads).
- Public external sharing of Control.
- Versioned Control revisions/history.

## Acceptance Criteria

1. Canvas page shows a third tab named **Control**.
2. Users with edit access can configure markdown, table, and button blocks for a canvas.
3. Users with run access can click configured buttons to trigger manual runs on that canvas.
4. Button payload templating can resolve memory and runtime values safely.
5. Control can read memory-backed values and render them in markdown/table blocks.
6. On successful button runs with configured mappings, memory values are updated and reflected in UI.
7. Errors in run trigger, run execution, or memory refresh are surfaced without breaking the tab.
8. Markdown rendering is sanitized and does not allow script execution.

## Success Metrics

- Reduction in average clicks/time to trigger common manual canvas actions.
- Percentage of canvases that configure at least one Control button.
- Success rate of button-triggered manual runs.
- Reuse rate of memory-backed dynamic blocks.
- Lower support volume for "how to run this canvas manually" workflows.

## Risks and Mitigations

- **Risk:** Control becomes too complex without layout constraints.  
  **Mitigation:** keep v1 to a constrained block model and defer advanced composition.

- **Risk:** Incorrect templating leads to bad run payloads.  
  **Mitigation:** add template preview/validation before save and safe fallback rules.

- **Risk:** Memory data races with concurrent runs.  
  **Mitigation:** define deterministic write semantics and display latest-write source.

- **Risk:** Users trigger dangerous actions accidentally.  
  **Mitigation:** support confirmation prompts and clear button labeling conventions.

## Open Questions

1. Should Control configuration be represented as canvas JSON metadata or a dedicated table?
2. Do we need per-button permission scopes beyond existing run/edit permissions?
3. Should table dynamic bindings support pagination in v1 or always render a fixed cap?
4. Should post-run memory writes happen only on success, or optionally on failure too?
5. How should multiple collaborators viewing the same Control tab receive memory update events (polling vs realtime)?
