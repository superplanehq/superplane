# Contract: Asset Read Access

## Purpose

Define the read-only asset lookup contract for the Read phase.

## Request Shape

| Field | Required | Description |
| --- | ---: | --- |
| `assetRef` | Yes | Canonical workflow asset reference |
| `resourceId` | No | ARM resource ID when available |
| `subscriptionId` | No | Azure subscription identifier |
| `resourceGroup` | No | Azure resource group containing the registry |
| `registryName` | No | Azure Device Registry or AIO registry name |

## Response Shape

| Field | Required | Description |
| --- | ---: | --- |
| `assetRef` | Yes | Echoed asset reference |
| `resourceId` | Yes | Resolved ARM resource ID |
| `name` | Yes | Human-readable asset name |
| `status` | No | Operational status or connection state |
| `latestTelemetryAt` | No | Timestamp of the latest known telemetry |
| `attributes` | No | Additional normalized metadata |

## Behavior Rules

- The lookup must be read-only.
- Missing assets must return an explicit empty result or not-found state.
- The workflow must continue to log the lookup attempt even when the asset is unavailable.

## Example Response

```json
{
  "assetRef": "press-4",
  "resourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/plant-rg/providers/Microsoft.DeviceRegistry/fabrics/factory-1/assets/press-4",
  "name": "Press 4",
  "status": "online",
  "latestTelemetryAt": "2026-05-07T11:59:45Z",
  "attributes": {
    "line": "packaging-2",
    "temperature": 82.1
  }
}
```
