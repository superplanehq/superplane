# Track C Reference — Demo & Glue (Fedja)

## Mock PagerDuty Incident Payload

Use this to trigger the Incident Copilot without a real PagerDuty account.
Send as POST to the canvas webhook URL.

```json
{
  "event": {
    "id": "01DEN4HPBQAAAG05V5QQYBRZMF",
    "event_type": "incident.triggered",
    "resource_type": "incident",
    "occurred_at": "2026-03-28T14:30:00.000Z",
    "agent": {
      "html_url": "https://acme.pagerduty.com/users/PLH1HKV",
      "id": "PLH1HKV",
      "self": "https://api.pagerduty.com/users/PLH1HKV",
      "summary": "Monitoring Bot",
      "type": "user_reference"
    },
    "data": {
      "id": "PGR0VU2",
      "type": "incident",
      "self": "https://api.pagerduty.com/incidents/PGR0VU2",
      "html_url": "https://acme.pagerduty.com/incidents/PGR0VU2",
      "number": 42,
      "status": "triggered",
      "incident_key": "hackathon-demo-incident-001",
      "created_at": "2026-03-28T14:30:00Z",
      "title": "API Gateway: 5xx error rate spike to 15% on /api/v1/orders",
      "urgency": "high",
      "service": {
        "html_url": "https://acme.pagerduty.com/services/PF9KMXH",
        "id": "PF9KMXH",
        "self": "https://api.pagerduty.com/services/PF9KMXH",
        "summary": "API Gateway (Production)",
        "type": "service_reference"
      },
      "assignees": [
        {
          "html_url": "https://acme.pagerduty.com/users/PTUXL6G",
          "id": "PTUXL6G",
          "self": "https://api.pagerduty.com/users/PTUXL6G",
          "summary": "Dragan Petrovic (On-Call SRE)",
          "type": "user_reference"
        }
      ],
      "escalation_policy": {
        "html_url": "https://acme.pagerduty.com/escalation_policies/PUS0KTE",
        "id": "PUS0KTE",
        "self": "https://api.pagerduty.com/escalation_policies/PUS0KTE",
        "summary": "Production - Critical",
        "type": "escalation_policy_reference"
      },
      "teams": [
        {
          "html_url": "https://acme.pagerduty.com/teams/PFCVPS0",
          "id": "PFCVPS0",
          "self": "https://api.pagerduty.com/teams/PFCVPS0",
          "summary": "Platform Engineering",
          "type": "team_reference"
        }
      ],
      "priority": {
        "html_url": "https://acme.pagerduty.com/priorities/PSO75BM",
        "id": "PSO75BM",
        "self": "https://api.pagerduty.com/priorities/PSO75BM",
        "summary": "P1",
        "type": "priority_reference"
      },
      "conference_bridge": {
        "conference_number": "+1 555-123-4567,,987654321#",
        "conference_url": "https://meet.google.com/abc-defg-hij"
      },
      "body": {
        "type": "incident_body",
        "details": "5xx error rate on API Gateway spiked from 0.1% to 15.3% at 14:28 UTC. Affects /api/v1/orders endpoint. 1,247 users impacted in last 2 minutes. Correlated with deployment deploy-api-v2.14.3 at 14:25 UTC."
      }
    }
  }
}
```

## Mock GitHub Release (Latest Deploy)

If using HTTP component to simulate GitHub data, return this:

```json
{
  "id": 12345678,
  "tag_name": "v2.14.3",
  "name": "Release v2.14.3 - Order Service Refactor",
  "body": "## Changes\n- Refactored order validation logic\n- Migrated to new payment gateway client\n- Updated database connection pooling\n\n## Authors\n- @braca (order validation)\n- @fedja (payment gateway)\n\n## Risk: Medium\nDatabase connection pool size changed from 20 to 50",
  "draft": false,
  "prerelease": false,
  "created_at": "2026-03-28T14:25:00Z",
  "published_at": "2026-03-28T14:25:30Z",
  "author": {
    "login": "braca",
    "id": 87654321
  }
}
```

## Mock Datadog Metrics Response

For the HTTP component fetching Datadog metrics:

```json
{
  "series": [
    {
      "metric": "api.gateway.error_rate_5xx",
      "points": [
        [1711633200, 0.1],
        [1711633260, 0.3],
        [1711633320, 2.1],
        [1711633380, 8.7],
        [1711633440, 15.3],
        [1711633500, 14.8]
      ],
      "tags": ["service:api-gateway", "env:production"]
    },
    {
      "metric": "api.gateway.latency_p99",
      "points": [
        [1711633200, 120],
        [1711633260, 145],
        [1711633320, 890],
        [1711633380, 2340],
        [1711633440, 4500],
        [1711633500, 4200]
      ],
      "tags": ["service:api-gateway", "env:production"]
    }
  ],
  "status": "ok",
  "query": "avg:api.gateway.error_rate_5xx{env:production} by {service}"
}
```

## Mock PagerDuty Log Entries

```json
{
  "log_entries": [
    {
      "type": "trigger_log_entry",
      "created_at": "2026-03-28T14:30:00Z",
      "summary": "Triggered by Datadog monitor: API 5xx Error Rate > 5%"
    },
    {
      "type": "notify_log_entry",
      "created_at": "2026-03-28T14:30:05Z",
      "summary": "Notified Dragan Petrovic via push notification"
    },
    {
      "type": "annotate_log_entry",
      "created_at": "2026-03-28T14:30:10Z",
      "summary": "Correlated with deploy-api-v2.14.3 (14:25 UTC)"
    }
  ]
}
```

## Slack Evidence Pack Template

What the final Slack message should look like:

```
:rotating_light: *INCIDENT TRIAGE — AUTO-GENERATED*

*API Gateway: 5xx error rate spike to 15% on /api/v1/orders*
Priority: P1 | Service: API Gateway (Production)
Assignee: Dragan Petrovic (On-Call SRE)

---

*SEVERITY: P1 — Critical*
Customer-facing order flow is down for ~1,200 users.

*LIKELY ROOT CAUSE:*
1. (85%) Deploy v2.14.3 changed DB connection pool 20->50, likely exhausting DB connections
2. (10%) Payment gateway client migration introduced timeout regression
3. (5%) Unrelated infrastructure issue

*AFFECTED SYSTEMS:*
- API Gateway /api/v1/orders endpoint
- Order Service (downstream)
- ~1,247 active users in checkout flow

*RECOMMENDED ACTIONS:*
1. :arrow_right: Rollback deploy v2.14.3 immediately (ETA: 3 min)
2. Check DB connection count: `SELECT count(*) FROM pg_stat_activity`
3. Monitor error rate after rollback for 5 min
4. If not resolved, escalate to Database Team

*ESCALATION:*
- Current: Platform Engineering (Dragan)
- Next: Database Team (@db-oncall) if rollback doesn't resolve

---
:clock1: Triage generated by SuperPlane Incident Copilot in 47 seconds
:link: <https://acme.pagerduty.com/incidents/PGR0VU2|View in PagerDuty>
```

## Demo Script — Detailed

### Setup (before demo starts)
1. Have the canvas open in browser, zoomed to show full flow
2. Have Slack channel open in a second tab/window
3. Have a `curl` command ready to fire the mock webhook

### Act 1: The Problem (30 seconds)
"It's 3am. PagerDuty fires. Your engineer opens 5 tabs: PagerDuty, Datadog, GitHub, the runbook, Slack. Spends 20 minutes gathering context before understanding the problem. We fixed that."

### Act 2: The Copilot (90 seconds)
1. Show the canvas: "Here's our Incident Copilot — built entirely in SuperPlane's Canvas"
2. Walk through the flow: trigger, parallel data collection, AI triage, Slack output
3. Fire the webhook: `curl -X POST <webhook-url> -H "Content-Type: application/json" -d @mock-incident.json`
4. Watch nodes light up in real-time (canvas execution visualization)
5. Switch to Slack: show the evidence pack arriving
6. "47 seconds. From alert to actionable triage."

### Act 3: The Safety Net (60 seconds)
1. "But how do you know this workflow is safe before it goes live?"
2. Run the linter: show green pass
3. Delete an edge in the canvas
4. Run linter again: red fail — "Orphan node detected"
5. Remove the approval gate
6. Run linter again: warning — "Destructive action without approval"
7. "The linter catches mistakes before they reach production."

### Act 4: What's Next (30 seconds)
- Linter as a built-in pre-publish hook
- Template library for common incident types
- Self-healing: AI suggests fixes when linter finds issues

## Screenshot Checklist

Capture these during the build (2:15-2:30):
- [ ] Full canvas view with all nodes connected
- [ ] Canvas with nodes executing (green highlights)
- [ ] Slack evidence pack message
- [ ] Linter output: passing (green)
- [ ] Linter output: failing (red)
- [ ] Before/after side-by-side

## Curl Command for Demo

Save this as `mock-incident.json` and use:
```bash
curl -X POST http://localhost:8000/api/v1/webhooks/<webhook-id> \
  -H "Content-Type: application/json" \
  -d @docs/mock-incident.json
```

(Get the webhook-id from the canvas trigger configuration after setup)
