---
app: "github"
label: "Update Issue"
name: "github.updateIssue"
type: "component"
---

# Update Issue

Update a GitHub issue

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| issueNumber | Issue Number | number | yes | - |
| title | Title | string | no | - |
| body | Body | text | no | - |
| state | State | select | no | - |
| assignees | Assignees | list | no | - |
| labels | Labels | list | no | - |

## Example Output

```json
{
  "data": {
    "html_url": "https://github.com/acme/widgets/issues/42",
    "id": 101,
    "number": 42,
    "state": "closed",
    "title": "Fix flaky build",
    "updated_at": "2026-01-16T17:40:00Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

