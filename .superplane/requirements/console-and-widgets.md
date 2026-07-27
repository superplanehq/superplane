# Console and Widgets

## Overview

Each App can provide a Console that turns workflow state into an operational
view. Authorized authors compose panels and live data; operators inspect status
and invoke explicitly exposed workflow controls without editing the Canvas.

## Terminology

- **Console:** The user-facing operational view belonging to one App.
- **Panel:** A draggable Console unit such as text, nodes, table, board, chart,
  number, or scorecard.
- **Widget data source:** Canvas memory, executions, or runs used by a panel.

## Requirements

### REQ-CON-001: View an App Console

**User story:** As an operator, I want a purpose-built App Console, so that I
can understand current workflow state without reading the Canvas graph.

**Acceptance criteria:**

- **AC-CON-001.1:** When an App has configured panels, SuperPlane shall render
  them in Console mode using the saved layout.
- **AC-CON-001.2:** When an operator has read-only access, SuperPlane shall
  allow Console viewing while preventing edit and run actions that require
  update permission.

### REQ-CON-002: Author and exchange Console definitions

**User story:** As an App author, I want to add, edit, arrange, import, and
export Console panels, so that the operational view can be maintained and
shared.

**Acceptance criteria:**

- **AC-CON-002.1:** When an authorized author changes panels or layout,
  SuperPlane shall persist and render the updated Console.
- **AC-CON-002.2:** When imported Console content is invalid, SuperPlane shall
  report the validation problem and leave the existing Console unchanged.

### REQ-CON-003: Present live operational data

**User story:** As an operator, I want tables, boards, charts, numbers, and
scorecards based on App data, so that I can monitor status and trends.

**Acceptance criteria:**

- **AC-CON-003.1:** When a panel references an available memory, execution, or
  run data source, SuperPlane shall render matching data according to the
  panel's filters and presentation.
- **AC-CON-003.2:** When a referenced value is absent, SuperPlane shall render
  a stable empty or unavailable state instead of failing the entire Console.

### REQ-CON-004: Invoke guarded Console actions

**User story:** As an authorized operator, I want to run eligible trigger
actions from Console panels, so that routine operations are available near
their context.

**Acceptance criteria:**

- **AC-CON-004.1:** When an operator invokes an eligible action, SuperPlane
  shall collect declared parameters or confirmation and start the associated
  trigger with the resolved payload.
- **AC-CON-004.2:** When concurrent invocation is disallowed and matching work
  is already active, SuperPlane shall prevent a duplicate submission and
  explain why.

## Traceability

- **Product context:** [Console dashboards](../../README.md#what-it-does)
- **Detailed behavior:** [Console and widgets PRD](../../docs/prd/console-and-widgets.md)
- **API evidence:** [widget catalog](../../protos/widgets.proto) and
  [Canvas run and memory data](../../protos/canvases.proto)
- **UI evidence:** [Console implementation](../../web_src/src/pages/app/console/)
- **Feature blueprint:** [Console and Widgets](../blueprints/features/console-and-widgets.feature.md)

## Open Questions

- Which panel types and data limits form the stable compatibility contract?
- Should Console changes share the App's staging and publishing lifecycle?
