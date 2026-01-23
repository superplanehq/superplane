---
app: "github"
label: "On Release"
name: "github.onRelease"
type: "trigger"
---

# On Release

Listen to release events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| actions | Actions | multi-select | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "published",
    "release": {
      "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
      "id": 3001,
      "name": "Release 1.2.3",
      "tag_name": "v1.2.3"
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
  "type": "github.release"
}
```

