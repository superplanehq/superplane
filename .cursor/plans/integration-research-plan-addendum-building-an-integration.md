# Plan addendum: building-an-integration and linked docs

Merge these points into the main integration-research plan ([integration_research_two_modes](https://github.com/superplanehq/superplane/blob/main/.cursor/plans) or local plan file).

## Source docs

- [building-an-integration.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/building-an-integration.md) — high-level steps to add a new integration
- [integrations.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/integrations.md) — integration development guide (structure, triggers, components)
- [component-implementations.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/component-implementations.md) — implementation patterns (spec structs, field types)
- [integration-prs.md](https://github.com/superplanehq/superplane/blob/main/docs/contributing/integration-prs.md) — PR title, description, video demo, backend/frontend expectations

---

## 1. Connection method = same three pillars as building-an-integration

In **building-an-integration**, step 2 is "Research the connection method":

- **Auth**: API key, OAuth, or other (where users get credentials, how they're passed).
- **API**: REST/GraphQL endpoints, rate limits, **webhooks vs polling for triggers**.
- **Constraints**: Any limitations or quirks that affect design.

**Action for plan**: In the new skill `superplane-integration-research`, require that the "Connection method" in the research output uses exactly these three pillars. That way the base-issue description is directly usable for "Research the connection method" and for the PR description (integration-prs.md: "Describe the implementation… Authorization is via API key…").

---

## 2. Implementer workflow after issues are created

building-an-integration says:

1. Pick an integration and **claim the existing issue** (comment on it); see [Integrations Board](https://github.com/orgs/superplanehq/projects/2/views/19).
2. **Research the connection method** (document in ticket or PR).
3. **Build**: Backend in `pkg/integrations/<name>/`, frontend mappers in `web_src/.../mappers/`, docs in the integration package (`make gen.components.docs`), tests in `pkg/integrations/<name>/`. Same structure and patterns as other integrations.
4. Open a PR and follow [integration-prs.md](docs/contributing/integration-prs.md) (title, description with issue link and video demo, backend/frontend expectations, CI, DCO).

**Action for plan**: In the research skill or command, optionally add one sentence that after issues are created, implementers follow building-an-integration (and link it). Research should be concrete enough that the resulting ticket fits this workflow.

---

## 3. Backend/frontend structure (for consistency, not research output)

integrations.md and component-implementations.md describe:

- Backend: `pkg/integrations/<app-name>/` with main integration file, client, trigger/component files.
- Frontend: `web_src/src/pages/workflowv2/mappers/<integration-name>/` with index and per-trigger/component renderers.
- Component patterns: strongly typed spec structs, conditionally visible fields as pointers, semantic field types.

**Action for plan**: No need to make research output mention file paths. The skill can note that suggested components will be implemented per integrations.md and component-implementations.md so research stays concrete (real API operations/events).

---

## 4. PR expectations (integration-prs.md)

- Title: `feat: Add <Integration>` or `feat: Add <Integration> <Trigger/Action>`.
- Description: issue link, what was implemented and how (e.g. auth, API), limitations; **video demo** (setup, workflow in action, config; 1–2 min).
- Backend in `pkg/integrations/<name>/`, frontend expectations, docs, tests.

**Action for plan**: Connection method from research (Auth, API, Constraints) feeds directly into "Describe the implementation" in the PR; no change to research flow beyond aligning connection method with building-an-integration.

---

## Summary: what to add to the main plan

1. **Skill content**: In "Connection method", require **Auth**, **API** (including webhooks vs polling for triggers), and **Constraints** as in building-an-integration. Add one optional line: "After issues are created, implementers follow docs/contributing/building-an-integration.md (claim ticket, research connection, build, then integration-prs.md for PR)."
2. **Commands**: In integration-research-new and integration-research-extension, step "Suggest base" or "Connection method" should explicitly say: document Auth, API (endpoints, rate limits, webhooks vs polling), and Constraints so the base issue matches building-an-integration step 2.
3. **Documentation**: In integration-pm-workflow.md, add a short "Next steps for implementers" that links to building-an-integration and the Integrations Board.
