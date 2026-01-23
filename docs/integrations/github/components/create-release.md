---
app: "github"
label: "Create Release"
name: "github.createRelease"
type: "component"
---

# Create Release

Create a new release in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| versionStrategy | Version Strategy | select | yes | How to determine the release version |
| tagName | Tag Name | string | no | The name of the tag to create the release for |
| name | Release Name | string | no | The title of the release |
| draft | Draft | boolean | no | Mark this release as a draft |
| prerelease | Prerelease | boolean | no | Mark this release as a prerelease |
| generateReleaseNotes | Generate release notes | boolean | no | Automatically generate release notes from commits since the last release |
| body | Additional notes | text | no | Optional text to append after auto-generated release notes. If auto-generation is off, this becomes the entire release description. |

## Example Output

```json
{
  "data": {
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_name": "v1.2.3"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

