# Plugin System

## Overview

SuperPlane plugins are self-contained, packaged extensions that add custom components, triggers,
and integrations to the platform. Plugins are developed as TypeScript projects, packaged into
distributable archive files (`.spx`), and installed via a CLI command. Once installed, each
plugin declares what it contributes through a manifest, runs in an isolated host process, and
interacts with SuperPlane through a versioned API module.

Today, extending SuperPlane requires writing Go code in the main repository and redeploying the
entire application. This limits contributions to developers who know Go and have access to the
codebase. The plugin system removes that barrier entirely.

The plugin lifecycle has four stages:

1. **Develop** — write TypeScript against the `@superplane/sdk` API.
2. **Package** — run `npx @superplane/cli plugin pack` to produce a `.spx` archive.
3. **Install** — run `superplane plugin install my-plugin-1.0.0.spx`.
4. **Run** — the server discovers installed plugins on startup, reads their manifests, and
   activates them lazily.

Plugins run in a separate Node.js process — the Plugin Host — which communicates with the Go
backend over a local JSON-RPC channel. This process-level isolation means a crashing plugin
cannot bring down the server, and the Node.js runtime gives plugin authors access to the full
npm ecosystem.

## Goals

1. Allow operators and community contributors to extend SuperPlane with custom components,
   triggers, and integrations — without touching Go code or recompiling the binary.
2. Provide a TypeScript-first development experience with a well-defined, versioned API.
3. Run plugins in an isolated Node.js process so that a faulty plugin cannot crash or degrade
   the main server.
4. Support lazy activation: plugins load only when their contributions are actually needed.
5. Use declarative manifests (`package.json`) so the server knows what a plugin provides
   without executing any of its code.
6. Register plugin-provided components, triggers, and integrations in the existing registry so
   they are indistinguishable from built-in ones in the canvas and API.

## Non-Goals

- **Marketplace / remote registry**: Plugins are packaged as `.spx` files and installed via the
  CLI. A hosted package registry, in-app store, or auto-update mechanism is out of scope.
- **Hot-reloading during execution**: Installing a new version of a plugin restarts its
  activation, but in-flight executions complete on the old code. There is no mid-execution
  code swap.
- **Browser-side plugin code**: Plugins contribute backend logic only. Frontend rendering uses
  the same generic mappers that built-in components use.
- **Multi-language plugins**: The Plugin Host runs Node.js. Go, Python, or other runtimes are
  not supported as plugin languages.
- **Sandboxed network access**: Plugins can make arbitrary HTTP calls from Node.js. Network
  policy enforcement (allowlists, egress controls) is deferred to a later phase.

## Architecture

The plugin system is built on five core principles:

1. **Manifest-driven contributions.** A plugin's `package.json` contains a `superplane.contributes`
   section that declaratively lists everything the plugin provides — components, triggers,
   integrations, their configuration schemas, output channels, and display metadata. The Go
   server reads this JSON to populate the registry and render the canvas UI without ever
   executing the plugin's code.

2. **Packaged distribution.** Plugins are compiled, bundled, and packaged into `.spx` archives
   before installation. The server never touches source files, `node_modules`, or build
   tooling. It loads only the pre-built `extension.js` bundle from the installed archive.

3. **Lazy activation.** Plugins declare `activationEvents` — conditions under which they should
   be loaded. A plugin that contributes a data transformation component activates only when a
   canvas node actually uses that component, not at startup. This keeps startup fast and memory
   usage low.

4. **Process isolation.** All plugins run in a dedicated child process (the Plugin Host),
   separate from the main Go server. Communication happens over a structured JSON-RPC protocol.
   If a plugin crashes, the Plugin Host catches the error and reports a failure — it does not
   bring down the server.

5. **Versioned API.** Plugins `import * as superplane from '@superplane/sdk'` to access
   platform capabilities. The API surface is explicitly versioned — plugins declare which
   SuperPlane version they require via `engines.superplane` in their manifest.

### Process Architecture

```
┌─────────────────────────────────────────┐
│             Go Server (main)            │
│                                         │
│  ┌──────────┐  ┌──────────────────────┐ │
│  │ Registry │  │  Plugin Manager      │ │
│  │          │◄─┤                      │ │
│  │Components│  │  - Reads installed    │ │    JSON-RPC over stdio
│  │Triggers  │  │    plugin manifests  │─┼──────────────────────────┐
│  │Integr.   │  │  - Manages lifecycle │ │                          │
│  └──────────┘  └──────────────────────┘ │                          │
│                                         │                          │
└─────────────────────────────────────────┘                          │
                                                                     │
                                                    ┌────────────────▼──────────────────┐
                                                    │         Plugin Host (Node.js)      │
                                                    │                                    │
                                                    │  ┌──────────────────────────────┐  │
                                                    │  │       Plugin Loader          │  │
                                                    │  │  - Requires each plugin      │  │
                                                    │  │  - Calls activate()          │  │
                                                    │  │  - Routes RPC to handlers    │  │
                                                    │  └──────────────────────────────┘  │
                                                    │                                    │
                                                    │  ┌────────┐ ┌────────┐ ┌────────┐ │
                                                    │  │Plugin A│ │Plugin B│ │Plugin C│ │
                                                    │  └────────┘ └────────┘ └────────┘ │
                                                    │                                    │
                                                    └────────────────────────────────────┘
```

The Go server spawns one Plugin Host process. The Plugin Host loads all activated plugins in the
same Node.js runtime. If a plugin throws an unhandled exception, the Plugin Host catches it and
reports a failure for that execution — it does not crash the process. If the Plugin Host process
dies unexpectedly, the Plugin Manager on the Go side detects the broken pipe, restarts the
process, and re-activates all plugins.

## Plugin Archive Format (`.spx`)

A `.spx` file (SuperPlane Extension) is a zip archive containing the compiled, ready-to-run
plugin. It contains no source code, no `node_modules`, and no build tooling. The packaging step
bundles everything into a single JavaScript file.

### Archive Contents

```
superplane-plugin-transform-1.0.0.spx   (zip archive)
├── package.json          ← manifest (contributes, activationEvents, engines)
├── extension.js          ← single bundled entry point (all dependencies inlined)
├── README.md             ← optional
└── icon.png              ← optional plugin icon
```

The key properties:

- **`package.json`** — the manifest. The `main` field always points to `extension.js`.
- **`extension.js`** — the entire plugin compiled and bundled into one file. The packaging
  tool uses a bundler (esbuild) to resolve all npm dependencies and produce a single
  self-contained output. No `node_modules` directory exists in the archive.
- **No source files** — TypeScript source, `tsconfig.json`, `src/`, test files, and dev
  dependencies are excluded. The archive is a distribution artifact, not a development project.

### Naming Convention

Archive files follow the pattern `<name>-<version>.spx`:

- `superplane-plugin-transform-1.0.0.spx`
- `superplane-plugin-pagerduty-custom-2.3.1.spx`

### Development Project Structure

During development, the plugin is a standard TypeScript project. The `.spx` archive is produced
from it by the packaging CLI:

```
superplane-plugin-transform/       ← development project (not shipped)
├── package.json
├── tsconfig.json
├── node_modules/
├── src/
│   └── index.ts
└── dist/                          ← build output (intermediate, not shipped directly)
    └── index.js
```

The developer runs `npm run build` (or `tsc`) during development, then
`npx @superplane/cli plugin pack` to produce the `.spx` archive for distribution.

## Plugin Manifest (`package.json`)

The manifest is the single source of truth for what a plugin provides. The Go server reads
manifests at startup without running any JavaScript. This is the declarative core of the system.

```json
{
  "name": "superplane-plugin-transform",
  "version": "1.0.0",
  "description": "Data transformation components for SuperPlane",
  "main": "dist/index.js",
  "engines": {
    "superplane": "^1.0.0"
  },
  "superplane": {
    "activationEvents": [
      "onComponent:transform.filter",
      "onComponent:transform.reshape"
    ],
    "contributes": {
      "components": [
        {
          "name": "transform.filter",
          "label": "Filter Data",
          "description": "Filter incoming event data by field values",
          "icon": "filter",
          "color": "blue",
          "configuration": [
            {
              "name": "field",
              "label": "Field",
              "type": "string",
              "required": true,
              "description": "The field to filter on"
            },
            {
              "name": "value",
              "label": "Value",
              "type": "string",
              "required": true,
              "description": "The value to match"
            }
          ],
          "outputChannels": [
            { "name": "matched", "label": "Matched" },
            { "name": "unmatched", "label": "Unmatched" }
          ]
        },
        {
          "name": "transform.reshape",
          "label": "Reshape Data",
          "description": "Transform the structure of incoming data",
          "icon": "shuffle",
          "color": "purple",
          "configuration": [
            {
              "name": "mapping",
              "label": "Field Mapping",
              "type": "code",
              "required": true,
              "description": "JavaScript expression for the mapping"
            }
          ],
          "outputChannels": [
            { "name": "default", "label": "Default" }
          ]
        }
      ],
      "triggers": [],
      "integrations": []
    }
  }
}
```

### Manifest Fields

| Field                             | Required | Description                                                   |
|-----------------------------------|----------|---------------------------------------------------------------|
| `name`                            | Yes      | npm package name. Convention: `superplane-plugin-<name>`.     |
| `version`                         | Yes      | Semver version.                                               |
| `main`                            | Yes      | Entry point (compiled JS).                                    |
| `engines.superplane`              | Yes      | Compatible SuperPlane version range.                          |
| `superplane.activationEvents`     | Yes      | When to activate the plugin (see Activation Events below).    |
| `superplane.contributes`          | Yes      | Declarative list of components, triggers, and integrations.   |

### Activation Events

Activation events control when a plugin's `activate()` function is called. Until activation,
the Plugin Host does not `require()` the plugin's code at all — it exists only as parsed
manifest data.

| Event Pattern                     | Fires When                                                    |
|-----------------------------------|---------------------------------------------------------------|
| `onComponent:<name>`              | A canvas node using this component is set up or executed.     |
| `onTrigger:<name>`                | A canvas node using this trigger is set up or fires.          |
| `onIntegration:<name>`            | An integration of this type is configured or synced.          |
| `*`                               | Immediately at Plugin Host startup.                           |

A plugin with `"activationEvents": ["*"]` loads at startup. This is appropriate for plugins
that need to perform initialization (e.g., register webhook handlers) regardless of whether
their components are currently in use.

A plugin with `"activationEvents": ["onComponent:transform.filter"]` loads only when a canvas
node references `transform.filter`. If no canvas on the instance uses that component, the
plugin never loads.

### Contribution Points

Each contribution type maps directly to a core interface:

**Components** declare the same fields as `core.Component`: name, label, description, icon,
color, configuration fields, and output channels. The Go server reads these from the manifest
and creates placeholder entries in the registry. When a component is actually executed, the
Plugin Manager delegates to the Plugin Host.

**Triggers** declare name, label, description, icon, color, configuration fields, and example
data — matching `core.Trigger`.

**Integrations** declare name, label, description, icon, instructions, configuration fields,
and the list of component/trigger names they expose — matching `core.Integration`.

## Plugin API (`@superplane/sdk`)

Plugins import `@superplane/sdk` to interact with SuperPlane. This module is provided by the
Plugin Host runtime and is not an npm dependency that plugins install themselves — it is
available globally in the Plugin Host process at runtime.

### Entry Point

```typescript
import * as superplane from '@superplane/sdk';

export function activate(context: superplane.PluginContext) {
  // Register component handlers
  context.components.register('transform.filter', {
    setup(ctx) {
      if (!ctx.configuration.field) {
        throw new Error('field is required');
      }
    },

    execute(ctx) {
      const items = ctx.input.items || [];
      const field = ctx.configuration.field;
      const value = ctx.configuration.value;

      const matched = items.filter((item: any) => item[field] === value);
      const unmatched = items.filter((item: any) => item[field] !== value);

      if (matched.length > 0) {
        ctx.emit('matched', 'filtered', { items: matched, total: matched.length });
      }

      if (unmatched.length > 0) {
        ctx.emit('unmatched', 'filtered', { items: unmatched, total: unmatched.length });
      }

      if (matched.length === 0 && unmatched.length === 0) {
        ctx.pass();
      }
    }
  });

  context.components.register('transform.reshape', {
    execute(ctx) {
      const mapping = new Function('data', ctx.configuration.mapping);
      const result = mapping(ctx.input);
      ctx.emit('default', 'reshaped', result);
    }
  });
}

export function deactivate() {
  // Clean up resources if needed
}
```

### API Surface

#### `superplane.PluginContext`

Passed to `activate()`. This is the plugin's handle to the SuperPlane platform.

```typescript
interface PluginContext {
  /** Register handlers for components declared in the manifest. */
  components: ComponentRegistry;

  /** Register handlers for triggers declared in the manifest. */
  triggers: TriggerRegistry;

  /** Register handlers for integrations declared in the manifest. */
  integrations: IntegrationRegistry;

  /** Track disposable resources for cleanup on deactivate. */
  subscriptions: Disposable[];

  /** Plugin-scoped logger. */
  log: Logger;
}
```

#### `ComponentRegistry`

```typescript
interface ComponentRegistry {
  register(name: string, handler: ComponentHandler): Disposable;
}

interface ComponentHandler {
  setup?(ctx: SetupContext): void | Promise<void>;
  execute(ctx: ExecutionContext): void | Promise<void>;
  cancel?(ctx: ExecutionContext): void | Promise<void>;
  cleanup?(ctx: SetupContext): void | Promise<void>;
  processQueueItem?(ctx: QueueItemContext): string | null | Promise<string | null>;
  handleAction?(ctx: ActionContext): void | Promise<void>;
  handleWebhook?(ctx: WebhookContext): WebhookResponse | Promise<WebhookResponse>;
}
```

#### `TriggerRegistry`

```typescript
interface TriggerRegistry {
  register(name: string, handler: TriggerHandler): Disposable;
}

interface TriggerHandler {
  setup?(ctx: TriggerSetupContext): void | Promise<void>;
  cleanup?(ctx: TriggerSetupContext): void | Promise<void>;
  handleWebhook?(ctx: WebhookContext): WebhookResponse | Promise<WebhookResponse>;
  handleAction?(ctx: TriggerActionContext): Record<string, any> | Promise<Record<string, any>>;
}
```

#### `IntegrationRegistry`

```typescript
interface IntegrationRegistry {
  register(name: string, handler: IntegrationHandler): Disposable;
}

interface IntegrationHandler {
  sync?(ctx: SyncContext): void | Promise<void>;
  cleanup?(ctx: IntegrationCleanupContext): void | Promise<void>;
  handleAction?(ctx: IntegrationActionContext): void | Promise<void>;
  handleWebhook?(setup: WebhookSetupContext): WebhookSetupResult | Promise<WebhookSetupResult>;
  listResources?(type: string, ctx: ListResourcesContext): IntegrationResource[] | Promise<IntegrationResource[]>;
  handleRequest?(ctx: HTTPRequestContext): void | Promise<void>;
}
```

#### Execution Context

The execution context mirrors the Go `core.ExecutionContext` — same capabilities, TypeScript
types:

```typescript
interface ExecutionContext {
  /** Unique execution ID. */
  id: string;

  /** Workflow that owns this execution. */
  workflowId: string;

  /** Organization that owns this workflow. */
  organizationId: string;

  /** Node that is executing. */
  nodeId: string;

  /** The upstream node that produced the input. */
  sourceNodeId: string;

  /** The base URL of the SuperPlane instance. */
  baseUrl: string;

  /** Input data from the upstream node or trigger. */
  input: any;

  /** Configuration values set by the user in the canvas. */
  configuration: Record<string, any>;

  /** Evaluate a SuperPlane expression. */
  eval(expression: string): Promise<Record<string, any>>;

  /** Emit a payload to a named output channel. */
  emit(channel: string, payloadType: string, data: any | any[]): void;

  /** Pass the execution without emitting data. */
  pass(): void;

  /** Fail the execution with a reason and message. */
  fail(reason: string, message: string): void;

  /** Set a key-value pair on the execution state. */
  setKV(key: string, value: string): void;

  /** Execution-scoped metadata (persisted across retries). */
  metadata: MetadataAccessor;

  /** Node-scoped metadata (persisted across executions). */
  nodeMetadata: MetadataAccessor;

  /** Make HTTP requests through SuperPlane's managed HTTP client. */
  http: HTTPClient;

  /** Read secrets from the organization's secret store. */
  secrets: SecretsAccessor;

  /** Access integration context when running under an integration. */
  integration: IntegrationAccessor;

  /** Logger scoped to this execution. */
  log: Logger;
}

interface MetadataAccessor {
  get(): Promise<any>;
  set(value: any): Promise<void>;
}

interface HTTPClient {
  request(method: string, url: string, options?: HTTPOptions): Promise<HTTPResponse>;
}

interface HTTPOptions {
  headers?: Record<string, string>;
  body?: string;
  timeout?: number;
}

interface HTTPResponse {
  status: number;
  headers: Record<string, string>;
  body: any;
}

interface SecretsAccessor {
  getKey(secretName: string, keyName: string): Promise<string>;
}

interface Logger {
  info(message: string, ...args: any[]): void;
  warn(message: string, ...args: any[]): void;
  error(message: string, ...args: any[]): void;
  debug(message: string, ...args: any[]): void;
}

interface Disposable {
  dispose(): void;
}
```

### Async Support

All handler methods can return `void` or `Promise<void>`. The Plugin Host awaits promises and
translates the resolved/rejected state back to the Go server. This means plugins can use
`async/await` naturally:

```typescript
async execute(ctx: ExecutionContext) {
  const token = await ctx.secrets.getKey('api-credentials', 'token');

  const response = await ctx.http.request('GET', 'https://api.example.com/data', {
    headers: { 'Authorization': `Bearer ${token}` },
    timeout: 5000,
  });

  if (response.status !== 200) {
    ctx.fail('api_error', `API returned ${response.status}`);
    return;
  }

  ctx.emit('default', 'api_response', response.body);
}
```

## Packaging CLI

The `@superplane/cli` package provides a `plugin pack` command that builds a `.spx` archive
from a TypeScript plugin project.

### Usage

```bash
# From within the plugin project directory:
npx @superplane/cli plugin pack

# Output:
#   ✓ Compiled TypeScript → dist/index.js
#   ✓ Bundled with esbuild → extension.js (148 KB)
#   ✓ Validated package.json manifest
#   ✓ Created superplane-plugin-transform-1.0.0.spx
```

### What `plugin pack` Does

1. **Compile** — runs `tsc` (or uses the project's build script) to produce JavaScript from
   TypeScript source.
2. **Bundle** — runs esbuild to bundle the compiled entry point and all its npm dependencies
   into a single `extension.js` file. The `@superplane/sdk` import is marked as external
   (it's provided by the Plugin Host at runtime, not bundled).
3. **Validate** — checks that `package.json` contains a valid `superplane` field with
   `contributes` and `activationEvents`. Checks that `engines.superplane` is set. Checks that
   the bundled entry point exports `activate`.
4. **Package** — creates a zip archive with the `.spx` extension containing `package.json`,
   `extension.js`, and any declared static assets (README, icon).

### Validation Errors

The CLI rejects packaging if:

- `package.json` is missing or lacks the `superplane` field.
- `engines.superplane` is not set.
- The entry point doesn't export an `activate` function.
- Contributed names are empty or contain invalid characters.
- Configuration fields reference unsupported types.

## Plugin Installation

### Install Command

```bash
# Install from a local .spx file:
superplane plugin install superplane-plugin-transform-1.0.0.spx

# Install from a URL:
superplane plugin install https://releases.example.com/superplane-plugin-transform-1.0.0.spx

# List installed plugins:
superplane plugin list

# Uninstall:
superplane plugin uninstall superplane-plugin-transform

# Show details:
superplane plugin info superplane-plugin-transform
```

### What `plugin install` Does

1. **Extract** — unzips the `.spx` archive into a subdirectory under the managed plugins
   directory: `<SUPERPLANE_PLUGINS_DIR>/<name>/`.
2. **Validate** — reads the extracted `package.json`, checks `engines.superplane` compatibility,
   checks for name collisions with built-in or other installed plugins.
3. **Register** — writes a record of the installed plugin (name, version, install time) to a
   `plugins.json` manifest file in the plugins directory.
4. **Signal** — if the server is running, sends a `SIGHUP` to trigger a plugin reload. The
   server re-scans the plugins directory, picks up the new plugin, registers its contributions,
   and activates it if needed.

### Version Management

Installing a plugin that is already installed replaces it:

```bash
# Upgrade: install the new version over the old one
superplane plugin install superplane-plugin-transform-2.0.0.spx
# → Replaces 1.0.0, deactivates old plugin, activates new one
```

Only one version of a plugin can be installed at a time. The install command extracts the new
archive, deactivates the old version (if the server is running), and activates the new one.

### Managed Directory Layout

The plugins directory is fully managed by the CLI and the server. Operators do not manually
create or modify files in it.

```
plugins/                                    ← SUPERPLANE_PLUGINS_DIR
├── plugins.json                            ← registry of installed plugins
├── superplane-plugin-transform/
│   ├── package.json
│   └── extension.js
├── superplane-plugin-slack-utils/
│   ├── package.json
│   └── extension.js
└── superplane-plugin-custom-approval/
    ├── package.json
    └── extension.js
```

The `plugins.json` file tracks what is installed:

```json
{
  "plugins": [
    {
      "name": "superplane-plugin-transform",
      "version": "1.0.0",
      "installedAt": "2026-02-20T10:30:00Z"
    },
    {
      "name": "superplane-plugin-slack-utils",
      "version": "0.5.2",
      "installedAt": "2026-02-19T14:00:00Z"
    }
  ]
}
```

## Plugin Discovery and Loading

### Startup Sequence

1. **Read installed plugins.** The Plugin Manager reads `SUPERPLANE_PLUGINS_DIR/plugins.json`.
   For each entry, it reads the `package.json` from the corresponding subdirectory.

2. **Validate manifests.** For each installed plugin:
   - Check `engines.superplane` compatibility with the running server version.
   - Validate that all contributed names are unique across all plugins (and don't collide with
     built-in names).
   - Validate that `extension.js` exists.
   - If validation fails, log an error and skip the plugin.

3. **Register contributions.** For each valid plugin, create adapter entries in the Go registry:
   - `PluginComponentAdapter` implements `core.Component`. Its metadata methods (Name, Label,
     Configuration, OutputChannels, etc.) return data from the manifest. Its execution methods
     (Setup, Execute, etc.) delegate to the Plugin Host over JSON-RPC.
   - `PluginTriggerAdapter` implements `core.Trigger`, same pattern.
   - `PluginIntegrationAdapter` implements `core.Integration`, same pattern.

4. **Spawn Plugin Host.** Start the Node.js child process. The Plugin Host receives the list of
   installed plugin paths and their activation events.

5. **Activate eager plugins.** Plugins with `"activationEvents": ["*"]` are activated
   immediately — the Plugin Host calls `require()` on their `extension.js` and invokes
   `activate()`.

### Lazy Activation Flow

When a component from a not-yet-activated plugin is first needed:

```
Canvas node references "transform.filter"
  → Node executor calls registry.GetComponent("transform.filter")
  → Returns PluginComponentAdapter
  → Adapter calls PluginManager.EnsureActivated("superplane-plugin-transform")
  → Plugin Manager sends "activate" RPC to Plugin Host
  → Plugin Host calls require("<plugins-dir>/superplane-plugin-transform/extension.js")
  → activate(context) is called
  → Plugin registers its handlers
  → Adapter sends "execute" RPC to Plugin Host
  → Plugin Host routes to the registered handler
  → Result returned to Go server
```

Subsequent calls to the same plugin skip activation — it's already loaded.

### Live Install / Uninstall

When a plugin is installed or uninstalled while the server is running (via `superplane plugin
install` / `uninstall`), the CLI sends `SIGHUP` to the server process. The server:

1. Re-reads `plugins.json`.
2. Compares the current set of installed plugins against what is loaded.
3. For new plugins: validates, registers contributions, and signals the Plugin Host to prepare
   for activation.
4. For removed plugins: deactivates in the Plugin Host, unregisters from the Go registry.
5. For updated plugins (same name, new version): deactivates old, loads new manifest, registers
   updated contributions, re-activates.

In-flight executions on old plugin versions complete normally.

## Communication Protocol

The Go server and Plugin Host communicate over the child process's stdin/stdout using JSON-RPC
2.0. Each message is a single JSON object terminated by a newline.

### Message Flow

**Go → Plugin Host (requests):**

```json
{"jsonrpc":"2.0","id":1,"method":"plugin/activate","params":{"pluginId":"superplane-plugin-transform"}}
{"jsonrpc":"2.0","id":2,"method":"component/execute","params":{"pluginId":"superplane-plugin-transform","component":"transform.filter","context":{...}}}
{"jsonrpc":"2.0","id":3,"method":"component/setup","params":{"pluginId":"superplane-plugin-transform","component":"transform.filter","context":{...}}}
{"jsonrpc":"2.0","id":4,"method":"trigger/handleWebhook","params":{"pluginId":"superplane-plugin-events","trigger":"events.github","context":{...}}}
```

**Plugin Host → Go (responses):**

```json
{"jsonrpc":"2.0","id":2,"result":{"action":"emit","channel":"matched","payloadType":"filtered","data":{...}}}
{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"field is required"}}
```

**Plugin Host → Go (requests for context operations):**

When plugin code calls `ctx.secrets.getKey()`, `ctx.http.request()`, `ctx.metadata.get()`, or
similar context methods, the Plugin Host sends a request to the Go server and blocks the handler
until the response arrives:

```json
{"jsonrpc":"2.0","id":100,"method":"ctx/secrets.getKey","params":{"executionId":"...","secretName":"api-creds","keyName":"token"}}
{"jsonrpc":"2.0","id":101,"method":"ctx/http.request","params":{"executionId":"...","method":"GET","url":"https://api.example.com","options":{...}}}
```

The Go server fulfills these using the same `SecretsContext`, `HTTPContext`, etc. that built-in
components use — same authorization, same audit trail, same rate limits.

### RPC Methods

| Method                        | Direction          | Description                                    |
|-------------------------------|--------------------|------------------------------------------------|
| `plugin/activate`             | Go → Plugin Host   | Activate a plugin (require + call activate)    |
| `plugin/deactivate`           | Go → Plugin Host   | Deactivate a plugin (call deactivate)          |
| `component/setup`             | Go → Plugin Host   | Call a component's setup handler               |
| `component/execute`           | Go → Plugin Host   | Call a component's execute handler             |
| `component/cancel`            | Go → Plugin Host   | Call a component's cancel handler              |
| `component/cleanup`           | Go → Plugin Host   | Call a component's cleanup handler             |
| `component/handleAction`      | Go → Plugin Host   | Call a component's action handler              |
| `component/handleWebhook`     | Go → Plugin Host   | Call a component's webhook handler             |
| `component/processQueueItem`  | Go → Plugin Host   | Call a component's queue item processor        |
| `trigger/setup`               | Go → Plugin Host   | Call a trigger's setup handler                 |
| `trigger/cleanup`             | Go → Plugin Host   | Call a trigger's cleanup handler               |
| `trigger/handleWebhook`       | Go → Plugin Host   | Call a trigger's webhook handler               |
| `trigger/handleAction`        | Go → Plugin Host   | Call a trigger's action handler                |
| `integration/sync`            | Go → Plugin Host   | Call an integration's sync handler             |
| `integration/cleanup`         | Go → Plugin Host   | Call an integration's cleanup handler          |
| `integration/handleAction`    | Go → Plugin Host   | Call an integration's action handler           |
| `integration/listResources`   | Go → Plugin Host   | Call an integration's resource lister          |
| `ctx/secrets.getKey`          | Plugin Host → Go   | Read a secret key                              |
| `ctx/http.request`            | Plugin Host → Go   | Make an HTTP request through managed client    |
| `ctx/metadata.get`            | Plugin Host → Go   | Read execution/node metadata                   |
| `ctx/metadata.set`            | Plugin Host → Go   | Write execution/node metadata                  |
| `ctx/eval`                    | Plugin Host → Go   | Evaluate a SuperPlane expression               |
| `ctx/integration.getConfig`   | Plugin Host → Go   | Read integration configuration                 |
| `ctx/integration.setMetadata` | Plugin Host → Go   | Write integration metadata                     |
| `ctx/integration.subscribe`   | Plugin Host → Go   | Subscribe to integration events                |
| `ctx/webhook.setup`           | Plugin Host → Go   | Set up a webhook endpoint                      |

## Registry Integration

Plugin-contributed items appear in the registry identically to built-in ones. The adapter
pattern makes this transparent:

```
┌─────────────────────────────────────────────────────────────┐
│                        Registry                             │
│                                                             │
│  Components:                                                │
│    "filter"            → built-in FilterComponent (Go)      │
│    "http"              → built-in HTTPComponent (Go)        │
│    "transform.filter"  → PluginComponentAdapter             │
│    "transform.reshape" → PluginComponentAdapter             │
│                                                             │
│  Triggers:                                                  │
│    "webhook"           → built-in WebhookTrigger (Go)       │
│    "events.github"     → PluginTriggerAdapter               │
│                                                             │
│  Integrations:                                              │
│    "github"            → built-in GitHubIntegration (Go)    │
│    "custom-service"    → PluginIntegrationAdapter           │
└─────────────────────────────────────────────────────────────┘
```

The `PluginComponentAdapter` implements every method of `core.Component`:

- **Metadata methods** (`Name()`, `Label()`, `Configuration()`, `OutputChannels()`, etc.)
  return data parsed from the manifest JSON. These never cross the process boundary.
- **Execution methods** (`Setup()`, `Execute()`, `Cancel()`, etc.) send an RPC to the Plugin
  Host and wait for the response. If the plugin is not yet activated, the adapter triggers
  activation first.

This means:

- The component listing API (`ListComponents`) works without any changes — it iterates the
  registry and returns all entries.
- The canvas node editor renders plugin components using the same configuration schema format.
- The node executor calls `component.Execute(ctx)` and gets back a result — it has no idea
  whether the component is Go or JavaScript.

## Naming Conventions

Plugin-contributed names use dot-separated prefixes derived from the plugin name:

- Plugin `superplane-plugin-transform` contributes `transform.filter`, `transform.reshape`.
- Plugin `superplane-plugin-slack-utils` contributes `slack-utils.format-message`.

This follows the existing convention where integration components use the integration name as a
prefix (e.g., `github.create-issue`, `slack.send-message`).

The Plugin Manager validates at startup that no contributed name collides with a built-in name
or a name from another plugin.

## Resource Limits

| Limit                        | Default    | Environment Variable                    |
|------------------------------|------------|-----------------------------------------|
| Execution timeout            | 30 seconds | `SUPERPLANE_PLUGIN_EXECUTION_TIMEOUT`   |
| Plugin activation timeout    | 10 seconds | `SUPERPLANE_PLUGIN_ACTIVATION_TIMEOUT`  |
| HTTP requests per execution  | 10         | `SUPERPLANE_PLUGIN_MAX_HTTP_REQUESTS`   |
| HTTP request timeout         | 10 seconds | `SUPERPLANE_PLUGIN_HTTP_TIMEOUT`        |
| Plugin Host memory limit     | 512 MB     | `SUPERPLANE_PLUGIN_HOST_MEMORY_LIMIT`   |
| Max plugins                  | 50         | `SUPERPLANE_PLUGIN_MAX_COUNT`           |

When a limit is exceeded:
- **Execution timeout**: The Go server cancels the pending RPC and fails the execution.
- **Activation timeout**: The plugin is marked as failed and its contributions are unavailable.
- **Plugin Host memory**: The Node.js process is killed and restarted via `--max-old-space-size`.

## Error Handling

### Plugin Crashes

If a plugin throws an unhandled exception during `execute()`, the Plugin Host catches it and
returns an RPC error response. The Go server fails the execution with the error message. The
plugin remains loaded — a single execution failure does not unload the plugin.

### Plugin Host Crashes

If the Plugin Host process exits unexpectedly:

1. The Plugin Manager detects the broken stdin/stdout pipe.
2. All pending RPC calls are failed with "Plugin Host unavailable."
3. The Plugin Manager waits 1 second and respawns the Plugin Host.
4. All previously activated plugins are re-activated.
5. If the Plugin Host crashes 5 times within 60 seconds, the Plugin Manager stops restarting
   and logs a critical error. Plugin-based components/triggers return errors until an operator
   intervenes.

### Malformed Plugins

- **Bad manifest**: Rejected at install time by the CLI. If a previously valid plugin becomes
  incompatible after a server upgrade, it is logged and skipped at startup.
- **Missing `extension.js`**: Rejected at install time. If the file is somehow missing from an
  installed plugin directory, activation fails and the plugin is marked as errored.
- **activate() throws**: Plugin marked as errored. Its contributions remain in the registry
  but return errors when used.
- **Incompatible engine version**: Rejected at install time. Logged and skipped at startup if
  the server version changes.

## Configuration

| Environment Variable                     | Default       | Description                                         |
|------------------------------------------|---------------|-----------------------------------------------------|
| `SUPERPLANE_PLUGINS_DIR`                 | `plugins`     | Path to managed directory for installed plugins.    |
| `SUPERPLANE_PLUGIN_HOST_MEMORY_LIMIT`    | `512MB`       | Max memory for the Plugin Host process.             |
| `SUPERPLANE_PLUGIN_EXECUTION_TIMEOUT`    | `30s`         | Max execution time per handler call.                |
| `SUPERPLANE_PLUGIN_ACTIVATION_TIMEOUT`   | `10s`         | Max time for a plugin to activate.                  |
| `SUPERPLANE_PLUGIN_MAX_HTTP_REQUESTS`    | `10`          | Max HTTP requests per execution.                    |
| `SUPERPLANE_PLUGIN_HTTP_TIMEOUT`         | `10s`         | Timeout for individual HTTP requests.               |
| `SUPERPLANE_PLUGIN_MAX_COUNT`            | `50`          | Max number of plugins that can be installed.        |
| `SUPERPLANE_PLUGIN_NODE_PATH`            | (auto-detect) | Path to the Node.js binary.                         |

If `SUPERPLANE_PLUGINS_DIR` does not exist or contains no `plugins.json`, the plugin system is
disabled — no Plugin Host is spawned, and the server operates exactly as it does today.

## Canvas Integration

Plugin components, triggers, and integrations appear in the canvas alongside built-in ones.

- **Component picker**: Plugin components appear in a "Plugins" section, grouped by plugin
  name. The icon and color from the manifest are used.
- **Configuration panel**: The sidebar renders configuration fields from the manifest using the
  existing `configuration.Field` schema — same UI as built-in components.
- **Run history**: Execution states, subtitles, and details use the default mappers. Plugin
  components don't need custom frontend mappers.
- **Integration panel**: Plugin integrations appear in the integrations list with their declared
  configuration fields and instructions.

Canvas nodes reference plugin components using the existing `Ref` structure:

```json
{
  "component": {
    "name": "transform.filter"
  }
}
```

## Complete Plugin Example

A full plugin that contributes a component and a trigger, showing the full lifecycle from
source to installation. This plugin creates GitHub issues and listens for new issues via
webhooks.

### Source Project

**`package.json`:**

```json
{
  "name": "superplane-plugin-github-issues",
  "version": "1.0.0",
  "description": "Create and listen for GitHub issues",
  "main": "dist/index.js",
  "engines": {
    "superplane": "^1.0.0"
  },
  "superplane": {
    "activationEvents": [
      "onComponent:github-issues.create-issue",
      "onTrigger:github-issues.on-issue-created"
    ],
    "contributes": {
      "components": [
        {
          "name": "github-issues.create-issue",
          "label": "Create GitHub Issue",
          "description": "Open a new issue in a GitHub repository",
          "icon": "github",
          "color": "gray",
          "configuration": [
            { "name": "owner", "label": "Repository Owner", "type": "string", "required": true },
            { "name": "repo", "label": "Repository Name", "type": "string", "required": true }
          ],
          "outputChannels": [
            { "name": "default", "label": "Default" }
          ]
        }
      ],
      "triggers": [
        {
          "name": "github-issues.on-issue-created",
          "label": "GitHub Issue Created",
          "description": "Fires when a new issue is opened in a GitHub repository",
          "icon": "github",
          "color": "gray",
          "configuration": [
            { "name": "owner", "label": "Repository Owner", "type": "string", "required": true },
            { "name": "repo", "label": "Repository Name", "type": "string", "required": true }
          ],
          "exampleData": {
            "action": "opened",
            "issue": {
              "number": 42,
              "title": "Bug: login page returns 500",
              "body": "Steps to reproduce...",
              "state": "open",
              "html_url": "https://github.com/acme/app/issues/42",
              "user": { "login": "octocat" },
              "labels": [{ "name": "bug" }],
              "created_at": "2026-02-20T10:30:00Z"
            },
            "repository": {
              "full_name": "acme/app"
            }
          }
        }
      ],
      "integrations": []
    }
  }
}
```

**`src/index.ts`:**

```typescript
import * as superplane from '@superplane/sdk';
import * as crypto from 'crypto';

export function activate(context: superplane.PluginContext) {
  context.components.register('github-issues.create-issue', {
    async setup(ctx) {
      const token = await ctx.secrets.getKey('github', 'token');
      if (!token) {
        throw new Error('GitHub token not configured — add a "github" secret with a "token" key');
      }
    },

    async execute(ctx) {
      const token = await ctx.secrets.getKey('github', 'token');
      const { owner, repo } = ctx.configuration;

      const response = await ctx.http.request(
        'POST',
        `https://api.github.com/repos/${owner}/${repo}/issues`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Accept': 'application/vnd.github+json',
            'Content-Type': 'application/json',
            'X-GitHub-Api-Version': '2022-11-28',
          },
          body: JSON.stringify({
            title: ctx.input.title,
            body: ctx.input.body || '',
            labels: ctx.input.labels || [],
            assignees: ctx.input.assignees || [],
          }),
        }
      );

      if (response.status !== 201) {
        ctx.fail('api_error', `GitHub API returned ${response.status}: ${response.body?.message}`);
        return;
      }

      ctx.emit('default', 'issue_created', {
        number: response.body.number,
        title: response.body.title,
        html_url: response.body.html_url,
        state: response.body.state,
      });
    }
  });

  context.triggers.register('github-issues.on-issue-created', {
    async setup(ctx) {
      const webhookUrl = await ctx.webhook.setup();
      const token = await ctx.secrets.getKey('github', 'token');
      const webhookSecret = await ctx.webhook.getSecret();
      const { owner, repo } = ctx.configuration;

      const metadata = await ctx.metadata.get();
      if (metadata?.webhookId) {
        return;
      }

      const response = await ctx.http.request(
        'POST',
        `https://api.github.com/repos/${owner}/${repo}/hooks`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Accept': 'application/vnd.github+json',
            'Content-Type': 'application/json',
            'X-GitHub-Api-Version': '2022-11-28',
          },
          body: JSON.stringify({
            config: {
              url: webhookUrl,
              content_type: 'json',
              secret: webhookSecret,
            },
            events: ['issues'],
            active: true,
          }),
        }
      );

      if (response.status !== 201) {
        throw new Error(`Failed to create GitHub webhook: ${response.body?.message}`);
      }

      await ctx.metadata.set({ webhookId: response.body.id });
    },

    async cleanup(ctx) {
      const metadata = await ctx.metadata.get();
      if (!metadata?.webhookId) {
        return;
      }

      const token = await ctx.secrets.getKey('github', 'token');
      const { owner, repo } = ctx.configuration;

      await ctx.http.request(
        'DELETE',
        `https://api.github.com/repos/${owner}/${repo}/hooks/${metadata.webhookId}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Accept': 'application/vnd.github+json',
            'X-GitHub-Api-Version': '2022-11-28',
          },
        }
      );
    },

    async handleWebhook(ctx) {
      const signature = ctx.headers['x-hub-signature-256'];
      const webhookSecret = await ctx.webhook.getSecret();

      const expected = 'sha256=' + crypto
        .createHmac('sha256', webhookSecret)
        .update(ctx.body)
        .digest('hex');

      if (signature !== expected) {
        return { status: 401 };
      }

      const payload = JSON.parse(ctx.body);

      if (payload.action !== 'opened') {
        return { status: 200 };
      }

      ctx.events.emit('issue_created', payload);
      return { status: 200 };
    }
  });
}

export function deactivate() {}
```

### Package and Install

```bash
# Build and package:
cd superplane-plugin-github-issues
npm install
npm run build
npx @superplane/cli plugin pack
# → superplane-plugin-github-issues-1.0.0.spx

# Install on the server:
superplane plugin install superplane-plugin-github-issues-1.0.0.spx
# → Extracted to plugins/superplane-plugin-github-issues/
# → Registered: github-issues.create-issue (component)
# → Registered: github-issues.on-issue-created (trigger)
# → Server signaled to reload plugins

# Verify:
superplane plugin list
# NAME                                VERSION   INSTALLED
# superplane-plugin-github-issues     1.0.0     2026-02-20T10:30:00Z
```

### What Gets Installed

```
plugins/superplane-plugin-github-issues/
├── package.json       ← manifest only (no devDependencies, no scripts)
└── extension.js       ← single bundled file (all deps inlined)
```

No `node_modules/`, no `src/`, no `tsconfig.json`, no build artifacts beyond the single
`extension.js` bundle.

## Implementation Plan

### Phase 1: Packaging CLI and Archive Format

1. Define the `.spx` archive format (zip structure, required files).
2. Implement `plugin pack` in `@superplane/cli` — compile, bundle with esbuild (marking
   `@superplane/sdk` as external), validate manifest, produce `.spx` archive.
3. Implement `plugin install` / `plugin uninstall` / `plugin list` commands — extract archive,
   manage `plugins.json`, signal server via `SIGHUP`.
4. Implement manifest validation (engine compatibility, name format, configuration schema).

At the end of Phase 1, plugins can be packaged and installed on disk, but the server doesn't
load them yet.

### Phase 2: Plugin Manager and Registry Adapters

1. Implement `pkg/plugins/manifest.go` — parse `package.json` from installed plugin
   directories, extract contribution definitions.
2. Implement `pkg/plugins/manager.go` — read `plugins.json`, load manifests, manage plugin
   lifecycle state, handle `SIGHUP` for live reload.
3. Implement `pkg/plugins/adapters.go` — `PluginComponentAdapter`, `PluginTriggerAdapter`,
   `PluginIntegrationAdapter` that implement the core interfaces using manifest data for
   metadata and RPC for execution.
4. Wire manifest-based registration into `cmd/server/main.go` (register adapters after
   built-in init).

At the end of Phase 2, installed plugin components appear in the registry and the API — but
executing them returns an error because the Plugin Host doesn't exist yet.

### Phase 3: Plugin Host and JSON-RPC

1. Implement the Plugin Host entry point in TypeScript (`plugin-host/src/index.ts`) — JSON-RPC
   server over stdio, plugin loader, activation lifecycle.
2. Implement `@superplane/sdk` — the API module that plugins import. Context proxy objects that
   translate method calls into RPC requests back to the Go server.
3. Implement the Go-side RPC client (`pkg/plugins/rpc.go`) — spawn the Node.js process,
   send/receive JSON-RPC messages, handle context callbacks.
4. Connect the adapters to the RPC client so execution methods delegate to the Plugin Host.
5. Implement execution timeout enforcement on the Go side.
6. Implement lazy activation — track which plugins are activated, trigger activation on first
   use.
7. Handle Plugin Host crashes — detection, restart, re-activation.

At the end of Phase 3, plugins can be packaged, installed, and executed end-to-end.

### Phase 4: Canvas Integration and Hardening

1. Add "Plugins" section to the component/trigger picker in the frontend.
2. Add unit tests for manifest parsing, adapter behavior, RPC protocol, and packaging CLI.
3. Add E2E tests for a sample plugin in a workflow.
4. Document plugin authoring guide, SDK reference, and deployment instructions.
5. Publish `@superplane/sdk` type definitions as an npm package for plugin development.

## Security Considerations

- **Process isolation.** Plugins run in a separate Node.js process. A crashing or misbehaving
  plugin cannot corrupt the Go server's memory or state.
- **Controlled context access.** Plugin code cannot access the database, file system, or
  internal Go APIs directly. All platform interactions go through the JSON-RPC protocol, which
  exposes only the same context interfaces that built-in components use.
- **Secret access.** Plugins access secrets through `ctx.secrets.getKey()`, which delegates to
  the Go server's `SecretsContext`. Secrets are transmitted over the local stdio pipe (not a
  network socket) and are never logged.
- **HTTP request controls.** HTTP requests from plugins go through the Go server's managed HTTP
  client (via `ctx/http.request` RPC), inheriting connection pooling, TLS settings, and rate
  limits. Plugins can also make direct HTTP calls from Node.js — network policy enforcement is
  a future consideration.
- **Resource limits.** Execution timeouts, memory limits, and request caps prevent a single
  plugin from consuming unbounded resources.
- **Trust model.** Plugins are `.spx` archives explicitly installed by the operator via the CLI.
  There is no user-facing upload mechanism or auto-install. The operator controls which `.spx`
  files are installed — the same trust model as built-in Go components.

## Decisions

- **Distribution format**: Plugins are packaged as `.spx` archives (zip files containing
  `package.json` + bundled `extension.js`) and installed via a CLI command. The server never
  reads source files, `node_modules`, or build artifacts — only the pre-built bundle from the
  archive.
- **Bundling**: The packaging CLI uses esbuild to produce a single `extension.js` with all npm
  dependencies inlined. This eliminates `node_modules` from the installed plugin, keeps the
  archive small, and makes loading fast (one `require()` call, no module resolution).
- **Runtime**: Node.js child process (Plugin Host) for full language support, npm ecosystem
  access, native async/await, and TypeScript compatibility — at the cost of a cross-process
  communication layer.
- **Communication**: JSON-RPC 2.0 over stdio. Simple, well-understood protocol. No need for
  gRPC or HTTP between the processes — stdio is the lowest-latency local IPC option and
  requires no port allocation.
- **Single Plugin Host**: All plugins share one Node.js process. This keeps resource usage
  predictable. If isolation between plugins becomes important, we can move to one process per
  plugin later.
- **Manifest-driven registration**: Metadata is read from `package.json` without executing
  plugin code. This means the Go server knows about all contributions at startup, even for
  plugins that haven't been activated yet.
- **Lazy activation**: Plugins activate on first use. This keeps startup fast and memory low
  when only a subset of installed plugins are needed.
- **TypeScript-first**: The `@superplane/sdk` ships TypeScript definitions. Plugins can be
  written in JavaScript, but the primary authoring experience targets TypeScript.
- **Naming**: Plugin contributions use dot-separated names. No special prefix like `js.` — the
  name reflects the plugin's domain, not its implementation language.
- **Global scope**: Like built-in components, plugin contributions are available to all
  organizations on the instance. Per-organization plugin scoping is out of scope.
- **No file watching**: There is no file watcher polling for changes. Plugins are installed and
  uninstalled explicitly via the CLI, which signals the server to reload. This is simpler and
  more predictable than watching directories for changes.
