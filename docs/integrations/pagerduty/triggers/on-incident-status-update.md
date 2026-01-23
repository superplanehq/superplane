---
app: "pagerduty"
label: "On Incident Status Update"
name: "pagerduty.onIncidentStatusUpdate"
type: "trigger"
---

# On Incident Status Update

Listen to incident status update events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| service | Service | app-installation-resource | yes | The PagerDuty service to monitor for incident status updates |
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
      "html_url": "https://acme.pagerduty.com/incidents/PGR0VU2",
      "id": "PGR0VU2",
      "self": "https://api.pagerduty.com/incidents/PGR0VU2",
      "summary": "A little bump in the road",
      "type": "incident_reference"
    },
    "status_update": {
      "created_at": "2026-01-19T12:30:00Z",
      "id": "P1234567",
      "message": "We have identified the issue and are working on a fix.",
      "sender": {
        "html_url": "https://acme.pagerduty.com/users/PLH1HKV",
        "id": "PLH1HKV",
        "self": "https://api.pagerduty.com/users/PLH1HKV",
        "summary": "Tenex Engineer",
        "type": "user_reference"
      }
    }
  },
  "timestamp": "2026-01-19T12:30:00Z",
  "type": "pagerduty.incident.status_update_published"
}
```

