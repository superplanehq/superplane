# Parallel Branches Converge With Merge

Keywords: parallel, fan-in, fan-out, converge, wait for both, multiple branches, synchronize, merge, join

Use this pattern when a user wants two or more steps to run in parallel and converge before downstream work proceeds.

## Decision Checklist

1. Detect parallel-then-converge intent: "in parallel", "wait for both", "after both finish", "fan out then merge".
2. Pick one upstream node as the fan-out point; subscribe each parallel branch directly to it.
3. Build each branch independently; branches need not be the same shape or length.
4. Add one `merge` node and subscribe it to the last node of every parallel branch.
5. Subscribe downstream consumers to `merge` on the `success` channel.
6. Do not chain branches sequentially (`A -> B -> merge`); each branch must hang off the fan-out node.

## Canonical workflow

```
trigger: any trigger (e.g. `start`, `github.onPullRequest`)
├─ branch A: <action> -> ...
├─ branch B: <action> -> ...
=> `merge` (channels: success | timeout | fail)
-> downstream consumer (subscribe on `success`)
```

## Notes

- `merge` channels are `success`, `timeout`, `fail`. There is no `stopped` channel.
- Optional `enableTimeout` bounds wait time and emits on `timeout`. Use only if the user asked for a deadline.
- Optional `enableStopIf` + `stopIfExpression` short-circuit and emit on `fail`. Use only if the user asked for an early-fail rule.
