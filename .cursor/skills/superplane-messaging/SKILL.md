---
name: superplane-messaging
description: Use when configuring messaging or notification components (Discord embeds, Slack mrkdwn, Telegram Markdown, Teams, SendGrid, SMTP). Covers per-provider fields, character limits, formatting rules, and rich content examples.
---

# SuperPlane Messaging Components

Load this skill when configuring a messaging or notification component. For Expr syntax, payload access, and YAML templating rules, also load the **superplane-expressions** skill.

---

## Provider reference

### Discord (`discord.sendTextMessage`)

| Field | Config key | Type | Limit |
|-------|-----------|------|-------|
| Plain text | `content` | `text` | 2000 chars |
| Embed title | `embedTitle` | `string` | 256 chars |
| Embed description | `embedDescription` | `text` | 4096 chars |
| Embed color | `embedColor` | `string` | Hex: `#5865F2`, `#RGB`, `#RRGGBB` |
| Embed URL | `embedUrl` | `string` | Linked from title |

Either `content` or an embed (title/description) is required. Discord embed descriptions support a markdown subset (bold, italic, links, code blocks). Always truncate dynamic values to stay within limits.

### Slack (`slack.sendTextMessage`)

| Field | Config key | Type |
|-------|-----------|------|
| Message text | `text` | `text` |

Supports [Slack mrkdwn](https://api.slack.com/reference/surfaces/formatting): `*bold*`, `_italic_`, `~strikethrough~`, `` `code` ``, ` ```code block``` `, `<url|label>` links. No embed or attachment fields are exposed.

`slack.waitForButtonClick` sends messages with up to 4 interactive buttons (each with name + value).

### Telegram (`telegram.sendMessage`)

| Field | Config key | Type |
|-------|-----------|------|
| Message text | `text` | `text` |
| Parse mode | `parseMode` | `select`: `None`, `Markdown` |

When `parseMode` is `Markdown`, use Telegram Markdown: `*bold*`, `_italic_`, `` `code` ``, ` ```code block``` `, `[text](url)`.

`telegram.waitForButtonClick` sends messages with interactive inline buttons.

### Microsoft Teams (`teams.sendTextMessage`)

| Field | Config key | Type |
|-------|-----------|------|
| Message text | `text` | `text` |

Plain text only. No rich formatting, cards, or embeds exposed.

### Email: SendGrid (`sendgrid.sendEmail`)

| Field | Config key | Type |
|-------|-----------|------|
| Subject | `subject` | `string` |
| Text body | `body` | `text` |
| HTML body | `htmlBody` | `text` |
| Mode | `mode` | `select`: `text`, `html`, `template` |

Supports plain text, HTML, or SendGrid dynamic templates (`templateId` + `templateData`).

### Email: SMTP (`sendEmail`)

| Field | Config key | Type |
|-------|-----------|------|
| Subject | `subject` | `string` |
| Body | `body` | `text` |

Plain text email. No HTML support.

### Capability summary

| Provider | Markdown | Rich embeds | Buttons | HTML |
|----------|----------|-------------|---------|------|
| Discord | Subset (in embed desc) | Yes | No | No |
| Slack | mrkdwn | No | Yes | No |
| Telegram | Optional | No | Yes | No |
| Teams | No | No | No | No |
| SendGrid | No | No | No | Yes |
| SMTP | No | No | No | No |

---

## Examples

### Discord embed with dynamic title, description, and color

```yaml
content: ""
embedTitle: "{{ let t = $['Generate title'].data.text; len(t) > 256 ? t[:253] + '...' : t }}"
embedDescription: "{{ $['Generate description'].data.text }}"
embedColor: "#E74C3C"
embedUrl: "{{ $['Create issue'].data.url }}"
```

### Slack notification with links

```yaml
text: "New P1 incident!\n\nLink to the incident: {{$['Create PD incident'].data.incident.html_url}}\n\nLink to the issue: {{$['Create issue on GitHub'].data.url}}"
```

### Telegram message with Markdown

```yaml
text: "*Alert*: {{ $['Check'].data.name }} is _down_\n\n[View dashboard]({{ $['Check'].data.dashboard_url }})"
parseMode: "Markdown"
```

### SendGrid HTML email with dynamic subject

```yaml
subject: "Endpoint is down (status {{ $[\"Health check request\"].data.status }})"
mode: "html"
htmlBody: "<h2>Health Check Failed</h2><p>Status: {{ $[\"Health check request\"].data.status }}</p><p><a href=\"{{ $['Dashboard'].data.url }}\">View details</a></p>"
```

### Multi-provider: same data, different formatting

For a PagerDuty incident routed to multiple channels, adapt formatting per provider:

**Slack** (mrkdwn):

```yaml
text: "*{{ $['Generate title'].data.text }}*\n\n{{ $['Generate description'].data.text }}\n\n<{{ $['Create PD incident'].data.incident.html_url }}|View incident>"
```

**Discord** (embed):

```yaml
embedTitle: "{{ $['Generate title'].data.text }}"
embedDescription: "{{ $['Generate description'].data.text }}"
embedUrl: "{{ $['Create PD incident'].data.incident.html_url }}"
embedColor: "#E74C3C"
```

**Teams** (plain text):

```yaml
text: "{{ $['Generate title'].data.text }}\n\n{{ $['Generate description'].data.text }}\n\nIncident: {{ $['Create PD incident'].data.incident.html_url }}"
```
