# Events and Data Flow

## Table of Contents

- [Overview](#overview)
- [Events](#events)
  - [Event Structure](#event-structure)
  - [Event Sources](#event-sources)
  - [Event Processing](#event-processing)
- [Inputs](#inputs)
  - [Input Definitions](#input-definitions)
  - [Input Value Sources](#input-value-sources)
  - [Input Mappings](#input-mappings)
  - [Single Connection Stages](#single-connection-stages)
  - [Forwarding Inputs Between Stages](#forwarding-inputs-between-stages)
- [Outputs](#outputs)
  - [Output Definitions](#output-definitions)
  - [Using Outputs as Inputs](#using-outputs-as-inputs)
  - [Pushing Outputs from Executions](#pushing-outputs-from-executions)
- [Data Flow Patterns](#data-flow-patterns)

---

## Overview

Events are the core mechanism that drives workflow execution in Superplane. Every component interaction, data transfer, and workflow progression happens through events. Data flows through workflows via inputs and outputs, allowing components to pass information and coordinate their operations.

**Key concepts:**
- **Events** carry data between components and trigger workflow execution
- **Inputs** define what data a component expects to receive
- **Outputs** define what data a component produces after execution
- **Input mappings** extract and transform data from events into component inputs

---

## Events

Events are JSON messages that carry data and trigger component execution. They serve as the communication mechanism between all components in a workflow.

### Event Structure

Events contain three main parts:

**Payload:** The actual data being passed
```json
{
  "ref": "refs/heads/main",
  "commit_sha": "abc123def456",
  "repository": {
    "name": "my-app",
    "url": "https://github.com/org/my-app"
  }
}
```

**Headers:** Metadata about the event
```json
{
  "X-GitHub-Event": "push",
  "X-GitHub-Delivery": "12345-67890-abcdef",
  "Content-Type": "application/json"
}
```

**Context:** Superplane-specific information
```json
{
  "source": "github-webhook",
  "timestamp": "2024-01-15T10:30:00Z",
  "execution_id": "exec_abc123"
}
```

### Event Sources

Events originate from multiple sources:

**External triggers:** Webhooks from integrated tools or custom systems
```json
{
  "alert": {
    "severity": "critical",
    "service": "payment-api",
    "region": "us-east-1",
    "error_rate": 15.7,
    "triggered_at": "2025-01-15T14:30:00Z"
  },
  "runbook": "https://wiki.company.com/incidents/payment-api",
  "on_call_engineer": "alice@company.com"
}
```

**Component completions:** Events emitted when stages finish execution
```json
{
  "execution": {
    "created_at": "2025-09-02T11:14:28.94643Z",
    "finished_at": "2025-09-02T22:33:00.237551Z",
    "id": "98bbeffa-08d1-4075-8e3d-60c214f2df04",
    "result": "failed",
    "started_at": "2025-09-02T21:17:41.290889Z"
  },
  "stage": {
    "id": "b57cbec8-e36a-4d49-8f58-10a0fcaba888"
  },
  "type": "execution_finished",
  "inputs": {
    "VERSION": "v1.2.3",
    "ENVIRONMENT": "staging"
  },
  "outputs": {
    "BUILD_URL": "https://builds.example.com/123",
    "IMAGE_TAG": "my-app:v1.2.3"
  }
}
```

**Manual triggers:** Events created through user actions or API calls

### Event Processing

When a component receives an event:
1. **Filter evaluation** - Event is checked against component filters
2. **Input mapping** - Event data is extracted and mapped to component inputs
3. **Queue addition** - Qualifying events are added to the execution queue
4. **Execution** - Queue items are processed based on conditions

---

## Inputs

Inputs define what data your components expect to receive and how to use that data during execution.

### Input Definitions

Define inputs in your component specification:

```yaml
spec:
  inputs:
    - name: VERSION
      description: "Application version to deploy"
    - name: ENVIRONMENT
      description: "Target environment (staging/production)"
    - name: IMAGE_TAG
      description: "Docker image tag"
```

**Input properties:**
- `name` - Unique identifier for the input
- `description` - Human-readable explanation of the input's purpose

### Input Value Sources

Inputs can get their values from multiple sources:

**Event data parsing:** Extract values from incoming events
```yaml
valueFrom:
  eventData:
    connection: github-repo
    expression: $.head_commit.id[0:7]  # First 7 chars of commit ID
```

**Static values:** Fixed values set during configuration
```yaml
value: "production"
```

**Previous execution results:** Values from the last successful execution
```yaml
valueFrom:
  lastExecution:
    result: [RESULT_PASSED]
```

**Canvas secrets:** Encrypted values stored at Canvas level
```yaml
valueFrom:
  secret:
    name: deployment-credentials
    key: api-token
```

### Input Mappings

Input mappings define how to populate inputs based on which connection triggered the component:

```yaml
spec:
  inputMappings:
    - when:
        triggeredBy:
          connection: github-repo
      values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: github-repo
              expression: $.head_commit.id[0:7]
        - name: ENVIRONMENT
          value: "staging"
    
    - when:
        triggeredBy:
          connection: approval-stage
      values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: approval-stage
              expression: $.inputs.VERSION
        - name: ENVIRONMENT
          value: "production"
```

**Expression syntax examples:**
- `$.ref` - Extract the `ref` field from event root
- `$.head_commit.id[0:7]` - Extract first 7 characters of commit ID
- `$.outputs.BUILD_URL` - Extract output from previous stage execution
- `$.commits[0].message` - First commit message from an array
- `$.files[*].name` - All file names from an array
- `$.tags[-1]` - Last item in the tags array

### Single Connection Stages

When a stage has only one connection, you can omit the `when` condition:

```yaml
spec:
  connections:
    - type: TYPE_EVENT_SOURCE
      name: github-repo

  inputs:
    - name: VERSION
    - name: BRANCH

  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: github-repo
              expression: $.head_commit.id[0:7]
        - name: BRANCH
          valueFrom:
            eventData:
              connection: github-repo
              expression: $.ref
```

### Forwarding Inputs Between Stages

You can pass inputs from one stage to another since inputs are included in completion events:

```yaml
# First stage
kind: Stage
metadata:
  name: build-stage
spec:
  inputs:
    - name: version
  outputs:
    - name: image_url
      required: true
  inputMappings:
    - values:
        - name: version
          valueFrom:
            eventData:
              expression: $.head_commit.id

---
# Second stage  
kind: Stage
metadata:
  name: deploy-stage
spec:
  connections:
    - name: build-stage
      type: TYPE_STAGE
  inputs:
    - name: version      # Forward input from previous stage
    - name: image_url    # Use output from previous stage
  inputMappings:
    - values:
        - name: version
          valueFrom:
            eventData:
              connection: build-stage
              expression: $.inputs.version  # Access forwarded input
        - name: image_url
          valueFrom:
            eventData:
              connection: build-stage
              expression: $.outputs.image_url  # Access stage output
```

---

## Outputs

Outputs define what data your components produce after execution and make it available to downstream components.

### Output Definitions

Define expected outputs in your component specification:

```yaml
spec:
  outputs:
    - name: BUILD_URL
      required: true
      description: "URL to the build artifacts"
    - name: TEST_RESULTS
      required: false
      description: "Test execution summary"
    - name: IMAGE_TAG
      required: true
      description: "Docker image tag created"
```

**Output properties:**
- `name` - Unique identifier for the output
- `required` - Whether this output must be provided for execution to succeed
- `description` - Human-readable explanation

**Required outputs:** If a required output is not provided, the execution is marked as failed even if the underlying operation succeeded.

### Using Outputs as Inputs

When a stage completes, it emits an event containing its outputs. Downstream stages can use these outputs as their inputs:

```yaml
# Consuming stage
spec:
  connections:
    - type: TYPE_STAGE
      name: build-stage
  
  inputs:
    - name: BUILD_URL
    - name: IMAGE_TAG
  
  inputMappings:
    - values:
        - name: BUILD_URL
          valueFrom:
            eventData:
              connection: build-stage
              expression: $.outputs.BUILD_URL
        - name: IMAGE_TAG
          valueFrom:
            eventData:
              connection: build-stage
              expression: $.outputs.IMAGE_TAG
```

### Pushing Outputs from Executions

Executors can push outputs back to Superplane using the `/outputs` API endpoint. This is typically done from within the executing system (CI/CD pipeline, script, etc.).

**Using OIDC tokens (Semaphore example):**
```bash
curl \
  "https://app.superplane.com/api/v1/outputs" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SEMAPHORE_OIDC_TOKEN" \
  --data '{
    "execution_id": "'$SUPERPLANE_STAGE_EXECUTION_ID'",
    "external_id": "'$SEMAPHORE_WORKFLOW_ID'", 
    "outputs": {
      "BUILD_URL": "https://builds.example.com/123",
      "TEST_COVERAGE": "87%",
      "IMAGE_TAG": "my-app:v1.2.3"
    }
  }'
```

**Using GitHub Actions OIDC:**
```bash
curl \
  "https://app.superplane.com/api/v1/outputs" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GITHUB_ID_TOKEN" \
  --data '{
    "execution_id": "'$SUPERPLANE_EXECUTION_ID'",
    "external_id": "'$GITHUB_RUN_ID'",
    "outputs": {
      "DEPLOY_URL": "https://app-staging.example.com",
      "HEALTH_CHECK": "passing"
    }
  }'
```

**Parameters:**
- `execution_id` - Superplane execution identifier (passed as parameter to executor)
- `external_id` - External system's run/workflow identifier
- `outputs` - Key-value pairs of output data

---

## Data Flow Patterns

### Conditional Routing
Use output values to determine workflow paths:
```yaml
# Stage only executes if previous stage produced specific output
connections:
  - type: TYPE_STAGE
    name: security-scan
    filters:
      - type: FILTER_TYPE_DATA
        data:
          expression: $.outputs.SECURITY_STATUS == "passed"
```