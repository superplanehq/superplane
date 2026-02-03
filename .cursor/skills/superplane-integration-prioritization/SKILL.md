---
name: superplane-integration-prioritization
description: When suggesting or prioritizing integration components for SuperPlane (base integrations, triggers, actions). Use when researching a tool to integrate, assigning P1–P4 priorities, or deciding which operations to implement first.
---

# SuperPlane Integration Prioritization

Use this skill when you are researching a tool for SuperPlane integration, suggesting which components (triggers and actions) to implement, or assigning priorities from P1 to P4.

## Criteria for prioritizing integrations and components

Apply these four criteria when evaluating a tool and its operations:

1. **Popularity in devops or software development** — Is this tool widely used in devops, CI/CD, or general software development? Leading or common tools in their category rank higher.

2. **Unlocks common devops workflow processes** — Does integrating this tool enable workflows that teams commonly need (e.g. deploy on merge, notify on failure, sync state, run tests)?

3. **Commonly used** — Would the integration and its operations be used frequently in real workflows, not just in edge cases?

4. **Usefulness of operations (components)** — Among the tool’s possible triggers and actions, which operations are most useful? Rank each suggested component (trigger or action) by how often and how critically it would be used in real workflows.

## Priority levels (P1–P4)

Assign each **base integration** and each **component** (trigger or action) to one priority based on the criteria above.

| Priority | Meaning | When to use |
|----------|---------|-------------|
| **P1** | High — core, frequently used | Core integrations and operations that are widely used and unlock essential workflows (e.g. GitHub base, On Push, Get Issue; Slack base, Send Message). |
| **P2** | Medium — important, common | Important integrations and operations that support common workflows but are not always required first (e.g. many CI/CD triggers, common read/write actions). |
| **P3** | Low — useful, less common | Useful operations for specific or less common workflows; implement after P1 and P2. |
| **P4** | Lowest — edge cases, rarely used | Edge cases, rarely used operations, or nice-to-haves; implement last or defer. |

## How to apply

- **Base integration**: Assign a single P1–P4 for the whole integration (e.g. GitHub → P1; lesser-known tool → P2 or P3).
- **Components**: Assign P1–P4 per trigger and per action. The same integration can have a mix (e.g. [GitHub] On Push → P1, [GitHub] Get Issue → P1, [GitHub] On Deployment → P2).
- **Ordering**: When suggesting a list of components, order or group by priority (P1 first, then P2, then P3, then P4) so the user sees what to implement first.

## Output when suggesting components

When you suggest a base integration and its components:

1. Name the integration and suggest the **base** (how it would connect to SuperPlane).
2. List suggested **triggers** and **actions** with a short rationale for each.
3. Assign **P1–P4** to the base and to each component.
4. Provide a brief **summary** so the user can review and request changes before issue templates or GitHub issues are created.
