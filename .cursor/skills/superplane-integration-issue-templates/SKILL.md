---
name: superplane-integration-issue-templates
description: When creating or reviewing SuperPlane integration issue content (base integration issues or component trigger/action issues). Use when generating issue templates, drafting issue bodies, or validating issue structure and guidelines.
---

# SuperPlane Integration Issue Templates

Use this skill when creating or reviewing issue content for SuperPlane integrations: **base integration issues** (one per tool) and **component issues** (triggers and actions, one per operation). Follow the integration-issue-conventions rule for IMPORTANT blocks, hierarchy, and title format.

---

## Base integration issue

**Purpose:** One parent issue per tool. Establishes how the tool connects to SuperPlane. All component issues are children of this base.

### Structure

1. **IMPORTANT block** (at top) — use the base version: "Review and rethink the suggested connection method before implementing. If unsure, reach out on **Discord** first."
2. **Title**: `[{Integration Name}] Base` (e.g. `[GitHub] Base`, `[ArgoCD] Base`)
3. **Description**: 2–3 sentences on what the tool does and primary use cases. Always include **Link**: {URL}.
4. **Suggested Connection Method**: Primary auth method (steps to generate credentials, required scopes, what to store in SuperPlane). Alternatives if applicable.
5. **Acceptance Criteria**: has proper tests, documentation, code quality review, functionality review, ui/ux review.
6. **Follow up tasks**: Once all components are done — announcement, outreach, marketplace/docs, templates/examples.
7. **Reference**: Integration & component checklist.

### Guidelines (base)

- **Connection method**: Describe recommended auth (API Token, OAuth 2.0, App Installation, Service Account, Personal Access Token, Webhook Secret). Include where/how to generate credentials and required permissions/scopes.

---

## Component issue (trigger or action)

**Purpose:** One issue per trigger or action. Each is a child of the base integration issue. Parent reference and IMPORTANT block at top (see integration-issue-conventions rule).

### Structure

1. **IMPORTANT block** (at top) — use the component version: "Review and rethink configuration options and output channels before implementing. If unsure, reach out on **Discord** first."
2. **Parent reference**: `**Parent (base integration):** #BASE_ISSUE_NUMBER` (immediately after IMPORTANT block).
3. **Title**: `[{Integration Name}] {Operation Name}` — Title Case; triggers use "On {Event}" (e.g. `[GitHub] On Push`, `[ArgoCD] Sync Application`).
4. **Priority**: `P1 - High` / `P2 - Medium` / `P3 - Low` / `P4 - Lowest` (from agreed prioritization).
5. **Description**: 1–2 sentences on **what the component does to external systems**, not that it exists. No "Enable X to Y as part of SuperPlane workflows."
6. **Use Cases**: 2–3 **specific, realistic scenarios** (e.g. "Send Slack notification when production deployment pipeline completes"). Not generic "Automate {operation} in CI/CD workflow."
7. **Configuration**: All fields with (required/optional), defaults, example values. **Triggers = FILTERS only. Actions = EXECUTION PARAMETERS only.**
8. **Outputs**: Channel(s) with when emitted and what data. See Output channels below.
9. **Acceptance Criteria** (optional): tests, documentation, reviews.
10. **Reference**: Integration & component checklist.

### Triggers vs Actions (critical)

- **Actions** (execute operations): Configuration = **execution parameters** — what resource, how to perform, what data to send. Action config answers: "What should I DO?"
- **Triggers** (listen for events): Configuration = **filters** — what events to listen for, what values to match (repository, status, labels). Trigger config answers: "What events should I LISTEN FOR?"

**Wrong (trigger):** `[ArgoCD] On Sync Completed` with config "Revision", "Prune" — those are for performing a sync, not filtering events.  
**Right (trigger):** "Application Filter", "Sync Status" (Succeeded/Failed), "Health Status".

### Output channels (components)

- **Triggers**: Single **default** channel (outcomes determined downstream).
- **Actions**: Depends on whether the user would model different workflow paths:
  - **Multiple channels** when the result should branch: e.g. `success`/`failed`, or `clear`/`degraded`/`critical` (e.g. by highest urgency), or `approved`/`rejected`.
  - **Single default** when there is one outcome stream or when "success" varies by user. Do not force success/failure on every action.
- **Channel names**: Lowercase, single-word (`success`, `failed`, `approved`, `timeout`). Be consistent.

### Anti-patterns to avoid

- **Generic use cases:** "Automate {operation} within a CI/CD workflow", "Sync {service} changes into internal systems", "Create a workflow that reacts to {service} events."
- **Generic descriptions:** "Enable {service} to {operation} as part of SuperPlane workflows.", "Trigger workflows in SuperPlane when {service} emits {event}."
- **Wrong config for triggers:** Trigger with execution parameters (e.g. On Alarm with "Alarm Name", "Metric", "Threshold" for creating alarms). Use filter fields (Alarm Name Filter, State Transition, Severity).
- **Wrong title case:** `Create user` → use `Create User`.
- **Vague example values:** `example`, `my-thing`, `test` → use `backend-api`, `prod-deployment`, `v1.2.3`.

### Validation checklist (before submitting)

- [ ] Title uses Title Case.
- [ ] Description explains WHAT happens to external systems.
- [ ] Use cases are specific scenarios, not generic patterns.
- [ ] Triggers: config has FILTER fields only.
- [ ] Actions: output channels match whether user would branch (multiple vs default).
- [ ] Configuration includes important API parameters; example values are realistic.
