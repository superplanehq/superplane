# Organization suspension cancellation

The regular worker polls pending suspension events every five seconds.
Without pending events, it does not cancel work.

Each processing pass uses one short transaction for a bounded batch.
It locks the organization first, then the pending suspension event, with
`FOR UPDATE SKIP LOCKED`. Other replicas skip busy rows and retry later.
The worker checks that the event still matches the current suspension before
it changes any work. All row locks are released when the transaction ends.
No dedicated database connection, advisory lock, or processing status is needed.

The worker discards queued factory work and requests cancellation of workflow
runs and runner tasks. It publishes notifications after the transaction commits.
Pending work remains in the database, so later passes retry notification delivery
and recover after process restarts. In-memory cursors let a pass reach later work
while earlier work stops. A restart can safely repeat earlier batches.

The existing execution terminator, run finalizer, and runner cancellation paths
stop the work. The suspension event remains pending until that work stops.
Then the worker sets `cancellation_completed_at`, which allows reactivation.
Schedules retain their future configuration. Built-in agent sessions are outside
the scope of suspension cancellation.

## Process configuration

Regular workers use their existing database and `RABBITMQ_URL` configuration.
Keep `START_RUN_FINALIZER=yes` and `START_EXECUTION_TERMINATOR=yes`.
Runner gateways use their existing database and RabbitMQ configuration.
Control notifications wake gateways, which read durable task cancellation requests.
Runner completion, log reporting, and cleanup use their existing paths.

This worker adds no configuration keys, Secret references, or rollout gate.
Deploy it after the model foundation and before the suspension API, UI, and
admission enforcement.
