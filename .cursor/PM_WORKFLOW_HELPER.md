# Cursor PM Workflow Helper

Use this file when adding or changing **Rules**, **Skills**, **Commands**, or **Subagents** for the SuperPlane integration PM workflow, or when you need a quick reference for which primitive to use.

---

## 1. Definitions and comparison

| Primitive | What it is | Where it lives |
|-----------|------------|----------------|
| **Rules** | System-level instructions; part of the prompt. Applied always, by file glob, or when the agent decides they're relevant. | `.cursor/rules/` (`.md` or `.mdc` with frontmatter). Also `AGENTS.md` in project root. |
| **Commands** | Slash-invoked workflows (`/name`). Repeatable workflows you or the agent trigger. | `.cursor/commands/` as `.md` files. |
| **Skills** | Portable knowledge packages (open standard). Domain-specific instructions the agent can apply when relevant or when you invoke via `/`. Can include `scripts/`, `references/`, `assets/`. | `.cursor/skills/<name>/SKILL.md` (folder per skill). |
| **Subagents** | Isolated AI assistants with their own context. Can run in foreground or background. | `.cursor/agents/` as `.md` files with YAML frontmatter. |

**When to use which (short):**

- **Rules** — Short, permanent or context-scoped guidance (e.g. issue conventions, title format, triggers vs actions). Keep under ~500 lines.
- **Commands** — Repeatable workflows you invoke step-by-step (e.g. `/integration-research`, `/integration-issue-templates`, `/integration-log-issues`).
- **Skills** — Substantial reference or procedure the agent loads when doing a task (e.g. prioritization criteria, issue templates, log-issues procedure).
- **Subagents** — Isolated context for research or long issue-creation runs so the main chat doesn't get bloated; return a concise summary to the parent.

---

## 2. Formats and locations quick reference

**Rules** (`.cursor/rules/*.mdc` or `.md`):

- Frontmatter: `description`, `globs` (array of path patterns), `alwaysApply` (boolean).
- Types: Always Apply, Apply Intelligently (by description), Apply to Specific Files (globs), Apply Manually (@mention).

**Commands** (`.cursor/commands/*.md`):

- Plain Markdown; optional YAML frontmatter (e.g. `description`).
- Filename (without `.md`) becomes the slash command (e.g. `integration-research.md` → `/integration-research`).

**Skills** (`.cursor/skills/<name>/SKILL.md`):

- Frontmatter: `name` (required, matches folder), `description` (required), optional `disable-model-invocation`, `license`, `compatibility`, `metadata`.
- Optional dirs: `scripts/`, `references/`, `assets/`.

**Subagents** (`.cursor/agents/*.md`):

- Frontmatter: `name`, `description`, `model` (e.g. `inherit`, `fast`), optional `readonly`, `is_background`.
- Body: Instructions for the subagent (what to do and what to return to the parent).

---

## 3. SuperPlane integration PM workflow mapping

This repo's **integration PM workflow** (research → templates → log issues) is implemented as follows.

### Flow

1. **Research** — User specifies integration/tool → check existing GitHub issues (MCP) → suggest base + components (triggers/actions) + P1–P4 → summarize for user feedback.
2. **Templates** — After approval → generate temp issue files (base + components) under `tmp/integrations_pm/issues/` → user reviews and requests changes.
3. **Log issues** — After approval → create issues on GitHub via MCP (base first, then children sequentially), set Board Integration Status = Backlog and Priority (P1–P4), attach children as sub-issues of base; track progress in tmp; after user confirms, cleanup tmp (progress doc, etc.).

### Primitives in use

| Phase | Rule | Skills | Commands | Subagents |
|-------|------|--------|----------|-----------|
| **Conventions** | `integration-issue-conventions` (globs: `tmp/integrations_pm/**`) | — | — | — |
| **Prioritization** | — | `superplane-integration-prioritization` | — | — |
| **Issue content** | — | `superplane-integration-issue-templates` | — | — |
| **Log to GitHub** | — | `superplane-integration-log-issues-github` | — | — |
| **Research workflow** | — | prioritization (+ templates for norms) | `/integration-research` | `integration-researcher` (optional) |
| **Templates workflow** | conventions | issue-templates | `/integration-issue-templates` | — |
| **Log issues workflow** | — | log-issues-github | `/integration-log-issues` | `integration-issue-logger` (optional) |

### File locations (this repo)

- **Rule:** `.cursor/rules/integration-issue-conventions.mdc`
- **Skills:** `.cursor/skills/superplane-integration-prioritization/`, `superplane-integration-issue-templates/`, `superplane-integration-log-issues-github/`
- **Commands:** `.cursor/commands/integration-research.md`, `integration-issue-templates.md`, `integration-log-issues.md`
- **Subagents:** `.cursor/agents/integration-researcher.md`, `integration-issue-logger.md`
- **Source templates / temp files:** `tmp/integrations_pm/issues/` (base under `base/p1/` or `base/p2/`, components under `p1/` or `p2/`). Progress tracking: `tmp/integrations_pm/log-progress-{integration}.md` (removed after user confirms).

### Invocation order

1. `/integration-research` (optionally with tool name) → user approves summary.
2. `/integration-issue-templates` → user reviews temp files and approves.
3. `/integration-log-issues` → create issues via MCP; user reviews on GitHub and confirms → cleanup.

The main agent can delegate to **integration-researcher** or **integration-issue-logger** subagents for isolated context (e.g. `/integration-researcher Consul` or "Use the integration-issue-logger subagent to log the Consul issues").

---

## 4. References

- [Rules](https://cursor.com/docs/context/rules)
- [Commands](https://cursor.com/docs/context/commands)
- [Agent Skills](https://cursor.com/docs/context/skills)
- [Subagents](https://cursor.com/docs/context/subagents)
- [Skills vs Commands vs Rules (forum)](https://forum.cursor.com/t/skills-vs-commands-vs-rules/148875)

---

## 5. Alignment with this repo

- **Project guidelines:** [AGENTS.md](AGENTS.md) at project root (build, test, formatting, migrations, etc.). Integration PM primitives live under `.cursor/` and follow the structure above.
- **Existing command pattern:** The component-review command uses a command file plus a rules file (`.cursor/commands/component-review.md` and `component-review.rules.md`). Integration PM uses Rules + Skills + Commands + Subagents as in the table above; no separate `.rules.md` next to commands.
