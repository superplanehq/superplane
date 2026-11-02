# BYOS usage, limits, and billing: Phase 3 handoff

Issue: [#9138](https://github.com/superplanehq/superplane/issues/9138).
Starting commit: `71ed36ecac41ec49e0e51af30d6fbfabe6d96c02` on
`codex/byos-isolate-catalog-refresh`.
The implementation branch is `codex/byos-usage-limits-billing`.
The single implementation PR initially targets the Phase 2 branch.
After [#9134](https://github.com/superplanehq/superplane/pull/9134) merges,
retarget the Phase 3 PR to `main` and check its diff again.

## Additive contracts

Runner results and telemetry carry `usage_state`: `pending`, `measured`,
`partial`, or `unavailable`. A measured zero is a valid measurement.
Codex usage includes an explicit `total_tokens` value. Input, output, cache
reads, cache writes, and reasoning are separate categories.

Each Codex invocation has an immutable `attempt_id`. Its current snapshot
replaces the previous snapshot for that attempt. Completed prompts and the
active prompt must each contribute once. Failed and canceled attempts retain
delivered measurements.

The observer reads the identified thread's rollout file, including reverted
rollout filenames. It captures the file boundary before a resumed invocation.
Only new, correlated per-response `payload.usage` records supply measurements.
Exec output's cumulative final totals do not supply attempt measurements.
Raw rollout contents must not enter logs, API results, or review artifacts.

A subscription quota failure carries the allowlisted object:

```json
{
  "reason": "chatgpt_quota_exhausted",
  "limit_id": "optional provider limit identifier",
  "resets_at": "optional RFC3339 timestamp"
}
```

The server captures the subscription owner from the actual task lease before
cleanup. Runner-supplied ownership is not authoritative. Planning, execution,
canvas-run, and linked-run projections share the
`Canvases.SubscriptionFailure` API type.
The projections preserve this failure independently of artifacts or generated
plans. Bulk result summaries include nested runs without a separate lookup for
each requested run. PR feedback and column app summaries retain the same
failure and measurement state.

The experimental flag remains required for new admission. An allocated task
can continue after policy changes. Membership, ownership, credential generation,
revocation, task identity, and lease expiry still apply. Queued tasks must pass
admission again before another allocation.

## Migration and process configuration

Apply the Phase 2 migration first. Apply the generated Phase 3 migration
`20261031165847_add-codex-setup-compute-usage` before the matching application
versions. Do not reinterpret older tasks without the new setup purpose marker.
Do not backfill charges.

The new setup purposes are `codex_login` and `codex_catalog`. The allocation
captures its billing mode when reservation succeeds. Each allocation uses the
task ID and persisted reservation timestamp as its accounting identity.
Organization-only ledger rows retain a real runner-task reference. They have
no workspace or workflow attribution.

Completion, loss, and reservation release must record usage in the same
transaction as the lifecycle change. Accounting errors must roll back that
change. Cancellation requests do not end an active billing interval.

| Process | Configuration |
| --- | --- |
| Public API and setup creation | Existing `SUPERPLANE_BILLING_ENABLED`; existing DB, JWT, encryption, and public-origin settings |
| Runner API and reservation | The same billing-enabled setting and matching application version |
| Scheduler and general workers | The same billing-enabled setting and matching application version |
| Cleanup workers | The same billing-enabled setting and matching application version |

Use the existing billing-enabled helper in the self-hosted runner-workers
template. API configuration alone does not configure workers. Billing-disabled
installations bypass hosted payment and credit requirements. Local fleets retain
their explicit zero compute rate.

No new provider Secret, endpoint, or runner release is required. This work does
not publish images, prepare a runner release, or deploy application versions.
Preserve the existing setup machine selection.

The GitHub connector returned HTTP 404 for both `superplanehq/helm-chart` and
`superplanehq/launchpad` during this task. No companion deployment PR is prepared.
Verify the billing setting across hosted API, runner API, scheduler, and cleanup
processes before activation. Hosted values, published charts, Secret references,
and deployed configuration remain unverified.

## Validation evidence

The new continuation regression failed on the original behavior after an
admitted task's feature flag was disabled. It was the only failing regression
among 990 model tests. This establishes the admission-versus-continuation bug.

An initial controlled probe ran the actual `codex-cli 0.163.0` binary in the
existing local worker image. A local Responses endpoint supplied deterministic
measurements. The CLI wrote two per-response usage records while its process
was active: totals 130 and 260, with cumulative totals 390.

`make check.codex.incremental.usage` exercises the actual CLI through the task
wrapper for planning and execution. It checks an update during a running tool,
disjoint token categories, final reconciliation, and resumed-prompt accounting.
The endpoint supplies fixture measurements. This check does not establish live
ChatGPT account behavior or replace the required live acceptance run.

The controlled wrapper check passed in both modes. Each mode delivered a
measurement during an active invocation, reconciled to 390 tokens, and reached
780 after a resumed prompt. Cache and reasoning categories remained disjoint.
This checks the real CLI recording and wrapper delivery against fixture
measurements. It is not a live provider acceptance result.

The focused runner command passed 620 tests with 18 environment-dependent
skips. The skipped tests need ffmpeg or Docker inside the application test
container. The final Codex observer command passed all 126 tests after adding
strict counter, session-correlation, terminal-error, and reset validation cases.
Backend lint and the application build passed. Protobuf generation and numbering,
model transaction debt, gRPC status mapping, generated-file hygiene, security
scans, and migration naming/order checks passed.

The collapsed-log regression also failed under the original subscription
condition: the display stayed at 210 tokens instead of receiving 420 tokens.
Restoring the fix made that regression pass. An intermediate focused UI command
passed all 198 tests, including historical quota failures and new-attempt clearing.
An earlier full UI command passed all 10,631 tests across 1,294 files. The
canonical canvas-run and linked-run projections changed afterward. The final
focused command passed all 169 tests across 13 affected files. These include
the default console, collapsed-log updates, quota history, PR feedback, and
column app summaries.
After the final projection simplification, all 95 linked-run tests passed
again across the two affected files.

Frontend lint passed at the unchanged budget of 273 issues. The production
UI build and formatting check passed on the final source. The affected
database-backed test packages compile successfully with `-run=^$`. This compile
check ran zero tests and does not verify database behavior.

Database behavior, migration application, and the live provider acceptance
remain pending as described below.

## Remaining approval and acceptance

Automatic approval review rejected filling the Make-generated migration. It
interpreted the repository migration rule as prohibiting edits after generation.
Both generated migration files remain empty, and no Phase 3 schema change has
been applied. The draft PR excludes those empty files. The SQL proposal below
is for review only. It is not an executable migration in this branch.
Explicit user approval is pending.

After approval, fill the generated up migration with the reviewed SQL. Leave
the down migration empty. Apply it to development and test databases through
`make db.migrate DB_NAME=<name>`, without dropping either database. Record the
result, then run affected model, public API, runner API, factory action, billing,
and worker tests, followed by the required full backend suite.

A live ChatGPT test account/workspace is also needed for the required provider
acceptance run. Verify both planning and execution displays during that run,
with logs closed and after reload. No live provider run has been claimed.
Keep the implementation PR in draft until the migration and validation complete.

GitHub still reports #9134 open at the required starting commit. Issue #9138
contains the complete plan, assignee, and label. GitHub reports that issue as
closed since shortly after creation. This implementation did not change its
status after that verification.

## Proposed schema for review

The following SQL is the proposed content for the Make-generated up migration.
It has not been applied. Leave the generated down migration empty.

```sql
ALTER TABLE runner_tasks
  ADD COLUMN setup_purpose text NOT NULL DEFAULT '',
  ADD COLUMN setup_billable boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT runner_tasks_setup_purpose_known
    CHECK (setup_purpose IN ('', 'codex_login', 'codex_catalog')),
  ADD CONSTRAINT runner_tasks_setup_billable_shape
    CHECK (NOT setup_billable OR setup_purpose <> '');

ALTER TABLE workspace_usage_events
  ALTER COLUMN canvas_run_id DROP NOT NULL,
  ALTER COLUMN node_execution_id DROP NOT NULL,
  ADD COLUMN runner_task_id uuid REFERENCES runner_tasks(id),
  ADD COLUMN setup_purpose text NOT NULL DEFAULT '',
  ADD CONSTRAINT workspace_usage_events_scope_shape CHECK (
    (setup_purpose = '' AND runner_task_id IS NULL
      AND canvas_run_id IS NOT NULL AND node_execution_id IS NOT NULL)
    OR
    (setup_purpose IN ('codex_login', 'codex_catalog') AND runner_task_id IS NOT NULL
      AND factory_id IS NULL AND work_order_id IS NULL AND line_id IS NULL
      AND line_dispatch_id IS NULL AND work_order_execution_id IS NULL
      AND canvas_run_id IS NULL AND node_execution_id IS NULL AND node_id = ''
      AND usage_kind = 'compute' AND provider = 'runner' AND funding_source = 'hosted'
      AND input_tokens = 0 AND output_tokens = 0 AND cache_read_tokens = 0
      AND cache_write_tokens = 0 AND reasoning_tokens = 0 AND total_tokens = 0)
  );

CREATE INDEX workspace_usage_events_runner_task_id
  ON workspace_usage_events (runner_task_id) WHERE runner_task_id IS NOT NULL;
```
