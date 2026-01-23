---
app: "github"
label: "Update Release"
name: "github.updateRelease"
type: "component"
---

# Update Release

Update an existing release in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| releaseStrategy | Release Strategy | select | yes | How to identify which release to update |
| tagName | Tag Name | string | no | Git tag identifying the release to update. Supports template variables from previous steps. |
| name | Release Name | string | no | Update the release title (leave empty to keep current) |
| generateReleaseNotes | Generate release notes | boolean | no | Automatically generate release notes from commits since the last release. If body is also provided, custom text is appended. |
| body | Release Notes | text | no | Update release description (leave empty to keep current) |
| draft | Draft | boolean | no | Mark release as draft or publish it |
| prerelease | Prerelease | boolean | no | Mark as prerelease or stable release |

## Example Output

```json
{
  "data": {
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_name": "v1.2.3",
    "updated_at": "2026-01-16T17:50:00Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

