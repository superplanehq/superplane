---
description: Generate temp issue template files (base + components) in tmp/integrations_pm/issues/ after research is approved. User reviews; when satisfied, run integration-log-issues.
---

# Integration Issue Templates

You are generating **temporary issue template files** for a SuperPlane integration. The user has already run `/integration-research` and approved the summary (base integration + list of components with P1–P4). Your job is to write one base file and one file per component under `tmp/integrations_pm/issues/`, following the template structure and conventions.

**Use the skill `superplane-integration-issue-templates`** for structure, guidelines, triggers vs actions, and output channels. Follow the **integration-issue-conventions** rule for IMPORTANT blocks, title format, and hierarchy.

## Input

- The **agreed research summary**: integration name, base (connection method), and list of components (triggers/actions) with assigned P1–P4. If the user did not paste it, ask them to provide the agreed list (or run `/integration-research` first and get approval).
- **Integration name** for file naming: use lowercase with underscores for filenames (e.g. `consul`, `grafana`, `git_hub` only if needed to avoid ambiguity; usually `github`).

## File locations and naming

- **Base**: `tmp/integrations_pm/issues/base/{p1|p2}/{integration}_base.md` — use `p1` or `p2` folder based on the base's priority (P1 → base/p1/, P2 → base/p2/).
- **Components**: `tmp/integrations_pm/issues/{p1|p2}/{integration}_{component_slug}.md` — use p1/p2 folder by each component's priority. Component slug: operation name in lowercase with spaces replaced by underscores (e.g. `Get KV` → `get_kv`, `On Deployment` → `on_deployment`, `Sync Application` → `sync_application`).
- Create the directories if they do not exist.

## Process

1. **Resolve input**: Confirm integration name and the full list (base + components with priorities). If missing, ask the user.
2. **Generate base file**: One file in `tmp/integrations_pm/issues/base/p1/` or `base/p2/`. Content: IMPORTANT block (base version), then `### Title` and `` `[{Integration Name}] Base` ``, then Description, Suggested Connection Method, Acceptance Criteria, Follow up tasks, Reference. The log-issues skill will strip the `### Title` section when creating the GitHub issue body.
3. **Generate component files**: One file per component in `tmp/integrations_pm/issues/p1/` or `p2/` by priority. Content: IMPORTANT block (component version), then `**Parent (base integration):** #TBD` (placeholder; replaced with actual base issue number when logging), then `### Title` and the title line, then Priority, Description, Use Cases, Configuration, Outputs, Acceptance Criteria (optional), Reference. Use the skill for triggers vs actions (config = filters vs execution parameters) and output channels. File name: `{integration}_{component_slug}.md`. The log-issues skill will strip the `### Title` section when creating the GitHub issue body.
4. **Optional — list.md**: If `tmp/integrations_pm/issues/list.md` exists, add an entry for this integration (base + components with checkboxes `[ ]`). If it does not exist, you may create a minimal list or skip; the log-issues skill will update checkboxes when issues are created.

## Output

- **List of created files**: Paths to each file (base + components).
- **Ask**: "Review the files above. Request any changes; when you're satisfied, run `/integration-log-issues` to create the issues on GitHub."

## Constraints

- Do **not** create GitHub issues here; that is the next command (`/integration-log-issues`).
- Parent reference in component files must be `**Parent (base integration):** #TBD`; the actual number is set when creating issues.
- Follow the issue-templates skill strictly (triggers = filter config, actions = execution params; output channels by whether user would branch).
- Use realistic example values in Configuration (e.g. `backend-api`, `prod-deployment`), not `example` or `my-thing`.
