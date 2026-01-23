---
app: "github"
label: "Publish Commit Status"
name: "github.publishCommitStatus"
type: "component"
---

# Publish Commit Status

Publish a status check to a GitHub commit

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| sha | Commit SHA | string | yes | The full SHA of the commit to attach the status to |
| state | State | select | yes | - |
| context | Context | string | yes | A label to identify this status check |
| description | Description | text | no | Short description of the status (max ~140 characters) |
| targetUrl | Target URL | string | no | e.g. Link to build logs, test results, ... |

## Example Output

```json
{
  "data": {
    "context": "ci/build",
    "created_at": "2026-01-16T17:45:00Z",
    "description": "All checks passed",
    "state": "success",
    "target_url": "https://ci.example.com/builds/123",
    "updated_at": "2026-01-16T17:45:10Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.commitStatus"
}
```

