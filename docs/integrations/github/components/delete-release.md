---
app: "github"
label: "Delete Release"
name: "github.deleteRelease"
type: "component"
---

# Delete Release

Delete a release from a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| releaseStrategy | Release to Delete | select | yes | How to identify which release to delete |
| tagName | Tag Name | string | no | Git tag identifying the release to delete. Supports template variables from previous steps. |
| deleteTag | Also delete Git tag | boolean | no | When enabled, also deletes the associated Git tag from the repository |

## Example Output

```json
{
  "data": {
    "deleted_at": "2026-01-16T17:55:00Z",
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_deleted": true,
    "tag_name": "v1.2.3"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

