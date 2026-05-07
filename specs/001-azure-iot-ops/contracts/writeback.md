# Contract: Governed Write-Back Actions

## Purpose

Define the write-back contract for the Write phase, including the approval requirement and audit fields.

## Request Shape

| Field | Required | Description |
| --- | ---: | --- |
| `actionType` | Yes | The governed action to perform |
| `targetResource` | Yes | ARM resource or asset target |
| `payload` | Yes | Action payload to send to the edge or control plane |
| `approvalGateId` | Yes | Approval gate that authorizes the action |
| `workflowRunId` | Yes | Workflow run that owns the action |
| `sourceEventId` | Yes | Originating trigger event identifier |

## Response Shape

| Field | Required | Description |
| --- | ---: | --- |
| `status` | Yes | `sent`, `acknowledged`, or `failed` |
| `activityLogEntryId` | No | Azure activity log reference for the write |
| `result` | No | Raw response or failure details |
| `approvedBy` | No | User or system that approved the action |
| `approvedAt` | No | Approval timestamp |

## Behavior Rules

- The action must not be sent until the approval gate is approved.
- Every execution must preserve the approval actor, payload, response, and activity log ID in run history.
- A rejected policy or permissions check must fail clearly and keep the audit trail intact.

## Example Request

```json
{
  "actionType": "invoke-management",
  "targetResource": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/plant-rg/providers/Microsoft.DeviceRegistry/fabrics/factory-1/assets/press-4",
  "payload": {
    "method": "resetFault",
    "reason": "Operator approved restart after cooling period"
  },
  "approvalGateId": "gate-456",
  "workflowRunId": "run-123",
  "sourceEventId": "evt-123"
}
```
