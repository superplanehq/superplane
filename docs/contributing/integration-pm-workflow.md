# Integration PM Workflow (Cursor)

This guide explains how to use the **integration product management workflow** in Cursor to plan new integrations, generate issue templates, and create GitHub issues on the SuperPlane Board. The workflow uses Cursor Rules, Skills, Commands, and optional Subagents so you can run it consistently and repeatably.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Workflow overview](#workflow-overview)
- [Step 1: Research the integration](#step-1-research-the-integration)
- [Step 2: Generate issue templates](#step-2-generate-issue-templates)
- [Step 3: Create issues on GitHub](#step-3-create-issues-on-github)
- [Using subagents (optional)](#using-subagents-optional)
- [Where everything lives](#where-everything-lives)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- **Cursor** with Agent (Chat) available.
- **GitHub MCP** enabled and connected to the SuperPlane repo, with **projects** permissions so the agent can create issues, set Board fields (Integration Status, Priority), and attach sub-issues to the base issue.
- The integration PM primitives are already in the repo: `.cursor/rules/`, `.cursor/skills/`, `.cursor/commands/`, `.cursor/agents/` (see [Where everything lives](#where-everything-lives)).

## Workflow overview

The workflow has three steps. You run one command per step, review the output, and then move to the next.

| Step | What you do | What the agent does |
|------|-------------|----------------------|
| **1. Research** | Run `/integration-research` with the tool name and links. | Checks existing GitHub issues for that integration, researches the tool, suggests base + components (triggers/actions) and P1–P4 priorities, and gives you a summary to review. |
| **2. Templates** | After you approve the research, run `/integration-issue-templates` (or ask the agent to generate templates). | Creates temp issue files under `tmp/integrations_pm/issues/` (one base + one per component). You review the files and request changes if needed. |
| **3. Log issues** | After you approve the templates, run `/integration-log-issues` with the integration name. | Creates the base issue on GitHub, then each component issue as a child; sets labels, Board Integration Status = Backlog, Priority (P1–P4), and attaches children as sub-issues. Tracks progress in a temp file; after you confirm on GitHub, cleans up the progress file. |

You must complete each step and approve before moving to the next. The agent will not generate templates until you approve the research summary, and will not create GitHub issues until you approve the template files.

## Step 1: Research the integration

1. Open a new Agent chat in Cursor (or use an existing one).
2. Type **`/integration-research`** and add the tool name and links. For example:

   ```
   /integration-research Rootly

   Tool: https://rootly.com/ — end-to-end incident management platform.
   Docs: https://docs.rootly.com/help-and-documentation
   ```

   Or in plain language:

   ```
   I want to integrate Rootly (https://rootly.com/, docs: https://docs.rootly.com/help-and-documentation) into SuperPlane. Run the integration research: check existing GitHub issues for this integration, then suggest the base integration and components (triggers/actions) with P1–P4 priorities, and give me a summary to review.
   ```

3. The agent will:
   - Use GitHub MCP to check for existing issues with that integration name (e.g. `[Rootly]`).
   - Research the tool and suggest how it would connect to SuperPlane (base), plus a list of triggers and actions with P1–P4 priorities.
   - Present a summary: existing issues (if any), base, and list of components with priorities.

4. **Review the summary.** Ask for changes (e.g. add/remove components, change priorities). When you are satisfied, say you approve and want to generate the issue templates.

## Step 2: Generate issue templates

1. After you have approved the research summary, ask the agent to generate the issue template files. For example:

   ```
   Looks good. Generate the issue template files for this integration (base + components) in tmp/integrations_pm/issues/.
   ```

   Or run **`/integration-issue-templates`** and confirm the integration name if asked.

2. The agent will create:
   - One **base** file: `tmp/integrations_pm/issues/base/p1/{integration}_base.md` (or `base/p2/` depending on priority).
   - One **component** file per trigger/action: `tmp/integrations_pm/issues/p1/{integration}_{component}.md` (or `p2/`).

3. **Review the files** in `tmp/integrations_pm/issues/`. Open them in the editor and ask the agent to fix anything (e.g. description, use cases, configuration, output channels). When you are satisfied, say you want to create the issues on GitHub.

## Step 3: Create issues on GitHub

1. After you have approved the template files, run **`/integration-log-issues`** and specify the integration name if you have more than one in `tmp/integrations_pm/issues/`. For example:

   ```
   /integration-log-issues Rootly
   ```

   Or: “Create the GitHub issues from the Rootly template files in tmp/integrations_pm/issues/. Use the integration-log-issues workflow.”

2. The agent will:
   - Create a **progress file** in `tmp/integrations_pm/` (e.g. `log-progress-rootly.md`) so it can resume after context summary if needed.
   - Create the **base issue** first (labels `integration`, `refinement`; Board Integration Status = Backlog, Priority from your agreed P1–P4).
   - Create **one child issue per component** sequentially (to avoid rate limits), each with the same labels and Board fields, and attach each child as a **sub-issue** of the base issue.
   - Optionally update `tmp/integrations_pm/issues/list.md` if it exists (checkboxes for created issues).

3. **Review the issues on GitHub.** Check the SuperPlane Board: base issue, child issues, labels, Integration Status = Backlog, Priority, and that children are linked as sub-issues of the base.

4. **Confirm in chat** when everything looks correct. The agent will then delete the progress file (and any other temp files it created for this run). It will **not** delete the source template files in `tmp/integrations_pm/issues/` unless you ask.

**Note:** The agent strips the **Title** and **Priority** sections from the issue body when creating GitHub issues. Title becomes the issue title; Priority is set only via the Board project field, so it is not duplicated in the body.

## Using subagents (optional)

Subagents run in an **isolated context** and return a concise summary to the main chat. They are useful when research or issue creation would produce a lot of output and you want to keep the main conversation short.

- **Integration researcher** — Use when you want the research step (existing-issues check + base + components + P1–P4) done in a separate context. Invoke by name, e.g. “Use the integration-researcher subagent to research Consul” or `/integration-researcher Consul`. The subagent returns the summary; you review and then continue with templates in the main chat.
- **Integration issue logger** — Use when you want the “create N issues via MCP” step done in isolation (e.g. many components). Invoke by name, e.g. “Use the integration-issue-logger subagent to create the GitHub issues for Rootly from the template files in tmp/integrations_pm/issues/.” The subagent creates the issues, tracks progress, and returns the list of issue numbers; you then review on GitHub and confirm so it can clean up.

You can run the full workflow **without** subagents by using the three commands in order; subagents are optional.

## Where everything lives

| Purpose | Location |
|--------|----------|
| **Rule** (issue conventions, title format, triggers vs actions) | `.cursor/rules/integration-issue-conventions.mdc` |
| **Skills** (prioritization, issue templates, log-issues procedure) | `.cursor/skills/superplane-integration-prioritization/`, `superplane-integration-issue-templates/`, `superplane-integration-log-issues-github/` |
| **Commands** (slash workflows) | `.cursor/commands/integration-research.md`, `integration-issue-templates.md`, `integration-log-issues.md` |
| **Subagents** (optional) | `.cursor/agents/integration-researcher.md`, `integration-issue-logger.md` |
| **Temp template files** | `tmp/integrations_pm/issues/` (base in `base/p1/` or `base/p2/`, components in `p1/` or `p2/`) |
| **Progress file** (during log-issues; deleted after you confirm) | `tmp/integrations_pm/log-progress-{integration}.md` |
| **Helper for agents/maintainers** | `.cursor/PM_WORKFLOW_HELPER.md` (reference for Rules/Skills/Commands/Subagents and when to use which) |

## Troubleshooting

- **Commands don’t appear when I type `/`** — Ensure you are in Agent (Chat) and that the repo has `.cursor/commands/` with the integration PM command files. You can also describe what you want in plain language (e.g. “research Rootly for SuperPlane integration and suggest base + components + priorities”).
- **GitHub MCP can’t set Board fields or sub-issues** — The GitHub MCP connection must have **projects** permissions for the SuperPlane repo. Check your Cursor/MCP configuration. Without projects permission, the agent can still create issues and labels; you would set Integration Status, Priority, and sub-issue links manually on the Board.
- **Agent didn’t strip Title or Priority from the issue body** — The command and skill instruct the agent to omit the `### Title` and `### Priority` sections when building the GitHub issue body. If a run missed this, you can edit the issue body on GitHub to remove those lines, or re-run the workflow after fixing the instructions in `.cursor/commands/integration-log-issues.md` and `.cursor/skills/superplane-integration-log-issues-github/SKILL.md`.
- **I want to change how priorities or templates work** — Edit the relevant Skill or Rule under `.cursor/` (see [Where everything lives](#where-everything-lives)). For a quick reference on what each primitive does, see `.cursor/PM_WORKFLOW_HELPER.md`.
