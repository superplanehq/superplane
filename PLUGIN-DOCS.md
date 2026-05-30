# SuperPlane Plugin SDK

Build custom SuperPlane integrations without modifying the SuperPlane codebase. Write a plugin server with the TypeScript SDK, point SuperPlane at it, and your actions appear natively in the canvas UI.

## Architecture

```
┌──────────────┐         ┌──────────────────┐         ┌────────────────┐
│  SuperPlane  │──GET───▶│  Plugin Server   │         │   Your Code    │
│  (Plugin     │ /manifest│  (SDK-powered)   │◀────────│   (actions,    │
│  Integration)│         │                  │         │    logic)      │
│              │──POST──▶│ /actions/:name/  │         │                │
│              │         │    execute       │         │                │
│              │◀──POST──│ POST events to   │         │                │
│              │         │ SuperPlane       │         │                │
└──────────────┘         └──────────────────┘         └────────────────┘
```

1. You build a plugin server using the SDK
2. SuperPlane's Plugin integration connects to your server
3. On setup, it fetches your manifest to discover available actions
4. When a canvas node runs your action, SuperPlane proxies the execution to your server
5. Your server can push events back to SuperPlane to trigger workflows

## Quick Start

### 1. Create a new project

```bash
mkdir my-plugin && cd my-plugin
bun init -y
bun add @superplane/plugin-sdk
```

### 2. Write your plugin

```typescript
// index.ts
import { createPlugin } from "@superplane/plugin-sdk";

const plugin = createPlugin({
  name: "my-plugin",
  label: "My Plugin",
  description: "Does useful things",
});

plugin.action("hello", {
  label: "Say Hello",
  description: "Generates a greeting",
  fields: {
    name: {
      label: "Name",
      type: "string",
      required: true,
      description: "Who to greet",
    },
  },
  execute: async (params) => {
    return { message: `Hello, ${params.name}!` };
  },
});

plugin.listen(3001);
```

### 3. Run it

```bash
bun run index.ts
# Plugin server "my-plugin" listening on port 3001
```

### 4. Connect to SuperPlane

1. In SuperPlane, add a new **Plugin** integration
2. Set **Server URL** to your plugin server's address (e.g. `https://my-plugin.example.com`)
3. Optionally set an **Auth Token**
4. Save — SuperPlane fetches your manifest and the integration goes ready

### 5. Use in a canvas

Add a **Run Plugin Action** node to your canvas. Select your action from the dropdown. Fill in the fields. Done.

## Protocol Reference

The SDK handles all of this for you, but if you want to build a plugin server in another language, here's the protocol.

### `GET /manifest`

Returns the plugin's metadata and available actions.

**Response:**

```json
{
  "name": "my-plugin",
  "label": "My Plugin",
  "icon": "puzzle",
  "description": "Does useful things",
  "actions": [
    {
      "name": "hello",
      "label": "Say Hello",
      "description": "Generates a greeting",
      "fields": [
        {
          "name": "name",
          "label": "Name",
          "type": "string",
          "description": "Who to greet",
          "required": true
        }
      ]
    }
  ]
}
```

### `POST /actions/{name}/execute`

Executes an action.

**Request:**

```json
{
  "parameters": {
    "name": "World"
  },
  "input": { ... }
}
```

- `parameters`: the field values configured by the user
- `input`: data from the upstream node in the canvas (may be `null`)

**Success response (200):**

```json
{
  "success": true,
  "data": {
    "message": "Hello, World!"
  }
}
```

**Error response (4xx/5xx):**

```json
{
  "success": false,
  "error": "Something went wrong"
}
```

### Events (Triggers)

To push events from your plugin server into SuperPlane, POST to:

```
POST /api/v1/integrations/{integration_id}/events
Content-Type: application/json

{
  "eventType": "my.event.type",
  "payload": { ... }
}
```

Use the **On Plugin Event** trigger in your canvas to listen for these events. You can filter by `eventType`.

## SDK API Reference

### `createPlugin(options)`

Creates a new plugin builder.

```typescript
const plugin = createPlugin({
  name: "my-plugin",       // required, unique identifier
  label: "My Plugin",      // optional, display name (defaults to name)
  icon: "puzzle",           // optional, icon identifier
  description: "...",       // optional
});
```

### `.action(name, definition)`

Registers an action.

```typescript
plugin.action("do-thing", {
  label: "Do Thing",                    // required, display name
  description: "Does a thing",         // optional
  fields: {                            // required, config fields
    myField: {
      label: "My Field",
      type: "string",
      required: true,
      description: "What it does",
    },
  },
  execute: async (params, ctx) => {    // required, execution handler
    // params = { myField: "value" }
    // ctx.input = data from upstream node
    return { result: "ok" };           // must return an object
  },
});
```

Returns `this` for chaining.

### `.listen(port, callback?)`

Starts the HTTP server.

```typescript
plugin.listen(3001);
plugin.listen(3001, () => console.log("Ready!"));
```

## Field Types

| Type | Description | Extra options |
|------|-------------|---------------|
| `string` | Single-line text input | — |
| `text` | Multi-line text input | — |
| `number` | Numeric input | — |
| `bool` | Boolean checkbox | — |
| `select` | Dropdown selection | `options: [{label, value}]` |
| `object` | JSON object editor | — |

### Field definition

```typescript
{
  label: "Display Name",        // required
  type: "string",               // required
  description: "Help text",     // optional
  required: true,               // optional, default false
  default: "value",             // optional
  options: [                    // required for "select" type
    { label: "Option A", value: "a" },
    { label: "Option B", value: "b" },
  ],
}
```

## Authentication

If your plugin server requires authentication, set the **Auth Token** in the SuperPlane Plugin integration config. The token is sent as a `Bearer` token in the `Authorization` header on every request from SuperPlane to your server.

To validate it server-side, add middleware to your Express app (the SDK doesn't enforce auth by default):

```typescript
// Example: add auth middleware before creating the plugin
// This is outside the SDK — use standard Express patterns
```

## Deploying

Your plugin server is a standard HTTP server. Deploy it anywhere SuperPlane can reach:

- **Local development**: `localhost` or tunnel (ngrok, cloudflared)
- **Cloud**: any container platform (Railway, Fly.io, Cloud Run, ECS)
- **Self-hosted**: any server with a public or VPN-accessible URL

The server must be reachable from SuperPlane at the configured URL. HTTPS is recommended for production.

## Example

See [`sdk/example/`](sdk/example/) for a complete example plugin with two actions (random quotes and greetings). Run it:

```bash
cd sdk/example
bun install
bun run index.ts
```

Then connect SuperPlane to `http://localhost:3001`.
