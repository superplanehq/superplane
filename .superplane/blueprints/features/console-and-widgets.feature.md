# Console and Widgets Feature Blueprint

## Feature Summary

Users turn a canvas into an operational console of markdown, node, table, board,
chart, number, and scorecard panels backed by memory, executions, and runs.
Authorized users edit/import/export layouts and invoke configured trigger row
actions. This feature implements the
[Console and Widgets](../../requirements/console-and-widgets.md) requirements.

Evidence: [`docs/prd/console-and-widgets.md`](../../../docs/prd/console-and-widgets.md),
[`pkg/models/console.go`](../../../pkg/models/console.go),
[`web_src/src/pages/app/console/`](../../../web_src/src/pages/app/console/), and
[`protos/canvases.proto`](../../../protos/canvases.proto).

## Component Blueprint Composition

- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  canvas APIs expose console/version/memory/run/execution data and realtime
  updates invalidate displayed state.
- [Git Staging and Versioning](../components/git-staging-and-versioning.component.md):
  console YAML/layout edits share staging and the commit boundary with canvas
  edits.
- [Workflow Execution](../components/workflow-execution.component.md):
  run/execution rows and node status are console data, and node/row actions emit
  normal trigger events.
- [Registry and Runtime](../components/registry-and-runtime.component.md):
  widget definitions provide configurable data/presentation contracts.

## Feature-Specific Flow

The frontend enters console mode, resolves the effective staged console, and
renders a draggable grid. `useWidgetData` fetches memory, executions, or runs,
derives rows, filters/transforms them with bounded expressions/templates, and
renders typed panels. Editors stage panel/layout changes. Trigger actions merge
configured payload templates with a selected row and invoke the authorized
trigger endpoint.

## System Contracts

- A console belongs to one canvas; editable changes follow stage then commit.
- Read permission allows viewing/export; update permission gates editing and
  runtime actions.
- Frontend and backend/YAML validators must agree on panel shapes.
- Import is replace-all for panels/layout, and invalid YAML must not partially
  apply.
- Data sources are read models over canonical memory/run/execution APIs.
- Expressions/templates operate on row data and cannot bypass API
  authorization.
- Realtime messages can be missed; widgets refetch canonical data.

## Requirement Coverage

- **REQ-CON-001:** Console mode renders the saved panel grid and separates read
  access from editing and trigger invocation permissions.
- **REQ-CON-002:** Staged console YAML and layout APIs support panel editing,
  import, export, validation, and atomic replacement.
- **REQ-CON-003:** `useWidgetData` reads Canvas memory, executions, and runs,
  then applies bounded filters and presentation to typed panels.
- **REQ-CON-004:** Row and trigger actions resolve templates, collect declared
  inputs or confirmation, enforce update permission, and invoke trigger hooks.

## Architecture Decision Records

### ADR-001: Store console configuration with workflow versioned content

**Context:** Operational views must evolve with the workflow nodes and data they
reference.

**Decision:** Edit console configuration through the same staging/commit loop as
the canvas.

**Consequences:** Console and workflow can be reviewed/versioned together;
uncommitted console changes remain user-specific.

## Open Questions

- What server-side query/filter support is needed as run and execution data
  grows?
- How should broken node names or field paths be diagnosed after workflow
  changes?
