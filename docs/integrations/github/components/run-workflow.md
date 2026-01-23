---
app: "github"
label: "Run Workflow"
name: "github.runWorkflow"
type: "component"
---

# Run Workflow

Run GitHub Actions workflow

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| passed | Passed | - |
| failed | Failed | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| workflowFile | Workflow file | string | yes | - |
| ref | Branch or tag | git-ref | yes | - |
| inputs | Inputs | list | no | - |

## Example Output

```json
{
  "data": {
    "workflow_run": {
      "conclusion": "success",
      "html_url": "https://github.com/acme/widgets/actions/runs/9001",
      "id": 9001,
      "status": "completed"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.workflow.finished"
}
```

