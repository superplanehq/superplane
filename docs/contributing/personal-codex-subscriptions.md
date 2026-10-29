# Personal Codex subscriptions

Each active human member connects a ChatGPT subscription separately in each
workspace. Members manage their connection in **Account → Codex usage**. The
existing Codex subscriptions feature gate must remain enabled.

Subscription tasks require one eligible owner. A member can start another
member's task without a personal connection. Unassigned tasks remain in the
backlog. Assignment does not resume a pending handoff. Use **Start** to resume it.
If the saved pull request is no longer open, or the provider reports that it
does not exist, **Start** discards the pending handoff. Temporary provider
errors keep the handoff. The task stays in the backlog with its plan, and a member can start it
on any line.

An automatic Verify automation that cannot start because of an owner or model
error fails its run and adds a status note to the task. The event does not retry
until the choice is corrected and the automation runs again.

Task refinement handles the same errors in the same way. The refinement run
fails with the reason, and the event does not retry. A draft task cannot show
status notes, so the failed run shows the reason.

The Start check compares only literal model choices with the owner's catalog.
A model expression is resolved later, and dispatch checks the resolved value.

A saved model can become unavailable after an owner change. Select another model
or clear the override to Auto. The previous choice remains visible until it is
changed. API-key and hosted-provider workspaces keep their existing behavior.

## Member removal

Member removal and account deletion disconnect the member's workspace
connections in the same transaction. They clear credentials and account data,
revoke lease authentication, and record pending cleanup. Rejoining requires a new
sign-in after cleanup finishes.

The runner cleanup worker scans pending connections even when no lease exists.
It cancels queued dispatches, associated runs, and login tasks. Cleanup remains
pending while cancellation or runner termination is incomplete. Inspect
`factory_codex_connections.revocation_pending_at` and
`factory_codex_leases.revoked_at` when cleanup needs investigation. Do not clear
these fields manually.

Revocation blocks further credential delivery immediately. Tokens already sent
to a provider cannot be recalled. Runner cancellation stops the remaining work.

## Deployment and activation

Drain old API, public callback, runner API, execution worker, and runner cleanup
worker versions before multiple connections can be created. Old workspace-only
lookups and cleanup code are incompatible with multiple personal connections.
Apply the migration and deploy the compatible processes before activation.

Keep the existing configuration consistent across processes:

- `JWT_SECRET` signs callback scopes. API, public callback, execution worker, and
  runner API processes must use the same signing key.
- `ENCRYPTION_MASTER_KEYS` protects credentials and lease secrets. API, callback,
  execution worker, and runner API processes need the existing encryption keys.
- `RUNNER_API_BASE_URL`, or its `BASE_URL` fallback, must be reachable from
  runners. The server supplies each task's callback URLs.
- `START_RUNNER_CLEANUP_WORKER=yes` must remain enabled on the runner workers.
  Execution termination, run finalization, and Fleet Manager cleanup must remain
  active until affected work and provider VMs stop.

No new configuration keys are required. Local Compose and the bundled chart
supply the existing domain, JWT, encryption, and runner worker configuration.
The hosted chart and Launchpad repositories were unavailable to the connected
GitHub account during this change. Their values, Secret references, published
chart versions, and deployed configuration remain unverified. Verify them before
activation. No companion deployment changes were published.
