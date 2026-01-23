---
app: "github"
label: "Get Issue"
name: "github.getIssue"
type: "component"
---

# Get Issue

Get a GitHub issue by number

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| issueNumber | Issue Number | string | yes | - |

## Example Output

```json
{
  "data": {
    "comments": 3,
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

