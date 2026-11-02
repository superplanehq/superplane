# Durable Sentry webhooks

SuperPlane stores every verified Sentry webhook in PostgreSQL before it
answers Sentry. HTTP 200 means that the delivery is committed and survives
a process restart. A worker processes the stored delivery later and retries
unfinished work until it is done.

This applies to both entry points:

- `POST /api/v1/sentry/app/webhook` for the public SuperPlane Sentry app.
  SuperPlane verifies the signature with `SUPERPLANE_SENTRY_APP_CLIENT_SECRET`.
- `POST /api/v1/integrations/{id}/events` (or `/webhook`) for a connection
  that uses its own internal integration. SuperPlane verifies the signature
  with the client secret of that connection.

The browser routes `/sentry/app/install` and `/sentry/app/setup` do not
change.

## Data

| Table | Content |
| --- | --- |
| `sentry_webhook_inbox` | One row for each incoming request. The encrypted payload, resource, routing identity, receipt ID, retry schedule, lease, and last error. |
| `sentry_webhook_delivery_targets` | One row for each trigger subscription that must receive the delivery, with its own state. |
| `sentry_app_installation_deletions` | Installations that Sentry deleted. A delayed `installation.created` cannot store a grant for them. |
| `sentry_webhook_receipts` | The admin receipt. `http_status` is the answer that Sentry received. `outcome` and `processing_error` show the processing state. |

The payload uses the application encryptor with the inbox ID as associated
data. Key rotation includes `sentry_webhook_inbox.payload_ciphertext`.
SuperPlane removes the payload when every target is processed, filtered, or
removed. Finished metadata stays for 14 days. Cleanup never deletes an
unfinished delivery or its receipt.

SuperPlane stores every request, also when the body is the same. Sentry
creates a new `Request-ID` for each retry, so a header or a payload hash
cannot identify a duplicate event. The inbox ID and the target identity
prevent duplicates from worker retries. Separate Sentry deliveries can
still start separate runs.

## Outcomes

| Outcome | Meaning |
| --- | --- |
| Queued | Committed. No worker has claimed it yet. |
| Processing | A worker holds the lease. |
| Retrying | A temporary failure. The next attempt is scheduled. |
| Blocked | An operator or user must act. SuperPlane still retries. |
| Processed | At least one trigger emitted an event, or the installation event finished. |
| Filtered | A configured filter excluded the delivery, for example an action or project filter, a paused intake, a removed trigger, or no subscription for the resource. |

Retries start after five seconds, double with up to 20% jitter, and stop
growing at five minutes. No retry limit deletes a delivery.

## Processes and configuration

| Process | Configuration |
| --- | --- |
| API | `SENTRY_WEBHOOK_DURABLE_ACCEPTANCE=yes` turns on durable acceptance. Database access and the encryption keys (`ENCRYPTION_MASTER_KEYS` or the configured key source). |
| Workers | `START_SENTRY_WEBHOOK_WORKER=yes`. Database access, the encryption keys, outbound HTTPS access to `sentry.io`, RabbitMQ for event notifications, and `SUPERPLANE_SENTRY_APP_SLUG`, `SUPERPLANE_SENTRY_APP_CLIENT_ID`, and `SUPERPLANE_SENTRY_APP_CLIENT_SECRET` where the public app is configured. |
| Runners | No change. |

The worker polls every second, claims up to 25 deliveries, and runs five at
a time. A lease lasts five minutes, and the worker renews it every minute.
A lost lease cancels the Sentry API requests of that attempt.

The bundled chart sets `START_SENTRY_WEBHOOK_WORKER=yes` on the workers and
reads `sentryWebhooks.durableAcceptance` for the API. The hosted chart in
`superplanehq/helm-chart` and the values in `superplanehq/launchpad` need the
same keys. The hosted workers also need the public Sentry app Secret, which
today only the API reads.

## Rollout

1. Apply the `20261102172647_add-sentry-webhook-inbox` migration.
2. Deploy workers with `START_SENTRY_WEBHOOK_WORKER=yes` and the public app
   Secret. Confirm the log line `Starting Sentry Webhook Worker`.
3. Deploy the API with `SENTRY_WEBHOOK_DURABLE_ACCEPTANCE=no`.
4. Set `SENTRY_WEBHOOK_DURABLE_ACCEPTANCE=yes` on the API.
5. Confirm that new receipts in Installation Admin move from Queued to
   Processed or Filtered.

The API never falls back to synchronous processing after a failed commit. It
answers 500, and Sentry sends the delivery again.

## Rollback

1. Set `SENTRY_WEBHOOK_DURABLE_ACCEPTANCE=no` on the API. New deliveries use
   the synchronous path again.
2. Keep the workers running until no delivery is unfinished:

   ```sql
   SELECT count(*) FROM sentry_webhook_inbox WHERE completed_at IS NULL;
   ```

3. Keep the tables and the encryption keys that encrypted the payloads.

## Metrics and alerts

| Metric | Use |
| --- | --- |
| `sentry.webhooks.acceptance.duration.seconds` | Time until the answer to Sentry. The p99 must stay below one second. |
| `sentry.webhooks.processing.duration.seconds` | Duration of one attempt, by recorded state. |
| `sentry.webhooks.attempts` | Attempts by recorded state. `retrying` and `blocked` count retries. |
| `sentry.webhooks.inbox.unfinished` | Queue depth. |
| `sentry.webhooks.inbox.blocked` | Blocked deliveries. |
| `sentry.webhooks.inbox.oldest_unfinished.age` | Age in seconds of the oldest unfinished delivery. |

Alert when `sentry.webhooks.inbox.blocked` is above zero. Alert when
`sentry.webhooks.inbox.oldest_unfinished.age` is above 300 seconds. The
worker also logs `Sentry webhook delivery is blocked` at error level.

## Operator procedure for blocked deliveries

1. Find blocked deliveries and their errors. Installation Admin shows them
   on the Sentry webhooks page. In SQL:

   ```sql
   SELECT id, source, hook_resource, action, installation_uuid,
          integration_id, attempts, next_attempt_at, last_error
   FROM sentry_webhook_inbox
   WHERE completed_at IS NULL AND state = 'blocked'
   ORDER BY received_at;
   ```

2. Find the waiting targets of one delivery:

   ```sql
   SELECT workflow_id, node_id, state, last_error
   FROM sentry_webhook_delivery_targets
   WHERE inbox_id = '<inbox id>' AND completed_at IS NULL;
   ```

3. Repair the cause:
   - `no connection uses Sentry installation`: connect the Sentry
     organization again, or confirm that the installation was removed.
   - `the canvas has no live version`: publish the canvas.
   - `reconnect required`: ask the user to install the SuperPlane Sentry app
     again. A later `installation.deleted` finishes the old delivery.
   - `this worker cannot process public Sentry app installations`: add the
     public app Secret to the workers.
   - `decrypt payload`: restore the encryption key that encrypted the payload.
   - `unsupported sentry webhook payload`: inspect the receipt and fix the
     parser before you continue.

4. The worker retries every blocked delivery at least every five minutes.
   To retry now after the repair:

   ```sql
   UPDATE sentry_webhook_inbox
   SET next_attempt_at = now()
   WHERE id = '<inbox id>' AND completed_at IS NULL;
   ```

Do not delete inbox rows to clear an alert. A deleted row loses the event.

## Limits

SuperPlane can recover deliveries that it committed. It cannot recover
events that Sentry never sends, and it cannot compensate for unrecoverable
database loss.
