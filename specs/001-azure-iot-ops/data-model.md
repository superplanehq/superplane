# Data Model: Azure IoT Operations Integration

## Entities

### AioSource

- Purpose: Describes a configured Azure IoT Operations source that can emit events or receive governed actions.
- Key fields: `id`, `name`, `mode` (`trigger|read|write`), `scope`, `assetRegistryRef`, `dedupeWindow`, `enabled`, `createdAt`, `updatedAt`.
- Relationships: Owns many `EdgeEvent` records and may be referenced by `AssetSnapshot` and `GovernedAction`.
- Validation: `scope` must be present for read/write sources; `dedupeWindow` is fixed project-wide and cannot be overridden by the source.

### EdgeEvent

- Purpose: A normalized inbound signal from AIO or the edge.
- Key fields: `id`, `sourceId`, `eventId`, `eventType`, `subject`, `receivedAt`, `payload`, `dedupeKey`, `deliveryMode` (`webhook|mqtt|rest`), `rawHeaders`.
- Relationships: Belongs to one `AioSource`; may create or update one `WorkflowRun`.
- Validation: `eventId` must be present when supplied by the source; `dedupeKey` must be stable for repeated deliveries of the same signal.
- State transitions: `received -> validated -> deduped|dispatched -> completed|failed`.

### AssetSnapshot

- Purpose: Read-only asset context captured from Azure Device Registry / ARM.
- Key fields: `id`, `sourceId`, `assetRef`, `resourceId`, `name`, `status`, `latestTelemetryAt`, `attributes`, `retrievedAt`.
- Relationships: Belongs to an `EdgeEvent` or a `WorkflowRun` context and can be reused across a single run.
- Validation: Missing assets should resolve to an explicit empty result rather than a silent failure.

### WorkflowRunContext

- Purpose: The execution-time bundle that ties inbound edge events, read context, and write actions together.
- Key fields: `workflowRunId`, `edgeEventId`, `assetSnapshotId`, `approvalGateId`, `auditTrailRef`.
- Relationships: One workflow run may have many events and many governed actions.
- Validation: A write-back action cannot be attached without an approval gate reference.

### ApprovalGate

- Purpose: Human decision checkpoint required before a physical-world action.
- Key fields: `id`, `workflowRunId`, `requestedBy`, `approvedBy`, `status`, `decisionAt`, `notes`.
- State transitions: `requested -> approved|rejected -> consumed`.
- Validation: `approvedBy` and `decisionAt` are required before a write-back can execute.

### GovernedAction

- Purpose: A write-back request to AIO or a connected operational system.
- Key fields: `id`, `sourceId`, `workflowRunId`, `actionType`, `targetResource`, `payload`, `status`, `approvalGateId`, `activityLogEntryId`, `result`.
- Relationships: Belongs to one `WorkflowRunContext` and must reference one `ApprovalGate`.
- State transitions: `draft -> awaiting_approval -> approved -> sent -> acknowledged|failed`.
- Validation: Write-back must not proceed unless the approval gate is approved and present.

## Conceptual Relationships

- An `AioSource` emits an `EdgeEvent`.
- An `EdgeEvent` may enrich a `WorkflowRunContext` with one `AssetSnapshot`.
- A `WorkflowRunContext` may create one or more `GovernedAction` records, each of which requires an `ApprovalGate`.
- Every write-back attempt should retain the activity log reference for auditability.

## Validation Rules

- Duplicate edge deliveries should collapse to the same dedupe key within the fixed project-wide window.
- Trigger events are append-only; later processing can add context or action records, but the original event payload is preserved.
- Read-only asset snapshots never mutate the source system.
- Governed actions are rejected if the approval gate is missing or not approved.
