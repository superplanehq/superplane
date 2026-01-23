---
app: "github"
label: "On PR Review Comment"
name: "github.onPullRequestReviewComment"
type: "trigger"
---

# On PR Review Comment

Listen to pull request review comment events

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
      "body": "Looks good to me",
      "html_url": "https://github.com/acme/widgets/pull/101#discussion_r7001",
      "id": 7001
    },
    "pull_request": {
      "html_url": "https://github.com/acme/widgets/pull/101",
      "number": 101,
      "title": "Add new endpoint"
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
  "type": "github.pullRequestReviewComment"
}
```

