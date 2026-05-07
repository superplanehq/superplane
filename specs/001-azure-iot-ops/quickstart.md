# Quickstart: Azure IoT Operations Integration

## Prerequisites

- SuperPlane development environment running locally or in Docker Compose.
- Azure credentials configured through the existing SuperPlane Azure authentication flow.
- Access to an Azure IoT Operations source that can POST webhooks for the Trigger phase.

## Local Setup

1. Start the dev stack with `make dev.start`.
2. Confirm the feature files exist under `specs/001-azure-iot-ops/`.
3. Review `specs/001-azure-iot-ops/plan.md` and the contracts in `specs/001-azure-iot-ops/contracts/` before implementing.

## Trigger Phase Verification

1. Configure an AIO source to send a representative alarm or dataflow output event to the SuperPlane webhook URL.
2. Verify the workflow starts with the normalized edge event payload and a stable dedupe key.
3. Re-send the same event and confirm it does not create a duplicate workflow run within the fixed project-wide window.

## Read Phase Verification

1. Configure an asset reference that resolves to Azure Device Registry or ARM.
2. Trigger a workflow that reads asset context.
3. Confirm the run history records the lookup and that missing assets return a clear empty result instead of a silent failure.

## Write Phase Verification

1. Add an approval gate before the write-back step.
2. Confirm the action does not execute until approval is granted.
3. Confirm the run history captures the approver, payload, returned result, and Azure activity log entry ID.

## Test And Validation Commands

- `make test PKG_TEST_PACKAGES=./pkg/integrations/azureiotoperations`
- `make check.build.app`
- `make check.build.ui`
- `make format.go`
- `make format.js` when UI mapper files change

## Operator Checks

- Confirm trigger names and descriptions use industrial terms first.
- Confirm write-back steps are visually marked as physical-action steps.
- Confirm offline or delayed delivery is explained in the workflow template and docs.
