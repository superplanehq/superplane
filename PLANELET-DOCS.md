# SuperPlane Planelet SDK

Build custom SuperPlane integrations without modifying the SuperPlane codebase. Write a Planelet server with the TypeScript SDK, point SuperPlane at it, and your actions appear natively in the canvas UI.

## Architecture

```
┌──────────────┐           ┌──────────────────┐         ┌────────────────┐
│  SuperPlane  │ ──GET───▶ │ Planelet Server  │         │   Your Code    │
│  (Planelets  │ /manifest │  (SDK-powered)   │◀────────│   (actions,    │
│  Integration)│           │                  │         │    logic)      │
│              │ ──POST──▶ │ /actions/:name/  │         │                │
│              │           │    execute       │         │                │
│              │ ◀──POST── │ POST events to   │         │                │
│              │           │ SuperPlane       │         │                │
└──────────────┘           └──────────────────┘         └────────────────┘
```

1. You build a Planelet server using the SDK
2. SuperPlane's Planelets integration connects to your server
3. On setup, it fetches your manifest to discover available actions
4. When a canvas node runs your action, SuperPlane proxies the execution to your server
5. Your server can push events back to SuperPlane to trigger workflows

## Quick Start

### 1. Create a new project

```bash
mkdir my-planelet && cd my-planelet
bun init -y
bun add @superplane/planelet-sdk
```

### 2. Write your Planelet

```typescript
// index.ts
import { createPlanelet } from "@superplane/planelet-sdk";

const planelet = createPlanelet({
  name: "my-planelet",
  label: "My Planelet",
  description: "Does useful things",
});

planelet.action("hello", {
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

planelet.listen(3001);
```

### 3. Run it

```bash
bun run index.ts
# Planelet server "my-planelet" listening on port 3001
```

### 4. Connect to SuperPlane

1. In SuperPlane, add a new **Planelets** integration
2. Set **Server URL** to your Planelet server's address (e.g. `https://my-planelet.example.com`)
3. Optionally set an **Auth Token**
4. Save — SuperPlane fetches your manifest and the integration goes ready

### 5. Use in a canvas

Add a **Run Planelet Action** node to your canvas. Select your action from the dropdown. Fill in the fields. Done.

## Protocol Reference

The SDK handles all of this for you, but if you want to build a Planelet server in another language, here's the protocol.

### `GET /manifest`

Returns the Planelet's metadata and available actions.

**Response:**

```json
{
  "name": "my-planelet",
  "label": "My Planelet",
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

To push events from your Planelet server into SuperPlane, POST to:

```
POST /api/v1/integrations/{integration_id}/events
Content-Type: application/json

{
  "eventType": "my.event.type",
  "payload": { ... }
}
```

Use the **On Planelet Event** trigger in your canvas to listen for these events. You can filter by `eventType`.

## SDK API Reference

### `createPlanelet(options)`

Creates a new Planelet builder.

```typescript
const planelet = createPlanelet({
  name: "my-planelet",       // required, unique identifier
  label: "My Planelet",      // optional, display name (defaults to name)
  icon: "puzzle",           // optional, icon identifier
  description: "...",       // optional
});
```

### `.action(name, definition)`

Registers an action.

```typescript
planelet.action("do-thing", {
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
planelet.listen(3001);
planelet.listen(3001, () => console.log("Ready!"));
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

If your Planelet server requires authentication, set the **Auth Token** in the SuperPlane Planelets integration config. The token is sent as a `Bearer` token in the `Authorization` header on every request from SuperPlane to your server.

To validate it server-side, add middleware to your Express app (the SDK doesn't enforce auth by default):

```typescript
// Example: add auth middleware before creating the Planelet
// This is outside the SDK — use standard Express patterns
```

## Deploying

Your Planelet server is a standard HTTP server. Deploy it anywhere SuperPlane can reach:

- **Local development**: `localhost` or tunnel (ngrok, cloudflared)
- **Cloud**: any container platform (Railway, Fly.io, Cloud Run, ECS)
- **Self-hosted**: any server with a public or VPN-accessible URL

The server must be reachable from SuperPlane at the configured URL. HTTPS is recommended for production.

## Example

See [`sdk/example/`](sdk/example/) for a complete example Planelet with two actions (random quotes and greetings). Run it:

```bash
cd sdk/example
bun install
bun run index.ts
```

Then connect SuperPlane to `http://localhost:3001`.
