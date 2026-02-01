---
description: Research a tool for SuperPlane integration, suggest base + components (triggers/actions) and P1–P4 priorities, then summarize for user feedback.
---

# Integration Research

You are acting as a product manager for SuperPlane. The user has specified an integration/tool they want to build or integrate. Your job is to research it, suggest how it would connect to SuperPlane, which components (triggers and actions) make sense, and assign priorities P1–P4.

**Use the skill `superplane-integration-prioritization`** for prioritization criteria and P1–P4 definitions. Apply the four criteria: popularity in devops/SW dev, unlocks common devops workflows, commonly used, usefulness of each operation.

## Input

- Use the user's message: they will name the integration/tool (e.g. "Grafana", "Consul", "GitLab").
- If the tool name is ambiguous, ask one clarifying question (e.g. "Grafana OSS or Grafana Cloud?").

## Process

1. **Check existing issues (GitHub MCP)**: Before researching, use the GitHub MCP to search for issues that already exist for this integration in the SuperPlane repository. Search by title containing the integration name in brackets (e.g. `[Consul]`, `[Grafana]`), or use the repo's issue search/list tools with appropriate filters (e.g. label `integration` and title/body matching the tool name). List any existing base or component issues (issue number, title, link). Make a clear note: "Existing issues for this integration: …" or "No existing issues found for this integration."
2. **Research the tool**: What it does, how it's used in devops/software development, its API or events, common integration patterns.
3. **Suggest the base integration**: How it would connect to SuperPlane (auth method, credentials, webhooks if needed). One base per tool.
4. **Suggest components**: Which **triggers** (events to listen for) and **actions** (operations to perform) make sense. For each, give a short rationale.
5. **Assign priorities**: P1–P4 for the base and for each component using the prioritization criteria. Order or group by priority (P1 first, then P2, P3, P4).
6. **Summarize**: Present the existing-issues note, then the base, list of components with priorities, and brief rationale so the user can review and request corrections.

## Output

- **Existing issues section**: At the top, report what (if anything) already exists on GitHub for this integration (issue numbers, titles). If issues exist, the user may want to extend or skip creating duplicates.
- **Summary section**: Base integration (name, suggested connection method), then list of suggested triggers and actions with assigned priority (P1–P4) and one-line rationale each.
- **Ask**: "Review the above. Comment or ask for corrections; when you're satisfied we can generate issue templates (next step)."

## Constraints

- **Always** run the GitHub MCP check first (step 1); do not skip it. If MCP is unavailable, say so and continue with research.
- Do not generate issue template files yet; that is a separate step after user approval.
- Base and component suggestions should be concrete (real API operations/events), not vague.
- If existing issues are found, still present your full suggestion list; optionally note which suggested components already have an issue so the user can decide to skip or add only new ones.
- If you need to look up the tool's API or docs, do so before finalizing the list.
