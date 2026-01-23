---
app: "github"
label: "On Branch Created"
name: "github.onBranchCreated"
type: "trigger"
---

# On Branch Created

Listen to GitHub branch creation events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| branches | Branches | any-predicate-list | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "description": "Example repository for webhook payloads",
    "master_branch": "main",
    "pusher_type": "user",
    "ref": "feature/new-endpoint",
    "ref_type": "branch",
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
  "type": "github.branchCreated"
}
```

