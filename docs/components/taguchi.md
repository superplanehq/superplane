# Taguchi multi-variate testing

Four native components that let a SuperPlane canvas run a Taguchi-style
multi-variate deployment experiment: define factors, fan out one deployment
per arm, collect trial observations, gate on sample size, analyze, and
promote the winner.

All math (orthogonal arrays, S/N ratios, ANOVA) is delegated to
[`github.com/aleksicmarija/taguchi`](https://github.com/aleksicmarija/taguchi)
(declared import path `github.com/marijaaleksic/taguchi`).

## Components

| Name | Role | Output channels |
|---|---|---|
| `taguchiPlan` | Define factors/levels, pick orthogonal array, emit arms | `default` |
| `taguchiArms` | Fan out one payload per arm read from memory | `each`, `empty` |
| `taguchiStatus` | Gate on min trials per arm | `sampleSizeMet`, `pending` |
| `taguchiAnalyze` | Rank arms, compute main effects, pick a winner | `confident`, `inconclusive` |

## Memory schema

Namespace: `taguchi:{experimentId}`. Rows:

- `{kind: "arm", arm_id, params: {...}, deployed_at}` — written by `taguchiPlan`.
- `{kind: "trial", arm_id, trial_id, metric, value}` — written by the canvas
  upstream of `taguchiStatus` / `taguchiAnalyze`. Typically a `webhook`
  trigger → `addMemory` node accepts external trial events and writes them
  into this namespace.

## Wiring (conceptual)

```
webhook (experiment-start)
  → taguchiPlan
  → taguchiArms (fan-out `each`)
  → http (deploy arm to target)

webhook (trial-ingest)
  → addMemory (namespace = taguchi:{experimentId}, kind = "trial")
  → taguchiStatus (minPerArm = 30)
      sampleSizeMet → taguchiAnalyze
                        confident → approval → http (promote winner)
                        inconclusive → notify
      pending → (loop back / wait for more events)
```

## Example `taguchiPlan` config

```yaml
component:
  name: taguchiPlan
configuration:
  experimentId: "{{ $['webhook'].data.deployment_id }}"
  factors:
    - name: icons
      levels: [classic, themed_a, themed_b]
    - name: board_size
      levels: ["7x6", "9x7", "11x8"]
    - name: time_limit
      levels: [none, blitz_30s, classical_5min]
```

Outputs a 9-arm plan (L9) and writes one `{kind: arm, ...}` row per arm
to canvas memory.

## Example `taguchiAnalyze` config

```yaml
component:
  name: taguchiAnalyze
configuration:
  experimentId: "{{ $['plan'].experimentId }}"
  metric: rematch_rate
  direction: larger
  confidenceThreshold: 1.0
```

Emits `{winner, ranking, mainEffects, confidence}`. Routes to `confident`
when `|winnerMean − runnerUpMean| ≥ confidenceThreshold × pooledStdDev`.
