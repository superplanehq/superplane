---
app: "github"
label: "On Issue"
name: "github.onIssue"
type: "trigger"
---

# On Issue

Listen to issue events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| actions | Actions | multi-select | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "opened",
    "assignee": null,
    "issue": {
      "html_url": "https://github.com/acme/widgets/issues/42",
      "number": 42,
      "state": "open",
      "title": "Fix flaky build",
      "user": {
        "login": "octocat"
      }
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

