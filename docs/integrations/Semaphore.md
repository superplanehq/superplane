---
title: "Semaphore"
sidebar:
  label: "Semaphore"
type: "application"
name: "semaphore"
label: "Semaphore"
---

Run and react to your Semaphore workflows

### Components

- [Run Workflow](#run-workflow)

### Triggers

- [On Pipeline Done](#on-pipeline-done)

## Components

### Run Workflow

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

## Triggers

### On Pipeline Done

Listen to Semaphore pipeline done events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| project | Project | app-installation-resource | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "blocks": [
      {
        "jobs": [
          {
            "id": "00000-00000-00000-00000-00000",
            "index": 0,
            "name": "Report result to SuperPlane",
            "result": "passed",
            "status": "finished"
          }
        ],
        "name": "Block #1",
        "result": "passed",
        "result_reason": "test",
        "state": "done"
      }
    ],
    "organization": {
      "id": "00000000-0000-0000-0000-000000000000",
      "name": "test"
    },
    "pipeline": {
      "created_at": "2026-01-19T12:00:00Z",
      "done_at": "2026-01-19T12:00:00Z",
      "error_description": "",
      "id": "00000000-0000-0000-0000-000000000000",
      "name": "Initial Pipeline",
      "pending_at": "2026-01-19T12:00:00Z",
      "queuing_at": "2026-01-19T12:00:00Z",
      "result": "passed",
      "result_reason": "test",
      "running_at": "2026-01-19T12:00:00Z",
      "state": "done",
      "stopping_at": "1970-01-01T00:00:00Z",
      "working_directory": ".semaphore",
      "yaml_file_name": "semaphore.yml"
    },
    "project": {
      "id": "00000000-0000-0000-0000-000000000000",
      "name": "test"
    },
    "repository": {
      "slug": "test/test",
      "url": "https://github.com/test/test"
    },
    "revision": {
      "branch": {
        "commit_range": "0000000000000000000000000000000000000000^...0000000000000000000000000000000000000000",
        "name": "test"
      },
      "commit_message": "Merge branch 'test' into test",
      "commit_sha": "0000000000000000000000000000000000000000",
      "pull_request": null,
      "reference": "refs/heads/test",
      "reference_type": "branch",
      "sender": {
        "avatar_url": "https://avatars2.githubusercontent.com/u/0000000000000000000000000000000000000000?s=460\u0026v=4",
        "email": "test@test.com",
        "login": "test"
      },
      "tag": null
    },
    "version": "1.0.0",
    "workflow": {
      "created_at": "2026-01-19T12:00:00Z",
      "id": "00000000-0000-0000-0000-000000000000",
      "initial_pipeline_id": "00000000-0000-0000-0000-000000000000"
    }
  },
  "timestamp": "2026-01-19T12:00:00Z",
  "type": "semaphore.pipeline.done"
}
```

