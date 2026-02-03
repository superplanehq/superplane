---
name: superplane-integration-log-issues-github
description: When creating GitHub issues from prepared SuperPlane integration issue templates. Use when logging base and component issues via GitHub MCP, assigning SuperPlane Board project fields (Integration Status, Priority), attaching sub-issues, and tracking progress in tmp.
---

# SuperPlane Integration: Log Issues to GitHub

Use this skill when creating integration issues on GitHub from prepared template files. You need **GitHub MCP** (e.g. `user-github`) with **projects** permissions so you can create issues, set SuperPlane Board fields (Integration Status, Priority), and attach child issues as sub-issues of the base. No manual Board or sub-issue setup is required. Source content lives under **tmp/integrations_pm/issues/**.

---

## Prerequisites

- **GitHub MCP** available with issue and **projects** permissions (for Board fields and sub-issue links).
- **Repository**: SuperPlane GitHub repo (owner/repo where issues are created).
- **Source content**: Markdown files under `tmp/integrations_pm/issues/` — base under `base/p1/` or `base/p2/`, components under `p1/` or `p2/`. File names: `{integration}_base.md`, `{integration}_{component}.md`.
- **Progress tracking**: Create or update a temporary document in `tmp/integrations_pm/` (e.g. `progress.md` or `log-progress-{integration}.md`) at the start and after each issue created, so you don’t lose track after context summary.

---

## Workflow overview

1. Create/update a **progress doc** in `tmp/integrations_pm/` listing the integration, base file, component files, and (as you go) created issue numbers.
2. **Create the base issue** first (one per integration). Assign labels and SuperPlane Board fields. Log base issue number in the progress doc.
3. **Create child issues** one per component, **sequentially** (one by one) to avoid API rate limiting. For each: create issue, set parent/sub-issue link, assign labels and Board fields, then update progress doc.
4. **Update list.md** (if present): in `tmp/integrations_pm/issues/list.md`, set checkbox to `[x]` for each created issue.
5. Ask the **user to review** on GitHub and confirm.
6. Once the user confirms, **clean up**: remove the progress doc and any other temporary files created for this run (e.g. under `tmp/integrations_pm/` for this integration).

---

## Step 1: Progress tracking

- **Before creating any issue**, create or open a progress document in `tmp/integrations_pm/` (e.g. `log-progress-consul.md`).
- **Contents**: Integration name, base file path, list of component file paths, then as you create issues: base issue #, then each child issue #.
- **After each created issue** (base or child), append to this doc so that after context summary you can resume from the last created issue.

---

## Step 2: Create the base issue

1. **Locate the base file**: `tmp/integrations_pm/issues/base/p1/{integration}_base.md` or `base/p2/` (e.g. `consul_base.md`).
2. **Build the issue** from the file:
   - **Title**: From `### Title` (e.g. `[Consul] Base`). Do **not** put the title line in the body.
   - **Body**: IMPORTANT block, Description, Suggested Connection Method, Acceptance Criteria, Follow up tasks, Reference — **no** `### Title` or backtick title line; **no** `### Priority` or priority value line (Priority is set via Board).
3. **Create the issue** via GitHub MCP with title and body as above.
4. **Labels**: Add `integration` and `refinement` to the issue.
5. **SuperPlane Board** (via MCP projects):
   - **Add** the issue to the SuperPlane Board project (`add_project_item` with owner, project_number 2, item_owner, item_repo, item_type `issue`, issue_number).
   - **Set Board fields** (required — do not skip): Adding an issue does **not** set Integration Status or Priority. Get the new item's numeric id via `list_project_items` (e.g. query `title:*[IntegrationName]*`), find the item whose Title matches the issue you just created; then call `update_project_item` twice for that item — once Integration Status = Backlog (field id and Backlog option id from `list_project_fields`), once Priority = agreed P1–P4 (field id and P1/P2/P3/P4 option id from `list_project_fields`). Field ids in `updated_field` must be **numbers**.
6. **Note the base issue number** (e.g. `#1912`) and write it in the progress doc. You need it for every child issue body and for linking sub-issues.

---

## Step 3: Create child issues (one per component, sequentially)

1. **Find component files**: Under `tmp/integrations_pm/issues/p1/` or `p2/`, all `{integration}_*.md` except `{integration}_base.md` (e.g. `consul_get_kv.md`, `consul_register_service.md`). Use `tmp/integrations_pm/issues/list.md` if present for the exact list.
2. **For each component file, one at a time** (to avoid rate limiting):
   - **Build the issue**:
     - **Title**: From `### Title` in the file (e.g. `[Consul] Get KV`). Do **not** put the title in the body.
     - **Body**: First line must be `**Parent (base integration):** #BASE_ISSUE_NUMBER`, then a blank line, then the rest of the file content **without** the `### Title` and backtick title line and **without** the `### Priority` and priority value line (Priority is set via Board). Include IMPORTANT block, Description, Use Cases, Configuration, Outputs, Acceptance Criteria, Reference.
   - **Create the issue** via MCP with this title and body.
   - **Labels**: Add `integration` and `refinement`.
   - **SuperPlane Board**: Add to project with `add_project_item`. Then **set Board fields** (required — do not skip): get the new item's numeric id via `list_project_items` (e.g. query `title:*[IntegrationName]*`), then call `update_project_item` twice for that item — once Integration Status = Backlog (field id and Backlog option id from `list_project_fields`), once Priority = agreed P1–P4 (field id and P1/P2/P3/P4 option id from `list_project_fields`). Field ids in `updated_field` must be numbers.
   - **Sub-issue link** (via MCP projects): Attach this issue as a **sub-issue of the base issue** so the Board shows the hierarchy. The parent reference in the body is also required.
   - **Update progress doc** with this child issue number, then proceed to the next component.
3. Create issues **sequentially**; do not batch many creates in parallel to avoid GitHub API rate limits.

---

## Step 4: Update list.md (if present)

- In `tmp/integrations_pm/issues/list.md`, for each issue created (base and each child), change the corresponding line from `- [ ]` to `- [x]` (same indentation and filename).
- This keeps the checklist in sync with what exists on GitHub.

---

## Step 5: User review and cleanup

- **Ask the user** to review the created issues on GitHub (base, children, labels, Board fields, sub-issue links) and confirm.
- **After user confirms**: Delete the progress document and any other temporary files created for this logging run (e.g. `tmp/integrations_pm/log-progress-{integration}.md`). Do **not** delete the source issue template files unless the user explicitly asks to clean those too.

---

## Summary checklist (per integration)

- [ ] Create/update progress doc in `tmp/integrations_pm/` before starting.
- [ ] Create **one base issue** first; exclude `### Title` and `### Priority` from body; add labels `integration`, `refinement`; add to Board then **set Board fields** via `update_project_item` (Integration Status = Backlog, Priority = agreed P1–P4) — get item id from `list_project_items`; do not skip this.
- [ ] Create **one child issue** per component **sequentially**; body starts with `**Parent (base integration):** #BASE_ISSUE_NUMBER`; exclude `### Title` and `### Priority` from body; same labels; add to Board then **set Board fields** via `update_project_item` for that item; attach as sub-issue of base via MCP (projects).
- [ ] After each issue, update progress doc; then update `list.md` checkboxes if the file exists.
- [ ] Ask user to review on GitHub; after confirmation, remove progress doc and other tmp files created for this run.
