# Track B Reference — Workflow Linter / Quality Gate (Braca)

## Existing Validation (What Superplane Already Does)

There's already validation in `pkg/grpc/actions/canvases/serialization.go` (lines 197-326). The linter should go **beyond** this with deeper semantic checks.

**Already validated (don't duplicate):**
- Unique node IDs
- Node names present
- Component/trigger references exist in registry
- Edge source/target IDs exist
- Widgets not used as edge source/target
- Cycle detection (`CheckForCycles`)
- Group widget validation (no nesting, no self-reference)
- Basic config validation against component schema

## What the Linter Should Add

### 1. Orphan Node Detection
Nodes not reachable from any trigger (no path from root).

```go
// Find root nodes (triggers with no incoming edges)
func FindOrphanNodes(nodes []Node, edges []Edge) []Node {
    // Build adjacency: reachable set from all triggers
    triggers := findTriggerNodes(nodes)
    reachable := bfs(triggers, edges)

    var orphans []Node
    for _, n := range nodes {
        if n.Type == "widget" { continue } // groups are OK
        if !reachable[n.ID] {
            orphans = append(orphans, n)
        }
    }
    return orphans
}
```

### 2. Dead-End Detection
Nodes with no outgoing edges that aren't terminal (Slack, email, approval, etc.).

### 3. Missing Approval Before Destructive Actions
Destructive components that should have an approval gate upstream:
- `pagerduty.resolveIncident`
- `pagerduty.escalateIncident`
- `github.deleteRelease`
- `github.createRelease`
- Any HTTP DELETE/PUT to production URLs
- SSH commands

Check: walk the graph backwards from these nodes — is there an `approval` component in the path?

### 4. Missing Required Configuration
Go beyond basic "field required" — check semantic requirements:
- Claude `textPrompt` with empty `prompt` field
- HTTP component with no `url`
- Slack `sendTextMessage` with no `channel`
- Merge component with only 1 incoming edge (pointless merge)

### 5. Expression Syntax Validation
Validate expression strings without executing them:
- Balanced `{{ }}` delimiters
- Valid `$['Node Name']` references point to actual node names
- `root()`, `previous()` used correctly

### 6. Unreachable Branches
After an `if` component, check that both true/false branches lead somewhere.

## Canvas Data Model

### Node Structure (`pkg/models/blueprint.go`)
```go
type Node struct {
    ID            string
    Name          string
    Type          string         // "trigger", "component", "blueprint", "widget"
    Ref           NodeRef        // exactly one of: Component, Blueprint, Trigger, Widget
    Configuration map[string]any
    Metadata      map[string]any
    Position      Position       // {X, Y}
    IsCollapsed   bool
    IntegrationID *string
    ErrorMessage  *string
    WarningMessage *string
}

type NodeRef struct {
    Component *ComponentRef  // {Name: "http"}
    Blueprint *BlueprintRef  // {ID: "..."}
    Trigger   *TriggerRef    // {Name: "pagerduty.onIncident"}
    Widget    *WidgetRef     // {Name: "group"}
}
```

### Edge Structure
```go
type Edge struct {
    SourceID string  // upstream node ID
    TargetID string  // downstream node ID
    Channel  string  // "default", "success", "fail", "approved", "rejected", etc.
}
```

### Canvas Version (where nodes/edges live)
```go
type CanvasVersion struct {
    Nodes []Node
    Edges []Edge
    // ... metadata
}
```

### Accessing Canvas via API
```
GET /api/v1/canvases/{id}        -> Canvas with live version spec
GET /api/v1/canvases/{id}/spec   -> Just the nodes and edges
```

**Proto:** `protos/canvases.proto` — `Canvas.Spec` contains `repeated Node nodes` and `repeated Edge edges`

## Component Configuration Schema

Each component defines its config via `Configuration() []configuration.Field`:

```go
type Field struct {
    Name       string
    Label      string
    Type       string  // "string", "number", "boolean", "select", "expression", "text", etc.
    Required   bool
    Default    any
    Sensitive  bool
}
```

**Existing validation:** `pkg/configuration/validation.go` → `ValidateConfiguration(fields, config)`

The registry at `pkg/registry/registry.go` has all components:
```go
Registry.Components  // map[string]core.Component
```

## Graph Traversal Helpers

Already in `pkg/models/blueprint.go`:
```go
FindEdges(sourceID, channel string) []Edge  // outgoing edges from node
FindRootNode() *Node                        // node with no incoming edges
```

## Linter Output Format

Suggested structure:
```json
{
  "status": "fail",
  "errors": [
    {
      "severity": "error",
      "rule": "orphan-node",
      "nodeId": "abc123",
      "nodeName": "Unused HTTP Call",
      "message": "Node is not reachable from any trigger"
    }
  ],
  "warnings": [
    {
      "severity": "warning",
      "rule": "missing-approval-gate",
      "nodeId": "def456",
      "nodeName": "Delete Release",
      "message": "Destructive action 'github.deleteRelease' has no upstream approval gate"
    }
  ],
  "info": [
    {
      "severity": "info",
      "rule": "single-input-merge",
      "nodeId": "ghi789",
      "nodeName": "Wait for all",
      "message": "Merge node has only 1 incoming edge — consider removing"
    }
  ],
  "summary": {
    "total": 3,
    "errors": 1,
    "warnings": 1,
    "info": 1
  }
}
```

## Implementation Options

### Option A: Go Package (recommended)
Add `pkg/linter/linter.go` with:
```go
func LintCanvas(nodes []models.Node, edges []models.Edge, registry *registry.Registry) *LintResult
```
- Can access component registry for config validation
- Can be called from gRPC action (new API endpoint)
- Can be wired into pre-publish hook

### Option B: TypeScript (frontend-only)
Add `web_src/src/utils/canvasLinter.ts`:
- Operates on the React Flow node/edge data already in memory
- Shows results inline in Canvas UI immediately
- No backend changes needed
- BUT: no access to component config schema

### Option C: Both
- Go backend for deep validation (config schema, expression parsing)
- TypeScript frontend for instant visual feedback (orphans, dead-ends)

For 3 hours, **Option A or B alone is sufficient**. Pick based on comfort.

## Key Source Files

| File | What to look at |
|------|-----------------|
| `pkg/models/blueprint.go:121-167` | Node, Edge, NodeRef structs |
| `pkg/grpc/actions/canvases/serialization.go:197-326` | Existing validation to extend |
| `pkg/configuration/field.go` | Config field schema |
| `pkg/configuration/validation.go` | Config validation logic |
| `pkg/core/component.go:70` | Component interface (Configuration method) |
| `pkg/registry/registry.go` | Component registry |
| `pkg/components/approval/approval.go` | Approval component |
| `pkg/components/merge/merge.go` | Merge component |
| `protos/canvases.proto` | Canvas proto definition |
| `protos/components.proto` | Node/Edge proto definition |

## "Eat Our Own Dogfood" Demo

At 2:15, run the linter against Dragan's Incident Copilot canvas:
1. It should PASS (green) — copilot is well-formed
2. Remove an edge → run again → catches orphan node (red)
3. Remove the approval gate → run again → warns about missing approval before destructive action
4. Fix → green again

This is the money shot for the demo.
