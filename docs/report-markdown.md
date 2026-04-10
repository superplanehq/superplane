# Report Markdown Reference

The report template field on components and triggers supports enriched markdown. Templates are resolved at execution time — use `{{ root().data.field }}` for trigger data or the full expression syntax for component data.

## Inline Badges

Use backtick notation with a `type:label` pattern to render colored pill badges:

```
`status:passed` `success:deployed` `error:3 failures` `warning:deprecated` `info:v2.1.0` `duration:12s`
```

Supported badge types:

| Type | Color |
|------|-------|
| `status` / `success` | Green |
| `warning` | Amber |
| `error` | Red |
| `info` | Blue |
| `duration` | Gray |

Any backtick text that doesn't match the `type:label` pattern renders as normal inline code.

## Syntax-Highlighted Code Blocks

Fenced code blocks with a language tag get syntax highlighting:

~~~
```json
{ "status": "ok", "count": 42 }
```

```yaml
deploy:
  target: production
  replicas: 3
```
~~~

## GitHub-Style Admonitions

Use blockquote alert syntax for colored callout boxes:

```markdown
> [!NOTE]
> Deployment completed successfully to production.

> [!TIP]
> You can speed this up by enabling parallel builds.

> [!IMPORTANT]
> This release includes breaking API changes.

> [!WARNING]
> 3 deprecation warnings detected during build.

> [!CAUTION]
> Database migration exceeded the 30s threshold (took 45s).
```

Supported types: `NOTE` (blue), `TIP` (green), `IMPORTANT` (purple), `WARNING` (amber), `CAUTION` (red).

## Collapsible Sections

Use HTML `<details>` / `<summary>` tags:

```markdown
<details>
<summary>Full error log (click to expand)</summary>

Error: connection refused at db:5432
  at connect (src/db.ts:42)
  at main (src/index.ts:10)

</details>
```

## Tables

Standard GFM table syntax. Tables render with styled headers and horizontal scroll on overflow:

```markdown
| File | Status | Lines |
|------|--------|------:|
| src/main.go | modified | +12 -3 |
| README.md | added | +45 |
| config.yaml | deleted | -8 |
```

## Links

All links open in a new tab with an external link icon:

```markdown
[View workflow run](https://github.com/org/repo/actions/runs/123)
```

## Images

Images render with rounded corners, a border, and a max height:

```markdown
![Screenshot](https://example.com/screenshot.png)
```

## Checkboxes (Task Lists)

```markdown
- [x] Build passed
- [x] Tests passed (142/142)
- [ ] Deploy to staging
- [ ] Smoke tests
```

## Full Example

A realistic report template combining several features:

```markdown
## Deploy `status:passed` `duration:34s`

Deployed **{{ root().data.head_commit.message }}** to production.

> [!NOTE]
> All 142 tests passed. No regressions detected.

| Step | Result | Duration |
|------|--------|----------|
| Build | `success:passed` | `duration:12s` |
| Test | `success:passed` | `duration:18s` |
| Deploy | `success:passed` | `duration:4s` |

<details>
<summary>Build output</summary>

GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server
Binary size: 24MB

</details>
```
