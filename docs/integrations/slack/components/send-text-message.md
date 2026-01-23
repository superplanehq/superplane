---
app: "slack"
label: "Send Text Message"
name: "slack.sendTextMessage"
type: "component"
---

# Send Text Message

Send a text message to a Slack channel

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| channel | Channel | app-installation-resource | yes | - |
| text | Text | text | yes | - |

## Example Output

```json
{
  "data": {
    "channel": "C123456",
    "text": "Hello from SuperPlane",
    "ts": "1700000000.000100",
    "user": "U123456"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "slack.message.sent"
}
```

