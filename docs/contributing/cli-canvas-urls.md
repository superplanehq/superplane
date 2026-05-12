# Canvas URL Pattern

When working with canvases programmatically (scripts, AI agents, CI jobs), the correct URL for viewing a canvas in the SuperPlane UI follows this pattern:

```
https://app.superplane.com/{orgId}/canvases/{canvasId}
```

**Both segments are required.** Constructing a URL without the `{orgId}` prefix will redirect to the homepage and will not open the intended canvas.

## Getting the URL from the CLI

The `canvases create` command prints the full URL on success:

```
$ superplane canvases create my-canvas
Canvas "my-canvas" created (ID: 0f3c...)
URL: https://app.superplane.com/<org-id>/canvases/0f3c...
```

The `canvases get` command supports a `--url` flag that outputs only the URL, suitable for piping into other tools:

```
$ superplane canvases get my-canvas --url
https://app.superplane.com/<org-id>/canvases/0f3c...
```

## Use cases

- **AI agents** that need to surface a clickable canvas link in their output
- **CI jobs** that post canvas links into chat or issue trackers after creation
- **Shell scripts** that open a canvas in the browser: `open $(superplane canvases get my-canvas --url)`