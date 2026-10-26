# Runtime configuration

SuperPlane reads a small set of environment variables at runtime to tune execution limits. These apply to self-hosted deployments.

## Execution limits

| Variable | Default | Description |
| --- | --- | --- |
| `SUPERPLANE_MAX_EMIT_COUNT` | `100` | Maximum number of events a single component execution may emit at once. Applies to fan-out components such as **For Each** (one event per array item) and **Read Memory** when emit mode is **One By One**. |
| `SUPERPLANE_MAX_PAYLOAD_SIZE` | `524288` (512 KiB) | Maximum serialized size of an emitted event payload, in bytes. |

Invalid or non-positive values are ignored; the default is used instead.

Set these on the SuperPlane API and worker processes. Changes take effect on the next process start (or immediately on the next read, depending on the variable).

## Parallel runner limit

Installation admins set **Parallel runner limit** in **Billing plans**. The
existing `parallel_task_limit` field limits reserved or running runner jobs
across all workspaces and fleets in an organization. Each job consumes one
slot, including jobs from the same automation. Billing disabled or an absent
limit means unlimited capacity.

Jobs wait in the existing fleet queue when the organization has no capacity.
The idle-runner reconciliation loop retries them. Other organizations retain
their own capacity. Completion, cancellation, and runner-loss rules release
capacity through job state changes. Cancellation requests alone do not
release a slot.

Reservation attempts read the current effective plan. A lower limit preserves
current reservations and blocks new reservations until usage falls below it.
Automation and line-step limits, credential expiry, and timeout behavior stay
unchanged. Long-wait reliability is separate work.

Pending and idle instances consume no job slots. Fleet Manager can still
provision idle instances for blocked queues. Full enforcement starts after
all application instances that handle reservations use this code. No Fleet
Manager protocol change is required.
