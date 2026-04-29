# Await Parallel Upstream Nodes

Keywords: merge, merge component, wait for all, branches, parallel branches, combine, sync, fan in

Use this pattern when a user wants two or more nodes to run in parallel, but to wait for all to emit before a single downstream node executes.

## Decision Checklist

1. Identify the trigger that starts the workflow.
2. Identify two or more action nodes to run in parallel — these sit between the trigger and `merge`, each performing independent work. Each node must have a distinct name; two subscriptions from the same node to `merge` count as one input regardless of channel.
3. Subscribe each parallel node to the trigger.
4. Add a `merge` node downstream of all parallel nodes — `merge` collects emissions from each parallel node before allowing any downstream work to proceed.
5. Subscribe each parallel node to `merge`.
6. If a downstream node exists, subscribe it to `merge`'s `success` channel.

## Canonical workflow

trigger: <trigger>
-> action_1
-> action_2

action_1 -> `merge`
action_2 -> `merge`

`merge` (success) -> downstream_node

## Notes

- `merge` supports a configurable timeout; if a parallel node may not always emit, enable it and subscribe a downstream node to `merge`'s `timeout` channel to handle the case.
- `merge` supports conditional early stop via `stopIfExpression`; if a failure in one parallel node should stop the workflow, enable it, subscribe `merge` to each parallel node's `fail` channel, and subscribe the downstream node to `merge`'s `fail` channel. When upstream nodes emit different shapes, write the expression defensively — evaluation errors emit on the `fail` channel rather than blocking the queue.
