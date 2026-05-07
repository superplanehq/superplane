# Implementation Plan: Azure IoT Operations Integration

**Branch**: `001-azure-iot-ops` | **Date**: 2026-05-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-azure-iot-ops/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a first-class Azure IoT Operations integration that ingests industrial edge events, enriches workflows with asset context, and keeps governed write-back behind an explicit approval gate. The implementation is split into three contractual phases: Trigger, Read, and Write.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.26.2 backend; TypeScript/Vite workflow UI  
**Primary Dependencies**: Existing SuperPlane core/registry, Azure ARM REST patterns, webhook handlers, workflow v2 mapper layer  
**Storage**: Existing PostgreSQL-backed application state, integration secrets, webhook records, and run history  
**Testing**: `make test`, `make check.build.app`, `make check.build.ui`, `make format.go`, `make format.js` when touching UI code  
**Target Platform**: AWS-hosted SuperPlane services with Azure Arc-managed edge environments and Linux containers  
**Project Type**: Backend service integration with workflow UI mapping and documentation  
**Performance Goals**: Trigger handling must complete within normal webhook request budgets; read lookups should fail fast on edge unavailability; write-back stays gated and auditable  
**Constraints**: No Azure-hosted SuperPlane runtime dependency; open standards only; fixed project-wide deduplication window; write-back requires approval before execution  
**Scale/Scope**: One new integration package, three user-facing phases, and corresponding UI mapper and contract documentation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution is satisfied by design:

- The Edge is Sovereign: webhook-first trigger handling, offline-safe semantics, and no hard Azure runtime dependency.
- Human Oversight is Non-Negotiable for Write-Back: write phase is blocked behind an explicit approval gate and audit trail capture.
- Open Standards Over Convenience: payloads and interfaces remain ARM/HTTP/JSON-based with no proprietary SDK dependency.
- AWS-First Deployment, Azure as Signal: SuperPlane remains AWS-hosted and uses standard credential flows for Azure access.
- Phases are Contracts, Not Suggestions: Trigger, Read, and Write are implemented in order and validated separately.
- Operational Safety and Operator Experience: docs and errors stay industrial-first.

## Project Structure

### Documentation (this feature)

```text
specs/001-azure-iot-ops/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
pkg/integrations/azureiotoperations/
├── integration.go
├── webhook_handler.go
├── events.go
├── component_get_asset.go
├── component_publish_dataflow.go
├── component_invoke_management.go
└── *_test.go

web_src/src/pages/workflowv2/mappers/azure_iot_operations/
├── index.ts
├── types.ts
├── on_asset_alarm.ts
├── on_dataflow_output.ts
└── on_edge_health_degraded.ts

specs/001-azure-iot-ops/contracts/
├── trigger-events.md
├── asset-read.md
└── writeback.md

test/e2e/
└── workflows/
```

**Structure Decision**: Use a dedicated `pkg/integrations/azureiotoperations/` package rather than extending the existing Azure resource-management integration. The AIO feature has different trigger semantics, read-only asset lookup, and write-back safety constraints, so isolating it keeps the registration, tests, and UI mapping aligned to the new domain.

## Complexity Tracking

No constitution violations identified. The implementation stays within the required phase ordering, keeps write-back gated, and uses the existing AWS-hosted SuperPlane runtime.
