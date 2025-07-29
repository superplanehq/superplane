An executor is what the stage calls when the event passes all the conditions and exits the queue. When defining your executor specification, you can use the syntax `${{ inputs.* }}` and `${{ secrets.* }}` to access the inputs and secrets defined in the stage.

Most executors are intended for use with integrations. See [integrations](integrations.md) for more information.

The available executor types are:
- [HTTP Executor](#http-executor)
- [Semaphore Executor](#semaphore-executor)
  - [Automatic Parameters](#automatic-parameters)
- [GitHub Executor](#github-executor)
  - [Example](#example)
  - [Specification](#specification)
  - [Automatic Inputs](#automatic-inputs)

### HTTP Executor

The HTTP Executor allows you to make HTTP POST requests to external services when a stage is executed.

<b>Example</b>

```yaml
executor:
  type: http
  spec:
    url: https://api.example.com/endpoint
    payload:
      key1: value1
      key2: ${{ inputs.KEY2 }}
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
    responsePolicy:
      statusCodes: [200, 201, 202]
```

- `url`: the URL to which the HTTP request will be sent.
- `payload`: used to send data to the external service through the request body. If nothing is specified, request body will be empty.
- `headers`: used to set headers for the request. If nothing is specified, no headers are sent.
- `responsePolicy`: defines what the successful response looks like. Currently, you can specify which HTTP status codes that are considered successful.

### Semaphore Executor

The Semaphore Executor allows you to trigger Semaphore pipelines when a stage is executed.

<b>Example</b>

```yaml
executor:
  type: semaphore
  integration:
    name: semaphore
  resource:
    type: project
    name: my-semaphore-project
  spec:
    task: my-task
    branch: sxmoon
    pipelineFile: .semaphore/pipeline_3.yml
    parameters:
      VERSION_A: ${{ inputs.VERSION_A }}
      VERSION_B: ${{ inputs.VERSION_B }}
```

If the `task` is not specified, the executor will use the [workflows API](https://docs.semaphoreci.com/reference/api#run-workflow) to run a workflow. If the `task` is specified, the executor will use the [tasks API](https://docs.semaphoreci.com/reference/api#run-task) to run the task.

#### Automatic Parameters

The executor automatically adds these parameters:
- `SUPERPLANE_STAGE_ID`: Current stage identifier
- `SUPERPLANE_STAGE_EXECUTION_ID`: Current execution identifier
- `SUPERPLANE_STAGE_EXECUTION_TOKEN`: Execution token (if available)

### GitHub Executor

The GitHub executor can trigger GitHub Actions workflows via workflow dispatch events.

#### Example

```yaml
executor:
  type: github
  integration:
    name: github-integration
  resource:
    type: repository
    name: owner/repository
  spec:
    workflow: .github/workflows/deploy.yml
    ref: main
    inputs:
      environment: production
      version: ${{ inputs.VERSION }}
```

#### Specification

- `workflow` (required): Workflow file name (e.g., "deploy.yml") or workflow name to trigger
- `ref` (required): Git branch, tag, or commit SHA to run the workflow against
- `inputs` (optional): Input parameters to pass to the workflow

#### Automatic Inputs

The executor automatically adds these workflow inputs:
- `superplane_stage_id`: Current stage identifier
- `superplane_execution_id`: Current execution identifier
- `superplane_execution_token`: Execution token (if available)

Note: they must be defined in your workflow file.
