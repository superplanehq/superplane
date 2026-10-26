# Runner queue and organization limits

SuperPlane limits concurrent integrated runner jobs across an organization. The
limit applies to every workspace and fleet. The billing plan retains the stored
`parallel_task_limit` key. The UI calls this setting **Parallel runner limit**.
Billing disabled or an absent limit means unlimited runner capacity.

Each job in `reserved` or `running` state uses one slot. A planning job and three
verification jobs use four slots, even when they belong to one task. Reservations
include provisioning and credential preparation. Queued jobs, idle runners,
approvals, and tasks without assigned runner jobs use no slots.

The reservation transaction locks the organization before the runner and job.
Automatic assignment and explicit administrator reservations use this boundary.
Usage comes from job state. SuperPlane does not maintain a separate slot counter.
A limit reduction keeps existing reservations. New jobs wait until usage falls
below the new limit.

## Waiting states

Automation concurrency and line-step concurrency retain their independent limits.
Their queues can contain work before a runner job exists. The UI shows that work
as an automation wait and does not assign an organization runner position.

After an automation creates a runner job, `queued_runner_jobs` identifies each
queued, unassigned job. Entries include the job, automation, node, run, execution,
and organization position. Positions use creation time and ID. Jobs without a
task association still count in the same queue. Available fleets can start a
later job first, so a queue position describes waiting order rather than a start
guarantee.

A task can have running work and several queued jobs at the same time. Desktop
and mobile views keep the task status visible beside individual runner waits.
Execution details distinguish **Waiting for runner**, **Starting runner**, and
**Running**. Reserved jobs include provisioning and credential preparation.
The workspace header shows organization usage, for example **4 of 10 active
runners**.

Start and Analyze continue through the existing automation flow. Organization
runner capacity does not change task lifecycle or hide Start.

## Cancel one queued job

Queue cancellation targets one exact job ID. SuperPlane checks organization and
task ownership inside the reservation boundary. Cancellation succeeds only while
the job remains queued and unassigned. A stale request returns
`FailedPrecondition`. It does not stop an assigned job, its siblings, or its parent
run. Cancellation requests for assigned jobs retain capacity until stopping is
confirmed or the runner is lost.

Completion, failure, acknowledged cancellation, runner loss, and safe release of
an undelivered reservation release capacity. Queue and capacity notifications
refresh organization usage and queue positions across workspaces. Polling recovers
updates missed during disconnection.

## Runner provisioning

Configured warm capacity is independent of queued demand. Fleet Manager subtracts
unassigned pending and idle supply before it provisions additional runners. A
runner created for demand reserves its job before the infrastructure provider is
called. Capacity denial or a stale candidate creates no logical runner or
provider resource.

Reserved runners cannot scale down. Deletion checks assignment again inside its
transaction. Provisioning failure or the ten-minute registration deadline
terminates the pending runner, requeues its undelivered job, and starts a durable
60-second fleet cooldown. Manager restarts preserve the cooldown. Connected
healthy runners can still accept work during cooldown.

## Credentials and preparation failures

Saved refresh descriptors identify stable resources and permitted scopes. They
refresh temporary credentials while preserving commands, prompts, endpoints,
and configuration. Resource deletion, revoked access, or an incompatible account
or recipient change fails preparation.

Preparation starts when a runner is available for delivery. Its durable deadline
is five minutes from the first attempt. Reconnects and restarts do not reset the
deadline. Each attempt has at most 30 seconds. Temporary provider failures retry
with increasing delays. Permanent failures or deadline exhaustion fail a
certainly-unsent job, release its slot, and retain normal execution finalization
and credential cleanup. Execution start stays empty when delivery never starts.

A persisted dispatch marker separates certainly-unsent jobs from writes that
could have reached the runner. After this marker, cancellation follows assigned
job acknowledgment and runner-loss rules.

Provider key creation has an attempt identity and durable cleanup record. Rejected
and superseded keys are revoked. An uncertain creation result is reconciled
before another mint attempt. Expiring keys limit exposure when provider access
prevents immediate cleanup.

## Automation timeout accounting

New Run App invocations exclude the union of queued, unassigned runner-job
intervals in the child run and its descendants. Overlapping waits pause the
parent once. Sequential waits accumulate. Reservation, provisioning, preparation,
running time, approvals, and automation-concurrency waits remain charged.

Queue intervals persist when jobs enter the queue. Reservation or terminal
cancellation closes an interval atomically. Requeueing opens another interval.
The durable timeout hook recomputes remaining time and checks every 15 seconds
while a relevant wait remains open. Late timeout hooks leave completed children
unchanged. Existing invocations without complete history keep their previous
timeout behavior. Individual runner execution timeouts start at actual execution.

## API and operations

Organization workspace usage returns `active_runner_count` and optional
`parallel_runner_limit`. Work order details and list rows return
`queued_runner_jobs`. Canvas execution details and references expose runner job
state and queue, reservation, start, and finish timestamps.

Monitor preparation deadline expiry, cleanup failures, reservation age, and fleet
provisioning cooldowns. A growing runner queue can indicate an organization limit,
a fleet shortage, or a provider failure. Compare queue age with reservation and
preparation state before changing capacity.

Deploy the updated runner gateways, administrator API, and Fleet Manager together.
Treat the organization limit as enforced only after every reservation writer uses
the shared boundary and every manager sends the job ID when it creates a runner
for queued demand.
