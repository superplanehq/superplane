# BYOS catalog refresh: Phase 2 handoff

Issue: [#9133](https://github.com/superplanehq/superplane/issues/9133).
Starting commit: `6b6a053affa1d0601d0c1846b78bf5a5c405b0db`.
This is the merged Phase 1 baseline from
[PR #9131](https://github.com/superplanehq/superplane/pull/9131).
The implementation uses one integration branch and one final PR.

## Lease and API contract

- A `catalog` lease belongs to one organization connection and one runner task.
  It has no workspace and has its own encrypted, random refresh secret.
- Catalog authority expires 15 minutes after lease creation. Credential fetch,
  refresh proxy, checkpoints, and runner admission enforce that deadline.
- Runs and catalog refresh can overlap in either admission order. Sign-in stays
  exclusive. New admissions do not invalidate other leases' generations.
- Duplicate refresh requests return the current catalog task. A revoked catalog
  row remains unique until task cleanup removes it.
- Catalog credential responses substitute the lease secret for the provider
  refresh token. The existing server refresh transaction serializes exchanges
  and preserves Phase 1 persistence and permanent-error behavior.
- Catalog checkpoints accept only usage and model metadata. Model and quota
  validation remain independent. Invalid or missing data preserves the previous
  value. A valid empty model list can clear the catalog.
- The catalog script requires complete model discovery and an accepted
  checkpoint. It stops the app-server process group and removes local
  credentials before success. Quota retrieval remains best effort.
- `FactoryCodexConnection.catalog_refresh_task_id` identifies active catalog
  activity. `catalog_refresh_canceling` identifies cancellation in progress.
  Connection status, `active_task_id`, and `reservation_held` retain their
  connection/sign-in meaning.
- Both cancellation request types accept `catalog_refresh_task_id`. A nonempty
  target affects only that task on the caller's connection. An empty target
  preserves sign-in cancellation. Stale targets never select a replacement.
- Cancellation revokes catalog authority and requests runner cancellation.
  Progress remains visible until terminal, lost, or missing-task cleanup.
  Local catalog cancellation does not wait for provider VM deletion.
  Connection-wide revocation retains its stricter cleanup requirements.
- Catalog failure does not clear healthy credentials or change unrelated runs.
  Genuine permanent authentication errors still require reconnection.
- Activity updates publish after start, cancellation, cleanup, and expiry commit.
  The UI polls during activity and refreshes model and usage queries on completion.

Phase 3 can use these fields for quota presentation and live token reporting.
Phase 2 must merge first. Rebase Phase 3 and rerun affected checks afterward.
Quota policy, entitlements, setup charging, and task-wide secret history remain
outside this phase. Phase 1's process-local secret retention remains unchanged.

## Migration and controlled rollout

Human security review of the integrated authorization change is required before
activation.

1. At ingress, pause POST requests to `/api/v1/me/codex-connection` and
   `/api/v1/factories/{factory_id}/codex-connection`. Restrict equivalent direct
   gRPC start calls. Keep cancellation and runner subscription routes available.
   Preserve the existing subscription feature flags for active runs.
2. Drain existing sign-in and catalog setup tasks. Do not reinterpret old login
   leases as catalog leases. Preserve existing run contracts.
3. Apply `20261031144703_add-codex-catalog-leases` through the normal migration
   process. It preserves existing login/run rows, scope, and secret uniqueness.
   It adds the catalog purpose, permits its null workspace, and creates a unique
   catalog index that includes locally revoked rows.
4. Update matching public API, general worker, runner API, cleanup worker, and UI
   application versions.
5. Verify configuration and endpoint access listed below.
6. Resume setup admissions. Confirm refresh can overlap a real workspace run,
   cancellation preserves that run, and catalog progress clears after cleanup.

Local validation applies the migration to `superplane_dev` and `superplane_test`
with `make db.migrate DB_NAME=<name>`. No database is dropped.
Runner image release preparation, image publication, version-pointer changes,
and deployment are outside this task. The script remains embedded in task files.

## Required process configuration

| Process | Required configuration or access |
| --- | --- |
| Public API and catalog task creation | Existing `JWT_SECRET`, `ENCRYPTION_MASTER_KEYS`, DB, RabbitMQ, and public origins |
| General workers and run creation | Matching existing JWT/encryption keys and reachable public origins |
| Runner API and task delivery | Matching JWT/encryption keys; preserve catalog creation deadlines |
| Cleanup workers | Matching application version and `START_RUNNER_CLEANUP_WORKER=yes` |
| Catalog task | Existing subscription JWT, `SUPERPLANE_BASE_URL`, and per-task `CODEX_REFRESH_TOKEN_URL_OVERRIDE` |
| Runner daemon | Existing `RUNNER_API_URL`, registration configuration, and public subscription endpoint access |

The task creator selects its public origin through `RUNNER_API_BASE_URL`, then
`WEBHOOKS_BASE_URL` or `BASE_URL`. Catalog tasks receive the refresh override at
`<origin>/api/v1/runner/codex-subscription/refresh`.
The runner agent replaces task `SUPERPLANE_BASE_URL` with its own configured
origin. It does not replace the refresh override.
Verify that both origins reach `/api/v1/runner/codex-subscription/*`.
Access to `/runner/v1` alone is insufficient.
The public API needs HTTPS egress to `https://auth.openai.com/oauth/token`.
Runners retain their existing ChatGPT network requirements.

Local Docker Compose and the self-host Helm chart already provide the shared
JWT/encryption configuration and public origins. The self-host ingress routes
public subscription endpoints to the API. The refresh override is task
configuration; it does not require a new deployment Secret.

The GitHub connector could not access `superplanehq/helm-chart` or
`superplanehq/launchpad` during this task. Hosted values, Secret references,
published chart state, and live network access remain unverified.
No companion deployment PR is prepared. Verify those sources before activation.

## Validation evidence

The admission regression failed on the starting commit in both orders:
run then catalog, and catalog then run. The baseline model suite ran 967 tests;
only those two new subtests and their parent failed.

The integrated focused backend command passed: 4,277 tests, with 10 unrelated
media skips because the application test container does not include ffmpeg.
The focused UI suite passed 29 tests. Tests cover the owning model, HTTP,
runner delivery, factory action, personal adapter, actual script, and React
query boundaries. Phase 1 redaction, cancellation persistence, and permanent
authentication-error coverage remain in these suites.

| Check | Local result |
| --- | --- |
| `make format.go` and `make check.format.go` | Passed |
| `make format.js` | Passed |
| `make lint` and `make check.build.app` | Passed |
| `make pb.gen` and `make check.proto.field.numbers` | Passed |
| `make check.models.tx.debt` | Passed; no debt increase |
| `make check.grpc.actions.status` | Passed |
| `make check.db.migrations` | Passed |
| `make check.generated.artifacts` | Passed; generated clients remain untracked |
| `make check.fast.security` | Passed |
| Migration application | Passed on development and test databases |
| Focused backend suites, including `./pkg/grpc` | Passed; 4,277 tests, 10 media skips |
| Focused UI suites | Passed; 29 tests |
| `make test` | Passed; 18,803 tests, 19 environment-dependent skips |
| `make check.lint.ui` | Passed; existing budget remains 273/273 |
| `make check.build.ui` and `make check.format.js` | Passed |
| `git diff --check` | Passed |

The focused backend command is:

```sh
make test PKG_TEST_PACKAGES='./pkg/models ./pkg/public ./pkg/public/runnerapi ./pkg/grpc ./pkg/grpc/actions/factories ./pkg/components/runner/... ./pkg/workers'
```

The full backend suite also passed. Its skips include media-tool tests and
Docker executor tests that require tools absent from the application test
container. These checks do not verify a live provider exchange or deployed
runner network access. Local checks do not replace human security review.

The focused UI files are:

```sh
make check.test.ui FILES='src/pages/factories/pages/onboarding/useFactoryCodexConnection.spec.tsx src/pages/factories/pages/settings/FactorySettingsAccountCodexUsagePage.spec.tsx src/pages/factories/codex/CodexActionProvider.spec.tsx'
```

## Implementation commits

The code snapshot validated before the handoff is `fcd414bc2`.

| Commit | Change |
| --- | --- |
| `6ce1c4e8a` | Add catalog leases, migration, cleanup, and activity snapshots |
| `f72994939` | Separate catalog script completion and UI activity |
| `df6a6524d` | Validate optional model metadata |
| `8ae6b1f96` | Add restricted API authority, task coalescing, and cancellation fields |
| `d2be445d0` | Apply repository formatting |
| `f3831b7b6` | Preserve models when quota data is malformed |
| `a6ef38b9a` | Reject malformed reasoning options |
| `4fa5f7673` | Separate UI activity and connected actions |
| `b6a970da5` | Publish cleanup and queued-expiry activity changes |
| `3d87b61d2` | Complete cancellation integration and correct regression fixtures |
| `fcd414bc2` | Separate connection retrieval and polling |
