# JavaScript Components

## Overview

JavaScript components are workflow components written in JavaScript that are loaded from files
on disk and executed at runtime by an embedded JavaScript engine. Unlike built-in Go components,
which are compiled into the binary and registered via `init()` functions, JavaScript components
are plain `.js` files in a configurable directory that SuperPlane discovers and loads
dynamically — both at startup and while the application is running.

Today, extending SuperPlane with custom logic requires one of two approaches:

- **Building a native Go component**: This requires knowledge of Go, access to the SuperPlane
  codebase, and a full redeploy for every change. The barrier to entry is high for teams that
  don't maintain their own SuperPlane fork.
- **Using the HTTP component**: Teams can call out to an external service that contains their
  custom logic. This works but introduces operational overhead — a separate service to deploy,
  monitor, and maintain — for what is often a small amount of transformation or decision logic.

JavaScript components solve this by allowing operators to drop `.js` files into a directory and
have them appear as first-class components in SuperPlane. They run in-process with well-defined
inputs and outputs, using the same execution model as built-in components. No recompilation, no
external services, no database migrations.

## Goals

1. Load JavaScript components from `.js` files on disk at server startup and dynamically while
   the application is running (new files are picked up, changed files are reloaded, deleted
   files are unregistered).
2. Execute them in-process using an embedded JavaScript runtime (goja) — no external
   dependencies.
3. Provide a JavaScript SDK that maps to the existing `core.Component` interface, giving JS
   components access to execution state, metadata, HTTP calls, secrets, and output channels.
4. Register JS components in the existing registry so they appear alongside built-in components
   in the canvas and can be used in workflows identically.
5. Enforce resource limits (execution time, memory) to prevent a single component from
   degrading the system.

## Non-Goals

- **Full Node.js compatibility**: The embedded runtime supports standard ECMAScript but not
  Node.js-specific APIs (`fs`, `net`, `child_process`, etc.). Components that need rich runtime
  capabilities should use the Daytona integration or the HTTP component.
- **npm package management**: No package manager or dependency resolution. Users write
  self-contained scripts. Third-party library support can be explored later.
- **TypeScript execution**: The runtime executes JavaScript only. TypeScript support
  (transpilation before loading) can be added as a future enhancement.
- **JavaScript triggers**: This PRD covers components only. JavaScript-based triggers follow a
  different lifecycle and will be addressed separately.
- **Web-based editing**: Components are managed as files (version-controlled, deployed via CI,
  edited in any text editor). There is no in-app code editor.

## Architecture Decision: Runtime

### Options Considered

**Option A — Embedded goja runtime**: JavaScript is executed directly inside the Go process
using [goja](https://github.com/dop251/goja), a pure-Go ECMAScript 5.1+ runtime. The Go host
creates a JS VM per execution, injects the SDK as global functions, runs the user's code, and
collects the result. No network hop, no external dependency, no cgo.

**Option B — External sandbox service**: JavaScript is sent to an external execution service
(e.g., Daytona sandboxes, Deno Deploy, or a custom container-based executor). The component
sends code + input over HTTP and receives the output.

**Option C — Embedded V8 via cgo**: Use a V8 binding (e.g., `rogchap/v8go`) for full ES2023+
support and higher performance. Requires cgo, which complicates the build and
cross-compilation.

### Decision: Option A (goja)

**Zero operational overhead.** No additional services to deploy, scale, or monitor. JavaScript
execution happens in the same process as all other component execution.

**Latency.** In-process execution eliminates the network round-trip. For lightweight
transformation or decision logic — the primary use case — execution stays in the low
milliseconds.

**Build simplicity.** goja is a pure-Go library with no cgo dependency. It doesn't change the
build or deployment process.

**Sufficient language support.** goja supports ECMAScript 5.1 with many ES6+ features (arrow
functions, let/const, template literals, destructuring). The target use case — data
transformation, conditional logic, API calls — doesn't require cutting-edge language features.

### Trade-offs Accepted

- **No full ES2023+ support**: Some modern JS features may not be available. We document
  supported features and provide clear error messages for unsupported syntax.
- **Single-threaded execution**: goja runs JavaScript synchronously on a single goroutine.
  Long-running computations block the goroutine. We mitigate this with execution timeouts.
- **No native async/await**: HTTP calls from JS components go through the Go host's HTTP
  context synchronously.
- **Memory overhead per VM**: Each execution creates a new goja VM instance. For high-throughput
  scenarios, this may need pooling in the future.

## Detailed Design

### File-Based Loading

JavaScript components live as `.js` files in a directory configured via the
`SUPERPLANE_JS_COMPONENTS_DIR` environment variable. At server startup, SuperPlane scans this
directory and registers each file as a component.

#### Directory Structure

```
js_components/
├── transform.js
├── slack-notify.js
└── validate-payload.js
```

Each `.js` file is one component. The file name (without extension) is used as the component's
registry name, prefixed with `js.` (e.g., `transform.js` becomes `js.transform`). File names
must be lowercase alphanumeric with hyphens (e.g., `my-component.js`).

#### Loading Behavior

On startup, and continuously while the application is running, SuperPlane watches the
configured directory for changes and keeps the registry in sync.

**Initial load (startup):**

1. Read `SUPERPLANE_JS_COMPONENTS_DIR` (default: no JS components loaded if unset).
2. Scan the directory for `*.js` files (non-recursive).
3. For each file:
   a. Read the source code.
   b. Create a temporary goja VM and execute the script to extract the component definition
      (metadata, configuration fields, output channels).
   c. Validate that the script calls `superplane.component()` with at least an `execute`
      handler.
   d. Register a `JSComponentAdapter` in the registry under the `js.<filename>` name.
4. Log the number of JS components loaded and their names.
5. If a file fails to parse or validate, log an error and skip it (don't crash the server).
6. Start the file watcher.

**File watcher (runtime):**

After the initial load, a background goroutine watches the directory for file system events
using a polling interval (default: 5 seconds). On each tick it detects three types of changes:

| Change           | Behavior                                                              |
|------------------|-----------------------------------------------------------------------|
| **New file**     | Parse and validate the file. If valid, register it in the registry.   |
| **Modified file**| Re-parse the file and replace the adapter in the registry. In-flight  |
|                  | executions using the old version complete normally; new executions use |
|                  | the updated code.                                                     |
| **Deleted file** | Unregister the component from the registry. Existing nodes that       |
|                  | reference it will fail on their next execution with a "component not  |
|                  | found" error.                                                         |

Change detection uses file modification timestamps. The watcher compares the current directory
state against the previously known state on each tick.

If a modified or new file fails to parse or validate, the watcher logs an error and keeps the
previous version (for modifications) or skips the file (for new files). A bad file never takes
down a previously working component.

#### Configuration

| Environment Variable            | Default | Description                                    |
|---------------------------------|---------|------------------------------------------------|
| `SUPERPLANE_JS_COMPONENTS_DIR`  | (unset) | Path to directory containing `.js` components. |
| `SUPERPLANE_JS_WATCH_INTERVAL`  | `5s`    | Polling interval for detecting file changes.   |

When `SUPERPLANE_JS_COMPONENTS_DIR` is unset or empty, no JavaScript components are loaded,
no watcher is started, and the feature is effectively disabled.

### JavaScript SDK

The runtime exposes a global `superplane` object. The user's code calls
`superplane.component()` with a definition object that declares metadata and handler functions.

#### Component Structure

```javascript
superplane.component({
  name: "transform",
  label: "Transform Data",
  description: "Filter and reshape incoming event data",
  icon: "shuffle",
  color: "blue",

  configuration: [
    { name: "filterField", label: "Filter Field", type: "string", required: true },
    { name: "filterValue", label: "Filter Value", type: "string", required: true },
  ],

  outputChannels: [
    { name: "default", label: "Default" },
  ],

  setup(ctx) {
    if (!ctx.configuration.filterField) {
      throw new Error("filterField is required");
    }
  },

  execute(ctx) {
    const data = ctx.input;
    const config = ctx.configuration;

    const result = {
      total: data.items.length,
      filtered: data.items.filter(item => item[config.filterField] === config.filterValue),
    };

    ctx.emit("default", "transform.result", result);
  },
});
```

#### Component Definition Fields

| Field              | Type       | Required | Description                                          |
|--------------------|------------|----------|------------------------------------------------------|
| `name`             | `string`   | No       | Override for the component name (defaults to filename)|
| `label`            | `string`   | Yes      | Human-readable display name for the UI.              |
| `description`      | `string`   | Yes      | Description of what the component does.              |
| `icon`             | `string`   | No       | Lucide icon name (default: `"code"`).                |
| `color`            | `string`   | No       | Color for the UI (default: `"blue"`).                |
| `configuration`    | `array`    | No       | Configuration field definitions.                     |
| `outputChannels`   | `array`    | No       | Output channels (default: single "default" channel). |
| `setup`            | `function` | No       | Called when the node is saved to validate config.     |
| `execute`          | `function` | Yes      | Called when the component should run.                 |

Configuration fields use the same schema as built-in components (`configuration.Field`), so
the existing configuration UI (field renderers, visibility conditions, expression support)
works without modification.

#### Execution Context (`ctx` in `execute`)

| Property / Method                       | Maps to (Go)                    |
|-----------------------------------------|---------------------------------|
| `ctx.id`                                | `ExecutionContext.ID`            |
| `ctx.workflowId`                        | `ExecutionContext.WorkflowID`    |
| `ctx.nodeId`                            | `ExecutionContext.NodeID`        |
| `ctx.input`                             | `ExecutionContext.Data`          |
| `ctx.configuration`                     | `ExecutionContext.Configuration` |
| `ctx.emit(channel, type, data)`         | `ExecutionState.Emit()`         |
| `ctx.pass()`                            | `ExecutionState.Pass()`         |
| `ctx.fail(reason, message)`             | `ExecutionState.Fail()`         |
| `ctx.metadata.get()`                    | `Metadata.Get()`                |
| `ctx.metadata.set(value)`               | `Metadata.Set()`                |
| `ctx.http.request(method, url, opts)`   | `HTTP.Do()`                     |
| `ctx.secrets.getKey(secret, key)`       | `Secrets.GetKey()`              |
| `ctx.log.info(message)`                 | `Logger.Info()`                 |
| `ctx.log.error(message)`               | `Logger.Error()`                |
| `ctx.eval(expression)`                  | `ExpressionEnv()`               |

#### Setup Context (`ctx` in `setup`)

| Property / Method                       | Description                      |
|-----------------------------------------|----------------------------------|
| `ctx.configuration`                     | Configuration values to validate |
| `ctx.http.request(method, url, opts)`   | Make an HTTP request             |
| `ctx.metadata.get()`                    | Get node metadata                |
| `ctx.metadata.set(value)`               | Set node metadata                |

#### HTTP Request API

```javascript
const response = ctx.http.request("POST", "https://api.example.com/data", {
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ key: "value" }),
  timeout: 10000,
});

// response.status  -> 200
// response.headers -> { "content-type": "application/json" }
// response.body    -> parsed JSON or string
```

The `ctx.http.request()` function is synchronous from the JavaScript perspective but delegates
to the Go `HTTPContext.Do()` under the hood.

### Runtime Execution Model

When the node executor encounters a JavaScript component, it follows this flow:

1. **Look up the JSComponentAdapter** from the registry.
2. **Create a new goja VM instance** for this execution.
3. **Inject the SDK** — register the `superplane` global object with Go callback functions that
   delegate to the real `ExecutionContext`.
4. **Run the component's source code** — this registers the handler via
   `superplane.component()`.
5. **Call the handler's `execute` function** with the execution context.
6. **Collect the result** — the script either calls `ctx.emit()`, `ctx.pass()`, or
   `ctx.fail()`, or throws an error.
7. **Tear down the VM** — the goja runtime is garbage-collected.

If the script exceeds the execution timeout, the VM is forcefully interrupted and the execution
is failed with an error.

#### Integration with the Registry

JavaScript components are registered at startup and kept in sync by the file watcher. Go
components register via `init()` at compile time; JS components register and update at runtime
as files appear, change, or are removed.

```
Server starts
  → Initial load: scan directory, register all valid .js files
  → Start file watcher goroutine

File watcher (every SUPERPLANE_JS_WATCH_INTERVAL):
  → New file detected      → registry.RegisterComponent("js.transform", adapter)
  → File modified detected → registry.RegisterComponent("js.transform", newAdapter)
  → File deleted detected  → registry.UnregisterComponent("js.transform")
```

After registration, JS components are indistinguishable from Go components in the registry.
`registry.GetComponent("js.transform")` works exactly like `registry.GetComponent("http")`.

#### Integration with the Node Executor

The node executor (`pkg/workers/node_executor.go`) requires no changes. It already calls
`registry.GetComponent(name)` and then `component.Execute(ctx)`. The `JSComponentAdapter`
handles translation to JavaScript execution internally.

### Resource Limits

| Limit                    | Default    | Configurable          |
|--------------------------|------------|-----------------------|
| Execution timeout        | 30 seconds | Yes (env var)         |
| Memory limit             | 64 MB      | Yes (env var)         |
| HTTP requests per exec   | 10         | Yes (env var)         |
| HTTP request timeout     | 10 seconds | Yes (env var)         |
| Code size per file       | 256 KB     | No                    |

When a limit is exceeded, the execution is failed with a clear error message indicating which
limit was hit.

| Environment Variable                | Default |
|-------------------------------------|---------|
| `SUPERPLANE_JS_EXECUTION_TIMEOUT`   | `30s`   |
| `SUPERPLANE_JS_MEMORY_LIMIT`        | `64MB`  |
| `SUPERPLANE_JS_MAX_HTTP_REQUESTS`   | `10`    |
| `SUPERPLANE_JS_HTTP_TIMEOUT`        | `10s`   |

### Canvas Integration

JavaScript components appear in the canvas component picker alongside built-in ones, under a
"Custom" section. They behave identically to built-in components:

- Configuration panel in the sidebar with the fields defined in the JS file.
- Run history with state badges, subtitles, and details.
- Output channels for downstream connections.

Canvas nodes reference JavaScript components using the existing `Ref` structure:

```json
{
  "component": {
    "name": "js.transform"
  }
}
```

The `js.` prefix ensures no collision with built-in component names.

### Frontend Component Mappers

JavaScript components use a generic mapper that reads display properties (icon, color, label)
from the component definition rather than having them hardcoded.

The default mapper handles:

- **State**: Derived from execution status (running, success, failed, error) — uses default
  states.
- **Subtitle**: Timestamp-based (same as most built-in components).
- **Details tab**: Shows execution duration, emitted payload summary, and any error messages.

Custom mappers for JS components are not needed initially.

## Implementation Plan

### Phase 1: Runtime and Loading

1. Add the goja dependency.
2. Implement the JS runtime in `pkg/jsruntime/`:
   - VM creation and teardown.
   - SDK injection (`superplane` global object).
   - Execution timeout enforcement.
   - Context bridging (JS ctx <-> Go `core.ExecutionContext`).
3. Implement `JSComponentAdapter` that implements `core.Component`.
4. Implement file-based component loader (scan directory, parse files, register components).
5. Implement the file watcher (polling-based, detects new/modified/deleted files, updates the
   registry).
6. Wire the loader and watcher into `cmd/server/main.go` (call after registry init, start
   watcher as a background goroutine).

### Phase 2: Canvas Integration

1. Ensure the component listing APIs include JS components (they should automatically since
   they're in the registry).
2. Add the generic frontend mapper for JS components.
3. Add the "Custom" section to the component picker.

### Phase 3: Hardening

1. Add unit tests for the JS runtime (timeout, SDK functions, error handling).
2. Add E2E tests for a sample JS component in a workflow.
3. Add telemetry for JS component execution (duration, error rates).
4. Document supported JavaScript features and SDK reference.

## Security Considerations

- **Sandboxed execution**: The goja runtime has no access to the file system, network, or OS
  APIs. All external interactions go through the explicitly injected SDK functions, which
  delegate to Go code with the same authorization and audit trail as built-in components.
- **Resource isolation**: Each execution gets its own VM instance. There is no shared state
  between executions. Execution timeouts and memory limits prevent resource exhaustion.
- **Secret access**: JavaScript components access secrets through `ctx.secrets.getKey()`, which
  uses the same `SecretsContext` as built-in components. Secrets are never exposed in logs or
  error messages.
- **File trust model**: The `.js` files are trusted the same way Go component source code is
  trusted — they are placed on disk by the operator who deploys SuperPlane. There is no
  user-facing upload mechanism. This is deployment-time configuration, not runtime user input.
- **HTTP request controls**: HTTP requests from JS components go through the existing
  `HTTPContext`, inheriting connection pooling, TLS settings, and rate limits. The per-execution
  request limit prevents abuse.

## Decisions

- **Runtime**: Embedded goja (pure-Go ECMAScript runtime) for zero operational overhead and
  build simplicity.
- **Storage**: File-based. Components are `.js` files on disk managed by the operator, not
  stored in the database. This keeps the system simple, avoids new migrations, and lets
  operators version-control components alongside their deployment configuration.
- **Naming convention**: JS components are prefixed with `js.` in the registry to distinguish
  from built-in components and prevent name collisions.
- **Global scope**: Unlike organization-scoped database resources, file-based components are
  available to all organizations on the instance. This matches how built-in Go components work.
- **Configuration**: Uses the same `configuration.Field` schema as built-in components, so the
  existing UI renders JS component configuration identically.
- **No Actions support initially**: JavaScript components support `execute` and `setup`
  handlers. Custom actions (`HandleAction`) and webhooks (`HandleWebhook`) are deferred to a
  future iteration.
- **Dynamic loading**: A file watcher polls the directory and picks up new, modified, and
  deleted files without requiring a server restart. Polling (not OS-level file notifications)
  is used for simplicity and portability across operating systems and container runtimes.
