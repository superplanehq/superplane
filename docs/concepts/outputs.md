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
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: XXXX
      projectId: 093f9ecd-ba40-420d-a085-77f2fbf953c1
      taskId: d76b6eb6-b1cc-40dd-bbf5-0b09980e184e
      branch: main
      pipelineFile: .semaphore/stage-2.yml
      parameters:
        - name: VERSION
          value: ${{ inputs.VERSION }}
```

### Pushing outputs from execution

The `POST /outputs` is available for executions to push outputs.

```
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SEMAPHORE_STAGE_EXECUTION_TOKEN" \
  --data "{\"execution_id\":$SEMAPHORE_STAGE_EXECUTION_ID,\"outputs\":{\"MY_OUTPUT\":\"hello\"}}" \
  "$SUPERPLANE_URL/api/v1/outputs"
```

The `SEMAPHORE_STAGE_EXECUTION_ID` and `SEMAPHORE_STAGE_EXECUTION_TOKEN` values are passed by Superplane to the executor. For example, in the case of the Semaphore executor type, those values are passed in the `parameters` field in the Semaphore Task API.
