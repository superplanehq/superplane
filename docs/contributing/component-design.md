# Component Design and Quality Review Guide

This guide establishes **product design principles and quality standards** for SuperPlane components. It focuses on the "what" and "why" — not the "how" of implementation.

**Who is this for?**
- Product managers reviewing component designs
- Engineers designing new components (before implementation)
- Anyone evaluating component quality and consistency

**What this covers:**
- User experience decisions and design patterns
- What information to display where and why
- Consistency guidelines across all components

**What this does NOT cover:**
- Implementation details (see technical guides below)
- Code patterns and syntax
- Backend/frontend architecture

For implementation, see: [integrations.md](integrations.md), [component-implementations.md](component-implementations.md), [component-customization.md](component-customization.md).

## Table of Contents

- [Glossary](#glossary)
- [Component Node Display](#component-node-display)
- [Component Configuration](#component-configuration)
- [Output Channels](#output-channels)
- [Run Item Display](#run-item-display)
- [Run Item Action Menu](#run-item-action-menu)
- [Run Item in Sidebar](#run-item-in-sidebar)
- [Error Handling](#error-handling)
- [Emitted Payload Structure](#emitted-payload-structure)
- [Summary Checklist](#summary-checklist)

---

## Glossary

Consistent terminology is essential for clear communication. Use these terms as defined:

| Term | Definition |
|------|------------|
| **Canvas** | The visual workspace where workflows are designed and monitored |
| **Component** | A building block definition (blueprint) that specifies capabilities, configuration, and outputs |
| **Component Node** | An instance of a component placed on the canvas with specific configuration |
| **Trigger Component** | A component that starts workflow executions by listening for external events |
| **Action Component** | A component that executes operations in response to upstream events |
| **Run** | A complete workflow execution from start to finish (collection of run items) |
| **Run Item** | A single execution within a single node (one received event or one action execution) |
| **Payload** | JSON data emitted by a node containing execution results or event data |
| **Output Channel** | Named output path for routing data to downstream nodes (e.g., "passed", "failed", "approved") |
| **Subscription** | Connection between nodes that defines how events flow from source to target |
| **Expression** | Dynamic value using Expr language to access payload data (`$['Node Name'].field`) |
| **Message Chain** | Accumulated payloads from all upstream nodes in a run, accessible via `$` |
| **State** | Execution status of a run item (running, success, failed, error, cancelled, etc.). Note: `error` means component couldn't execute; `failed` means it executed but the outcome was failure |
| **Metadata** | Key-value data stored with an execution for state determination and display |
| **Details Tab** | UI panel in the sidebar showing execution information (timestamps, results, errors) |
| **Event Section** | UI representation of a run item displayed on a node or in the sidebar run history |

---

## Component Node Display

Component nodes on the canvas should provide at-a-glance information about their configuration and status.

### Node Structure

A component node consists of the following visual elements:

| Element | Description |
|---------|-------------|
| **Header** | Contains the component icon, node title (user-defined name), and action menu (visible on hover) |
| **Main Content** | Displays node metadata - a quick-glance summary of the component's configuration |
| **Footer** | Shows the latest run item with its state badge, result, and timestamp |
| **Input Channel** | Handle on the left side; used to subscribe to events from upstream nodes (action components only) |
| **Output Channel(s)** | Handle(s) on the right side; downstream nodes connect here to receive emitted payloads |
| **Action Menu** | Appears on hover in the header; provides node operations (Run, Pause, Duplicate, etc.) |

### Node States

Nodes can be displayed in two visual states:

| State | What's Visible | When to Use |
|-------|----------------|-------------|
| **Expanded** (default) | Header, full metadata, footer with run item | Normal working view; shows all configuration details |
| **Collapsed** | Header only (icon, title, state indicator) | When canvas is crowded; user wants overview |

- Default to **expanded** for newly created nodes
- Respect the user's collapse preference (stored in `node.isCollapsed`)
- Collapsed nodes still show a state indicator (colored dot) for the latest run item

### Icon

Icons are a core part of a node's visual identity and should clearly represent the node's function.

- For **integration components**, always use the official SVG icon of the integration (e.g., GitHub, Slack). Never use generic or substitute icons; source and use the authentic SVG asset to maintain brand and user clarity.
- For **core components**, use an appropriate icon from the [Lucide icon set](https://lucide.dev/) that best conveys the action or meaning (such as filter, clock, hand, etc.).

Do not alter the colors or proportions of official integration SVGs. Style all icons consistently in terms of size and color usage. This ensures a visually coherent and intuitive interface.


### Main Content (Metadata)

The main content area provides at-a-glance information about the node's configuration. This section should be **clear, focused, and uncluttered**. For detailed configuration, users have the sidebar.

#### Design Principles

**The Golden Rule: Less is More**

- Display **0-3 items** maximum
- **Empty is OK** — if there's no critical configuration to show, show nothing
- This area is for **key information at a glance** or **quick actions**
- Everything else belongs in the sidebar

**Distinguish Between Node Instances**

The main content should help users **tell apart multiple instances of the same component** on the canvas. If a user has three GitHub "Run Workflow" nodes, the metadata should make each one identifiable at a glance.

- **Repository** for GitHub components → `superplanehq/superplane`
- **Channel** for Slack components → `#deployments`
- **URL/Endpoint** for HTTP components → `POST api.example.com/webhook`
- **Project** for CI/CD components → `backend-service`

Without this distinguishing information, users would need to click each node to understand which is which.

**Only Show What Matters**

Ask yourself: "Does the user need to see this at a glance to understand what this node does?"

- If YES → Show it in main content
- If NO → It belongs in the sidebar configuration panel

**Hide Empty/Default States**

Never show placeholder or zero-state information:

| Bad | Good |
|-----|------|
| `Filters: 0 items` | *(show nothing if no filters configured)* |
| `Tags: None selected` | *(show nothing if no tags selected)* |
| `Retries: 0` | *(only show if retries are enabled)* |

**Example: Optional Filters**

If a component has optional filters where users can select multiple items:
- **Filters configured**: Show `Filters: staging, production` or use badges `[staging] [production]`
- **No filters configured**: Show nothing — the absence indicates "all items" or "no filtering"

#### Display Patterns

| Pattern | Prop | When to Use |
|---------|------|-------------|
| **Inline metadata** | `metadata: MetadataItem[]` | 1-3 key configuration values |
| **Specs** | `specs: ComponentBaseSpec[]` | Lists of values shown as badges (with optional tooltip for overflow) |
| **Custom fields** | `customField: ReactNode` | Interactive elements (buttons, timers) or complex displays |

#### Inline Metadata

Simple icon + label pairs for the most important configuration:

```
🔗 POST https://api.example.com
⏱ Timeout: 30s
```

**Guidelines:**
- Maximum **3 items** — be selective about what's truly important
- Use clear icons that represent the information type
- Truncate long values — full details are in the sidebar
- Order by importance (most critical first)

#### Specs (Configuration Badges)

Use for displaying lists of selected values as compact badges:

```
[staging] [production]
[User: alice@example.com] [Role: Admin]
```

**Guidelines:**
- Use when users have selected multiple items from a list
- If more than 3-4 badges, show a summary with tooltip for the full list
- **Only show if values are configured** — no badges means no filtering/selection

#### Custom Fields

Interactive UI elements or rich displays that go beyond simple metadata:

**Examples:**
- **Start trigger**: Run buttons for each template
- **Approval**: Approve/Reject buttons (only when awaiting approval)
- **Wait**: Countdown timer display

**Guidelines:**
- Use `customFieldPosition: "before"` for context needed before viewing run items
- Use `customFieldPosition: "after"` (default) for actions related to current state
- Only show interactive elements when they're actionable
- Keep lightweight — if it's complex, consider if it belongs in the sidebar instead

#### Decision Framework

When implementing a component, follow this priority order to decide what appears in main content:

**Priority 1: Primary Target (almost always show)**

What external resource does this node operate on?

- Repository, branch → GitHub components
- Channel → Slack components  
- URL/endpoint → HTTP component
- Project, pipeline → CI/CD components
- Queue, topic → messaging components

This is typically the single most important piece of metadata. If a user has multiple nodes of the same component, this tells them which is which.

**Priority 2: Component Status Information (show when relevant)**

Does the component have dynamic information that helps users understand what happens next?

- Next trigger time → Schedule component ("Next run: Tomorrow at 9:00 AM")
- Time gate status → Time Gate component ("Active: Mon-Fri, 9AM-5PM")

This helps users understand what the component will do without opening the sidebar.

**Note:** Run-specific information like countdown timers, approval progress ("2/3 approved"), or execution duration belongs in the **Footer (Run Item)** section, not here. Main content shows component-level information, not execution-level information.

**Priority 3: Key Behavioral Setting (show if space permits)**

What's the main "how" of this component?

- HTTP method (GET, POST, etc.)
- Wait duration or mode
- Approval requirement (who needs to approve)

Only include if it adds value AND you haven't exceeded 3 items total.

**Priority 4: Active User Configuration (show only if configured)**

Has the user explicitly configured optional settings?

- Filters or selections → Show what's selected, hide if empty
- Tags or labels → Show if applied, hide if none
- Custom parameters → Show if set, hide if default

**Never show:**

- Empty states (`0 items`, `None`, `Not configured`)
- Default values that don't differentiate the node
- Sensitive data (tokens, secrets, credentials)
- Complex expressions (truncate heavily or omit)

**Quick Test**

Before adding something to main content, ask which purpose it serves:

| Purpose | Question to Ask | Example |
|---------|-----------------|---------|
| **Identification** | Does this help distinguish this node from other nodes of the same type? | Repository name, Slack channel, URL endpoint |
| **Quick Context** | Does this give users important status or timing information at a glance? | "Next run: Tomorrow 9:00 AM", Time gate schedule |
| **Quick Action** | Does this let users perform a common action without opening the sidebar? | Start template buttons, Approve/Reject buttons |

If the item doesn't serve any of these three purposes → **Don't show it** (sidebar is fine)

If it does serve one of these purposes → **Consider showing it**, but still respect the 0-3 item limit

#### What to Show

- **Primary target**: URL, repository, channel, project
- **Key behavior**: HTTP method, wait duration, approval requirements
- **Active filters**: Selected items, tags, branches (only if configured)
- **Quick actions**: Template run buttons, approval buttons

#### What NOT to Show

- Empty or zero states (`0 items`, `None selected`)
- Default values that don't add information
- Sensitive values (tokens, secrets, API keys)
- Advanced settings most users don't change
- Long expressions or complex values (truncate or move to sidebar)

### Footer (Latest Run Item)

The footer displays the most recent run item for at-a-glance execution status.

**Contents:**

| Element | Description |
|---------|-------------|
| **State badge** | Colored badge indicating run item state (success, failed, running, waiting, etc.) |
| **Title** | Trigger components generate this title; action components inherit it from the root event (trigger) |
| **Subtitle** | Run item metadata - always includes timestamp, optionally includes execution-specific info |

**Subtitle Guidelines:**

The subtitle (called `subtitle` in code) should always include a timestamp, and optionally short contextual data:

| State | Subtitle Format | Example |
|-------|-----------------|---------|
| **Finished** | Timestamp only (most common) | `5m ago` |
| **Finished with context** | Result + timestamp | `Response: 200 · 5m ago` |
| **Running with progress** | Progress indicator | `1/2 approved` |
| **Running with time** | Remaining/elapsed time | `remaining 13 min`, `running for 2m` |

**Keep subtitles short** — this is at-a-glance information. Detailed execution data belongs in the sidebar's Details tab.

**Guidelines:**
- See [Run Item Display](#run-item-display) for comprehensive subtitle and state guidelines

### Input & Output Channels

**Input Channel (left handle):**
- Only present on action components (not triggers)
- Single input channel per node
- Used to subscribe to events from upstream nodes

**Output Channel(s) (right handle/s):**
- Present on all components
- Can have single or multiple channels
- See [Output Channels](#output-channels) for when to use multiple channels

---

## Component Configuration

Configuration fields define how users set up components. Well-designed configuration improves usability and reduces errors.

### Field Type Selection

Choose the appropriate field type based on the data being collected:

| Field Type | When to Use | Example |
|------------|-------------|---------|
| `string` | Single-line text, short values | API endpoint path, branch name |
| `text` | Multi-line text, longer content | Message body, description |
| `expression` | Values that should reference payload data | Dynamic URLs, conditional values |
| `number` | Numeric values | Timeout seconds, retry count |
| `bool` | On/off toggles | Enable feature, send notification |
| `select` | Single choice from predefined options | HTTP method, environment |
| `multiSelect` | Multiple choices from predefined options | Event types to listen for |
| `list` | Dynamic collection of similar items | Headers, labels, assignees |
| `object` | Structured nested configuration | Advanced settings group |
| `integrationResource` | External resources from integrations | Repository, Slack channel, project |

### Field Behavior Options

Beyond field types, these options control how fields behave:

#### Togglable Fields (`Togglable: true`)

Makes a field optional with an explicit on/off toggle. When toggled off, the field is hidden and not included in configuration.

**When to use:**
- Optional features that add complexity (query params, headers, body)
- Settings most users don't need but power users want
- Features that have meaningful "off" state vs empty state

**Good candidates for togglable:**

| Component | Togglable Fields | Why |
|-----------|------------------|-----|
| HTTP | Query Params, Headers, Body, Retries | Most requests don't need all of these |
| Integrations | Advanced filters, service options | Power user features |
| Triggers | Optional filtering criteria | Start simple, add filtering later |

#### Expression language: `secret()`

Configuration values can reference organization secrets so sensitive data (API keys, tokens) is not stored in the canvas. Use the expr function `secret("name", "key")` inside `{{ }}` to resolve a secret key’s value (e.g. `{{ secret("my-api", "api_key") }}`). Secrets are resolved at execution time and are scoped to the organization. If the builder has no secret context (e.g. some gRPC paths), expressions that use `secret()` will fail to compile.

#### Disabling Expressions (`DisallowExpression: true`)

Prevents users from using `{{ }}` expressions in a field. The field will only accept static values.

**When to use:**
- Field values that must be known at configuration time (not runtime)
- Technical identifiers that shouldn't be dynamic
- Fields where expressions would cause errors or security issues

**Good candidates for DisallowExpression:**

| Field Type | Allow Expressions? | Why |
|------------|-------------------|-----|
| Header names | No | Names must be static for HTTP protocol |
| Header values | Yes | Values often come from payload data |
| Field keys (in key-value pairs) | No | Keys define structure, should be static |
| Field values (in key-value pairs) | Yes | Values are often dynamic |
| Resource type identifiers | No | Affects parsing/routing logic |

### Required vs Optional Fields

- **Required**: Fields essential for the component to function
- **Optional**: Fields that enhance behavior but have sensible defaults

**Guidelines:**
- Minimize required fields to reduce setup friction
- Every optional field should have a sensible default
- Use visibility conditions to hide advanced options until needed

### Default Values

**Core Principle: It's easier to edit existing configuration than to create from scratch.**

Ideally, every field should have a sensible default value — not just a placeholder. Users should be able to drop a component on the canvas and have it work (or nearly work) immediately, then customize as needed.

#### Provide Defaults for Common Patterns

If a field has a commonly-used value, make it the default:

| Field | Default | Reasoning |
|-------|---------|-----------|
| Branch | `main` | Most common branch name |
| HTTP Method | `POST` | Most common for webhooks/APIs |
| Timeout | `30s` | Reasonable default for most requests |
| Approval items | One item: "Any user" | Allows immediate testing, user adds specific approvers |
| Retry count | `3` | Common retry pattern |

#### Pre-populate Lists

For list fields (headers, approvers, filters), **pre-populate with one sensible item** rather than starting empty.

| Approach | User Experience |
|----------|-----------------|
| **Pre-populated** (recommended) | User sees example item, can edit or add more |
| **Empty list** (avoid) | User must figure out format and manually add first item |

**Examples:**
- Approval: Start with one "Any user" approver → user customizes
- Headers: Start with one common header → user adds more
- Filters: Start with one example filter → user modifies

**Why this matters:**
- Reduces clicks/effort to get started
- Shows users the expected format
- Makes the component immediately testable

#### When Empty Defaults Are OK

Some fields are inherently user-specific and have no universal default:

| Field Type | Why No Default | Use Instead |
|------------|----------------|-------------|
| Repository | Unique to user's setup | Placeholder: `owner/repo` |
| URL/Endpoint | Unique to integration | Placeholder: `https://api.example.com/webhook` |
| Slack Channel | Unique to workspace | Placeholder: `#channel-name` |
| Credentials | Unique and sensitive | Placeholder explaining format |

For these fields, provide a helpful **placeholder** that guides users on the expected format without being an actual default value.

#### Defaults Summary

| Scenario | Approach |
|----------|----------|
| Common pattern exists | Provide default value |
| List/collection field | Pre-populate with one example item |
| User-specific value | Empty default + helpful placeholder |
| On/off feature | Default to most common state |

**Avoid:**
- Defaults that trigger on every event (noisy)
- Empty lists when a pre-populated example would help
- Defaults that require immediate modification to work

### Visibility Conditions

Visibility conditions control when a field is shown or hidden based on the value of another field. This keeps the UI clean by only showing fields relevant to the user's choices.

#### Example: HTTP Component

| User Selection | Fields Shown | Fields Hidden |
|---------------|--------------|---------------|
| Method: GET | URL, Headers, Query Params | Body, Content Type |
| Method: POST | URL, Headers, Query Params, Body, Content Type | — |
| Method: POST + Content-Type: JSON | All above + JSON editor | Form fields, XML editor |
| Method: POST + Content-Type: Form | All above + Form fields | JSON editor, XML editor |

**How it works:**
- Body field only appears for POST, PUT, PATCH (not needed for GET, DELETE)
- JSON Payload field only appears when Method is POST/PUT/PATCH **AND** Content-Type is JSON
- Each choice progressively reveals relevant options

**Why this matters:**
- Keeps the UI clean — users only see fields relevant to their choices
- Reduces confusion — no need to explain "ignore this field if using GET"
- Progressive disclosure — start simple, reveal complexity as needed

**Guidelines:**
- Hide advanced options until they're relevant
- Group related conditional fields together
- Design the "happy path" to require minimal configuration

### Field Naming

- Use `camelCase` for field names in JSON
- Use clear, concise names that describe the value
- Avoid abbreviations unless universally understood

### Validation

Implement validation in `Setup()` to catch configuration errors early:

- Validate required fields are present
- Validate field formats (URLs, expressions, etc.)
- Validate integration resources are accessible
- Return clear, actionable error messages

---

## Output Channels

Output channels help users **model different paths in their workflow** based on component outcomes.

### Why Output Channels Exist

Users can always use **Filter** or **If** components to parse payload data and route traffic. However, if a component has **obvious outcome states** that users would naturally want different paths for, these should be built-in output channels.

**The question to ask:** *"Will most users want to do something different based on this outcome?"*

- If YES → Make it an output channel
- If NO → Single default channel is fine (users can filter if needed)

### Examples

| Component | Channels | Why Multiple Channels |
|-----------|----------|----------------------|
| GitHub Run Workflow | `passed`, `failed` | Users almost always want different handling for failed CI |
| Merge | `success`, `timeout`, `stopped` | Each outcome requires different follow-up actions |
| Approval | `approved`, `rejected` | Obvious decision paths |
| Dash0 List Issues | `clear`, `degraded`, `critical` | Different severity = different response |
| HTTP Request | `default` only | Success/failure is in payload; users may define "success" differently |

### When to Use Multiple Channels

**Use multiple channels when:**
- The outcome clearly falls into distinct categories
- Most users would want different downstream actions for each outcome
- The distinction is fundamental to the component's purpose

**Use single default channel when:**
- There's only one logical outcome path
- Success/failure criteria vary by user (let them use Filter)
- The component transforms data without branching logic

### Channel Naming

- Use lowercase, single-word names: `passed`, `failed`, `approved`, `timeout`
- Names should clearly indicate the outcome
- Be consistent with existing patterns in similar components

### Design Considerations

- Users must explicitly choose which channel to subscribe to
- Each channel can have different subscribers
- Fewer channels = simpler workflow design; only add channels that provide clear value

---

## Run Item Display

Run items represent individual executions displayed on nodes and in the sidebar. The display is controlled by **Component Mappers** in the frontend.

### Component Mappers

A **ComponentBaseMapper** defines how a component's run items are displayed. Each mapper provides:

| Function | Purpose |
|----------|---------|
| `props()` | Returns display properties (icon, title, metadata, event sections) |
| `subtitle()` | Returns the subtitle text for a run item |
| `getExecutionDetails()` | Returns data for the Details tab |

Mappers are registered in `web_src/src/pages/workflowv2/mappers/index.ts`.

### State Registry

Each component has a **State Registry** that controls:
1. **Which states exist** — The possible states for this component
2. **How states look** — Icon, colors, badge styling for each state
3. **When to show each state** — Logic to determine current state from execution data

Most components use the default states. Only create custom states when the defaults don't capture meaningful distinctions.

For implementation details, see [Component Customization Guide](component-customization.md#example-1-creating-a-custom-state-registry-approval-component).

### States

#### Default States

Most components use these default states (from `DEFAULT_EVENT_STATE_MAP`):

| State | Icon | Color | When to Use |
|-------|------|-------|-------------|
| `running` | loader | blue | Execution in progress |
| `success` | circle-check | green | Completed successfully |
| `failed` | circle-x | red | Completed with expected failure (component executed, outcome was failure) |
| `error` | triangle-alert | red | Unexpected error (component failed to execute) |
| `cancelled` | ban | gray | User cancelled execution |
| `neutral` | circle | gray | No execution yet |

#### Critical: Error vs Failed

**Do not confuse these two states — they mean different things:**

| State | Meaning | Example |
|-------|---------|---------|
| `failed` | **Expected outcome** — Component executed successfully, but the result was a failure | HTTP request returned 404, Filter didn't match, CI pipeline failed |
| `error` | **Unexpected failure** — Component could not complete execution | Network timeout, API credentials invalid, Internal exception |

**Key distinction:**
- `failed` = "I did my job, and the answer was 'no'"
- `error` = "I couldn't do my job"

#### Error State is Required

**Every component MUST support the `error` state.** If something goes wrong during execution (network issues, invalid credentials, unexpected exceptions), the component must be able to show the `error` state.

Components cannot exist without error handling. Even if your component only has happy-path custom states (like `approved`, `waiting`), the `error` state must still be available for when things go wrong.

#### When to Use Custom States

Add custom states when the default states don't capture meaningful distinctions for your component:

| Component | Custom States | Why Needed |
|-----------|---------------|------------|
| Approval | `waiting`, `approved`, `rejected` | Distinguishes "waiting for response" from "approved" vs "rejected" outcomes |
| Wait | `finished`, `pushed through` | Distinguishes natural completion from manual override |
| Filter | `passed`, `filtered` | Shows whether event passed or was filtered out |
| Dash0 List Issues | `clear`, `degraded`, `critical` | Shows issue severity level from monitoring |

**Guidelines for custom states:**
- Only add states that provide meaningful information to users
- Always extend `DEFAULT_EVENT_STATE_MAP` (don't replace it) — this ensures `error` state is always available
- Follow the state function pattern: check errors → cancellation → running → specific outcomes
- Choose icons and colors that clearly communicate the state

### Title

Each run item displays two pieces of identifying information:

| Element | Source | Styling | Example |
|---------|--------|---------|---------|
| **Short Run ID** | System-generated (truncated event ID) | Gray text | `a1b2c3d4` |
| **Title Text** | Trigger's `getTitleAndSubtitle()` function | Black text | `fix: resolve timeout issue` |

These come from different sources and are styled differently to create visual hierarchy — the title (black) is the primary identifier, while the run ID (gray) is secondary reference information.

#### Title Sources

| Component Type | Title Source |
|----------------|--------------|
| **Trigger components** | Generated by the trigger's `getTitleAndSubtitle()` function |
| **Action components** | Inherited from the root event (the trigger that started the run) |

#### Title Examples

Triggers should generate meaningful titles that describe the event:

| Trigger | Title Example | What it shows |
|---------|---------------|---------------|
| GitHub onPush | `fix: resolve connection timeout` | Commit message (truncated) |
| Slack onAppMention | `Hey @bot can you deploy...` | Message text (truncated) |
| PagerDuty onIncident | `P123 - Database timeout` | Incident ID + title |
| Semaphore onPipelineDone | `build-and-test` | Pipeline name |
| Schedule | `Jan 28, 2026, 9:00:00 AM` | Formatted trigger time |

### Subtitle (Run Item Metadata)

The subtitle shows brief status information. It should **always include a timestamp** and optionally include execution-specific context.

#### Subtitle Patterns by State

| State | Pattern | Example |
|-------|---------|---------|
| **Running (with progress)** | Progress + timestamp | `1/2 approved · 5m ago` |
| **Running (with duration)** | Elapsed time | `running for 2m 30s` |
| **Finished (simple)** | Timestamp only | `5m ago` |
| **Finished (with result)** | Result + timestamp | `Response: 200 · 5m ago` |
| **Finished (with outcome)** | Outcome + timestamp | `Approved · 5m ago` |
| **Cancelled** | Timestamp | `Cancelled · 1h ago` |

#### Subtitle Guidelines

- **Always include timestamp** — users need to know when things happened
- **Keep it short** — aim for under 20 characters
- **Show the most relevant info** — what does the user need to know at a glance?
- **Use `formatTimeAgo()`** — consistent relative time formatting
- **Progress for running states** — helps users understand how much is done

### Automatic Time Updates

For running states, components can enable live-updating duration display that refreshes automatically without page reload.

**Examples:**
- Wait component: `remaining 4m 32s` → `remaining 4m 31s` → `remaining 4m 30s`
- Long-running workflow: `running for 2m 15s` → `running for 2m 16s`

Use this when elapsed or remaining time is meaningful to the user.

---

## Run Item Action Menu

Run items have their own action menu separate from node actions. These actions operate on a specific execution.

### Run Item Actions

| Action | When Available | Purpose |
|--------|----------------|---------|
| **Cancel** | When execution is running | Stop a running execution |
| **Re-Emit** | On completed trigger events | Replay an event to re-trigger the workflow |
| **Push Through** | When component supports it (e.g., Wait) | Force a waiting execution to proceed immediately |

### Queue Item Actions

Items waiting in the queue also have actions:

| Action | Purpose |
|--------|---------|
| **Cancel** | Remove item from queue without executing |

### Custom Actions

Some components define additional actions specific to their behavior:

| Component | Custom Actions | When Available |
|-----------|----------------|----------------|
| Approval | Approve, Reject | When awaiting approval |
| Wait | Push Through | When waiting |

**Guidelines:**
- Actions should be verbs (Approve, Reject, Cancel)
- Only show actions when they're actionable
- Actions appear in the run item's dropdown menu

---

## Run Item in Sidebar

Run items appear in the sidebar in two contexts:

1. **Run History** — List of past executions for a component node
2. **Single Run Page (Chain)** — When viewing a specific workflow run, showing all component executions in the chain

### Consistency Across Views

**Important:** Run items must look identical across all three locations:
- Node footer (on canvas)
- Sidebar run history
- Single run page (chain view)

The same rules apply everywhere:
- Same states and state icons
- Same title (from trigger)
- Same subtitle format (always includes timestamp)
- Same action menu options

This consistency ensures users can recognize and understand run items regardless of where they see them.

### Where Run Items Appear

| Context | What User Sees | When |
|---------|----------------|------|
| **Run History** | List of run items for selected node | Click on a component node on canvas |
| **Chain View** | All run items in execution order | Open a specific workflow run |

### Run Item List View

In both contexts, run items are displayed as a list. Each item shows:

| Element | Description | Example |
|---------|-------------|---------|
| **State icon** | Visual indicator of execution state | Green checkmark, red X, spinning loader |
| **Title** | Event identifier (from trigger) | `fix: resolve timeout issue` |
| **Subtitle** | Timestamp + optional context | `5m ago` or `1/2 approved · 5m ago` |

**List View Guidelines:**
- Items are sorted newest first
- Running items appear at the top
- State icon provides instant visual status
- Title helps identify which event this execution belongs to
- Subtitle gives timing and brief status context

### Single Run Item View

When a user clicks on a run item, the sidebar expands to show detailed information:

| Section | Content |
|---------|---------|
| **Header** | State badge, title, short run ID |
| **Details Tab** | Execution details (see below) |
| **Payload Tab** | Payload data emitted |

### Details Tab

The Details tab provides **human-readable execution details** in the context of the component. While the Payload tab contains all raw execution data, it's not easy to consume — the Details tab makes key information accessible and actionable.

#### Purpose

- Help users understand what happened during execution
- Provide useful data like links to external resources
- Present information in a component-specific, meaningful way
- **Never be empty** — always show at least the timestamp

#### Timestamps

Every Details tab should start with a timestamp:

| Component Type | Timestamp Label | Why |
|----------------|-----------------|-----|
| Most components | "Started At" | Standard execution start |
| Filter/If | "Evaluated At" | More contextual — it's an evaluation, not a long-running process |
| Schedule | "Triggered At" | It's a trigger event |

**For longer-running components** (Wait, Approval, HTTP with retries), also include:
- "Finished At" — when execution completed
- Duration may be shown inline or calculated

#### Content Guidelines

| Guideline | Description |
|-----------|-------------|
| **2-5 items** | Reasonable amount of information |
| **Never exceed 10** | Too much information becomes overwhelming |
| **Never empty** | Always at least show timestamp |
| **Include links** | Link to external resources whenever possible |
| **No raw data dumps** | Curate what's shown, don't just dump payload |
| **Don't duplicate state** | State is already shown on the run item — don't repeat it |

**Note on state:** The Details tab should not display the execution state (e.g., "Rejected") since that's already visible on the run item. However, it *should* show **how that state was reached**. For example, an Approval component with state "Rejected" should show the timeline of individual decisions (Alice: Approved, Bob: Rejected) that led to the final state.

#### Examples by Component

| Component | Details Content | Why This Information |
|-----------|-----------------|----------------------|
| **Approval** | Timeline with actors, decisions, comments | Users need to see who approved/rejected and when |
| **GitHub Run Workflow** | Workflow name, run URL (link), status | Users want to jump to GitHub to see details |
| **HTTP** | Response status, URL called, retry count | Users need to see what was called and result |
| **Filter** | Expression result, evaluated values | Users need to understand why event passed/filtered |
| **Wait** | Wait duration, finish reason, actor (if pushed) | Users need to know if natural or manual completion |

#### Special Display Types

The Details tab supports rich formatting:

| Type | Use Case | Example |
|------|----------|---------|
| **Links** | External resources | GitHub run URL, Slack message link |
| **Timeline** | Multi-step processes | Approval history with actors and timestamps |
| **Evaluation badges** | Filter/If results | Shows values with pass/fail styling |
| **Error** | Error messages | Red styling, always shown last |

#### Error Display

When an execution fails, the error message in the Details tab is often the user's primary way to understand what went wrong.

**Error display requirements:**
- **Clear visibility** — Use distinct red styling so errors stand out
- **Fully readable** — Never truncate error messages; users need the complete text
- **Easy to copy** — Users often need to copy error messages for debugging or reporting
- **Always last** — Position errors at the bottom so they don't push other details out of view

The Details tab is where users come to diagnose problems — make errors impossible to miss and easy to read.

#### Field Ordering

1. **Timestamp** — When it started (and finished, if applicable)
2. **Primary result** — The main outcome (status, decision, link to resource)
3. **Supporting details** — Additional context (2-3 items max)
4. **Error** — Always last (if present)

For implementation details, see [Component Customization Guide](component-customization.md).

---

## Error Handling

Proper error handling ensures users understand what went wrong and how to fix it.

**Remember:** This section is about the `error` state (component couldn't execute), not the `failed` state (component executed but outcome was failure). See [States](#states) for the distinction.

### Error Types

| Type | When to Use | User Experience |
|------|-------------|-----------------|
| **Configuration error** | Invalid setup (missing fields, bad values) | Shown during workflow save/validation |
| **Execution error** | Component couldn't complete (network, auth, exception) | Shown in run item with `error` state |
| **Transient error** | Temporary issue (network timeout, rate limit) | Retry if configured, then show as `error` |

### Error Message Guidelines

| Guideline | Good Example | Bad Example |
|-----------|--------------|-------------|
| Be specific | `"Failed to create issue: repository 'org/repo' not found"` | `"Error occurred"` |
| Be actionable | `"API key expired. Update in integration settings."` | `"Authentication failed"` |
| Hide internals | `"Could not connect to GitHub"` | `"ECONNREFUSED 127.0.0.1:443"` |
| Be concise | `"Request timeout after 30s"` | `"The request to the external service did not complete within the configured timeout period of 30 seconds"` |

### Error Display

Errors appear in:
1. **Run item state** — Shows `error` state with red triangle icon
2. **Subtitle** — Brief error indication (e.g., `Error · 5m ago`)
3. **Details tab** — Full error message at the bottom of details

For implementation details, see [Component Implementation Patterns](component-implementations.md).

---

## Emitted Payload Structure

Payloads are the data components emit for downstream nodes to consume via expressions.

### Payload Format

All payloads have three parts:

| Part | Description | Example |
|------|-------------|---------|
| **data** | The actual content | `{ "issue": { "number": 123, "title": "Bug fix" } }` |
| **timestamp** | When it was emitted | `2026-01-28T10:30:00.000Z` |
| **type** | Payload type identifier | `github.issue` |

### Type Naming

| Pattern | Example | Use Case |
|---------|---------|----------|
| `integration.resource` | `github.issue`, `slack.message` | Integration resources |
| `integration.event` | `github.push`, `slack.appMention` | Trigger events |
| `component.result` | `http.response`, `filter.result` | Component outputs |

### Content Guidelines

| Do | Don't |
|----|-------|
| Include data users need in expressions | Include sensitive data (tokens, secrets) |
| Include resource identifiers | Create deeply nested structures |
| Include status/result information | Include internal implementation details |
| Keep structure flat and accessible | Create extremely large payloads |

### Expression-Friendly Design

Users access payload data via expressions like `$['Create Issue'].issue.number`.

| Structure | Expression | Verdict |
|-----------|------------|---------|
| `{ "issue": { "number": 123 } }` | `$['Node'].issue.number` | Good — direct access |
| `{ "result": { "data": { "issue": { "number": 123 } } } }` | `$['Node'].result.data.issue.number` | Bad — too nested |

**Rule of thumb:** If the expression path has more than 3 levels, the payload structure is too deep.

### External API Responses

For data received from external APIs (GitHub, Slack, PagerDuty, etc.), we're not always in control of the payload structure. In these cases:

- **Consider preserving the original format** — Users familiar with the external API may expect a certain structure
- **Don't alter just to flatten** — Changing the structure can confuse users who reference external documentation
- **Case-by-case decision** — Sometimes it's better to keep the API's native format even if it's deeply nested

For example, if GitHub returns `{ "pull_request": { "head": { "ref": "feature-branch" } } }`, it may be better to keep that structure rather than flattening to `{ "headRef": "feature-branch" }` — users reading GitHub's API docs will expect the original format.

For implementation details, see [Component Implementation Patterns](component-implementations.md).

---

## Summary Checklist

Use this checklist when designing or reviewing components.

### Component Node Display

- [ ] Main content shows 0-3 key items (empty is OK if nothing critical)
- [ ] Displayed info distinguishes this node from other instances of same component
- [ ] Specs used for lists/structured data, custom fields for interactive UI
- [ ] Node uses integration icon and color consistently

### Configuration

- [ ] Field types match the data being collected
- [ ] Sensible defaults that cover common use cases
- [ ] `Togglable` used for optional feature sections
- [ ] `DisallowExpression` used for static values (e.g., header names)
- [ ] Visibility conditions hide advanced options appropriately

### Run Item Display

- [ ] Title comes from trigger (meaningful event description)
- [ ] Subtitle shows timestamp + optional brief context
- [ ] Running states show live duration updates

### States

- [ ] `error` state exists and is reachable (required for all components)
- [ ] `error` vs `failed` used correctly (error = couldn't execute, failed = executed with failure outcome)
- [ ] Uses default states where applicable
- [ ] Custom states provide meaningful distinction users care about
- [ ] State styling (icon, colors) is visually clear

### Output Channels

- [ ] Single channel unless users clearly need different paths
- [ ] Multiple channels for obvious outcome states (pass/fail, approved/rejected)
- [ ] Channel names are lowercase, single-word, descriptive

### Error Handling

- [ ] Error messages are specific and actionable
- [ ] No internal details exposed to users
- [ ] Errors appear in Details tab

### Run Item in Sidebar

- [ ] Run items look identical across node footer, run history, and chain view
- [ ] Same states, title, subtitle format across all views

### Details Tab

- [ ] Never empty — always shows at least timestamp
- [ ] 2-5 items of information (never exceed 10)
- [ ] Includes links to external resources where possible
- [ ] Doesn't duplicate state (shows how state was reached instead)
- [ ] Contextual timestamp labels when appropriate (e.g., "Evaluated At" for Filter)
- [ ] Error messages are fully visible (not truncated) and shown last

### Payload Structure

- [ ] Follows `{ data, timestamp, type }` format
- [ ] Type follows naming convention (`integration.resource`)
- [ ] Structure is flat (≤3 levels for expression access)
- [ ] No sensitive data (tokens, secrets)

---

## References

- [Integration Development Guide](integrations.md) - Creating new integrations
- [Component Implementation Patterns](component-implementations.md) - Backend implementation best practices
- [Component Customization Guide](component-customization.md) - Frontend customization patterns
