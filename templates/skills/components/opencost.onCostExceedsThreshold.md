# opencost.onCostExceedsThreshold

## Purpose

Polls OpenCost and triggers a workflow when cost allocation data exceeds a configured dollar threshold.

## Required Configuration

| Field       | Type   | Required | Description                                |
|-------------|--------|----------|--------------------------------------------|
| window      | string | yes      | Time window (e.g., `1h`, `24h`, `7d`)     |
| aggregate   | select | yes      | Dimension: namespace, pod, cluster, etc.   |
| threshold   | string | yes      | Dollar threshold (e.g., `100.00`)          |

## Planning Rules

1. This trigger polls every 5 minutes — it does not use webhooks.
2. The threshold is compared against the **total cost** across all allocations for the given window and aggregate.
3. If total cost exceeds the threshold, the trigger fires and includes which items exceeded it.
4. Use `24h` window for daily budget checks, `1h` for near-real-time alerting.

## Event Semantics

- `type`: `opencost.costExceedsThreshold`
- `data.totalCost`: combined cost across all allocations
- `data.threshold`: the configured threshold
- `data.window`: the time window queried
- `data.aggregate`: the aggregation dimension
- `data.exceedingItems[]`: each allocation that individually exceeds the threshold

## Payload Field Mapping

| Expression                                | Description              |
|-------------------------------------------|--------------------------|
| `event.data.totalCost`                    | Total cost               |
| `event.data.threshold`                    | Configured threshold     |
| `event.data.window`                       | Time window              |
| `event.data.aggregate`                    | Aggregate dimension      |
| `event.data.exceedingItems[0].name`       | First exceeding item     |
| `event.data.exceedingItems[0].totalCost`  | Its total cost           |

## Common Patterns

- **Budget alert**: opencost.onCostExceedsThreshold → slack.sendMessage
- **Cost report with alert**: opencost.onCostExceedsThreshold → opencost.getCostAllocation → slack.sendMessage
- **Incident creation**: opencost.onCostExceedsThreshold → pagerduty.createIncident

## Mistakes to Avoid

- Setting threshold to `0` — this will always trigger.
- Using very short windows like `1m` — OpenCost may not have data at that granularity.
