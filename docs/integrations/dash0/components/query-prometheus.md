---
app: "dash0"
label: "Query Prometheus"
name: "dash0.queryPrometheus"
type: "component"
---

# Query Prometheus

Execute a PromQL query against Dash0 Prometheus API and return the response data

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| query | PromQL Query | text | yes | The PromQL (Prometheus Query Language) query to execute |
| dataset | Dataset | string | yes | The dataset to query |
| type | Query Type | select | yes | - |
| start | Start Time | string | no | Start time for range queries (e.g., 'now-5m', '2024-01-01T00:00:00Z') |
| end | End Time | string | no | End time for range queries (e.g., 'now', '2024-01-01T01:00:00Z') |
| step | Step | string | no | Query resolution step width for range queries (e.g., '15s', '1m', '5m') |

## Example Output

```json
{
  "data": {
    "data": {
      "result": [
        {
          "metric": {
            "service_name": "test"
          },
          "value": [
            1234567890,
            "1"
          ],
          "values": [
            [
              1234567890,
              "1"
            ],
            [
              1234567900,
              "2"
            ]
          ]
        }
      ],
      "resultType": "vector"
    },
    "status": "success"
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "dash0.prometheus.response"
}
```

