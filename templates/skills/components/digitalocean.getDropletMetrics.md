# DigitalOcean Get Droplet Metrics Skill

Use this guidance when planning or configuring the `digitalocean.getDropletMetrics` component.

## Purpose

`digitalocean.getDropletMetrics` fetches CPU usage, memory utilization, and public network bandwidth metrics for a droplet over a configurable lookback window.

It makes four API calls in sequence (CPU, memory, public outbound bandwidth, public inbound bandwidth) and emits all results in a single combined payload.

## Required Configuration

- `droplet` (required): the droplet to query metrics for. Supports expressions.
- `lookbackPeriod` (required): how far back to fetch data. Valid values: `"1h"`, `"6h"`, `"24h"`, `"7d"`, `"30d"`.

## Output Fields

The emitted `data` object contains:

- `data.dropletId`: the queried droplet ID.
- `data.start`: ISO 8601 timestamp of the start of the metrics window.
- `data.end`: ISO 8601 timestamp of the end of the metrics window (time of execution).
- `data.lookbackPeriod`: the selected lookback period string.
- `data.cpu`: CPU usage percentage time series (Prometheus matrix format — see below).
- `data.memory`: memory utilization percentage time series.
- `data.publicOutboundBandwidth`: public outbound bandwidth (Mbps) time series.
- `data.publicInboundBandwidth`: public inbound bandwidth (Mbps) time series.

Each metric object has:

- `.status`: `"success"` when data is available.
- `.data.resultType`: `"matrix"`.
- `.data.result[0].values`: array of `[unix_timestamp, "value_string"]` pairs.

## Common Mapping

- `droplet` ← `"{{ $.steps.createDroplet.data.id }}"` to query a droplet created earlier
- `lookbackPeriod` ← `"1h"` for near-real-time monitoring, `"7d"` for capacity trend analysis

Accessing metric values downstream:

- Latest CPU value: `{{ $["Get Droplet Metrics"].data.cpu.data.result[0].values[-1][1] }}`
- Latest memory value: `{{ $["Get Droplet Metrics"].data.memory.data.result[0].values[-1][1] }}`

## Planning Rules

1. Metrics are only available for droplets with the DigitalOcean Monitoring Agent installed. Official DigitalOcean images created after 2018 include the agent by default.
2. Shorter lookback periods (`"1h"`, `"6h"`) return finer-grained data points; longer periods (`"7d"`, `"30d"`) return coarser data.
3. Use this component before a scaling or alerting decision to sample current resource utilization.
4. When using metric values in downstream `if` nodes, extract the latest data point value and compare numerically.
5. Empty `result` arrays in any metric indicate that no data is available for that window — this is expected for new droplets or those without the monitoring agent.
