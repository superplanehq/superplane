# Loop component ticket (container-style)

> NOTE: This ticket describes UX and runtime behavior for a container-style Loop
> component in SuperPlane. Review with design and engineering before build.

## Description
The Loop component lets users repeat a subset of actions on the same canvas. It
is visually different from other components and acts as a container that expands
to wrap its body. Users can drop actions inside it, and it grows to envelope
those components. Triggers must not be allowed inside the Loop body.

This ticket is inspired by:
- https://github.com/superplanehq/superplane/issues/2347
- https://github.com/superplanehq/superplane/issues/1468

## Goals
- Provide a container-style Loop node that holds actions on the main canvas.
- Support repeat-until and for-each execution modes.
- Make the container auto-size to its contents with clear drop affordances.
- Enforce "no triggers inside Loop" in both UI and backend validation.

## Non-goals
- Adding or changing trigger behavior.
- Parallel or concurrent iteration modes (sequential only).
- Building new triggers inside the Loop body.

## UX and visual design
- Loop node is visually distinct (unique border, header, and background).
- Container expands to envelope its body nodes with consistent padding.
- Dragging actions into the container adds them to the Loop body.
- Moving actions out removes them from the Loop body.
- Empty Loop shows a placeholder like "Drop actions here".
- Triggers dropped into the container are rejected with a clear inline message.

## Canvas and graph semantics
- Loop body is a subgraph owned by the Loop node.
- Outside edges connect to Loop entry and exit ports on the container.
- Inside edges connect between actions within the Loop body.
- Direct edges that cross the container boundary are invalid and must be
  re-routed or blocked.

## Configuration
- `items` (optional, expr): evaluates to an array; enables for-each mode.
- `break_when` (optional, expr): stops iteration when true.
- `interval` or `wait` between iterations in repeat-until mode (define config
  shape, defaults, and UI control).
- Validation: `break_when` is required in repeat-until mode.

## Execution model
Loop has two modes depending on `items`:

### Mode A - repeat-until (default when `items` is not set)
- Run the body once immediately, producing `last`.
- Evaluate `break_when` (required in this mode).
  - If false, wait for the configured interval and run again.
  - If true, the Loop completes.

### Mode B - for-each (when `items` is set)
- Evaluate `items` once when the Loop starts.
  - If it is not an array (or is null), the Loop fails with a clear error.
- For `index` from `0..len(items)-1`:
  - Set `each.item = items[index]`.
  - Run the body once, producing `last` for that iteration.
  - If `break_when` is provided, evaluate it after the iteration:
    - If true, stop iterating and complete.
    - If false, continue to next item.

## Evaluation context
Expressions in the Loop (including `break_when`) can access:

- `in`: inbound payload entering the Loop (stable across iterations)
- `last`: most recent body output (changes each iteration)
- `loop`:
  - `loop.iteration` (0-based current iteration index)
  - `loop.elapsed_ms`
  - `loop.started_at`
- `each`:
  - `each.item` (current item when iterating; otherwise null)
  - `each.index` (0-based index when iterating; otherwise -1)
  - `each.total` (len(items) when iterating; otherwise 0)
- `items`: the evaluated array (only when `items` is set)

## Output payload
On completion, Loop emits a structured payload that includes `last` and loop
metadata:

```json
{
  "loop": {
    "status": "completed",
    "iterations": 7,
    "elapsed_ms": 420000,
    "stop_reason": "condition_true"
  },
  "last": {
    "status": 200,
    "body": {
      "error_rate": 0.004,
      "p95_latency_ms": 180,
      "baseline_error_rate": 0.002
    }
  }
}
```

## Validation and constraints
- Triggers must not exist inside Loop body (UI and server validation).
- On import or paste, invalid nodes in the Loop body are rejected or moved out.
- Confirm which node types are allowed inside the Loop (actions, widgets, etc).
- Provide clear errors when the Loop body is empty and no entry action exists.

## Acceptance criteria
- [ ] Loop node is a container with distinct visuals and auto-sizing.
- [ ] Users can drag actions into the Loop body; triggers are rejected.
- [ ] Loop config supports `items` and `break_when` with validation rules.
- [ ] Runtime executes repeat-until and for-each modes per the spec above.
- [ ] Evaluation context and output payload are available as documented.
- [ ] Canvas serialization supports nested Loop body nodes and entry/exit ports.
- [ ] Unit/integration tests cover validation and execution behavior.
- [ ] Documentation updated for component behavior and UX.

## Follow-up tasks
- [ ] Add a template/example canvas that uses Loop.
- [ ] UI/UX review pass for the container design.
- [ ] Review user education (docs, release notes, announcement).

## Open questions
- What is the default interval between iterations?
- Do we need max iterations or max elapsed time safety limits?
- Are nested Loops allowed?
- Are widgets allowed inside the Loop body or only action components?
- How should entry/exit ports render when the container is resized?

## References
- https://github.com/superplanehq/superplane/issues/1468
- https://github.com/superplanehq/superplane/issues/2347
- docs/contributing/component-design.md
- docs/contributing/component-implementations.md
- docs/contributing/integration_and_component_checklist.md
# Loop Component (Visual Container + Execution Model)

## Summary
Introduce a Loop core component in SuperPlane that visually wraps a set of
actions and repeatedly executes that subset of the canvas. The Loop node is a
distinct visual container that expands to envelope its child actions. It must
reject triggers inside the Loop body.

This ticket is inspired by:
- https://github.com/superplanehq/superplane/issues/2347
- https://github.com/superplanehq/superplane/issues/1468

## Goals
- Provide a Loop component that can run a body of actions repeatedly.
- Support two patterns: repeat-until and for-each (iterator).
- Make the Loop node visually distinct from other components.
- Allow users to place actions inside the Loop container and auto-expand its
  bounds as children are added or moved.
- Prevent triggers from being placed inside the Loop body (UI and validation).

## Non-goals
- Adding new triggers or trigger types.
- Automatic conversion of existing canvases into loops.
- Long-term scheduling or cron-like functionality (handled elsewhere).

## UX and Visual Behavior
- The Loop node renders as a container with a clear header (label, icon, and
  optional iteration summary).
- The container is visually distinct (border, background, and padding) from
  standard components.
- The Loop container expands/contract as child actions are added, moved, or
  removed, maintaining consistent padding around contained nodes.
- Drag-and-drop behavior:
  - Actions can be dropped into the Loop body.
  - Triggers cannot be dropped into the Loop body and should show a clear error
    message explaining the restriction.
  - If a trigger is moved into the Loop body via multi-select or paste, block
    the move and keep it outside the container.
- Edges:
  - Entry and exit edges for the Loop node are shown at the container boundary.
  - Internal edges between actions inside the Loop are shown normally.

## Configuration
Loop has two optional config expressions:
- items (optional): expr-lang that evaluates to an array; when set, Loop
  iterates over the array (for-each).
- break_when (optional): expr-lang that decides when to stop (repeat-until or
  early-exit in for-each).

Validation rules:
- If items is not set, break_when is required.
- If items is set, break_when is optional and is evaluated after each iteration.

## Execution Model (inspired by issue 1468)
Loop has two modes depending on items.

Mode A - repeat-until (default when items is not set)
- Run the body once immediately, producing last.
- Evaluate break_when after each iteration.
  - If break_when is false, pause for the configured interval and run again.
  - If break_when is true, complete the Loop.

Mode B - for-each (when items is set)
- Evaluate items once when the node starts, producing items[].
  - If items is null or not an array, fail with a clear error.
- For index from 0..len(items)-1:
  - Set each.item = items[index]
  - Run the body once, producing last
  - If break_when is provided, evaluate it after the iteration:
    - If true, stop iterating and complete.
    - If false, continue to the next item.

## Evaluation Context
Expressions for break_when and other node-level expressions should have access
to:
- in: inbound payload entering the Loop node (stable across iterations)
- last: most recent body output payload
- loop:
  - loop.iteration (0-based index)
  - loop.elapsed_ms
  - loop.started_at
- each:
  - each.item (current item when iterating; null otherwise)
  - each.index (0-based index when iterating; -1 otherwise)
  - each.total (len(items) when iterating; 0 otherwise)
- items: evaluated array (only when items is set)

## Output Payload
On completion, the Loop node emits:
```json
{
  "loop": {
    "status": "completed",
    "iterations": 7,
    "elapsed_ms": 420000,
    "stop_reason": "condition_true"
  },
  "last": {
    "status": 200,
    "body": {
      "error_rate": 0.004
    }
  }
}
```

## Acceptance Criteria
- Loop component is available in the core component list.
- Loop component renders as a distinct container and auto-resizes around
  contained actions.
- Users can add actions inside the Loop; triggers are rejected inside the Loop
  body with clear UX feedback.
- Loop execution matches the mode behaviors and validation rules above.
- Items and break_when expressions are evaluated with the documented context.
- Error messages are clear when items is invalid or break_when is missing.
- Add/update documentation for the Loop component.
- Add automated tests for UI behavior and execution semantics.

## Follow-up Tasks
- Consider templates and examples that demonstrate Loop usage.
- Review if any announcement or docs highlight is needed.
- Confirm any extra UX polish or accessibility adjustments.

## References
- https://github.com/superplanehq/superplane/issues/2347
- https://github.com/superplanehq/superplane/issues/1468
- docs/contributing/component-design.md
- docs/contributing/integration_and_component_checklist.md
