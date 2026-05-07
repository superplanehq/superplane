# Feature Specification: Azure IoT Operations Integration

**Feature Branch**: `001-azure-iot-ops`  
**Created**: 2026-05-07  
**Status**: Draft  
**Input**: User description: "Feature Spec: Azure IoT Operations Integration"

## Summary

Add Azure IoT Operations as a first-class SuperPlane integration so teams can react to industrial edge events, enrich workflows with asset context, and govern approved actions back to connected systems. The feature extends SuperPlane beyond software operations into factory-floor and edge operations while preserving the same canvas-based orchestration model.

## Clarifications

### Session 2026-05-07

- Q: How should the deduplication window be defined? → A: Fixed project-wide window for all AIO sources.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Edge Events Trigger Workflows (Priority: P1)

As an operator or platform engineer, I want edge and asset events from Azure IoT Operations to start SuperPlane workflows so that industrial issues and production signals can be handled without manual polling or custom glue code.

**Why this priority**: Event ingestion is the entry point for the integration and delivers immediate value on its own.

**Independent Test**: Send a representative edge event into the integration and verify that the expected workflow starts with the right event type and key metadata.

**Acceptance Scenarios**:

1. **Given** a registered Azure IoT Operations source, **When** an asset alarm or contextualized output event arrives, **Then** SuperPlane starts the mapped workflow with the correct trigger details.
2. **Given** repeated identical events within a short period, **When** they are received, **Then** the workflow does not create duplicate independent incident records for the same signal.

---

### User Story 2 - Asset Context Is Available In Workflow (Priority: P2)

As an incident responder, I want SuperPlane workflows to retrieve asset context from Azure IoT Operations so that I can make decisions with the latest operational details.

**Why this priority**: Context lookup improves triage quality and makes the integration useful beyond simple alerting.

**Independent Test**: Open a workflow for a known asset and verify that the workflow can retrieve asset identity, status, and recent operational context.

**Acceptance Scenarios**:

1. **Given** a known asset identifier, **When** a workflow requests asset context, **Then** the workflow receives the latest available asset details.
2. **Given** an asset that cannot be found, **When** the workflow requests its context, **Then** the workflow shows a clear missing-data result instead of failing silently.

---

### User Story 3 - Approved Actions Can Be Sent Back Safely (Priority: P3)

As an OT lead, I want approved workflows to send governed actions back to connected edge systems so that operational changes remain auditable and human-controlled.

**Why this priority**: Write-back is the highest-risk capability and must be introduced only after the read and trigger flows are working.

**Independent Test**: Run a workflow that includes an approval gate and confirm that the action is blocked until approval is granted, then recorded after execution.

**Acceptance Scenarios**:

1. **Given** a workflow with a required approval step, **When** approval has not been granted, **Then** no write-back action is sent.
2. **Given** approval has been granted, **When** the workflow continues, **Then** the action is sent and the execution is recorded in the run history.
3. **Given** a write-back is rejected by policy or permission checks, **When** execution is attempted, **Then** the workflow reports the rejection clearly and preserves the audit trail.

---

### Edge Cases

- An edge site is temporarily offline and events arrive later than expected.
- A source sends incomplete or malformed event data.
- A device or asset is registered after a workflow has already started.
- The same event arrives more than once.
- A workflow requests asset context for an unknown or retired asset.
- A write-back action is requested without the required approval gate.
- A connected system rejects a governed action because of policy, permissions, or state constraints.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow Azure IoT Operations events to start SuperPlane workflows.
- **FR-002**: The system MUST support mapping incoming industrial edge signals to distinct workflow trigger types.
- **FR-003**: The system MUST preserve the important source details needed for incident triage, including the originating asset or event reference.
- **FR-004**: The system MUST allow workflows to retrieve current asset context from Azure IoT Operations.
- **FR-005**: The system MUST return a clear missing-data result when requested asset context is unavailable.
- **FR-006**: The system MUST allow approved workflows to send governed actions back to connected edge systems.
- **FR-007**: The system MUST require a human approval gate before any action that can affect a physical asset is executed.
- **FR-008**: The system MUST record each inbound event, context lookup, approval outcome, and write-back attempt in the workflow run history.
- **FR-009**: The system MUST surface permission or policy rejections for write-back actions in a way that is understandable to operators.
- **FR-010**: The system MUST avoid creating duplicate workflow runs for the same event when the same source signal is delivered repeatedly within the fixed project-wide deduplication window.

### Key Entities *(include if feature involves data)*

- **Edge Event**: A signal originating from Azure IoT Operations that can start a workflow or enrich an existing one.
- **Asset**: A physical device or industrial resource represented in the integration for lookup, monitoring, or action.
- **Workflow Run**: A single execution of a SuperPlane canvas that may include inbound events, approvals, lookups, and actions.
- **Approval Gate**: A required human decision point before a workflow can trigger a physical-world action.
- **Governed Action**: A controlled request sent back to an operational system after policy checks and approval.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new Azure IoT Operations trigger can be configured and used to start a workflow in under 15 minutes by a first-time integrator.
- **SC-002**: At least 95% of valid inbound edge events produce the expected workflow trigger without manual intervention.
- **SC-003**: Operators can retrieve relevant asset context for a known asset during a workflow within 2 minutes in 9 out of 10 attempts.
- **SC-004**: 100% of approved write-back actions are recorded in run history with the originating event, approval outcome, and execution result.
- **SC-005**: No physical-world action can complete without an explicit approval step when one is required by the workflow.

## Assumptions

- The first release prioritizes event intake and workflow triggering before advanced bidirectional capabilities.
- Existing Azure-oriented authentication and access patterns can be reused for this integration.
- Industrial customers may operate intermittently connected edge sites, so delayed event delivery is an expected condition.
- The integration is intended for supervised operational use, not autonomous control of physical systems.
- SuperPlane remains an orchestration layer and does not replace the underlying industrial control system or asset registry.
