# Incident integration: commit split and PR description

Use this to split your changes into logical commits and open the PR. Run from repo root. **Sign off every commit:** use `git commit -s` (or `git ci -m "..."` if you have the alias).

---

## Before you commit

1. **Run quality checks** (with Docker/app running if needed):
   ```bash
   make lint
   make check.build.app
   make check.build.ui
   ```
2. **Regenerate component docs** (optional; if you changed backend descriptions):
   ```bash
   make gen.components.docs
   ```
   Then add any changed files under `docs/components/` to the docs commit below.

---

## Commit 1: chore – ignore local dev artifacts

```bash
git add .gitignore
git commit -s -m "chore: Ignore .local and ngrok.tgz for dev tunnel"
```

---

## Commit 2: feat – Incident integration (backend)

```bash
git add pkg/integrations/incident/
git add pkg/server/server.go
git commit -s -m "feat: Add Incident integration (incident.io)

- Integration: API key auth, Sync validates key via ListSeverities, ListResources for severity
- On Incident trigger: webhook (Svix), events created/updated, signing secret, no API to register endpoint
- Create Incident action: name, summary, severity, visibility via Incidents V2 API
- Webhook handler: Setup returns URL for manual registration in incident.io; HandleWebhook verifies Svix signature
- Tests: incident_test, on_incident_test, create_incident_test, webhook_handler_test"
```

---

## Commit 3: feat – Incident UI mappers and assets

```bash
git add web_src/src/pages/workflowv2/mappers/incident/
git add web_src/src/pages/workflowv2/mappers/index.ts
git add web_src/src/ui/BuildingBlocksSidebar/index.tsx
git add web_src/src/ui/componentSidebar/integrationIcons.tsx
git add web_src/src/utils/integrationDisplayName.ts
git add web_src/src/assets/icons/integrations/incident.svg
git commit -s -m "feat: Add Incident UI mappers and sidebar entry

- Mappers for On Incident and Create Incident (config, details, example data)
- On Incident: webhook URL copy button and HTTPS hint when URL is HTTP
- Incident icon and display name; BuildingBlocksSidebar and integrationIcons"
```

---

## Commit 4: docs – WEBHOOKS_BASE_URL and Incident testing

```bash
git add Makefile
git add docs/contributing/connecting-to-3rdparty-services-from-development.md
git add docs/development/manual-test-incident-integration.md
git add docs/components/Incident.mdx
git commit -s -m "docs: WEBHOOKS_BASE_URL options and Incident manual testing

- Makefile: dev.start.with-webhook-tunnel target
- connecting-to-3rdparty: how to set WEBHOOKS_BASE_URL (inline, .env, export) and re-save workflow
- manual-test-incident-integration: step-by-step for integration, On Incident, Create Incident, webhook in incident.io
- Incident.mdx: component docs (generated or hand-maintained)"
```

---

## PR title

Use the semantic format (see [integration-prs.md](../contributing/integration-prs.md)):

```
feat: Add Incident integration (incident.io)
```

---

## PR description (paste into GitHub)

```markdown
Implements #[ISSUE_NUMBER].

## What changed

- **Incident integration (incident.io):** API key connection, Sync validation, ListResources for severity.
- **On Incident (trigger):** Runs when incident.io sends webhooks (created/updated). User copies webhook URL from the UI into incident.io and pastes the signing secret back; Svix signature verification in HandleWebhook.
- **Create Incident (action):** Creates an incident via incident.io Incidents V2 API (name, summary, severity, visibility).
- **UI:** Mappers for both components, webhook URL copy button, and a short hint when the URL is HTTP (incident.io requires HTTPS in dev; set WEBHOOKS_BASE_URL and re-save).
- **Docs:** How to set WEBHOOKS_BASE_URL (inline, .env, export), manual test guide for Incident, and `make dev.start.with-webhook-tunnel` for local HTTPS webhooks.

## Why

To let users build workflows with incident.io: react to incident events (On Incident) and create incidents from SuperPlane (Create Incident).

## How

- **Connection:** API key in Settings → Integrations. incident.io has no API to register webhooks; user adds the SuperPlane webhook URL in incident.io Settings → Webhooks and pastes the signing secret into the trigger.
- **Webhooks:** Same pattern as other integrations: URL built from `WEBHOOKS_BASE_URL` when the canvas is saved. For local dev with incident.io (HTTPS-only), use a tunnel and set `WEBHOOKS_BASE_URL`; the UI shows the URL and an HTTPS hint when it’s HTTP.

## Testing

- On Incident: configured webhook in incident.io, created/updated incident, run appeared in SuperPlane.
- Create Incident: ran from workflow (trigger → Create Incident and Schedule → Create Incident), incident created in incident.io.
- Unit tests: `pkg/integrations/incident/*_test.go`.

## Demo

[Link to a short (1–2 min) video: setup integration, configure On Incident and webhook in incident.io, trigger run; configure and run Create Incident.]
```

Replace `[ISSUE_NUMBER]` and add your demo video link before opening the PR.
