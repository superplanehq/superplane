# Blueprints and Workflows

This document describes the new architecture introduced in the blueprints-and-workflows branch, which fundamentally changes how Superplane handles event processing through the introduction of **Components**, **Blueprints**, and **Workflows**.

## Overview

The new architecture replaces the previous canvas-based execution model with a more flexible, composable system that supports:

- **Reusable components** with full lifecycle control
- **Blueprints** as composable, reusable subgraphs
- **Workflows** as top-level orchestration plane
- **Unified execution model** for both simple and complex flows

## Core Concepts

### Components

Components are the fundamental building blocks of workflows and blueprints. They replace the previous Executor interface with a more powerful abstraction that provides full control over execution lifecycle.

#### Component Interface

Located in `pkg/components/component.go:5`, the Component interface defines:

```go
type Component interface {
    Name() string              // Unique identifier
    Label() string             // Display name
    Description() string       // Documentation

    OutputBranches(configuration any) []OutputBranch
    Configuration() []ConfigurationField

    Execute(ctx ExecutionContext) error

    Actions() []Action
    HandleAction(ctx ActionContext) error
}
```

#### Key Differences from Executor Interface

The previous Executor interface (in `pkg/executors/executor.go:7`) was simple and stateless:

```go
type Executor interface {
    Validate(context.Context, []byte) error
    Execute([]byte, ExecutionParameters) (Response, error)
}
```

**Components provide significantly more capabilities:**

1. **Execution Lifecycle Control**: Components have full control over their execution state through `ExecutionStateContext`:
   - Can mark executions as passed or failed
   - Control when execution completes
   - Support for asynchronous patterns

2. **Metadata Management**: Components can store and retrieve execution-specific metadata via `MetadataContext`:
   - Persist state across action invocations
   - Track component-specific information per execution

3. **Custom Actions**: Components can expose actions that can be invoked on specific executions:
   - Enables interactive workflows (e.g., approval flows)
   - Support for async operations
   - Parameters defined via ConfigurationField schema

4. **Dynamic Output Branches**: Components can define multiple output paths:
   - Configuration-dependent branches (e.g., switch/if components)
   - Enables complex routing logic

5. **Rich Configuration Schema**: Components define their configuration declaratively:
   - Type-safe field definitions
   - Support for complex types (lists, objects, nested schemas)
   - Validation and UI generation

#### ExecutionContext

Components receive an `ExecutionContext` (defined at `pkg/components/component.go:70`):

```go
type ExecutionContext struct {
    Data                  any                    // Input data
    Configuration         any                    // Component configuration
    MetadataContext       MetadataContext        // Store/retrieve metadata
    ExecutionStateContext ExecutionStateContext  // Control execution lifecycle
}
```

The `ExecutionStateContext` interface provides:

```go
type ExecutionStateContext interface {
    Pass(outputs map[string][]any) error  // Complete successfully with outputs
    Fail(reason, message string) error     // Mark as failed
}
```

#### Built-in Components

Several components are implemented in `pkg/components/`:

1. **HTTP Component** (`pkg/components/http/http.go:25`): Makes HTTP requests
   - Configurable method, URL, headers
   - Returns response with status, headers, and body
   - Synchronous execution

2. **Approval Component** (`pkg/components/approval/approval.go:60`): Collects approvals
   - Configurable approval count
   - Stores approval records in metadata
   - Exposes `approve` and `reject` actions
   - Asynchronous execution pattern

3. **If Component** (`pkg/components/if/if.go:16`): Conditional routing
   - Evaluates boolean expressions
   - Two output branches: `true` and `false`
   - Uses expr-lang for expression evaluation

4. **Switch Component** (`pkg/components/switch/switch.go:20`): Multi-way routing
   - Multiple configurable branches with expressions
   - Dynamic output branches based on configuration
   - Can route to multiple branches simultaneously

5. **Filter Component** (`pkg/components/filter/filter.go`): Filters events based on expressions

### Blueprints

Blueprints are reusable, composable subgraphs that can be embedded in workflows or other blueprints. They are defined in the `blueprints` table (see `db/migrations/20251006150635_add-blueprints-table.up.sql:3`).

#### Blueprint Model

Located in `pkg/models/blueprint.go:21`:

```go
type Blueprint struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    Name           string
    Description    string
    CreatedAt      *time.Time
    UpdatedAt      *time.Time
    Nodes          datatypes.JSONSlice[Node]
    Edges          datatypes.JSONSlice[Edge]
    Configuration  datatypes.JSONSlice[components.ConfigurationField]  // Exposed parameters
    OutputBranches datatypes.JSONSlice[components.OutputBranch]         // Exit points
}
```

#### Blueprint Characteristics

- **Parameterized**: Blueprints expose configuration fields that can be bound when used in workflows
- **Multiple Outputs**: Can define multiple output branches for different exit paths
- **Nested**: Blueprints can contain other blueprint nodes
- **Isolated**: Each blueprint execution maintains its own scope

#### Configuration Resolution

When a blueprint is used as a node in a workflow, its internal node configurations can reference the parent configuration. The `ConfigurationBuilder` (in `pkg/components/configuration_builder.go`) resolves these references.

For example, if a blueprint exposes a `url` parameter, internal nodes can use expressions to access it from the parent blueprint node's configuration.

### Workflows

Workflows are top-level plane for orchestrating event processing. They are defined in the `workflows` table (see `db/migrations/20251006150645_add-workflows-table.up.sql:7`).

#### Workflow Model

Located in `pkg/models/workflow.go:12`:

```go
type Workflow struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    Name           string
    Description    string
    CreatedAt      *time.Time
    UpdatedAt      *time.Time
    Nodes          datatypes.JSONSlice[Node]
    Edges          datatypes.JSONSlice[Edge]
}
```

#### Node Structure

Both workflows and blueprints contain nodes, defined in `pkg/models/blueprint.go:58`:

```go
type Node struct {
    ID            string
    Name          string
    RefType       string         // "component" or "blueprint"
    Ref           NodeRef
    Configuration map[string]any
}

type NodeRef struct {
    Component *ComponentRef  // Reference to a component by name
    Blueprint *BlueprintRef  // Reference to a blueprint by ID
}
```

#### Edge Structure

Edges connect nodes and define data flow:

```go
type Edge struct {
    SourceID   string  // Source node ID
    TargetType string  // "node" or "output_branch"
    TargetID   string  // Target node ID or output branch name
    Branch     string  // Output branch name from source
}
```

**Edge Types:**
- **Node-to-Node**: `TargetType = "node"` - connects to next node in flow
- **Node-to-OutputBranch**: `TargetType = "output_branch"` - exits blueprint via named output

## Execution Model

### Execution Records

All node executions are tracked in the `workflow_node_executions` table. The model is defined in `pkg/models/workflow_node_execution.go:25`:

```go
type WorkflowNodeExecution struct {
    ID         uuid.UUID
    WorkflowID uuid.UUID
    NodeID     string

    // Root event (shared by all executions in this workflow run)
    RootEventID uuid.UUID

    // Sequential flow - references to previous execution
    PreviousExecutionID  *uuid.UUID
    PreviousOutputBranch *string
    PreviousOutputIndex  *int

    // Blueprint hierarchy - parent blueprint node execution
    ParentExecutionID *uuid.UUID

    // Blueprint context
    BlueprintID *uuid.UUID

    // State machine
    State         string  // pending, started, routing, finished
    Result        string  // passed, failed
    ResultReason  string
    ResultMessage string

    // Data
    Outputs       datatypes.JSONType[map[string][]any]

    // Component metadata and configuration snapshot
    Metadata      datatypes.JSONType[map[string]any]
    Configuration datatypes.JSONType[map[string]any]

    CreatedAt *time.Time
    UpdatedAt *time.Time
}
```

### Execution States

Defined in `pkg/models/workflow_node_execution.go:13`:

1. **pending**: Execution created, waiting to be processed
2. **started**: Component is actively executing
3. **routing**: Execution completed, waiting to route to next nodes
4. **finished**: Execution and all routing complete

### Execution Flow

The execution lifecycle is managed by two workers:

#### 1. PendingNodeExecutionWorker

Located in `pkg/workers/pending_node_execution_worker.go:20`, this worker:

- Polls for executions in `pending` state
- Ensures only one execution per node runs at a time (per workflow)
- Handles both component nodes and blueprint nodes

**For Component Nodes:**
1. Marks execution as `started`
2. Retrieves component from registry
3. Gets input data from previous execution (or initial event)
4. Creates `ExecutionContext` with metadata and state contexts
5. Calls `component.Execute(ctx)`
6. Component controls when execution completes via `Pass()` or `Fail()`

**For Blueprint Nodes:**
1. Finds the first node in the blueprint (node with no incoming edges)
2. Creates a child execution for that node with:
   - `ParentExecutionID` pointing to the blueprint node execution
   - `BlueprintID` set to the blueprint's ID
   - `PreviousExecutionID` pointing to blueprint node for input resolution
   - Resolved configuration using ConfigurationBuilder
3. Marks blueprint node as `started` (waits for children to complete)

#### 2. ExecutionRouter

Located in `pkg/workers/execution_router.go:27`, this worker:

- Polls for executions in `routing` state
- Creates child executions for next nodes in the flow
- Handles blueprint completion

**Routing Logic:**

1. **Passed Execution:**
   - Finds all outgoing edges from the completed node
   - For each edge matching an output branch with data:
     - If `TargetType = "node"`: Creates child execution for each item in branch
     - If `TargetType = "output_branch"`: Marks as blueprint exit (handled in completion)
   - Marks execution as `finished`

2. **Failed Execution:**
   - If inside a blueprint: finishes child and fails parent blueprint execution
   - Otherwise: marks as `finished` (chain stops)

3. **Blueprint Completion:**
   - When a blueprint child execution has no more nodes to route to
   - Checks if any active executions remain in the blueprint
   - If none remain:
     - Collects outputs from all blueprint exit edges
     - Aggregates them by output branch name
     - Marks parent blueprint node execution as passed with collected outputs
     - Moves parent to `routing` state to continue workflow

### Input Resolution

The `GetInputs()` method (in `pkg/models/workflow_node_execution.go:231`) determines execution inputs:

1. **First Node in Flow**: Reads from `workflow_initial_events` table using `RootEventID`
2. **Entering a Blueprint**: Reads from parent blueprint node execution's inputs
3. **Normal Flow**: Reads from `PreviousExecutionID.Outputs[PreviousOutputBranch][PreviousOutputIndex]`

### Output Structure

Outputs are structured as `map[string][]any`:
- **Key**: Output branch name (e.g., "default", "true", "false")
- **Value**: Array of output items

This structure supports:
- Multiple output branches per node
- Multiple items per branch (for fan-out scenarios)
- Automatic creation of child executions for each item

## Database Schema

### Blueprints Table

```sql
CREATE TABLE blueprints (
  id              uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  name            VARCHAR(128) NOT NULL,
  description     TEXT,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  nodes           JSONB NOT NULL DEFAULT '[]',
  edges           JSONB NOT NULL DEFAULT '[]',
  configuration   JSONB NOT NULL DEFAULT '[]',  -- Exposed parameters
  output_branches JSONB NOT NULL DEFAULT '[]',  -- Exit points
  UNIQUE (organization_id, name)
);
```

### Workflows Table

```sql
CREATE TABLE workflows (
  id              uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  name            VARCHAR(128) NOT NULL,
  description     TEXT,
  created_at      TIMESTAMP NOT NULL,
  updated_at      TIMESTAMP NOT NULL,
  nodes           JSONB NOT NULL DEFAULT '[]',
  edges           JSONB NOT NULL DEFAULT '[]',
  UNIQUE (organization_id, name)
);
```

### Workflow Initial Events Table

```sql
CREATE TABLE workflow_initial_events (
  id          uuid PRIMARY KEY,
  workflow_id uuid NOT NULL REFERENCES workflows(id),
  data        JSONB NOT NULL,
  created_at  TIMESTAMP NOT NULL
);
```

Stores the initial trigger data for each workflow run. The `id` becomes the `root_event_id` for all executions in that run.

### Workflow Node Executions Table

```sql
CREATE TABLE workflow_node_executions (
  id                     uuid PRIMARY KEY,
  workflow_id            uuid NOT NULL REFERENCES workflows(id),
  node_id                VARCHAR(128) NOT NULL,

  -- Root event (shared by all executions in this run)
  root_event_id          uuid NOT NULL REFERENCES workflow_initial_events(id),

  -- Sequential flow
  previous_execution_id  uuid REFERENCES workflow_node_executions(id),
  previous_output_branch VARCHAR(64),
  previous_output_index  INTEGER,

  -- Blueprint hierarchy
  parent_execution_id    uuid REFERENCES workflow_node_executions(id),
  blueprint_id           uuid,  -- No FK, preserve history

  -- State machine
  state                  VARCHAR(32) NOT NULL,
  result                 VARCHAR(32),
  result_reason          VARCHAR(128),
  result_message         TEXT,

  -- Data
  outputs                JSONB,
  metadata               JSONB NOT NULL DEFAULT '{}',
  configuration          JSONB NOT NULL DEFAULT '{}',

  created_at             TIMESTAMP NOT NULL,
  updated_at             TIMESTAMP NOT NULL
);
```

**Key Indexes:**
- `(workflow_id, node_id)` - Find executions for a node
- `(root_event_id)` - Find all executions in a workflow run
- `(previous_execution_id)` - Trace execution chains
- `(parent_execution_id)` - Find blueprint children
- `(blueprint_id)` - Find executions within a blueprint
- `(state)` WHERE `state = 'pending'` - Pending execution polling
- `(state)` WHERE `state = 'routing'` - Routing execution polling

## API and Services

New gRPC services were added:

### Blueprint Service

Located in `pkg/grpc/blueprint_service.go`, provides:
- `CreateBlueprint` - Create new blueprint
- `UpdateBlueprint` - Update blueprint definition
- `DescribeBlueprint` - Get blueprint details
- `ListBlueprints` - List all blueprints

### Component Service

Located in `pkg/grpc/component_service.go`, provides:
- `ListComponents` - List all registered components
- `DescribeComponent` - Get component details (config fields, actions, etc.)
- `ListComponentActions` - Get available actions for a component

### Workflow Service

Located in `pkg/grpc/workflow_service.go`, provides:
- `CreateWorkflow` - Create new workflow
- `UpdateWorkflow` - Update workflow definition
- `DeleteWorkflow` - Delete workflow
- `DescribeWorkflow` - Get workflow details
- `ListWorkflows` - List all workflows
- `ListWorkflowEvents` - List initial events (workflow runs)
- `ListEventExecutions` - List executions for a workflow run
- `ListNodeExecutions` - List executions for a specific node
- `InvokeNodeExecutionAction` - Invoke a component action on an execution

## Component Registry

Components must be registered at startup. The registry (in `pkg/registry/`) maps component names to implementations.

Built-in components registered:
- `http` - HTTP component
- `approval` - Approval component
- `if` - If component
- `switch` - Switch component
- `filter` - Filter component

## Migration from Executors

The old Executor interface remains for backward compatibility with existing integrations, but new functionality should use the Component interface.

**Key Migration Points:**

| Executor | Component |
|----------|-----------|
| Simple execute method | Full lifecycle control via ExecutionContext |
| No state management | Metadata storage per execution |
| Binary success/failure | Rich result handling with reasons |
| No dynamic behavior | Actions for interactive flows |
| Fixed outputs | Dynamic output branches |
| Synchronous only | Support for async patterns |

## Example: Approval Component Flow

This example demonstrates the async pattern using the Approval component:

1. **Initial Execution** (`Execute` called):
   - Creates metadata with required count and empty approvals list
   - Does NOT complete execution
   - Execution remains in `started` state

2. **User Action** (via `InvokeNodeExecutionAction` API):
   - Calls `HandleAction` with action name "approve"
   - Adds approval record to metadata
   - If count not reached: updates metadata, execution stays in `started`
   - If count reached: calls `Pass()` to complete execution

3. **Routing**:
   - Execution moves to `routing` state
   - ExecutionRouter creates child executions
   - Execution marked as `finished`

## Example: Blueprint with HTTP Calls

Scenario: A blueprint that makes two HTTP calls in sequence.

**Blueprint Definition:**
```json
{
  "name": "api-chain",
  "configuration": [
    {"name": "base_url", "type": "string", "required": true}
  ],
  "output_branches": [
    {"name": "success", "label": "Success"}
  ],
  "nodes": [
    {
      "id": "http-1",
      "ref_type": "component",
      "ref": {"component": {"name": "http"}},
      "configuration": {
        "url": "${base_url}/endpoint1",
        "method": "GET"
      }
    },
    {
      "id": "http-2",
      "ref_type": "component",
      "ref": {"component": {"name": "http"}},
      "configuration": {
        "url": "${base_url}/endpoint2",
        "method": "POST"
      }
    }
  ],
  "edges": [
    {"source_id": "http-1", "target_type": "node", "target_id": "http-2", "branch": "default"},
    {"source_id": "http-2", "target_type": "output_branch", "target_id": "success", "branch": "default"}
  ]
}
```

**Workflow Using Blueprint:**
```json
{
  "name": "my-workflow",
  "nodes": [
    {
      "id": "blueprint-node",
      "ref_type": "blueprint",
      "ref": {"blueprint": {"id": "api-chain-uuid"}},
      "configuration": {
        "base_url": "https://api.example.com"
      }
    },
    {
      "id": "next-step",
      "ref_type": "component",
      "ref": {"component": {"name": "http"}},
      "configuration": {"url": "https://other.com", "method": "POST"}
    }
  ],
  "edges": [
    {"source_id": "blueprint-node", "target_type": "node", "target_id": "next-step", "branch": "success"}
  ]
}
```

**Execution Flow:**

1. Workflow triggered with initial event
2. `blueprint-node` execution created in `pending` state
3. PendingNodeExecutionWorker processes it:
   - Finds first node in blueprint: `http-1`
   - Resolves `${base_url}` in config to `"https://api.example.com"`
   - Creates child execution for `http-1` with `parent_execution_id` = blueprint-node ID
   - Marks blueprint-node as `started`
4. PendingNodeExecutionWorker processes `http-1`:
   - Executes HTTP GET to `https://api.example.com/endpoint1`
   - Calls `Pass()` with response outputs
   - Moves to `routing` state
5. ExecutionRouter routes `http-1`:
   - Finds edge to `http-2`
   - Creates child execution for `http-2`
   - Marks `http-1` as `finished`
6. PendingNodeExecutionWorker processes `http-2`:
   - Executes HTTP POST with previous response as body
   - Calls `Pass()` with response
   - Moves to `routing` state
7. ExecutionRouter routes `http-2`:
   - Finds exit edge to `success` output branch
   - No more nodes in blueprint
   - Checks for active executions in blueprint (none found)
   - Collects outputs from `http-2` to `success` branch
   - Marks `http-2` as `finished`
   - Marks blueprint-node as passed with collected outputs
   - Moves blueprint-node to `routing` state
8. ExecutionRouter routes blueprint-node:
   - Finds edge to `next-step`
   - Creates execution for `next-step`
   - Marks blueprint-node as `finished`
9. Execution continues with `next-step`

## Summary

The blueprints-and-workflows architecture provides:

✅ **Composability**: Blueprints are reusable subgraphs
✅ **Flexibility**: Components have full lifecycle control
✅ **Interactivity**: Actions enable async workflows
✅ **State Management**: Metadata storage per execution
✅ **Complex Routing**: Dynamic branches and fan-out support
✅ **Hierarchy**: Nested blueprints with configuration resolution
✅ **Traceability**: Complete execution history in database

This architecture replaces the previous canvas-based system with a more powerful, flexible model suitable for complex orchestration scenarios.
