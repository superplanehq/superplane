# opencost.getCostAllocation

## Purpose

Fetches cost allocation data from OpenCost, grouped by a specified dimension (namespace, pod, cluster, etc.).

## Required Configuration

| Field      | Type   | Required | Description                                 |
|------------|--------|----------|---------------------------------------------|
| window     | string | yes      | Time window (e.g., `1h`, `7d`, `30d`)      |
| aggregate  | select | yes      | Dimension: namespace, pod, cluster, etc.    |
| step       | string | no       | Interval for data points (e.g., `1h`, `1d`)|
| resolution | string | no       | Data granularity (e.g., `1m`, `10m`)        |

## Planning Rules

1. Window and aggregate are required. Step and resolution are optional refinements.
2. The output contains an array of allocations, each with cost breakdowns (CPU, GPU, RAM, PV, network).
3. Use this component when you need the full cost breakdown, not just a threshold check.

## Output Semantics

- `type`: `opencost.allocation`
- `data.window`: queried window
- `data.aggregate`: grouping dimension
- `data.totalCost`: sum of all allocations
- `data.allocations[]`: individual cost entries

## Payload Field Mapping

| Expression                                  | Description                  |
|---------------------------------------------|------------------------------|
| `output.data.totalCost`                     | Total cost                   |
| `output.data.window`                        | Time window                  |
| `output.data.aggregate`                     | Aggregate dimension          |
| `output.data.allocations[0].name`           | First allocation name        |
| `output.data.allocations[0].totalCost`      | Its total cost               |
| `output.data.allocations[0].cpuCost`        | CPU cost                     |
| `output.data.allocations[0].ramCost`        | RAM cost                     |

## Common Patterns

- **Daily cost report**: schedule → opencost.getCostAllocation → slack.sendMessage
- **Cost-triggered action**: opencost.onCostExceedsThreshold → opencost.getCostAllocation → slack.sendMessage

## Mistakes to Avoid

- Forgetting to set the window — empty window will fail validation.
- Using unsupported aggregate dimensions — stick to: namespace, pod, controller, service, label, cluster, node.
