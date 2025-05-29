This is what we had the last time we talked:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
spec:
  conditions: []

  connections:
    - type: TYPE_EVENT_SOURCE
      name: docs
      inputs:
        - name: DOCS_VERSION
          valueFrom:
            eventData:
              expression: ref

    - type: TYPE_EVENT_SOURCE
      name: terraform
      inputs:
        - name: TERRAFORM_VERSION
          valueFrom:
            eventData:
              expression: ref

  secrets:
    - name: semaphore
      inputs:
        - name: API_TOKEN
          valueFrom:
            key: API_TOKEN

  runTemplate:
    type: TYPE_SEMAPHORE

    inputs:
      - name: DOCS_VERSION
        type: string
        required: true
      - name: TERRAFORM_VERSION
        type: string
        required: true
      - name: API_TOKEN
        type: string
        required: true

    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: 093f9ecd-ba40-420d-a085-77f2fbf953c1
      taskId: d76b6eb6-b1cc-40dd-bbf5-0b09980e184e
      branch: main
      pipelineFile: .semaphore/stage-1.yml
      parameters:
        DOCS_VERSION: ${{ inputs.DOCS_VERSION }}
        TERRAFORM_VERSION: ${{ inputs.TERRAFORM_VERSION }}
```

Reasons I don't like this:
- Having API_TOKEN as an input seems wrong. Inputs are the things that I see in the event in my queue. I don't need to see that. I only need to see the DOCS_VERSION and TERRAFORM_VERSION ones.
- the input assignments are scattered around multiple places, which leads to the `valueFrom` field having multiple structures.
- runTemplate name sucks

### Iteration 1

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: stage-1
spec:
  connections:
    - type: TYPE_EVENT_SOURCE
      name: docs
      inputs:
        - name: DOCS_VERSION
          valueFrom:
            eventData:
              expression: ref

    - type: TYPE_EVENT_SOURCE
      name: terraform
      inputs:
        - name: TERRAFORM_VERSION
          valueFrom:
            eventData:
              expression: ref

  secrets:
    - name: API_TOKEN
      valueFrom:
        secret:
          name: semaphore
          key: API_TOKEN

  executor:
    type: TYPE_SEMAPHORE

    inputs:
      - name: DOCS_VERSION
        required: true
      - name: TERRAFORM_VERSION
        required: true

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

Changes:
- Since API_TOKEN is not really helpful for me, I move it to a different "namespace" – "${{ secrets.* }}"  – other than "${{ inputs.* }}"
- Similar to "${{ inputs.* }}", you have to define your "${{ secrets.* }}" to use it in the run template
- Input assignments are not scattered around multiple places anymore, since they are only on the connections.
- `runTemplate` was renamed to `executor`

But there's one problem here still:
- I should be able to define inputs manually – not from connections – since a stage can have 0 connections but still be able to run things.

### Iteration 2

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
      value: v1.0
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

    inputs:
      - name: DOCS_VERSION
        required: true
      - name: TERRAFORM_VERSION
        required: true

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

Changes:
- Inputs were moved out of the connections to a top-level `inputs` field.

Problems here still:
- I still don't like the `inputs` field inside the `executor` field. It makes it seem like these are the inputs for `SEMAPHORE` executor, which they are not.

### Iteration 3

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
      valueFrom:
        required: true
        eventData:
          connection: docs
          expression: ref

    - name: TERRAFORM_VERSION
      valueFrom:
        required: true
        eventData:
          connection: docs
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
      organizationUrl: https://lucaspin.sxmoon.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
      taskId: fd67cfb1-e06c-4896-a517-c648f878330a
      branch: sxmoon
      pipelineFile: .semaphore/pipeline_3.yml
      parameters:
        - name: DOCS_VERSION
          value: ${{ inputs.DOCS_VERSION }}
        - name: TERRAFORM_VERSION
          value: ${{ inputs.TERRAFORM_VERSION }}
```

Changes:
- We combine both `inputs` into the top-level stage `inputs` field.

Problems here:
- Not so clear (and configurable) how inputs are configured per source, that is, when new docs event comes in, the TERRAFORM_VERSION wouldn't come from `eventData` itself, but from some other sources - the last input set applied which has this input defined.

### Iteration 4

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
      required: true
      description: ""
    - name: TERRAFORM_VERSION
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
      organizationUrl: https://lucaspin.sxmoon.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
      taskId: fd67cfb1-e06c-4896-a517-c648f878330a
      branch: sxmoon
      pipelineFile: .semaphore/pipeline_3.yml
      parameters:
        - name: DOCS_VERSION
          value: ${{ inputs.DOCS_VERSION }}
        - name: TERRAFORM_VERSION
          value: ${{ inputs.TERRAFORM_VERSION }}
```
