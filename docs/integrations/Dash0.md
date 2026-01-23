---
title: "Dash0"
sidebar:
  label: "Dash0"
type: "application"
name: "dash0"
label: "Dash0"
---

Connect to Dash0 to query data using Prometheus API

### Components

- [List Issues](#list-issues)
- [Query Prometheus](#query-prometheus)

## List Issues

Query Dash0 to get a list of all current issues using the metric dash0.issue.status

### Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| checkRules | Check Rules | app-installation-resource | no | Select one or more check rules to filter issues |

### Example Output

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
  "type": "dash0.issues.list"
}
```

## Query Prometheus

Execute a PromQL query against Dash0 Prometheus API and return the response data

### Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| query | PromQL Query | text | yes | The PromQL (Prometheus Query Language) query to execute |
| dataset | Dataset | string | yes | The dataset to query |
| type | Query Type | select | yes | - |
| start | Start Time | string | no | Start time for range queries (e.g., 'now-5m', '2024-01-01T00:00:00Z') |
| end | End Time | string | no | End time for range queries (e.g., 'now', '2024-01-01T01:00:00Z') |
| step | Step | string | no | Query resolution step width for range queries (e.g., '15s', '1m', '5m') |

### Example Output

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

