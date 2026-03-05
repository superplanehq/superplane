# GCP Cloud Functions • Invoke Function Skill

Use this guidance when planning or configuring `gcp.cloudfunctions.invokeFunction`.

## Purpose

`gcp.cloudfunctions.invokeFunction` invokes a deployed Google Cloud Function and returns its response as workflow output.

## Required Configuration

- `location` (required): GCP region where the function is deployed (e.g. `us-central1`). Selected from the integration's available locations.
- `function` (required): Full resource name of the Cloud Function to invoke. Selected from functions in the chosen location.
- `payload` (optional): JSON object sent as input data to the function.
- `projectId` (optional): Override the GCP project ID from the integration. Leave empty to use the integration's project.

## Planning Rules

When generating workflow operations that include `gcp.cloudfunctions.invokeFunction`:

1. Always set `configuration.location` to a valid GCP region string (e.g. `"us-central1"`).
2. Always set `configuration.function` to the full resource name: `projects/{project}/locations/{location}/functions/{name}`.
3. Only set `payload` when the user wants to pass input data to the function.
4. Only set `projectId` when the user explicitly wants to override the integration's project.
5. `gcp.cloudfunctions.invokeFunction` emits on the `default` channel.
6. The output `data.result` contains the function's response parsed as JSON. If the function returns plain text, use `data.resultRaw` instead.
7. The invocation is synchronous — the workflow step completes when the function returns.

## Output Fields

- `data.functionName`: Full resource name of the invoked function.
- `data.executionId`: Unique execution ID assigned by Cloud Functions.
- `data.result`: Function response parsed as a JSON object (present when response is valid JSON).
- `data.resultRaw`: Raw string response (present when response is not valid JSON).

## Configuration Example

```yaml
location: "us-central1"
function: "projects/my-project/locations/us-central1/functions/my-function"
payload:
  userId: "123"
  action: "process"
```

## Accessing Output in Downstream Nodes

- Function result: `{{ $["Invoke Function"].data.result }}`
- Execution ID: `{{ $["Invoke Function"].data.executionId }}`
- Raw response: `{{ $["Invoke Function"].data.resultRaw }}`

## Mistakes To Avoid

- Using a short function name instead of the full resource path (`projects/{project}/locations/{location}/functions/{name}`).
- Setting `location` and `function` inconsistently (function must be in the specified location).
- Expecting `data.result` when the function returns plain text — check `data.resultRaw` in that case.
- Connecting from this component with a channel other than `default`.
