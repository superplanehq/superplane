# Manual testing: Incident integration

This guide walks you through testing the **Incident** (incident.io) integration in SuperPlane: what it is, how it works, and exactly what to do step by step.

---

## 1. What you’re testing

- **Integration (base)**  
  A connection from SuperPlane to incident.io using an **API key**. SuperPlane stores the key and uses it to:
  - Check that the key works (when you connect).
  - Call incident.io’s API when the **Create Incident** action runs.

- **On Incident (trigger)**  
  Starts a workflow when something happens in incident.io (e.g. incident created or updated).  
  incident.io sends those events to SuperPlane via a **webhook**: you give incident.io a **webhook URL** from SuperPlane, and optionally a **signing secret** so SuperPlane can verify the requests.

- **Create Incident (action)**  
  Creates a new incident in incident.io (name, severity, visibility, summary) when the workflow runs. It uses the API key from the integration you connected.

So in practice:

1. You **connect** the integration (API key) so SuperPlane can talk to incident.io.
2. You add the **On Incident** trigger and (optionally) configure the webhook in incident.io so runs start when incidents change.
3. You add the **Create Incident** action so a workflow can create incidents in incident.io (e.g. when the trigger fires or when you use another trigger like Schedule).

---

## 2. What you need before starting

- SuperPlane running locally (e.g. `make dev.start`), and the UI open (e.g. http://localhost:8000).
- **Local HTTPS for incident.io:** incident.io only accepts HTTPS. The webhook URL is shown in the UI (copy from the On Incident trigger). If it’s HTTP, the UI explains how to get HTTPS; see [Local development: HTTPS webhook URL](#local-development-https-webhook-url) below.
- An **incident.io account** and an **API key**:
  - Go to [incident.io](https://incident.io) and sign in.
  - Open **Settings → API keys** (or [app.incident.io/settings/api-keys](https://app.incident.io/settings/api-keys)).
  - Create an API key and copy it (you’ll paste it in SuperPlane in Step 4).

Optional for full trigger test:

- A **webhook signing secret** from incident.io (you get this when you create a webhook endpoint in incident.io; we’ll do that in Step 8).

---

## 3. Get to your organization

1. In the browser, open SuperPlane (e.g. `http://localhost:8000`).
2. Log in if needed.
3. You should see your organization’s home (e.g. “Canvases” or a list of workflows). The URL will look like:  
   `http://localhost:8000/<organizationId>`  
   Remember this **organization ID** (or keep the tab open); you’ll use it for Settings and for workflows.

---

## 4. Connect the Incident integration (API key)

This step stores an incident.io API key in SuperPlane so the **Create Incident** action (and severity list) can work.

1. Go to **Settings → Integrations**:
   - Either: click your **organization name/logo** (top left) and choose **Settings**, then in the left sidebar click **Integrations**.
   - Or: open this URL (replace `<organizationId>` with your org ID):  
     `http://localhost:8000/<organizationId>/settings/integrations`

2. On the Integrations page you’ll see a list of integration types (GitHub, Slack, **Incident**, etc.).

3. Find the card for **Incident** (label “Incident”, description about managing incidents in incident.io).

4. Click the **Connect** button on that card.

5. A **Connect Incident** modal opens:
   - **Integration name:** Give this connection a name, e.g. `incident-production` or `my-incident`. (This is only a label in SuperPlane.)
   - **API Key:** Paste the incident.io API key you copied earlier.

6. Click **Connect** in the modal.

7. You should be taken to the detail page for this integration. Check that the status is **Ready** (or similar). If there’s an error, the API key may be wrong or not have the right permissions in incident.io.

You’ve now completed the “base integration” test: SuperPlane can talk to incident.io with this API key.

---

## 5. Open or create a workflow (canvas)

1. From the organization home, either:
   - Open an existing workflow (canvas), or  
   - Click **New workflow** / **Create canvas** (or go to `/<organizationId>/canvases/new`) to create a new one.

2. You should see the **workflow editor**: a canvas with a **Components** (or “Building blocks”) sidebar on the left. The right side may show a panel when you select a node.

---

## 6. Add the “On Incident” trigger

This adds the trigger that can start the workflow when incident.io sends events (we’ll hook up the webhook later).

1. In the **left sidebar**, find the section **Incident** (it may show “Incident (2)” for 2 building blocks).

2. Expand it if needed. You should see:
   - **On Incident** (trigger)
   - **Create Incident** (action)

3. **Drag “On Incident”** onto the canvas and drop it.

4. The trigger node appears (e.g. “On Incident” with an incident icon).

5. **Configure the trigger** (click the node so the right sidebar opens):
   - **Events:** Choose at least one, e.g. **Incident created** (and optionally **Incident updated**).
   - **Signing secret:** Leave this **empty** for now. It is optional so you can save first and get the webhook URL. You’ll paste the signing secret here in Step 8 after creating the endpoint in incident.io.
   - Ensure an **Incident integration** is selected for this trigger (e.g. the one you connected in Step 4). If the sidebar asks you to “Select integration”, pick your connected Incident instance. Without it, the Settings form and webhook URL section may not show.

6. **Save the workflow** (Save button or Ctrl/Cmd+S).

7. **Get the webhook URL:**
   - Click the **On Incident** trigger node (select it).
   - In the **right sidebar**, switch to the **Settings** tab (not “Latest” or “Runs”).
   - Scroll down below the **Events** and **Signing secret** fields.
   - You should see a section **“incident.io Webhook Setup”** with numbered steps and a **“Webhook URL”** (a long URL). Copy that URL; you’ll use it in incident.io in Step 8.
   - If you only see “\[URL GENERATED ONCE THE CANVAS IS SAVED\]”, save the workflow again and reselect the trigger—the URL is created when the canvas is saved and the trigger is set up.

You’ve added and configured the trigger. The workflow can now be started by the On Incident webhook once the URL is registered in incident.io.

---

## 7. Add the “Create Incident” action and connect it

This step tests the **Create Incident** action and how it uses the integration’s API key.

1. In the same **left sidebar**, under **Incident**, drag **Create Incident** onto the canvas (e.g. to the right of the trigger).

2. **Connect the trigger to the action:**  
   Drag from the **output handle** (right side) of the **On Incident** node to the **input handle** (left side) of the **Create Incident** node, so that when the trigger runs, it runs the action.

3. **Configure the Create Incident node** (click it, use the right panel):
   - **Incident name:** e.g. `Test from SuperPlane` (or an expression like `$['On Incident'].incident.name` if you want to reuse data from the trigger).
   - **Summary:** optional, e.g. `Created by SuperPlane workflow`.
   - **Severity:** optional; if your integration is connected and ready, you can pick one from the list (from incident.io).
   - **Visibility:** e.g. **Public**.

4. **Save the workflow** again.

Now:
- When the **On Incident** trigger receives a webhook from incident.io, the workflow will run and **Create Incident** will create an incident using the API key you configured.
- You can also test **Create Incident** with another trigger (e.g. **Schedule** or **Webhook**) if you prefer.

---

## 8. Register the webhook in incident.io so the trigger actually runs

Until you do this, incident.io won’t send events to SuperPlane, so the **On Incident** trigger won’t start runs. This step links incident.io to your workflow.

1. In **incident.io**, go to **Settings → Webhooks** (or [app.incident.io/settings](https://app.incident.io/settings) and find Webhooks).

2. **Create a new webhook endpoint:**
   - **Endpoint URL:** Paste the **webhook URL** you copied from SuperPlane in Step 6 (the one shown for the On Incident trigger after saving).
   - **Subscriptions:** Subscribe to the events that match what you chose in the trigger, e.g.:
     - **Public incident created (v2)**
     - **Public incident updated (v2)**

3. Save the endpoint in incident.io. incident.io will show a **Signing secret** for this endpoint (often starting with `whsec_...`). **Copy it.**

4. Back in **SuperPlane**, open your workflow, click the **On Incident** trigger node, go to the **Settings** tab in the right sidebar, and in **Signing secret** paste the secret you just copied. **Save** the workflow.

After this, when you create or update an incident in incident.io (matching the events you subscribed to), incident.io will send a request to SuperPlane’s webhook URL, and your workflow should run (trigger → Create Incident, if connected).

---

## 9. How to “run” the workflow and what to check

### Option A: Trigger via incident.io (full path)

1. In **incident.io**, create a new incident (or update an existing one), so that the event type matches what your webhook is subscribed to (e.g. “Public incident created (v2)”).
2. In **SuperPlane**, open your workflow and check **runs** or **executions** (e.g. “Latest” or “Runs” in the right panel, or a run list view).
3. You should see a **new run** started by the On Incident trigger, with the trigger’s payload (e.g. `event_type`, `incident`).
4. If the trigger is connected to Create Incident, that run should also show the **Create Incident** node as executed, with output (incident id, name, reference, permalink, etc.). In incident.io you should see the newly created incident.

### Option B: Test only “Create Incident” (without real webhook)

1. Create a **new workflow** (or change this one).
2. Add a trigger that you can fire yourself, e.g. **Schedule** (run once) or **Webhook** (you’ll POST to a URL).
3. Add **Create Incident** and connect it to that trigger. Configure name, summary, severity, visibility. Ensure the workflow uses the **Incident** integration you connected (the one with the API key).
4. Save, then **run** the trigger (e.g. run the schedule or send a request to the webhook URL).
5. Check that a run appears and that **Create Incident** succeeded. Open the run and the Create Incident node’s output: you should see incident details (id, name, reference, permalink, etc.). In incident.io, the new incident should appear.

---

## 10. Quick checklist

- [ ] **Integration:** Settings → Integrations → Connect **Incident** with API key → status **Ready**.
- [ ] **Trigger:** Workflow has **On Incident** with at least one event; signing secret set if you use real webhooks; workflow saved and webhook URL copied.
- [ ] **Action:** Workflow has **Create Incident** connected to a trigger; name (and optional summary/severity/visibility) set; workflow saved.
- [ ] **Webhook (optional):** incident.io webhook endpoint created with SuperPlane’s URL and correct events; signing secret pasted into On Incident and saved.
- [ ] **Run:** Either create/update incident in incident.io and see a run in SuperPlane, or run the workflow with Schedule/Webhook and see Create Incident succeed and the incident in incident.io.

If all of the above work, the Incident integration and both components are working as intended.

---

## Local development: HTTPS webhook URL

incident.io only accepts **HTTPS** webhook URLs. The URL you use in incident.io is the one shown in the UI (On Incident trigger → Settings → Webhook URL). Copy it with the copy button and paste it into incident.io.

If the UI shows an HTTP URL, it will show a short hint: set **WEBHOOKS_BASE_URL** when starting the app and re-save the workflow to get an HTTPS URL. For how to set it (inline, `.env`, or export) and expose your local app over HTTPS (tunnel), see **[Connecting to third-party services during development](../contributing/connecting-to-3rdparty-services-from-development.md)**. Same approach as GitHub, PagerDuty, and other webhook integrations.

---

## Troubleshooting

- **incident.io rejects the URL (“Endpoint URL schemes must be https…”):** incident.io only accepts HTTPS. Follow [Connecting to third-party services during development](../contributing/connecting-to-3rdparty-services-from-development.md) (tunnel + `WEBHOOKS_BASE_URL`), then re-save the workflow to refresh the webhook URL.
- **Integration not “Ready”:** Check the API key in incident.io (Settings → API keys) and its permissions. Fix or regenerate the key, then in SuperPlane go to the integration detail page and update the configuration (or remove and reconnect).
- **Trigger never runs:** Confirm the webhook URL in incident.io is exactly the one from the On Incident node; that the events in incident.io match the trigger (e.g. “Public incident created (v2)”); and that the signing secret in SuperPlane matches the one in incident.io. Check app logs for webhook errors (e.g. 403 for bad signature).
- **Create Incident fails:** Ensure the integration is Ready and the workflow is using it. Check error message in the run (e.g. “unauthorized” → API key; “validation” → check name/visibility/severity).
- **No “Incident” in Integrations list:** Ensure the app is built with the Incident integration (e.g. `make check.build.ui` and restart the server). The integration is registered under the name **Incident** (label “Incident”).
