---
app: "pagerduty"
label: "Create Incident"
name: "pagerduty.createIncident"
type: "component"
---

# Create Incident

Create a new incident in PagerDuty

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| title | Incident Title | string | yes | A succinct description of the incident |
| description | Description | string | no | Additional details about the incident |
| urgency | Urgency | select | yes | - |
| service | Service | app-installation-resource | yes | The PagerDuty service to create the incident for |
| fromEmail | From Email | string | no | Email address of a valid PagerDuty user. Required for App OAuth and account-level API tokens, optional for user-level API tokens. |

## Example Output

```json
{
  "data": {
    "incident": {
      "assigned_via": "escalation_policy",
      "assignments": [
        {
          "assignee": {
            "html_url": "https://subdomain.pagerduty.com/users/PXPGF42",
            "id": "PXPGF42",
            "self": "https://api.pagerduty.com/users/PXPGF42",
            "summary": "Earline Greenholt",
            "type": "user_reference"
          },
          "at": "2015-11-10T00:31:52Z"
        }
      ],
      "created_at": "2015-10-06T21:30:42Z",
      "escalation_policy": {
        "html_url": "https://subdomain.pagerduty.com/escalation_policies/PT20YPA",
        "id": "PT20YPA",
        "self": "https://api.pagerduty.com/escalation_policies/PT20YPA",
        "summary": "Another Escalation Policy",
        "type": "escalation_policy_reference"
      },
      "first_trigger_log_entry": {
        "html_url": "https://subdomain.pagerduty.com/incidents/PT4KHLK/log_entries/Q02JTSNZWHSEKV",
        "id": "Q02JTSNZWHSEKV",
        "self": "https://api.pagerduty.com/log_entries/Q02JTSNZWHSEKV?incident_id=PT4KHLK",
        "summary": "Triggered through the API",
        "type": "trigger_log_entry_reference"
      },
      "html_url": "https://subdomain.pagerduty.com/incidents/PT4KHLK",
      "id": "PT4KHLK",
      "incident_key": "baf7cf21b1da41b4b0221008339ff357",
      "incident_number": 1234,
      "incident_type": {
        "name": "major_incident"
      },
      "last_status_change_at": "2015-10-06T21:38:23Z",
      "last_status_change_by": {
        "html_url": "https://subdomain.pagerduty.com/users/PXPGF42",
        "id": "PXPGF42",
        "self": "https://api.pagerduty.com/users/PXPGF42",
        "summary": "Earline Greenholt",
        "type": "user_reference"
      },
      "priority": {
        "id": "P53ZZH5",
        "self": "https://api.pagerduty.com/priorities/P53ZZH5",
        "summary": "P2",
        "type": "priority_reference"
      },
      "resolved_at": null,
      "self": "https://api.pagerduty.com/incidents/PT4KHLK",
      "service": {
        "html_url": "https://subdomain.pagerduty.com/service-directory/PWIXJZS",
        "id": "PWIXJZS",
        "self": "https://api.pagerduty.com/services/PWIXJZS",
        "summary": "My Mail Service",
        "type": "service_reference"
      },
      "status": "triggered",
      "summary": "[#1234] The server is on fire.",
      "teams": [
        {
          "html_url": "https://subdomain.pagerduty.com/teams/PQ9K7I8",
          "id": "PQ9K7I8",
          "self": "https://api.pagerduty.com/teams/PQ9K7I8",
          "summary": "Engineering",
          "type": "team_reference"
        }
      ],
      "title": "The server is on fire.",
      "type": "incident",
      "updated_at": "2015-10-06T21:40:23Z",
      "urgency": "high"
    }
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "pagerduty.incident"
}
```

