# Canvas Authoring

## Overview

Canvas authoring lets workflow builders compose an App's graph from triggers,
components, connections, configuration, notes, and files. The experience must
make graph changes understandable, valid, and safe for collaborators.

## Terminology

- **Node:** A trigger or component placed on the Canvas.
- **Channel:** A named output path that connects one node to another.
- **Files view:** The App view that exposes repository-backed definitions such
  as `canvas.yaml`.

## Requirements

### REQ-AUTHOR-001: Build and edit a workflow graph

**User story:** As a workflow builder, I want to add, name, arrange, configure,
connect, and remove nodes, so that the Canvas represents my intended process.

**Acceptance criteria:**

- **AC-AUTHOR-001.1:** When a builder adds multiple nodes of the same kind,
  SuperPlane shall give each node a distinguishable name and display it on the
  Canvas.
- **AC-AUTHOR-001.2:** When a builder removes a node or connection and saves
  the change, SuperPlane shall no longer show that graph element in the
  effective draft.

### REQ-AUTHOR-002: Configure data flow

**User story:** As a workflow builder, I want configuration assistance based on
upstream data, so that downstream nodes can consume event payloads correctly.

**Acceptance criteria:**

- **AC-AUTHOR-002.1:** When a builder edits an expression with upstream nodes,
  SuperPlane shall offer references to available node data.
- **AC-AUTHOR-002.2:** When node configuration is invalid, SuperPlane shall
  identify the affected configuration and prevent it from becoming the live
  workflow.

### REQ-AUTHOR-003: Switch among App representations

**User story:** As a workflow builder, I want to inspect the visual Canvas and
its definition files, so that I can understand the same App in the most useful
form.

**Acceptance criteria:**

- **AC-AUTHOR-003.1:** When the builder opens the Files view, SuperPlane shall
  show the App's available definition files and the current Canvas definition.
- **AC-AUTHOR-003.2:** When the builder returns from Files to the Canvas,
  SuperPlane shall preserve the current App and its saved authoring state.

### REQ-AUTHOR-004: Enforce authoring permissions

**User story:** As a read-only collaborator, I want to inspect an App without
accidentally changing it, so that collaboration does not alter production
behavior.

**Acceptance criteria:**

- **AC-AUTHOR-004.1:** When a collaborator has read but not update permission,
  SuperPlane shall allow App inspection while withholding or disabling graph
  mutation actions.
- **AC-AUTHOR-004.2:** When that collaborator attempts an update through a
  direct request, SuperPlane shall reject it without modifying the Canvas.

## Traceability

- **Product context:** [Canvas and component concepts](../../README.md#how-it-works)
- **API evidence:** [Canvas service](../../protos/canvases.proto),
  [component catalog](../../protos/components.proto), and
  [trigger catalog](../../protos/triggers.proto)
- **UI evidence:** [App page](../../web_src/src/pages/app/AppDefaultTabGate.tsx)
  and [workflow file helpers](../../web_src/src/pages/app/lib/workflow-spec-files.ts)
- **Behavior evidence:** [Canvas authoring and Files view](../../test/e2e/canvas_page_test.go),
  [autosave](../../test/e2e/canvas_auto_save_test.go), and
  [permission guards](../../test/e2e/canvas_permission_guards_test.go)
- **Feature blueprint:** [Canvas Authoring and Versioning](../blueprints/features/canvas-authoring-and-versioning.feature.md)

## Open Questions

- Which Canvas edits must support undo and redo across browser sessions?
- What graph-size limits should be communicated before authoring degrades?
