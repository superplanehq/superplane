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
