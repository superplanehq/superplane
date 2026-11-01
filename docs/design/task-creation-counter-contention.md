# Task creation counter contention

## Incident boundary

[PRODUCTION-BK](https://superplane.sentry.io/issues/7822956539/events/695e03422d2a42759a6b66817b8c9b80/)
reports three statement timeouts on November 1, 2026, at approximately 08:30 UTC.
The waiting operation allocates a task number in
`Factory.allocateNextWorkOrderNumber`. The timeout logger reports the error;
it does not identify the blocker.

The allocator updates one factory row by its primary key and organization ID.
It does not scan the task table. PostgreSQL holds the counter row lock until
the caller's transaction ends. Other counter allocations and factory updates
can wait for this lock.

## Local evidence

`TestFactoryContext_CreateWorkOrderRemoteFetchDoesNotLockCounter` executes real
creation through `FactoryContext` in two independent PostgreSQL transactions.
Synchronization channels hold the first creation during a GitHub image read,
remote object upload, or workspace-file copy. The second creation targets the
same factory with a local 500 ms statement timeout.

On the original code, the second transaction executes its setup and lookup
statements, then fails on `UPDATE factories ... RETURNING`. PostgreSQL reports
SQLSTATE `57014` after approximately 500 ms in all three cases. This identifies
counter contention, not connection-pool exhaustion, in the reproduction.

After the repair, the second creation commits before the external operation resumes.
Both tasks persist with distinct positive numbers. The first task retains its
source run, final description, and task file.

`TestFactoryContext_CreateWorkOrderRollbackCleansPreparedObjects` forces a
transaction failure after allocation, file persistence, and creation-event
insertion. The task, file rows, event, and counter roll back. Cleanup removes
the staged remote object and workspace copy. The original workspace object
and binding remain intact. The next successful creation reuses number 1.

`TestFactoryContext_CreateWorkOrderFailureInsideCommittingCaller` rejects file
persistence after number allocation, then commits the caller's transaction.
The creation savepoint rolls back the task and counter. Cleanup uses the local
creation error to delete staged objects despite the successful outer commit.

## Repair and lifecycle

Creation chooses a task UUID before allocating its number. It prepares remote
content and workspace-file copies for that UUID without inserting task files.
It then allocates the counter and inserts the task, file rows, and bindings
inside the original transaction. File quotas remain checked during persistence.
Creation events use the final description and bindings.

Creation uses a savepoint when the caller already owns a transaction. This
prevents partial creation when the executor records an action error and commits.

The GitHub, Productive, Jira, and Linear ingestion paths share this preparation
lifecycle. Duplicate-intake checks still run before preparation. Their locks
remain inside the caller's transaction.

Both `NodeExecutor` and `NodeRequestExecutor` already collect `FileBindCleanup`
jobs. They apply cleanup after transaction completion. Rollback deletes copied
objects; commit deletes obsolete workspace objects. The executors publish
queued updates and notifications only after commit.

The transactional allocator, unique task-number index, database timeouts, and
Sentry reporting remain enabled. This repair requires no new configuration.

## Production correlation required

The supplied event has no blocking-session data or effective timeout settings.
No full Sentry event or production database diagnostics were available during
this implementation. The local reproduction proves a code hazard. It does not
prove that remote files caused PRODUCTION-BK.

An operator must obtain the following evidence from the affected worker path:

- The effective statement and idle-transaction timeout settings.
- The waiting statement and its start time.
- Blocking session IDs, statements, and transaction start times.
- The affected task input, including remote files and workspace references.
- The blocking transaction duration and any external-operation timing.

During recurrence, use `pg_stat_activity` and `pg_blocking_pids(pid)` to connect
the waiting allocator to its blocking transaction. Settings from an unrelated
administrative connection do not establish the worker's effective settings.
Repository defaults are 60 seconds and 30 seconds; deployed values can differ.

Correlate these records with the event time and worker
`superplane-workers-65bc7b78b-6l6vm`. Check recurrence after deployment before
marking PRODUCTION-BK resolved. If another operation caused the production
block, request operator review before expanding the repair.
