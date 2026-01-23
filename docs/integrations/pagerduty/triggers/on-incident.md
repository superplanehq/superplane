---
app: "pagerduty"
label: "On Incident"
name: "pagerduty.onIncident"
type: "trigger"
---

# On Incident

Listen to incident events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| events | Events | multi-select | yes | - |
| service | Service | app-installation-resource | yes | The PagerDuty service to monitor for incidents |
| urgencies | Urgencies | multi-select | no | Filter incidents by urgency |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "agent": {
      "html_url": "https://acme.pagerduty.com/users/PLH1HKV",
      "id": "PLH1HKV",
      "self": "https://api.pagerduty.com/users/PLH1HKV",
      "summary": "Tenex Engineer",
      "type": "user_reference"
    },
    "incident": {
      "assignees": [
        {
          "html_url": "https://acme.pagerduty.com/users/PTUXL6G",
          "id": "PTUXL6G",
          "self": "https://api.pagerduty.com/users/PTUXL6G",
          "summary": "User 123",
          "type": "user_reference"
        }
      ],
      "conference_bridge": {
        "conference_number": "+1 1234123412,,987654321#",
        "conference_url": "https://example.com"
      },
      "created_at": "2020-04-09T15:16:27Z",
      "escalation_policy": {
        "html_url": "https://acme.pagerduty.com/escalation_policies/PUS0KTE",
        "id": "PUS0KTE",
        "self": "https://api.pagerduty.com/escalation_policies/PUS0KTE",
        "summary": "Default",
        "type": "escalation_policy_reference"
      },
      "html_url": "https://acme.pagerduty.com/incidents/PGR0VU2",
      "id": "PGR0VU2",
      "incident_key": "d3640fbd41094207a1c11e58e46b1662",
      "number": 2,
      "priority": {
        "html_url": "https://acme.pagerduty.com/account/incident_priorities",
        "id": "PSO75BM",
        "self": "https://api.pagerduty.com/priorities/PSO75BM",
        "summary": "P1",
        "type": "priority_reference"
      },
      "reopened_at": "2020-10-02T18:45:22Z",
      "resolve_reason": null,
      "self": "https://api.pagerduty.com/incidents/PGR0VU2",
      "service": {
        "html_url": "https://acme.pagerduty.com/services/PF9KMXH",
        "id": "PF9KMXH",
        "self": "https://api.pagerduty.com/services/PF9KMXH",
        "summary": "API Service",
        "type": "service_reference"
      },
      "status": "triggered",
      "teams": [
        {
          "html_url": "https://acme.pagerduty.com/teams/PFCVPS0",
          "id": "PFCVPS0",
          "self": "https://api.pagerduty.com/teams/PFCVPS0",
          "summary": "Engineering",
          "type": "team_reference"
        }
      ],
      "title": "A little bump in the road",
      "type": "incident",
      "urgency": "high"
    }
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "pagerduty.onIncident"
}
```

