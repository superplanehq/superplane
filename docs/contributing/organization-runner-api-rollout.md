# Organization runner API rollout

Organization fleet management uses
`/api/v1/organizations/{id}/runner-fleets`. The existing `/fleets` route
continues to list enabled fleet choices for task selection.

The `organization_runners` feature is unreleased and disabled by default.
Use the existing installation feature administration API to enable it for
selected internal organizations. Do not change organization defaults.

## Deployment order

1. Keep `organization_runners` disabled.
2. Apply the `scope-runner-idempotency-to-fleet` migration.
3. Deploy the API, runner gateway, and scheduler admission changes.
4. Verify that all affected instances run the new code.
5. Enable the feature for selected internal organizations.

The migration preserves existing runner IDs, request keys, hashes, and
registration grants. Existing Fleet Managers retain their fleet-prefixed
keys. Do not reuse a key across fleets while old API instances remain.
After deployment, arbitrary keys are isolated by the internal fleet UUID.

This change requires no new service, configuration key, Secret, hosted
infrastructure, or companion deployment PR.

## Admission and disablement

New admission locks the organization, fleet, runner, and task in that order.
Feature writes lock the organization row. Fleet configuration writes lock
the fleet row. These locks remain until transaction commit.

If reservation commits first, the registered runner can finish that work.
If disablement commits first, new registration and reservation fail.
Unused registration tokens remain rejected even for previously reserved
tasks. Timeout and termination cleanup release those reservations.

Idle runners receive no new work after feature or fleet disablement.
Queued tasks return to pending on their original fleet and release their
organization slots. Re-enablement permits scheduling on that same fleet.
Running tasks retain reconnect, log upload, cancellation, and completion
access. Inspection, termination, provider cleanup, logs, and history remain
available.

## Direct startup

Runner creation returns a registration token and `runnerApiUrl`. It does
not start a process. Set `RUNNER_REGISTRATION_TOKEN` and `RUNNER_API_URL`
from that response, then start the runner binary. Provider ownership fields
are optional and do not require a Fleet Manager process.

The runner entrypoint accepts the returned `/runner/v1` URL and the existing
server-root URL. Ephemeral runners exit after one acknowledged task.
Reusable runners wait for successive tasks.
