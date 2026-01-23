---
app: "github"
label: "On Pull Request"
name: "github.onPullRequest"
type: "trigger"
---

# On Pull Request

Listen to pull request events

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
    "number": 101,
    "pull_request": {
      "html_url": "https://github.com/acme/widgets/pull/101",
      "number": 101,
      "state": "open",
      "title": "Add new endpoint",
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
  "type": "github.pullRequest"
}
```

