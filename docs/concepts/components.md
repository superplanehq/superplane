# Components

## Table of Contents

- [Overview](#overview)
- [Event Sources](#event-sources)
  - [Types of Event Sources](#types-of-event-sources)
  - [Custom Webhook Usage](#custom-webhook-usage)
- [Stages](#stages)
  - [Stage Configuration](#stage-configuration)
  - [Stage Execution Model](#stage-execution-model)
  - [Conditions](#conditions)
- [Connection Groups](#connection-groups)
  - [Configuration](#configuration)
  - [Using Connection Groups](#using-connection-groups)
  - [Event Structure](#event-structure)
  - [Input Mapping from Connection Groups](#input-mapping-from-connection-groups)
- [Executors](#executors)
  - [HTTP Executor](#http-executor)
  - [Semaphore Executor](#semaphore-executor)
  - [GitHub Executor](#github-executor)
- [Event Filtering](#event-filtering)

---

## Overview

Components are the building blocks of workflows in SuperPlane. They operate on an event-driven model where components listen for events, process them, and emit new events to trigger downstream components.

**Component types:**
- **Event sources** - Listen to external systems and emit events to the Canvas
- **Stages** - Listen to events, perform actions on external resources, and emit completion events
- **Connection groups** - Coordinate events from multiple sources before emitting

All workflows operate through event passing. Components receive events containing payload data, process them based on their configuration, and emit new events to continue the workflow chain.

---

## Event Sources

Event sources listen to external systems (integrations, webhooks) and emit events when something happens. They serve as the entry points for workflows.

### Types of Event Sources

**Integration-based sources** automatically configure webhooks and monitoring:
```yaml
apiVersion: v1
kind: EventSource
metadata:
  name: github-repo
  canvasId: canvas-123
spec:
  integration:
    name: github-integration
  resource:
    type: repository
    name: my-repository
  events:
    - type: push
      filters:
        - type: FILTER_TYPE_DATA
          data:
            expression: $.ref=="$.refs/heads/main"
      filterOperator: FILTER_OPERATOR_AND
```

**Custom webhook sources** give you a URL to send events to manually:
```yaml
apiVersion: v1
kind: EventSource
metadata:
  name: custom-webhook
  canvasId: canvas-123
spec: {}
```

### Custom Webhook Usage

When you create a custom webhook event source, you push events to SuperPlane using the provided URL and signature key. The event should be a JSON object with HMAC-SHA256 signature:

```bash
export SOURCE_ID="<YOUR_SOURCE_ID>"
export SOURCE_KEY="<YOUR_SOURCE_KEY>"
export EVENT="{\"version\":\"v1.0\",\"app\":\"core\"}"
export SIGNATURE=$(echo -n "$EVENT" | openssl dgst -sha256 -hmac "$SOURCE_KEY" | awk '{print $2}')

curl -X POST \
  -H "X-Signature-256: sha256=$SIGNATURE" \
  -H "Content-Type: application/json" \
  --data "$EVENT" \
  https://app.superplane.com/api/v1/sources/$SOURCE_ID
```

---

## Stages

Stages are the primary execution components in workflows. They listen for events, manage execution queues, and coordinate with external systems through their configured executors.

### Stage Configuration

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: build-and-test
  canvasId: canvas-123
spec:
  # Event sources this stage listens to
  connections:
    - type: TYPE_EVENT_SOURCE
      name: github-repo
    - type: TYPE_STAGE
      name: previous-stage

  # Filter which events trigger execution
  filters:
    - type: FILTER_TYPE_DATA
      data:
        expression: "$.ref == 'refs/heads/main'"

  # Define stage inputs
  inputs:
    - name: VERSION
      description: "Version to build"

  # Map event data to inputs
  inputMappings:
    - when:
        triggeredBy:
          connection: github-repo
      values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: github-repo
              expression: "$.commits[0].id"

  # Define expected outputs
  outputs:
    - name: BUILD_URL
      required: true

  # Configure what action to perform
  executor:
    type: semaphore
    integration:
      name: semaphore
    resource:
      type: project
      name: my-project
    spec:
      branch: main
      pipelineFile: .semaphore/build.yml
      parameters:
        VERSION: ${{ inputs.VERSION }}
```

### Stage Execution Model

When a stage receives a qualifying event:
1. The event is processed and mapped to stage inputs
2. The event is added to the stage's execution queue
3. Queue conditions are evaluated (approvals, time windows)
4. The executor runs the external operation using the inputs
5. Upon completion, the stage emits a new event with execution status and outputs
6. Downstream stages can listen for these result events

### Conditions

Stages can have conditions that control when queue items execute:

```yaml
spec:
  conditions:
    - type: CONDITION_TYPE_APPROVAL
      approval:
        count: 1  # Number of approvals required
```

**Available conditions:**
- **Manual approval** - Require human approval before execution
- **Time windows** - Restrict execution to specific schedules

---

## Connection Groups

Connection groups allow you to coordinate events from multiple connections. Instead of every single event from every connection generating a new execution, connection groups wait for events from all specified connections with matching grouping fields before emitting a single coordinated event.

### Configuration

```yaml
apiVersion: v1
kind: ConnectionGroup
metadata:
  name: preprod
spec:
  # Define connections to coordinate
  connections:
    - type: TYPE_STAGE
      name: preprod1
    - type: TYPE_STAGE
      name: preprod2

  # Fields used to group events together
  groupBy:
    fields:
      - name: version
        expression: $.outputs.version

  # How long to wait for all connections (seconds)
  timeout: 86400

  # What to do when timeout is reached
  timeoutBehavior: TIMEOUT_BEHAVIOR_DROP  # or TIMEOUT_BEHAVIOR_EMIT
```

### Using Connection Groups

Use a connection group as a connection in another stage:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: prod-deploy
spec:
  connections:
    - type: TYPE_CONNECTION_GROUP
      name: preprod
```

### Event Structure

When all connections send events with matching grouping fields, the connection group emits:

```json
{
  "fields": {
    "version": "v1.2.3"
  },
  "events": {
    "preprod1": { "outputs": { "version": "v1.2.3" } },
    "preprod2": { "outputs": { "version": "v1.2.3" } }
  }
}
```

If timeout is reached with missing connections:

```json
{
  "fields": {
    "version": "v1.2.3"
  },
  "events": {
    "preprod1": { "outputs": { "version": "v1.2.3" } }
  },
  "missing": ["preprod2"]
}
```

### Input Mapping from Connection Groups

When using connection group events as stage inputs:

```yaml
spec:
  inputs:
    - name: VERSION
  
  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: preprod
              expression: $.fields.version
```

---

## Executors

Executors perform the actual work when stages execute. They integrate with external systems to run operations like triggering builds, making API calls, or deploying applications.

### HTTP Executor

Make REST API calls to external services:

```yaml
executor:
  type: http
  spec:
    url: https://api.example.com/deploy
    payload:
      version: ${{ inputs.VERSION }}
      environment: production
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
    responsePolicy:
      statusCodes: [200, 201, 202]
```

**Configuration:**
- `url` - The endpoint to call
- `payload` - Request body data (optional)
- `headers` - HTTP headers (optional)
- `responsePolicy` - Define successful response criteria

**Automatic inputs:** `stageId`, `executionId`

### Semaphore Executor

Trigger Semaphore CI/CD pipelines:

```yaml
executor:
  type: semaphore
  integration:
    name: semaphore-integration
  resource:
    type: project
    name: my-semaphore-project
  spec:
    branch: main
    pipelineFile: .semaphore/deploy.yml
    parameters:
      VERSION: ${{ inputs.VERSION }}
      ENVIRONMENT: ${{ inputs.ENVIRONMENT }}
```

**Configuration:**
- `branch` - Git branch to run against
- `pipelineFile` - Semaphore pipeline file to execute
- `parameters` - Parameters passed to the pipeline
- `task` - (optional) Specific task to run instead of full workflow

**Automatic parameters:** `SUPERPLANE_STAGE_ID`, `SUPERPLANE_STAGE_EXECUTION_ID`

### GitHub Executor

Trigger GitHub Actions workflows via workflow dispatch:

```yaml
executor:
  type: github
  integration:
    name: github-integration
  resource:
    type: repository
    name: my-repository
  spec:
    workflow: .github/workflows/deploy.yml
    ref: main
    inputs:
      environment: ${{ inputs.ENVIRONMENT }}
```

**Configuration:**
- `workflow` - Workflow file name or workflow name
- `ref` - Git branch, tag, or commit SHA
- `inputs` - Input parameters for the workflow

**Automatic inputs:** `superplane_execution_id`

#### GitHub Workflow Requirements

Your GitHub Actions workflow must:

1. **Accept workflow_dispatch events:**
```yaml
on:
  workflow_dispatch:
    inputs:
      superplane_execution_id:
        required: true
        type: string
      environment:
        required: false
        type: string
```

2. **Use the execution ID in run name:**
```yaml
run-name: "Deploy - ${{ inputs.superplane_execution_id }}"
```

3. **Configure OIDC permissions for outputs:**
```yaml
permissions:
  id-token: write
```

4. **Push outputs back to SuperPlane:**
```yaml
- name: Push outputs
  run: |
    curl -s \
      $SUPERPLANE_URL/api/v1/outputs \
      -X POST \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $GITHUB_ID_TOKEN" \
      --data "{\"execution_id\":\"$EXECUTION_ID\",\"external_id\":\"$GITHUB_RUN_ID\",\"outputs\":{\"deploy_url\":\"https://app.example.com\"}}"
  env:
    EXECUTION_ID: ${{ inputs.superplane_execution_id }}
    GITHUB_ID_TOKEN: ${{ steps.get-token.outputs.token }}
```

---

## Event Filtering

Components can filter incoming events using configurable rules:

### Data Filters
Filter based on event payload content:
```yaml
filters:
  - type: FILTER_TYPE_DATA
    data:
      expression: "$.ref == 'refs/heads/main'"
```

### Header Filters  
Filter based on event headers:
```yaml
filters:
  - type: FILTER_TYPE_HEADER
    header:
      expression: "headers['X-GitHub-Event'] == 'push'"
```

### Combining Filters
Use `filterOperator` to combine multiple filters:
```yaml
filterOperator: FILTER_OPERATOR_AND  # or FILTER_OPERATOR_OR
filters:
  - type: FILTER_TYPE_DATA
    data:
      expression: "$.ref == 'refs/heads/main'"
  - type: FILTER_TYPE_HEADER
    header:
      expression: "headers['X-GitHub-Event'] == 'push'"
```

Events that don't match filter criteria are discarded. Matching events proceed to input mapping and execution.
