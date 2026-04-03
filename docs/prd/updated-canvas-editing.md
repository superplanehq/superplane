# Updated canvas editing (draft vs live)

## Overview

This PRD defines **draft-first canvas editing**: you do not **edit the same graph that production is running**. Work happens on an **unpublished draft** in **edit mode**; saves go to the draft. **Going live** either **publishes** the draft (when change management is off) or **submits for review** (when change management is on). **Live** is for observing and operating what is actually live, not for rewiring the graph in place.

**Versioning** remains on for canvases. **Change management** is a separate setting: it only changes what **go live** means, not whether drafts exist.

Detailed behavior lives in engineering specs; this document is the product brief.

## Problem Statement

- Today it is easy to **change the canvas that is also executing**: triggers fire, queue items move, and users still drag nodes or save config. That feels like a **live operation**—hard to reason about, alarming when something breaks, and unclear whether edits applied to what **already ran** or what **is about to**.
- **Draft vs live** should not be **two roles smashed into one screen**; users need a clear split between **the thing that runs** and **the thing you edit**.
- **Going live** should stay **deliberate**; **bad config** should be able to **block publish** until fixed.
- **Version preview** should not feel like stray clicks become edits; after switching versions, the view should **fit the graph** (e.g. zoom “fit all”) even when nodes appear slightly late.

## Goals

1. **Separate execution from editing:** live graph is not the direct target of structural edits while you are “just editing.”
2. **Edit mode = draft:** saves persist to the **draft**; exit paths support **save and leave** or **discard** back to live, with prompts when there is unsaved work.
3. **Live mode = read-only structure** for the live definition: users can **run** and **dig in**, not rewire the live graph in place.
4. **Publishing:** with change management **off**, one action **promotes draft to live**; with change management **on**, publish is **submit for review**, not straight to live.
5. **Settings:** surface **change management** in canvas settings alongside existing policy.
6. **Pre-publish gate:** if there are **blocking warnings** on the graph, **publish is disabled** and the UI shows what is wrong **next to the button**.
7. **Versions UX:** versions panel and **Versions** control only in **edit**; on **live**, users get **Edit**, not the full version browser. In edit, picking another version (draft, live snapshot, history, open CRs—same rules as today) ends with **fit-all** behavior. When previewing **read-only** content, a **normal node click** does **not** open the component sidebar (avoid accidental config).
8. **Rollback / edit-from-version:** keep existing support for starting a draft from an **older published** version.
9. **New canvas:** creation flow ends in **edit** with the **component sidebar** open.

## Non-Goals

- **Dry-run**, simulated test runs, per-node test, and run-console affordances for exercising a **draft** without promoting it (see **Follow-ups**).
- **Large** runs/queue redesign beyond what **draft vs live** requires.

## Primary Users

- People who think like **CI pipelines, deployment tools, and workflow builders**: change the pipeline in one place; **watch runs, logs, and outcomes** elsewhere.
- Users who expect **editing configuration** and **execution detail** to be **different modes or screens**, not one view where the graph is both “what I change” and “what is live right now.”
- **Operators** who mostly want the live view can stay on **live**; **builders** step into **edit** when changing the workflow—similar to **pipeline editor** vs **run** views.

## User Stories

1. As an editor, I work on a **draft** in **edit mode** so I am not editing the **live-running** graph.
2. As an editor, I can **save** my draft, **leave** edit mode, or **discard** with clear prompts when there are unsaved changes.
3. If a **draft already exists** when I enter edit, I can **continue the draft** or **start over from live**.
4. As a viewer on **live**, the **structure is read-only**; I can still **run** and inspect execution without rewiring live in place.
5. As an editor with change management **off**, I can **publish** to promote the draft to live.
6. As an editor with change management **on**, **publish** means **submit for review**, not immediate live.
7. As an editor, I see **blocking graph warnings** next to publish and **cannot publish** until they are resolved.
8. As an editor, I use **Versions** only in edit; on live I see **Edit** instead of the full version browser.
9. As an editor previewing a read-only version, I do **not** open the component sidebar from a normal node click.
10. As a creator, finishing **new canvas** setup lands me in **edit** with the **component sidebar** open.

## Functional Requirements

### Live vs edit

| | **Live** | **Edit** |
|---|----------|----------|
| Structure | **Read-only** (live definition) | **Draft**; saves go to draft |
| Purpose | Run, observe, dig into what is live | Change workflow safely |
| Versions UI | **Edit** entry; not full version browser | Full version browser + panel |

### Draft lifecycle and exit

- Entering **edit** works with a **draft**; saves target the **draft**.
- Exiting: **save and leave**, or **discard** and return to live; **unsaved** work triggers prompts.
- If a draft **already exists** when entering edit: user chooses **continue draft** or **start over from live**.
- Users can also create a new draft from a **previous published version**.

### Publishing

- **Change management off:** one **publish** promotes draft → live.
- **Change management on:** **publish** = **submit for review** (not direct to live).

### Settings

- **Change management** appears in **canvas settings** where policy is already exposed.

### Before publish

- **Blocking warnings** on the graph **disable publish**.
- Copy or indicators **next to the publish action** explain what is wrong.

### Versions (edit mode)

- **Versions** panel and **Versions** button **only in edit**.
- User can select versions to inspect (draft, live snapshot, history, open CRs—**same product rules as today**).
- After switching version, viewport **fits all components** (equivalent to zoom fit-all), including when nodes appear slightly late.
- **Read-only preview:** normal node click **does not** open the component sidebar.

### Rollback / edit-from-version

- Preserve existing flows to **start a draft from an older published version** where the product already supports it.

### New canvas

- **Create** flow completes in **edit mode** with **component sidebar** open.

## Acceptance Criteria

1. Users can distinguish **live** vs **edit** without internal coaching (“no Discord translation” bar for internal dogfood).
2. **Live** does not allow the same **in-place structural editing** model as before; **edit** is where draft changes happen.
3. **Publish** behavior matches **change management** setting (direct promote vs submit for review).
4. **Publish** is blocked when **blocking warnings** exist; UI surfaces **why** next to the action.
5. **Versions** affordances match **edit-only** vs **live** rules above.
6. After **version switch** in edit, graph **fits** in view as specified.
7. **Read-only version preview** does not open component sidebar on ordinary node click.
8. **New canvas** creation ends in **edit** with component sidebar **open**.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Draft-first **slows** “try it on live” iteration | Communicate expectation; schedule **test events** and **dry run from edit** (see Follow-ups). |
| Users confuse **draft** vs **live** | Clear mode labels, entry/exit flows, and onboarding; measure internal + external comprehension. |
| **Publish** blocked by warnings feels blocking | Surface **actionable** messages next to publish; keep non-blocking guidance distinct from **blocking** rules in spec. |

## Follow-ups

- **Test events** and **dry run from edit mode** so builders can send **sample payloads** through the **draft** graph **without** promoting it—closes the prototyping gap without undoing draft/live separation.
- **Iteration speed** and dry-run success criteria stay tied to this follow-up, not to the draft/live launch checklist alone.

## How we’ll know it worked

- **Internal:** Team uses the model **without** constantly explaining draft vs live.
- **Prod dogfood:** Ship, use for ~1 week, watch for confusion, workarounds, and “feels wrong” feedback.
- **External sessions:** Reviews, hackathons—do people **understand the UI** without random clicking and confusion?
- **Cloud metrics:** Usage and funnel signals from the **hosted** instance, read **with** qualitative signals above.

## Reference

- [Video overview](https://drive.google.com/file/d/1M16kKRp_g9oE61m8FTnK7tonisFPk9Zx/view?usp=sharing)
- [Prototype branch (not mergeable, illustrative only)](https://github.com/superplanehq/superplane/tree/feat--new-canvas-edit)

# Technical Proposal

## High-level architecture

This section describes only the architectural changes introduced by this proposal.

### Data model changes

- The edited graph is no longer treated as the same thing as the live graph in the product flow.
- Entering **edit mode** always attaches the user to a **draft version**, and all edit-mode writes target that draft.
- Exiting edit mode no longer depends on a manual save as the persistence boundary; the draft is the persisted working state.
- Versioning becomes unconditional.
- The persisted policy is `enable_change_request`.
- The policy exists at the canvas level and at the organization level.
- Organization policy overrides canvas policy.

### Database changes

- **V1 has database schema changes**.
- Remove canvas-level `versioning_enabled`.
- Remove organization-level `versioning_enabled` for canvases.
- Add a canvas-level `enable_change_request` flag.
- Add an organization-level `enable_change_request` flag.
- Draft/version tables stay the same.
- That means:
  - **no new tables** for v1,
  - **one new canvas policy column** for v1,
  - **one new organization policy column** for v1,
  - **existing versioning columns are removed**,
  - **existing canvases are migrated to `enable_change_request = false`**,
  - **existing organizations are migrated to `enable_change_request = false`**.

Example: schema delta for v1

```sql
ALTER TABLE workflows
  DROP COLUMN versioning_enabled;

ALTER TABLE workflows
  ADD COLUMN enable_change_request boolean NOT NULL DEFAULT false;

ALTER TABLE organizations
  ADD COLUMN enable_change_request boolean NOT NULL DEFAULT false;

ALTER TABLE organizations
  DROP COLUMN versioning_enabled;
```

### API behavior changes

- Edit-mode saves continue to use `UpdateCanvasVersion`.
- The change is in how the UI calls it: in edit mode it always sends `version_id`.
- Entering edit mode continues to reuse the current draft when it exists and create a draft when it does not.
- The user can also create a draft from a selected previous published version.
- “Go live” splits into two behaviors:
  - **Publish draft directly** when `enable_change_request = false`.
  - **Create/submit a change request** when `enable_change_request = true`.
- The publish action enforces a **pre-publish validation gate** so blocking graph issues prevent promotion to live.
- Version browsing becomes **edit-mode scoped**:
  - live mode shows the entry point into edit,
  - edit mode shows versions, draft, history, and open change requests.

### API changes

- `UpdateCanvasRequest` changes.
- Remove `versioning_enabled`.
- Add `enable_change_request`.

```proto
message UpdateCanvasRequest {
  string id = 1;
  optional string name = 2;
  optional string description = 3;
  optional bool enable_change_request = 4;
  optional CanvasChangeRequestApprovalConfig change_request_approval_config = 5;
}
```

- `Canvas.Metadata` changes in the same way.

```proto
message Canvas {
  message Metadata {
    string id = 1;
    string organization_id = 2;
    string name = 3;
    string description = 4;
    google.protobuf.Timestamp created_at = 5;
    google.protobuf.Timestamp updated_at = 6;
    UserRef created_by = 7;
    bool is_template = 8;
    bool enable_change_request = 9;
    CanvasChangeRequestApprovalConfig change_request_approval_config = 10;
  }
}
```

- Organization settings add the same field.

```proto
message UpdateOrganizationRequest {
  // ...
  optional bool enable_change_request = <new_field_number>;
}
```

- `UpdateCanvasVersionRequest` stays unchanged.
- The frontend change is simple: in edit mode it always calls this endpoint with `version_id`.
- The empty-`version_id` live-update path is no longer part of the draft-first canvas flow.

```http
PUT /api/v1/canvases/{canvas_id}/versions/{version_id}
Content-Type: application/json

{
  "canvasId": "canvas_123",
  "versionId": "ver_draft_456",
  "canvas": {
    "metadata": {
      "name": "Incident Router",
      "description": "Routes incidents by severity"
    },
    "spec": {
      "nodes": [ ... ],
      "edges": [ ... ]
    }
  }
}
```

- Entering edit mode continues to use the existing draft creation endpoint.
- No request change is needed there.
- Creating a draft from a previous version needs API support to specify the source version.

Example: create draft from a previous published version

```proto
message CreateCanvasVersionRequest {
  string canvas_id = 1;
  string source_version_id = 2;
}
```

- If `source_version_id` is empty, the draft is created from the current live version.
- If `source_version_id` is set, the draft is created from that published version.

```http
POST /api/v1/canvases/{canvas_id}/versions
```

- `CreateCanvasChangeRequest` and `ActOnCanvasChangeRequest` stay unchanged for the change-request flow.
- The new API work is the direct-publish flow. That flow needs a direct draft publish endpoint because the current API only publishes through change requests.
- The CLI changes too.
- Canvas settings commands must stop reading and writing `versioning_enabled`.
- Canvas settings commands must read and write `enable_change_request`.
- Canvas publish commands must follow the new split:
  - publish draft directly when `enable_change_request = false`,
  - create and act on change requests when `enable_change_request = true`.

Example: new direct draft publish API

```proto
rpc PublishCanvasVersion(PublishCanvasVersionRequest) returns (PublishCanvasVersionResponse) {
  option (google.api.http) = {
    patch: "/api/v1/canvases/{canvas_id}/versions/{version_id}/publish"
    body: "*"
  };
}

message PublishCanvasVersionRequest {
  string canvas_id = 1;
  string version_id = 2;
}

message PublishCanvasVersionResponse {
  CanvasVersion version = 1;
  Canvas canvas = 2;
}
```

- This endpoint calls the existing draft-publish backend logic and moves the draft to live when `enable_change_request = false`.
- The backend model helper already exists. The missing piece is wiring it into the API/service layer.
- This endpoint is allowed only when `enable_change_request = false`.
- If `enable_change_request = true`, the API returns `FailedPrecondition` and the user must create a change request instead.

Example: direct-publish path

```text
Enter edit mode
-> create/reuse draft version
-> autosave draft with PUT /versions/{version_id}
-> PATCH /versions/{version_id}/publish
```

Example: change-request path

```text
Enter edit mode
-> create/reuse draft version
-> autosave draft with PUT /versions/{version_id}
-> POST /change-requests
-> PATCH /change-requests/{id}/actions { action: ACTION_APPROVE }
-> PATCH /change-requests/{id}/actions { action: ACTION_PUBLISH }
```

- Publish validation is a real API change.
- It does not exist today.
- Expose it on `CanvasVersion` so the UI gets publish blockers from the existing version fetch/save flow instead of calling a separate check endpoint.

Example: `CanvasVersion` response delta

```proto
message CanvasVersion {
  message PublishBlocker {
    string node_id = 1;
    string message = 2;
  }

  message Metadata {
    string id = 1;
    string canvas_id = 2;
    UserRef owner = 4;
    bool is_published = 6;
    google.protobuf.Timestamp published_at = 7;
    google.protobuf.Timestamp created_at = 8;
    google.protobuf.Timestamp updated_at = 9;
    bool can_publish = 10;
  }

  Metadata metadata = 1;
  Canvas.Spec spec = 2;
  repeated PublishBlocker publish_blockers = 3;
}
```

- Example response

```json
{
  "metadata": {
    "id": "ver_draft_456",
    "canvasId": "canvas_123",
    "canPublish": false
  },
  "publishBlockers": [
    {
      "nodeId": "slack-send-message",
      "message": "Slack channel is required"
    }
  ]
}
```
