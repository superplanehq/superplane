---
app: "github"
label: "On Tag Created"
name: "github.onTagCreated"
type: "trigger"
---

# On Tag Created

Listen to GitHub tag creation events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| tags | Tags | any-predicate-list | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "description": "Example repository for webhook payloads",
    "master_branch": "main",
    "pusher_type": "user",
    "ref": "v1.2.3",
    "ref_type": "tag",
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
  "type": "github.tagCreated"
}
```

