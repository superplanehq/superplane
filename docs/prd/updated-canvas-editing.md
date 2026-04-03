## 1. What ships

Draft-first canvases so you’re not **editing the graph that production is running**. You work on an unpublished draft in edit mode, save there, then publish to live if change management is off, or go through change request if it’s on. Live is for running and watching what’s actually live, not for rewiring it in place.

Versioning stays on for canvases. Change management is its own setting: it only changes what “go live” means.

---

## 2. Why (main problem)

Today it’s easy to end up **changing the same canvas that’s also executing**. Triggers fire, queue items move, and you’re still dragging nodes or saving config. That reads like a **live operation**: hard to reason about, scary when something breaks, and unclear whether you just edited what already ran or what’s about to. We’re separating **the thing that runs** from **the thing you’re editing** so that confusion goes away.

Also:

- Clear draft vs live, not two roles smashed into one screen.
- Going live stays a deliberate step; bad config can block publish until fixed.
- New canvas lands in edit with building blocks ready.
- Version preview shouldn’t feel like stray clicks turn into edits; switching versions re-zooms to fit the graph.

---

## 3. Users

People who already think in terms of **CI pipelines, deployment tools, and other workflow builders**: you define or change the pipeline in one place, and you **watch runs, logs, and outcomes somewhere else**. They expect editing configuration and digging into execution detail to be **two different modes or screens**, not one mushy view where the graph is both “what I’m changing” and “what’s live right now.”

That’s who draft vs live is for. Builders who mostly want the operational view can stay on live; when they need to change the workflow they step into edit, same mental model as opening a pipeline editor vs opening a run.

---

## 4. Behavior

**Live vs edit**  
Live: structure is read-only, you can run and dig in. Edit: your draft, saves go to the draft. Exit: save and leave, or discard and back to live, with prompts when there’s unsaved stuff. If a draft already exists when you hit Edit, choose continue draft or start over from live.

**Publishing**  
CM off: one publish action promotes the draft to live. CM on: the publish button is submit for review, not straight to live.

**Settings**  
Change management shows up in canvas settings where we already expose policy.

**Before publish**  
If there are blocking warnings on the graph, publish is disabled and we show what’s wrong next to the button.

**Versions**  
Versions panel and the Versions button only show in edit mode. On live you get Edit, not the full version browser.

In edit, you can pick another version to look at (draft, live snapshot, history, open CRs, same rules as today). After you switch, the view fits all components like the zoom “fit all”, even if nodes showed up a beat late.

When you’re previewing something you can’t edit, we don’t open the component sidebar from a normal node click so you don’t drift into config by mistake.

Rollback / edit-from-version: still where we already support starting a draft from an older published version.

**New canvas**  
Create flow ends in edit mode with component sidebar open

---

## 5. Tradeoffs (what we lose)

Draft-first tightens safety but **slows the old “try it on live” loop**. Live is what actually runs; the draft is not. So to exercise a change against real triggers, queues, and downstream behavior you mostly have to **publish (or complete review)** first. That’s fine for careful rollouts, rough for fast prototyping and “does this even fire?” checks.

**Follow-up we should schedule next:** **test events** plus **dry run from edit mode**, so builders can send sample payloads through the draft graph without promoting it. That closes the gap without undoing the draft/live split.

---

## 6. Not in this branch

Dry-run, simulated test runs, per-node test, and run-console affordances for that. Same as section 5: intentional deferral until the follow-up above. No big runs/queue redesign beyond what draft vs live already needs.

---

## 7. Did it work

How we’ll tell:

- **Internal:** Our own team uses the new model **without anyone having to explain it**. If we’re still translating draft vs live in Discord, we’re not there.
- **Prod dogfood:** Ship to production, use it ourselves for a week, and see **whether anything bubbles up** (confusion, workarounds, “this feels wrong” feedback).
- **Sessions with outsiders:** Product reviews, hackathons, similar. Do people **get the UI on their own** or do they wander, click randomly, and ask what they’re looking at?
- **Cloud metrics:** Pull usage and funnel-style signals from the **hosted cloud instance** and read them alongside the qualitative stuff above.

Iterate speed and dry run stay separate from this checklist until that follow-up ships.

---

Details live in specs; this is the short version.


## 8. Refference:
- [Video overview](https://drive.google.com/file/d/1M16kKRp_g9oE61m8FTnK7tonisFPk9Zx/view?usp=sharing)
- [Prototype branch (not mergable, just ilustrative)](https://github.com/superplanehq/superplane/tree/feat--new-canvas-edit)
