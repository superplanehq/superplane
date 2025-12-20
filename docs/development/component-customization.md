# Component Customization Guide

This guide explains how to add custom behaviors to workflow components by following the existing architecture patterns. The component system uses registries to manage different types of customizations.

## File Structure Overview

The component system is organized in the following directory structure:

```
web_src/src/pages/workflowv2/mappers/
├── index.ts                    # Main registry file - all registrations happen here
├── types.ts                    # TypeScript interfaces for all customization types
├── stateRegistry.ts            # Default state registry and fallback state logic
├── default.ts                  # Default trigger renderer implementation
├── approval.ts                 # Approval component with custom states, data builder
├── wait.tsx                    # Wait component with custom field renderer
├── schedule.ts                 # Schedule trigger with custom field renderer
├── github.ts                   # GitHub-specific trigger renderer
├── semaphore.ts                # Semaphore component mapper
├── http.ts                     # HTTP component mapper
├── if.ts                       # If/conditional component mapper
├── filter.ts                   # Filter component mapper
├── timegate.ts                 # Time gate component mapper
├── noop.ts                     # No-operation component mapper
└── semaphore/                  # Directory for semaphore app-specific mappers
    └── index.ts                # Semaphore app component and trigger registries
```

## Key Files and Their Purposes

- **`index.ts`** - This is where ALL component customizations are registered. Every new customization must be added here.
- **`types.ts`** - Contains TypeScript interfaces that define the contracts for all customization types.
- **`stateRegistry.ts`** - Provides the default state logic that most components inherit from.
- **Component files** (e.g., `approval.ts`, `wait.tsx`) - Individual component implementations with their specific customizations.

## Registry Types

The main registry file `web_src/src/pages/workflowv2/mappers/index.ts` manages 6 types of customizations:

### 1. Component Base Mappers (`componentBaseMappers`)
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 35-44
**Purpose:** Maps component rendering logic and properties.
**Current registrations:** noop, if, http, semaphore, time_gate, filter, wait, approval

```typescript
const componentBaseMappers: Record<string, ComponentBaseMapper> = {
  noop: noopMapper,              // from ./noop.ts
  if: ifMapper,                  // from ./if.ts
  http: httpMapper,              // from ./http.ts
  semaphore: oldSemaphoreMapper, // from ./semaphore.ts
  time_gate: timeGateMapper,     // from ./timegate.ts
  filter: filterMapper,          // from ./filter.ts
  wait: waitMapper,              // from ./wait.tsx
  approval: approvalMapper,      // from ./approval.ts
};
```

### 2. Trigger Renderers (`triggerRenderers`)
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 30-33
**Purpose:** Handles how triggers are displayed and behave.
**Current registrations:** github, schedule

```typescript
const triggerRenderers: Record<string, TriggerRenderer> = {
  github: githubTriggerRenderer,    // from ./github.ts
  schedule: scheduleTriggerRenderer, // from ./schedule.ts
};
```

### 3. Component Additional Data Builders (`componentAdditionalDataBuilders`)
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 56-58
**Purpose:** Builds component-specific data that requires external API calls or complex logic.
**Current registrations:** approval

```typescript
const componentAdditionalDataBuilders: Record<string, ComponentAdditionalDataBuilder> = {
  approval: approvalDataBuilder, // from ./approval.ts
};
```

### 4. Event State Registries (`eventStateRegistries`)
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 60-62
**Purpose:** Custom state logic and visual styling for different component states.
**Current registrations:** approval

```typescript
const eventStateRegistries: Record<string, EventStateRegistry> = {
  approval: APPROVAL_STATE_REGISTRY, // from ./approval.ts
};
```

### 5. Custom Field Renderers (`customFieldRenderers`)
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 64-67
**Purpose:** Renders additional UI elements in component settings.
**Current registrations:** schedule, wait

```typescript
const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  schedule: scheduleCustomFieldRenderer, // from ./schedule.ts
  wait: waitCustomFieldRenderer,        // from ./wait.tsx
};
```

### 6. App-specific Registries
**Location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 46-54
**Purpose:** For components that belong to specific applications (like semaphore.*, github.*)
**Current app registrations:** semaphore, github

```typescript
const appMappers: Record<string, Record<string, ComponentBaseMapper>> = {
  semaphore: semaphoreComponentMappers, // from ./semaphore/index.ts
  github: githubComponentMappers,       // from ./github/index.ts
};

const appTriggerRenderers: Record<string, Record<string, TriggerRenderer>> = {
  semaphore: semaphoreTriggerRenderers, // from ./semaphore/index.ts
  github: githubTriggerRenderers,       // from ./github/index.ts
};
```

## Step-by-Step Tutorial

### Example 1: Creating a Custom State Registry (Approval Component)

**Primary file:** `web_src/src/pages/workflowv2/mappers/approval.ts`
**State map definition:** Lines 37-69
**State function definition:** Lines 74-102
**State registry creation:** Lines 107-110
**Registration location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 60-62

The approval component demonstrates custom state logic:

#### 1. Define Custom State Map

```typescript
export const APPROVAL_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP, // Inherit defaults
  waiting: {
    icon: "clock",
    textColor: "text-black",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  approved: {
    icon: "circle-check",
    textColor: "text-black",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  rejected: {
    icon: "circle-x",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};
```

#### 2. Create Custom State Function

```typescript
export const approvalStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  // Error state - component could not evaluate
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
    return "error";
  }

  // Waiting state - actors haven't responded
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "waiting";
  }

  // Check execution metadata for approval decision
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const metadata = execution.metadata as Record<string, any> | undefined;
    if (metadata?.result === "approved") return "approved";
    if (metadata?.result === "rejected") return "rejected";
    return "approved"; // Default to success
  }

  return "error"; // Fallback
};
```

#### 3. Create State Registry

```typescript
export const APPROVAL_STATE_REGISTRY: EventStateRegistry = {
  stateMap: APPROVAL_STATE_MAP,
  getState: approvalStateFunction,
};
```

#### 4. Register in Main Registry

In `web_src/src/pages/workflowv2/mappers/index.ts:60-62`:

```typescript
const eventStateRegistries: Record<string, EventStateRegistry> = {
  approval: APPROVAL_STATE_REGISTRY,
};
```

### Example 2: Creating a Custom Field Renderer (Wait Component)

**Primary file:** `web_src/src/pages/workflowv2/mappers/wait.tsx`
**Custom field renderer definition:** Lines 242-294
**Registration location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 64-67
**Import statement location:** `web_src/src/pages/workflowv2/mappers/index.ts` line 22

The wait component shows custom UI in the settings panel:

#### 1. Implement CustomFieldRenderer

```typescript
export const waitCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: ComponentsNode, configuration: Record<string, unknown>) => {
    const mode = configuration?.mode as string;

    let content: string;
    let title: string;

    if (mode === "interval") {
      title = "Fixed Time Interval";
      content = `Component will wait for a fixed amount of time...

Example expressions:
{{ $.wait_time }}
{{ $.wait_time + 5 }}`;
    } else if (mode === "countdown") {
      title = "Countdown to Date/Time";
      content = `Component will countdown until the provided date/time...

Example expressions:
{{ $.run_time }}
{{ date($.date_string) }}`;
    } else {
      title = "Wait Component";
      content = "Configure the wait mode to see more details.";
    }

    return (
      <div className="border-t-1 border-gray-200">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700">{title}:</span>
            <div className="text-sm text-gray-900 mt-1 border-1 p-3 bg-gray-50 rounded-md font-mono whitespace-pre-line">
              {content}
            </div>
          </div>
        </div>
      </div>
    );
  },
};
```

#### 2. Register in Main Registry

In `web_src/src/pages/workflowv2/mappers/index.ts:64-67`:

```typescript
const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  schedule: scheduleCustomFieldRenderer,
  wait: waitCustomFieldRenderer,
};
```

### Example 3: Creating an Additional Data Builder (Approval Component)

**Primary file:** `web_src/src/pages/workflowv2/mappers/approval.ts`
**Data builder definition:** Lines 257-398
**Registration location:** `web_src/src/pages/workflowv2/mappers/index.ts` lines 56-58
**Import statement location:** `web_src/src/pages/workflowv2/mappers/index.ts` line 23
**Usage in component:** Lines 119 and 123 (passed as `additionalData` parameter)

For components that need to fetch external data:

#### 1. Implement ComponentAdditionalDataBuilder

```typescript
export const approvalDataBuilder: ComponentAdditionalDataBuilder = {
  buildAdditionalData(
    _nodes: ComponentsNode[],
    node: ComponentsNode,
    _componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    workflowId: string,
    queryClient: QueryClient,
    organizationId?: string,
  ) {
    const execution = lastExecutions[0];
    const usersById: Record<string, { email?: string; name?: string }> = {};
    const rolesByName: Record<string, string> = {};

    // Fetch user data from cache
    if (organizationId) {
      const usersResp = queryClient.getQueryData(organizationKeys.users(organizationId));
      if (Array.isArray(usersResp)) {
        usersResp.forEach((user) => {
          const id = user.metadata?.id;
          if (id) usersById[id] = { email: user.metadata?.email, name: user.spec?.displayName };
        });
      }
    }

    // Map execution records to approval items with interactive handlers
    const approvals = ((execution?.metadata?.records as any[]) || []).map((record) => ({
      id: `${record.index}`,
      title: record.user?.name || record.user?.email || "Unknown",
      approved: record.state === "approved",
      rejected: record.state === "rejected",
      interactive: record.state === "pending" && execution?.state === "STATE_STARTED",
      onApprove: async (artifacts?: Record<string, string>) => {
        await workflowsInvokeNodeExecutionAction({
          path: { workflowId, executionId: execution.id, actionName: "approve" },
          body: { parameters: { index: record.index, comment: artifacts?.comment } },
        });
        queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, node.id!) });
      },
      onReject: async (comment?: string) => {
        await workflowsInvokeNodeExecutionAction({
          path: { workflowId, executionId: execution.id, actionName: "reject" },
          body: { parameters: { index: record.index, reason: comment } },
        });
        queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, node.id!) });
      },
    }));

    return { approvals, usersById, rolesByName };
  },
};
```

#### 2. Register in Main Registry

In `web_src/src/pages/workflowv2/mappers/index.ts:56-58`:

```typescript
const componentAdditionalDataBuilders: Record<string, ComponentAdditionalDataBuilder> = {
  approval: approvalDataBuilder,
};
```

## Creating a New Custom Component

To create a new component with custom behaviors:

### 1. Create Component File

**Location:** `web_src/src/pages/workflowv2/mappers/mycomponent.ts`
**Required imports:** From `./types` and any UI components you need
**Follow naming convention:** File name should match component type name

Create `web_src/src/pages/workflowv2/mappers/mycomponent.ts`:

```typescript
import { ComponentBaseMapper, EventStateRegistry, CustomFieldRenderer } from "./types";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";

// Custom state map (optional)
export const MY_COMPONENT_STATE_MAP = {
  ...DEFAULT_EVENT_STATE_MAP,
  processing: {
    icon: "loader",
    textColor: "text-blue-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
};

// Custom state function (optional)
export const myComponentStateFunction = (execution) => {
  if (execution.metadata?.status === "processing") return "processing";
  // ... other logic
  return defaultStateFunction(execution);
};

// State registry (optional)
export const MY_COMPONENT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: MY_COMPONENT_STATE_MAP,
  getState: myComponentStateFunction,
};

// Base mapper (required)
export const myComponentMapper: ComponentBaseMapper = {
  props(nodes, node, componentDefinition, lastExecutions, nodeQueueItems, additionalData) {
    return {
      iconSlug: componentDefinition.icon || "box",
      iconColor: "text-blue-600",
      headerColor: "bg-white",
      title: node.name || "My Component",
      // ... other properties
    };
  },
  subtitle(node, execution, additionalData) {
    return execution.metadata?.customMessage || "Processing...";
  },
};

// Custom field renderer (optional)
export const myComponentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node, configuration) => {
    return (
      <div className="p-4">
        <p>Custom configuration UI for {node.name}</p>
        <pre>{JSON.stringify(configuration, null, 2)}</pre>
      </div>
    );
  },
};
```

### 2. Register in Main Registry

**File to modify:** `web_src/src/pages/workflowv2/mappers/index.ts`
**Add import statements:** Near the top of the file with other imports
**Add to registries:** In the appropriate registry objects (lines 35-67)
**Follow existing patterns:** Look at how other components are registered

In `web_src/src/pages/workflowv2/mappers/index.ts`, add imports and register:

```typescript
import {
  myComponentMapper,
  MY_COMPONENT_STATE_REGISTRY,
  myComponentCustomFieldRenderer
} from "./mycomponent";

// Add to registries
const componentBaseMappers: Record<string, ComponentBaseMapper> = {
  // ... existing mappers
  mycomponent: myComponentMapper,
};

const eventStateRegistries: Record<string, EventStateRegistry> = {
  // ... existing registries
  mycomponent: MY_COMPONENT_STATE_REGISTRY,
};

const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  // ... existing renderers
  mycomponent: myComponentCustomFieldRenderer,
};
```

## Best Practices

1. **Follow naming conventions**: Component file names should match the component type
2. **Extend defaults**: Always extend `DEFAULT_EVENT_STATE_MAP` rather than replacing it
3. **Error handling**: Include proper error states and fallbacks in state functions
4. **Type safety**: Use proper TypeScript types from `types.ts`
5. **Performance**: Cache expensive operations in additional data builders
6. **Consistency**: Follow existing patterns for UI styling and interactions

## Helper Functions

The registry provides several helper functions in `web_src/src/pages/workflowv2/mappers/index.ts`:

**Location of helper functions:** Lines 73-147
**Default fallbacks:** Defined in `web_src/src/pages/workflowv2/mappers/stateRegistry.ts` and `web_src/src/pages/workflowv2/mappers/default.ts`

Available helper functions:
- `getTriggerRenderer(name)`: Lines 73-87 - Get trigger renderer with app support
- `getComponentBaseMapper(name)`: Lines 93-107 - Get component mapper with app support
- `getComponentAdditionalDataBuilder(name)`: Lines 113-115 - Get data builder
- `getEventStateRegistry(name)`: Lines 121-123 - Get state registry with fallback
- `getStateMap(name)`: Lines 129-131 - Get state map
- `getState(name)`: Lines 137-139 - Get state function
- `getCustomFieldRenderer(name)`: Lines 145-147 - Get custom field renderer

These functions handle the lookup logic and provide fallbacks to default implementations.

## Adding Props to ComponentBase

When you need to add new visual properties or behaviors to components:

### 1. Add Prop to ComponentBaseProps Interface

**File:** `web_src/src/ui/componentBase/index.tsx`
**Interface location:** Lines 187-212
**Add your new prop** to the `ComponentBaseProps` interface:

```typescript
export interface ComponentBaseProps extends ComponentActionsProps {
  // ... existing props
  myCustomProp?: string;          // Add your new prop here
  myCustomBehavior?: boolean;     // Or multiple props as needed
}
```

### 2. Update ComponentBase Component

**File:** `web_src/src/ui/componentBase/index.tsx`
**Component definition:** Lines 214-401
**Add prop to destructuring** (around line 214) and **use it in JSX**:

```typescript
export const ComponentBase: React.FC<ComponentBaseProps> = ({
  // ... existing props
  myCustomProp,
  myCustomBehavior,
  // ... rest of props
}) => {
  // Use your prop in the component logic or JSX
  return (
    <div className={`${myCustomBehavior ? 'custom-class' : ''}`}>
      {myCustomProp && <span>{myCustomProp}</span>}
      {/* ... rest of component */}
    </div>
  );
};
```

### 3. Update Component Mappers

**Files:** Various mapper files (e.g., `web_src/src/pages/workflowv2/mappers/approval.ts`)
**Update the `props` method** in your component's mapper to include the new prop:

```typescript
export const myComponentMapper: ComponentBaseMapper = {
  props(nodes, node, componentDefinition, lastExecutions, nodeQueueItems, additionalData): ComponentBaseProps {
    return {
      // ... existing props
      myCustomProp: "Custom value based on component logic",
      myCustomBehavior: lastExecutions.length > 0,
      // ... rest of props
    };
  },
};
```

### Example: Adding a Custom Badge Prop

Let's say you want to add a `statusBadge` prop:

#### Step 1: Add to Interface
**Location:** `web_src/src/ui/componentBase/index.tsx:187-212`
```typescript
export interface ComponentBaseProps extends ComponentActionsProps {
  // ... existing props
  statusBadge?: {
    text: string;
    color: string;
  };
}
```

#### Step 2: Use in Component
**Location:** `web_src/src/ui/componentBase/index.tsx:214-401`
```typescript
export const ComponentBase: React.FC<ComponentBaseProps> = ({
  // ... existing props
  statusBadge,
}) => {
  return (
    <div>
      <ComponentHeader /* ... */ />
      {statusBadge && (
        <div className={`px-2 py-1 text-xs font-semibold rounded ${statusBadge.color}`}>
          {statusBadge.text}
        </div>
      )}
      {/* ... rest of component */}
    </div>
  );
};
```

#### Step 3: Update Mapper
**Location:** `web_src/src/pages/workflowv2/mappers/approval.ts:120-138`
```typescript
export const approvalMapper: ComponentBaseMapper = {
  props(nodes, node, componentDefinition, lastExecutions): ComponentBaseProps {
    const lastExecution = lastExecutions[0];

    return {
      // ... existing props
      statusBadge: lastExecution?.state === "STATE_STARTED" ? {
        text: "Awaiting Approval",
        color: "bg-yellow-100 text-yellow-800"
      } : undefined,
    };
  },
};
```

## Quick Reference Paths

For designers who need to quickly locate files:

- **Main registry:** `web_src/src/pages/workflowv2/mappers/index.ts`
- **Type definitions:** `web_src/src/pages/workflowv2/mappers/types.ts`
- **ComponentBase UI:** `web_src/src/ui/componentBase/index.tsx`
- **ComponentBaseProps interface:** `web_src/src/ui/componentBase/index.tsx` (lines 187-212)
- **Default state logic:** `web_src/src/pages/workflowv2/mappers/stateRegistry.ts`
- **Example custom states:** `web_src/src/pages/workflowv2/mappers/approval.ts` (lines 37-110)
- **Example custom field:** `web_src/src/pages/workflowv2/mappers/wait.tsx` (lines 242-294)
- **Example data builder:** `web_src/src/pages/workflowv2/mappers/approval.ts` (lines 257-398)
- **App-specific example:** `web_src/src/pages/workflowv2/mappers/semaphore/index.ts`