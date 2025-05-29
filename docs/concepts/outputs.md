Other than inputs, a stage can have outputs as well, which can be used as inputs by another stage when connecting to it:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
spec:

  connections:
    - type: TYPE_EVENT_SOURCE
      name: docs
    - type: TYPE_EVENT_SOURCE
      name: terraform

  inputs:
    - name: DOCS_VERSION
      required: true
      description: ""
    - name: TERRAFORM_VERSION
      required: true
      description: ""

  outputs:
    - name: VERSION
      required: true
      description: ""
    - name: URL
      required: true
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
              inputName: TERRAFORM_VERSION
              result: [RESULT_FAILED, RESULT_PASSED]

    - when:
        triggeredBy:
          connection: terraform
      values:
        - name: DOCS_VERSION
          valueFrom:
            lastExecution:
              inputName: DOCS_VERSION
              result: [RESULT_FAILED, RESULT_PASSED]

        - name: TERRAFORM_VERSION
          valueFrom:
            eventData:
              connection: terraform
              expression: ref

  secrets:
    - name: API_TOKEN
      valueFrom:
        secret:
          name: semaphore
          key: API_TOKEN

  executor:
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: 093f9ecd-ba40-420d-a085-77f2fbf953c1
      taskId: d76b6eb6-b1cc-40dd-bbf5-0b09980e184e
      branch: main
      pipelineFile: .semaphore/stage-1.yml
      parameters:
        - name: DOCS_VERSION
          value: ${{ inputs.DOCS_VERSION }}
        - name: TERRAFORM_VERSION
          value: ${{ inputs.TERRAFORM_VERSION }}
```

Then, I can use them as inputs in another stage:

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
      required: true
      description: ""

  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: stage-1
              expression: outputs.VERSION

  secrets:
    - name: API_TOKEN
      valueFrom:
        secret:
          name: semaphore
          key: API_TOKEN

  executor:
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: 093f9ecd-ba40-420d-a085-77f2fbf953c1
      taskId: d76b6eb6-b1cc-40dd-bbf5-0b09980e184e
      branch: main
      pipelineFile: .semaphore/stage-2.yml
      parameters:
        - name: VERSION
          value: ${{ inputs.VERSION }}
```
