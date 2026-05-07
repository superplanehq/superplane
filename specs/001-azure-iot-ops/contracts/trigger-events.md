# Contract: Azure IoT Operations Trigger Events

## Purpose

Define the inbound event envelope SuperPlane expects from Azure IoT Operations for the Trigger phase.

## Supported Event Types

- `asset.alarm`
- `dataflow.output`
- `edge.health.degraded`
- `asset.discovered`

## Envelope Fields

| Field | Required | Description |
| --- | ---: | --- |
| `id` | Yes | Stable event identifier used for deduplication |
| `eventType` | Yes | One of the supported AIO trigger event types |
| `subject` | No | Asset or source subject path |
| `source` | Yes | Source system identifier |
| `occurredAt` | Yes | Original event time in UTC |
| `payload` | Yes | Normalized JSON payload with event-specific details |
| `assetRef` | No | Canonical asset reference used by workflows |
| `correlationId` | No | Optional correlation value carried through run history |

## Behavior Rules

- The event handler must preserve the raw payload for auditability.
- Duplicate deliveries of the same `id` within the fixed project-wide deduplication window must resolve to the same workflow run.
- Missing optional fields must not fail the trigger if the required envelope fields are present.

## Example

```json
{
  "id": "evt-123",
  "eventType": "asset.alarm",
  "subject": "/sites/plant-1/lines/packaging-2/machines/press-4",
  "source": "aio-dataflow://press-events",
  "occurredAt": "2026-05-07T12:00:00Z",
  "assetRef": "press-4",
  "correlationId": "corr-789",
  "payload": {
    "severity": "critical",
    "alarmCode": "E-221",
    "message": "Press overheating detected"
  }
}
```
