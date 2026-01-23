---
app: "github"
label: "On Issue Comment"
name: "github.onIssueComment"
type: "trigger"
---

# On Issue Comment

Listen to issue comment events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| contentFilter | Content Filter | string | no | Optional regex pattern to filter comments by content |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "created",
    "comment": {
      "body": "I can reproduce this",
      "html_url": "https://github.com/acme/widgets/issues/42#issuecomment-5001",
      "id": 5001
    },
    "issue": {
      "html_url": "https://github.com/acme/widgets/issues/42",
      "number": 42,
      "title": "Fix flaky build"
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
  "type": "github.issueComment"
}
```

