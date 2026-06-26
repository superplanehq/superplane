## Summary

Fix EC2 scale-down so fleet-manager drains selected runners through task-broker before calling `TerminateInstances`.

The fix also keeps scale-down from terminating instances that currently own claimed broker tasks, and it fails closed when broker runner state is unavailable.

## What happened

The failed smoke task was `1dd54674-7bcb-4c61-a2f2-ec239b64a56c`.

From the logs:

1. At `2026-06-26T12:55:24Z`, fleet-manager launched 10 `e1-tiny-amd64` runners, including `i-0a7cd3e280fc52cc7` and `i-0ac896c389590de86`.
2. At `2026-06-26T13:16:16Z`, fleet-manager read task counts for the fleet. The broker returned no queued or claimed work, so dynamic scaling computed `want = headroom = 10`.
3. At `2026-06-26T13:16:17Z`, fleet-manager also swept 2 unhealthy runners, then still saw excess live capacity and logged `ec2 terminating excess runners` for `i-0a7cd3e280fc52cc7` and `i-0ac896c389590de86`.
4. EC2 termination is asynchronous. Those runner WebSocket connections were still alive in task-broker after fleet-manager requested termination.
5. At `2026-06-26T13:17:13Z`, task-broker assigned the smoke task to `i-0a7cd3e280fc52cc7`, immediately requeued it after runner infrastructure failure, then assigned it to `i-0ac896c389590de86`.
6. The task then failed with `context canceled`. The task CloudWatch stream was missing, which matches a runner dying before it could execute and ship task logs.

## Root Cause

The old scale-down path selected oldest excess EC2 instances and terminated them directly. It did not coordinate with task-broker before termination.

That left a race:

- Fleet-manager selected an idle connected runner for termination.
- EC2 termination started but had not yet closed the runner process or broker WebSocket.
- Task-broker still considered the runner available and assigned new work to it.
- The instance was already on its way down, so the task failed for infrastructure reasons.

Protecting only already-claimed tasks is necessary but not sufficient. In this incident, the selected runners were idle at scale-down time and became unsafe only because they were assigned work after termination had started.

## Fix

1. `GET /v1/fleets/{id}/task-counts` now includes `claimed_runner_ids`, so fleet-manager can avoid terminating instances that already own claimed tasks.
2. task-broker now has `POST /v1/runners/drain`. Fleet-manager calls it with the EC2 instance IDs it wants to remove.
3. task-broker marks those runner IDs as draining and returns per-runner status:
   - `drained`: the runner has no active or in-progress claim and can be terminated.
   - `busy`: the runner is running, claiming, or belongs to another fleet, so fleet-manager must not terminate it in this reconcile.
4. The WebSocket and legacy HTTP claim paths both check the drain hub before claiming a task. The check and claim reservation are protected by the same lock used by drain, so the broker cannot report a runner as drained while it is entering `ClaimTask`.
5. fleet-manager now calls `TerminateInstances` only for runners that task-broker reported as `drained`. If the broker drain request fails, scale-down fails closed and skips termination.

## Why This Fixes It

The fix moves the termination decision from "EC2 looks excess" to "EC2 is excess and task-broker has stopped assigning it work."

For the incident race, fleet-manager would have drained `i-0a7cd3e280fc52cc7` and `i-0ac896c389590de86` before terminating them. Once drained, task-broker would reject any new claim from those runner IDs and close idle WebSocket streams. The smoke task would remain queued for a non-draining runner instead of being assigned to an instance already being terminated.

The claim reservation also covers the narrow concurrent race: if a runner is between "available" and "claimed", drain returns `busy` instead of `drained`, and fleet-manager leaves that instance alive for the current reconcile.

## Tests

- `go test ./...`
- Added broker drain hub tests for idle, active, unknown, wrong-fleet, and currently-claiming runners.
- Added a WebSocket regression test proving a drained runner does not receive a queued task.
- Added fleet-manager tests proving scale-down terminates only broker-drained runners and fails closed when broker drain fails.
