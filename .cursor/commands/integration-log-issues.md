---
description: Create GitHub issues from prepared integration template files in tmp/integrations_pm/issues/. Uses MCP, SuperPlane Board fields, sub-issues; tracks progress in tmp; cleanup after user confirms.
---

# Integration Log Issues

You are creating integration issues on GitHub from prepared template files under `tmp/integrations_pm/issues/`. The user has already run `/integration-research` and `/integration-issue-templates` and approved the temp files. Your job is to create the base issue first, then one child issue per component, assign labels and SuperPlane Board fields, and track progress so you can resume after context summary.

**Use the skill `superplane-integration-log-issues-github`** for the full procedure: progress doc, base then children, body content (strip `### Title` and `### Priority` — both are helpers; Priority is set via Board project field), labels, Integration Status = Backlog, Priority (P1–P4), sub-issue links, sequential creation, list.md checkboxes, user review, cleanup.

## Input

- **Integration name** (e.g. "Consul", "Grafana"): the user will specify which integration to log, or you can infer from the base file in `tmp/integrations_pm/issues/base/p1/` or `base/p2/` (e.g. `consul_base.md` → integration "consul"). If multiple base files exist, ask which integration to log.
- **Source**: Template files under `tmp/integrations_pm/issues/`. Base: `base/p1/{integration}_base.md` or `base/p2/`. Components: `p1/{integration}_*.md` or `p2/` (excluding base).

## Process

1. **Create progress doc**: In `tmp/integrations_pm/`, create or open a progress file (e.g. `log-progress-{integration}.md`) with integration name, base file path, list of component file paths. You will append the base issue number and each child issue number after creating them.
2. **Create base issue**: Read base file; title from `### Title`, body = file content **without** `### Title` and the backtick title line, and **without** `### Priority` and the priority value line (if present; Priority is set via Board). Create issue via GitHub MCP; add labels `integration` and `refinement`; set SuperPlane Board **Integration Status** = `Backlog`, **Priority** = agreed P1–P4 for base. Write base issue number in progress doc.
3. **Create child issues sequentially**: For each component file (one at a time to avoid rate limits): read file; title from `### Title`; body = `**Parent (base integration):** #BASE_ISSUE_NUMBER` (use actual number from step 2), then blank line, then rest of file **without** `### Title` and title line and **without** `### Priority` and the priority value line (Priority is set via Board). Replace any `#TBD` parent reference with the real base issue number. Create issue via MCP; add labels; set Board Integration Status = Backlog, Priority from file; attach as sub-issue of base via MCP (projects permissions). Append child issue number to progress doc.
4. **Update list.md**: If `tmp/integrations_pm/issues/list.md` exists, set checkbox to `[x]` for the base and each component file you created.
5. **Ask user to review**: Tell the user to check the issues on GitHub (base, children, labels, Board fields, sub-issue links) and confirm.
6. **Cleanup after confirm**: Once the user confirms, delete the progress doc and any other tmp files created for this run (e.g. `log-progress-{integration}.md`). Do **not** delete the source template files unless the user explicitly asks.

## Output

- **After creating issues**: List base issue # and each child issue # (with links if possible). "Please review the issues on GitHub and confirm when done."
- **After user confirms**: "Cleanup complete. Progress doc removed."

## Constraints

- **Always** create the progress doc before the first issue; update it after every issue so you can resume after context summary.
- Create issues **sequentially** (base, then child 1, then child 2, …); do not batch creates to avoid API rate limiting.
- Body for GitHub: never include the `### Title` line or the backtick title line; never include the `### Priority` section or the priority value line (Priority is set via Board project field). For children, body must start with `**Parent (base integration):** #N`.
- Use GitHub MCP with **projects** permissions to set SuperPlane Board fields (Integration Status = Backlog, Priority) and to attach each child issue as a sub-issue of the base. No manual Board or sub-issue setup is required.
