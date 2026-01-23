---
title: "Slack"
sidebar:
  label: "Slack"
type: "application"
name: "slack"
label: "Slack"
---

# Slack

Send and react to Slack messages and interactions

## Installation

You can install the Slack app without the **Bot Token** and **Signing Secret**.
After installation, follow the setup prompt to create the Slack app and add those values.

## Components

### Send Text Message

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

## Triggers

### On App Mention

Listen to messages mentioning the Slack App

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| channel | Channel | app-installation-resource | no | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "api_app_id": "A123ABC456",
    "authed_users": [
      "U123ABC456",
      "U222222222"
    ],
    "event": {
      "channel": "C123ABC456",
      "event_ts": "1515449522000016",
      "text": "\u003c@U0LAN0Z89\u003e is it everything a river should be?",
      "ts": "1515449522.000016",
      "type": "app_mention",
      "user": "U061F7AUR"
    },
    "event_id": "Ev123ABC456",
    "event_time": 123456789,
    "team_id": "T123ABC456",
    "token": "XXYYZZ",
    "type": "event_callback"
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "slack.app.mention"
}
```

