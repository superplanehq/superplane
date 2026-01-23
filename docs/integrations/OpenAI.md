---
title: "OpenAI"
sidebar:
  label: "OpenAI"
type: "application"
name: "openai"
label: "OpenAI"
---

Generate text responses with OpenAI models

### Components

- [Text Prompt](#text-prompt)

## Components

### Text Prompt

Generate a text response using OpenAI

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| model | Model | app-installation-resource | yes | - |
| input | Prompt | text | yes | - |

## Example Output

```json
{
  "data": {
    "id": "cmpl-1234567890",
    "model": "gpt-5.2",
    "text": "Hello, world!",
    "usage": {
      "input_tokens": 10,
      "output_tokens": 10,
      "total_tokens": 20
    }
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "openai.api.response"
}
```

