An executor is what the stage calls when the event passes all the conditions and exits the queue. When defining your executor specification, you can use the syntax `${{ inputs.* }}` and `${{ secrets.* }}` to access the inputs and secrets defined in the stage.

The available executor types are:
- [HTTP Executor](#http-executor)
- [Semaphore Executor](#semaphore-executor)

### HTTP Executor

The HTTP Executor allows you to make HTTP POST requests to external services when a stage is executed.

<b>Example</b>

```yaml
executor:
  type: TYPE_HTTP
  http:
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
  type: TYPE_SEMAPHORE
  semaphore:
    organizationUrl: https://myorg.semaphoreci.com
    apiToken: ${{ secrets.API_TOKEN }}
    projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
    taskId: fd67cfb1-e06c-4896-a517-c648f878330a
    branch: sxmoon
    pipelineFile: .semaphore/pipeline_3.yml
    parameters:
      VERSION_A: ${{ inputs.VERSION_A }}
      VERSION_B: ${{ inputs.VERSION_B }}
```