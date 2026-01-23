---
app: "github"
label: "Create Issue"
name: "github.createIssue"
type: "component"
---

# Create Issue

Create a new issue in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| title | Title | string | yes | - |
| body | Body | text | no | - |
| assignees | Assignees | list | no | - |
| labels | Labels | list | no | - |

## Example Output

```json
{
  "data": {
    "html_url": "https://github.com/acme/widgets/issues/42",
    "id": 101,
    "number": 42,
    "state": "open",
    "title": "Fix flaky build",
    "user": {
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

