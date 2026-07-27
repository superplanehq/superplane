# Staging and Versioning

## Overview

SuperPlane separates in-progress App edits from the live workflow. Staging and
versioning let collaborators save drafts, review differences, commit changes,
publish a chosen version, and inspect earlier versions without disrupting
running automation.

## Terminology

- **Staging:** Saved App changes that are not yet live.
- **Commit:** A named, immutable App version created from staged changes.
- **Publish:** Promote a committed version to become the live workflow.

## Requirements

### REQ-STAGE-001: Isolate staged changes

**User story:** As a workflow builder, I want draft edits isolated from the
live App, so that I can iterate without changing active automation.

**Acceptance criteria:**

- **AC-STAGE-001.1:** When the builder saves an edit in staging, SuperPlane
  shall show it in edit mode while the live view continues to show the
  published version.
- **AC-STAGE-001.2:** When staged changes are discarded, SuperPlane shall
  restore the builder's draft view to the current committed state.

### REQ-STAGE-002: Collaborate on a shared draft

**User story:** As a collaborating workflow builder, I want to see current
staged changes from other editors, so that we do not unknowingly overwrite one
another.

**Acceptance criteria:**

- **AC-STAGE-002.1:** When one editor saves staged changes, another authorized
  editor shall be able to observe the updated staged App.
- **AC-STAGE-002.2:** When concurrent edits cannot be safely reconciled,
  SuperPlane shall surface a conflict or refresh requirement rather than
  silently losing a collaborator's changes.

### REQ-STAGE-003: Commit and publish

**User story:** As an App publisher, I want to review, commit, and publish
staged changes, so that live automation changes intentionally.

**Acceptance criteria:**

- **AC-STAGE-003.1:** When valid staged changes are committed, SuperPlane shall
  create a version that can be identified in version history.
- **AC-STAGE-003.2:** When an authorized publisher promotes that version,
  SuperPlane shall make it the live Canvas and clear the promoted staging
  state.

### REQ-STAGE-004: Inspect historical versions

**User story:** As an operator, I want to preview older App versions, so that I
can understand prior behavior and run history.

**Acceptance criteria:**

- **AC-STAGE-004.1:** When the operator selects a historical version,
  SuperPlane shall render that version without presenting it as the live
  Canvas.
- **AC-STAGE-004.2:** When version history exceeds the initial view,
  SuperPlane shall allow additional versions to be reached without losing the
  current selection.

## Traceability

- **API evidence:** [staging and version RPCs](../../protos/canvases.proto)
- **UI evidence:** [editable workflow snapshot](../../web_src/src/pages/app/lib/editable-workflow-snapshot.ts)
  and [staging indicators](../../web_src/src/pages/app/lib/local-staging-indicators.ts)
- **Behavior evidence:** [commit and publish](../../test/e2e/canvas_staging_commit_publish_test.go),
  [live isolation](../../test/e2e/canvas_staging_live_view_test.go),
  [multi-user staging](../../test/e2e/canvas_multi_user_staging_test.go), and
  [version preview](../../test/e2e/canvas_version_preview_test.go)
- **Feature blueprint:** [Canvas Authoring and Versioning](../blueprints/features/canvas-authoring-and-versioning.feature.md)

## Open Questions

- What conflict-resolution choices should editors receive for concurrent
  staging?
- Can a historical version be restored directly, or only copied into staging?
