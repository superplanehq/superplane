# Canvas Versioning

## Goals

- Keep one live version per canvas for execution.
- Track immutable revisions for every live publish.
- Store draft revisions per user without affecting live execution.
- Support conflict detection by base live version.

## Data model

- `workflows.live_version_id`: points to the currently published revision.
- `workflow_versions`: immutable canvas snapshots (`nodes`, `edges`, owner, revision, publish state, base revision).
- `workflow_user_drafts`: single draft pointer per `(workflow_id, user_id)`.

## Runtime contract

- Execution always reads canvas graph from `workflows.live_version_id`.
- If `live_version_id` is not set (legacy state), runtime falls back to `workflows.nodes`/`workflows.edges`.

## Current backend flow

- `CreateCanvas` creates the workflow and writes revision `1` as published.
- `CreateCanvasVersion` creates or returns the user draft version.
- `UpdateCanvasVersion` updates only the draft graph (no runtime setup).
- `PublishCanvasChangeRequest` applies setup/upserts runtime nodes and sets that version as live.
- `UpdateCanvas` still exists for compatibility and keeps its previous behavior.
- Draft APIs are modeled in `pkg/models/canvas_version.go` with:
  - `SaveCanvasDraftInTransaction`
  - `PublishCanvasDraftInTransaction`

## Conflict detection

`PublishCanvasDraftInTransaction` fails with `ErrCanvasVersionConflict` when draft `based_on_version_id` does not match current live revision.

## Next backend steps

- Add dedicated RPCs for draft save/publish and draft-vs-live diff.
- Return version metadata in canvas responses.
- Add org-level draft execution policy and enforce execution only for live revision by default.
