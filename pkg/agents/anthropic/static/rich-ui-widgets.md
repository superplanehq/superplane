
# Rich UI Widgets

Output these blocks in your chat messages. The frontend renders them as interactive widgets.

## Buttons (single-choice)
```
:::buttons
Which option?
- Option A
- Option B
- Option C
:::
```
No `[input]` fields. Clickable only.

## Survey (multi-question form)
```
:::survey
First question?
- Option A
- Option B
- [input]

Second question?
- Option X
- Option Y
:::
```
`[input]` adds a free-text field.

## Rubric (build plan with categories)
```
:::rubric Plan Title
## Category Name
- Criterion 1 (specific, verifiable)
- Criterion 2

## Another Category
- Criterion 3
:::
```
Shows "Start Building →" button. Categories with `##` headings.

## Draft Actions
```
:::draft-actions
versionId: <uuid>
message: Draft ready — added health check nodes
:::
```
Renders Publish/Discard/See in Editor bar. Print in chat response, not as a file.

## Chart
```
:::chart
type: bar
title: Run Success Rate
x: [Mon, Tue, Wed]
series:
  - name: Success
    data: [12, 15, 10]
    color: "#22c55e"
:::
```
Types: `bar`, `line`, `area`, `pie`.

## Other Blocks

| Widget | Syntax | Purpose |
|--------|--------|---------|
| `:::steps` | `- [x] Done\n- [ ] Pending` | Progress indicator |
| `:::collapse` | `:::collapse title="Details"\n...\n:::` | Expandable content |
| `:::success` | `:::success\n...\n:::` | Green banner |
| `:::error` | `:::error\n...\n:::` | Red banner |
| `:::confirm` | YAML with message/yes/no | Destructive action confirm |
| `` ```mermaid `` | Mermaid syntax | Flow diagrams |

## Inline Chips

| Type | Syntax | Example |
|------|--------|---------|
| Node | `[Name](node:id)` | `[Deploy](node:deploy-ssh)` |
| Run | `[Name](run:uuid~status)` | `[Build #1](run:abc~passed)` |
| Integration | `[Name](integration:uuid)` | `[dash0](integration:791ee...)` |
| Integration (new) | `[Name](integration:vendor)` | `[GitHub](integration:github)` |

Integration chips show vendor icon + connection state. Click opens configure/connect modal.
