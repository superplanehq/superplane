---
app: "semaphore"
label: "Run Workflow"
name: "semaphore.runWorkflow"
type: "component"
---

# Run Workflow

Run Semaphore workflow

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| passed | Passed | - |
| failed | Failed | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| project | Project | app-installation-resource | yes | - |
| pipelineFile | Pipeline file | string | yes | - |
| ref | Pipeline file location | git-ref | yes | - |
| commitSha | Commit SHA | string | no | - |
| parameters | Parameters | list | no | - |

## Example Output

```json
{
  "data": {
    "extra": {
      "triggeredBy": "SuperPlane"
    },
    "pipeline": {
      "id": "ppl_456",
      "result": "passed",
      "state": "done"
    },
    "workflow": {
      "id": "wf_123",
      "url": "https://semaphore.example.com/workflows/wf_123"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "semaphore.workflow.finished"
}
```

