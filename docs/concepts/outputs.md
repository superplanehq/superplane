Other than inputs, a stage can push outputs from the execution. Those outputs can be used as inputs by another stage when connecting to it.

### Definition

The `outputs` field is how you define the stage outputs:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
spec:
  outputs:
    - name: VERSION
      required: true
      description: ""
    - name: URL
      required: false
      description: ""
```

If a required output is not pushed from the execution, the execution is marked as failed, even if its underlying status is successful.

### Using outputs from one stage as input on another

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-2
spec:
  connections:
    - type: TYPE_STAGE
      name: stage-1
  inputs:
    - name: VERSION
      description: ""
  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: stage-1
              expression: outputs.VERSION
  executor:
    type: semaphore
    integration:
      name: semaphore
    resource:
      type: project
      name: my-semaphore-project
    spec:
      branch: main
      pipelineFile: .semaphore/stage-2.yml
      parameters:
        - name: VERSION
          value: ${{ inputs.VERSION }}
```

### Pushing outputs from execution

The `POST /outputs` endpoint is available for executions to push outputs. If the integration being used supports OpenID Connect ID tokens, you can use them to authenticate the request. For example, when running Semaphore workflows, you can use the `SEMAPHORE_OIDC_TOKEN` and `SEMAPHORE_WORKFLOW_ID` environment variables to authenticate the request:

```bash
curl \
  "$SUPERPLANE_URL/api/v1/outputs" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SEMAPHORE_OIDC_TOKEN" \
  --data @- << EOF
{
  "execution_id": "$SUPERPLANE_STAGE_EXECUTION_ID",
  "external_id": "$SEMAPHORE_WORKFLOW_ID",
  "outputs": {
    "output_1":"hello",
    "output_2":"world"
  }
}
EOF
```

Note: the `SUPERPLANE_STAGE_EXECUTION_ID` value is passed as a parameter by Superplane to the workflow run request.
