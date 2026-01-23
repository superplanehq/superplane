---
app: "dash0"
label: "List Issues"
name: "dash0.listIssues"
type: "component"
---

# List Issues

Query Dash0 to get a list of all current issues using the metric dash0.issue.status

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| clear | Clear | No active issues detected |
| degraded | Degraded | One or more degraded issues detected |
| critical | Critical | One or more critical issues detected |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| checkRules | Check Rules | app-installation-resource | no | Select one or more check rules to filter issues |

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
  "type": "dash0.issues.list"
}
```

