Inputs let you define what your stage expects to receive from its connections, and how you use those values to compute the executor specification. Here's an example:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
  canvasId: a88894a7-8043-4e55-a9f1-e2ca85887a42
spec:

  connections:
    - type: TYPE_EVENT_SOURCE
      name: docs
    - type: TYPE_EVENT_SOURCE
      name: terraform

  inputs:
    - name: DOCS_VERSION
      description: ""
    - name: TERRAFORM_VERSION
      description: ""

  inputMappings:
    - when:
        triggeredBy:
          connection: docs
      values:
        - name: DOCS_VERSION
          valueFrom:
            eventData:
              connection: docs
              expression: ref
        - name: TERRAFORM_VERSION
          valueFrom:
            lastExecution:
              result: [RESULT_PASSED]

    - when:
        triggeredBy:
          connection: terraform
      values:
        - name: DOCS_VERSION
          valueFrom:
            lastExecution:
              result: [RESULT_FAILED, RESULT_PASSED]

        - name: TERRAFORM_VERSION
          valueFrom:
            eventData:
              connection: terraform
              expression: ref

  executor:
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: XXXX
      projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
      taskId: fd67cfb1-e06c-4896-a517-c648f878330a
      branch: main
      pipelineFile: .semaphore/pipeline_3.yml
      parameters:
        - name: DOCS_VERSION
          value: ${{ inputs.DOCS_VERSION }}
        - name: TERRAFORM_VERSION
          value: ${{ inputs.TERRAFORM_VERSION }}
```

- The `inputs` field are the input definitions.
- The `inputMappings` field allow you to assign values to your inputs based on the connection that is currently hitting your stage. When an event comes from a connection, the input mapping for that connection is chosen and applied to compute the input set that will be used when the execution starts.
- The values that come from the `inputMappings` are made available in the executor spec field through the use of the `${{ inputs.* }}` syntax.

### Single connection

When the stage has a single connection, you can omit the `inputMapping.when` field:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
  canvasId: a88894a7-8043-4e55-a9f1-e2ca85887a42
spec:

  connections:
    - type: TYPE_EVENT_SOURCE
      name: source-1

  inputs:
    - name: VERSION
      description: ""

  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: source-1
              expression: ref

  executor:
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: XXXX
      projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
      taskId: fd67cfb1-e06c-4896-a517-c648f878330a
      branch: main
      pipelineFile: .semaphore/pipeline_3.yml
      parameters:
        - name: DOCS_VERSION
          value: ${{ inputs.DOCS_VERSION }}
        - name: TERRAFORM_VERSION
          value: ${{ inputs.TERRAFORM_VERSION }}
```