# Research: Azure IoT Operations Integration

## Decision 1: Use a dedicated integration package

- Decision: Implement the feature in `pkg/integrations/azureiotoperations/` and register it as a new integration.
- Rationale: Azure IoT Operations has different trigger semantics, asset lookups, and write-back safety rules than the existing Azure resource-management integration. A separate package keeps the safety surface isolated and easier to test.
- Alternatives considered: Extending `pkg/integrations/azure/` was rejected because it would mix resource-management triggers with industrial-edge orchestration rules.

## Decision 2: Start with webhook ingress for Trigger phase

- Decision: Phase 1 will accept inbound HTTP webhooks from AIO Data Flows and normalize them into SuperPlane trigger events.
- Rationale: This matches the existing webhook-based integration model, is compatible with the spec's lowest-risk slice, and avoids introducing a persistent MQTT client in the MVP.
- Alternatives considered: A direct MQTT client would reduce latency but adds connection management, offline reconnection, and edge-network complexity that are unnecessary for the first release.

## Decision 3: Read phase uses ARM REST semantics, not a new SDK path

- Decision: Asset context lookup will be implemented with Azure Resource Manager REST requests using the existing Azure authentication pattern.
- Rationale: The spec explicitly calls for a read-only component with no new SDK requirement, and the constitution prefers open standards and portable interfaces.
- Alternatives considered: A new AIO-specific SDK path was rejected because it would add dependency weight without increasing clarity or safety.

## Decision 4: Write-back remains phase-gated and approval-gated

- Decision: `aio.management.invoke` or any equivalent write-back action will only ship after trigger and read surfaces are stable, and it will require an approval gate in every canvas that uses it.
- Rationale: The constitution treats write-back as a safety-sensitive operation. The plan must preserve the hard separation between observation and action.
- Alternatives considered: Allowing write-back as part of the initial trigger slice was rejected because it would violate the phase contract and increase operator risk.

## Decision 5: Deduplication uses a fixed project-wide window

- Decision: The duplicate-event protection window will be fixed at the project level for all AIO sources.
- Rationale: A fixed window makes trigger behavior predictable, keeps the acceptance criteria testable, and avoids source-specific drift.
- Alternatives considered: Per-source or per-workflow windows were rejected because they would complicate operator expectations and test matrices without a clear benefit for the MVP.

## Decision 6: Keep UI mapping separate for AIO-specific renderer state

- Decision: Add workflow v2 mappers under `web_src/src/pages/workflowv2/mappers/azure_iot_operations/` with explicit trigger renderers for alarm, output, and health events.
- Rationale: Existing Azure mappers are VM/storage-specific. A dedicated AIO mapper directory keeps the industrial semantics readable in the UI.
- Alternatives considered: Reusing the existing Azure mapper directory was rejected because it would blur cloud-resource operations with edge-operations terminology.
