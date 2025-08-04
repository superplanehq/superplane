An executor is what the stage calls when the event passes all the conditions and exits the queue. When defining your executor specification, you can use the syntax `${{ inputs.* }}` and `${{ secrets.* }}` to access the inputs and secrets defined in the stage.

Most executors are intended for use with integrations. See [integrations](integrations.md) for more information.

The available executor types are:
- [HTTP](#http)
  - [Example](#example)
  - [Specification](#specification)
  - [Inputs](#inputs)
- [Semaphore](#semaphore)
  - [Example](#example-1)
  - [Specification](#specification-1)
  - [Inputs](#inputs-1)
- [GitHub](#github)
  - [Example](#example-2)
  - [Specification](#specification-2)
  - [Inputs](#inputs-2)

### HTTP

The HTTP Executor allows you to make HTTP POST requests to external services when a stage is executed.

#### Example

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

#### Specification

- `url`: the URL to which the HTTP request will be sent.
- `payload`: used to send data to the external service through the request body. If nothing is specified, request body will be empty.
- `headers`: used to set headers for the request. If nothing is specified, no headers are sent.
- `responsePolicy`: defines what the successful response looks like. Currently, you can specify which HTTP status codes that are considered successful.

#### Inputs

Other than the parameters defined in your spec, the HTTP executor will include:
- `stageId`: Current stage identifier
- `executionId`: Current execution identifier

### Semaphore

The Semaphore Executor allows you to trigger Semaphore pipelines when a stage is executed. You can use the Semaphore executor through the [Semaphore integration](integrations.md#semaphore-integration).

#### Example

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

#### Specification

- `branch`: the branch to run the workflow against.
- `pipelineFile`: the pipeline file to run the workflow against.
- `parameters`: the parameters to pass to the workflow.
- `task`: the task to run. If not specified, the executor will use the [workflows API](https://docs.semaphoreci.com/reference/api#run-workflow) to run a workflow. If specified, the execution will use the [tasks API](https://docs.semaphoreci.com/reference/api#run-task) to trigger a task.

#### Inputs

Other than the parameters defined in your spec, the Semaphore executor will include:
- `SUPERPLANE_STAGE_ID`: Current stage identifier
- `SUPERPLANE_STAGE_EXECUTION_ID`: Current execution identifier

> [!WARNING]
> If you are triggering a Semaphore task, and you need the value of the `SUPERPLANE_STAGE_ID` and `SUPERPLANE_STAGE_EXECUTION_ID` parameters, you must define them in your task.

### GitHub

The GitHub executor can trigger GitHub Actions workflows via workflow dispatch events. You can use the GitHub executor through the [GitHub integration](integrations.md#github-integration).

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

#### Inputs

Other than the inputs defined in your GitHub executor spec, the GitHub executor will always add these workflow inputs:
- `superplane_stage_id`: Current stage identifier
- `superplane_execution_id`: Current execution identifier

> [!WARNING]
> Your workflow must define the `superplane_stage_id` and `superplane_execution_id` parameters, otherwise the executor will fail.
