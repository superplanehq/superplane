# Runtime Context Draft

## Purpose

This document defines the first draft of the runtime context passed to extension handlers.

Unlike the manifest, this is not static metadata. It is the engine-provided capability surface made available at execution time.

Current direction:

- do not require manifest-level permissions in v1
- pass the same runtime context shape to all handlers
- keep privileged operations behind context objects rather than ambient runtime access

## Design Rules

- All privileged behavior flows through the runtime context.
- Extensions should not rely on ambient host capabilities.
- The same runtime context shape is passed to integrations, components, and triggers for now.
- Individual handlers may ignore parts of the context they do not need.

## Top-Level Runtime Context

```ts
interface RuntimeContext {
  logger: RuntimeLogger;
  http: HTTPContext;
  metadata: MetadataContext;
  requests: RequestContext;
  events: EventContext;
  executionState: ExecutionStateContext;
  integration: IntegrationContext;
  webhook: NodeWebhookContext;
}
```

This is intentionally broad in v1 so the SDK and engine stay simple.

## Generic Handler Contexts

### Standard Handler

```ts
interface HandlerContext<TConfiguration = RuntimeValue, TInput = RuntimeValue> {
  configuration: TConfiguration;
  input: TInput;
  runtime: RuntimeContext;
}
```

Use this for handlers such as:

- integration sync
- component setup
- component execute
- trigger setup

### Action Handler

```ts
interface ActionHandlerContext<TConfiguration = RuntimeValue, TParameters = Record<string, RuntimeValue>> {
  name: string;
  configuration: TConfiguration;
  parameters: TParameters;
  runtime: RuntimeContext;
}
```

Use this for:

- integration actions
- component actions
- trigger actions

### Webhook Handler

```ts
interface WebhookHandlerContext<TConfiguration = RuntimeValue> {
  configuration: TConfiguration;
  body: Uint8Array;
  headers: Record<string, string[]>;
  runtime: RuntimeContext;
  findExecutionByKV?(key: string, value: string): Promise<RuntimeValue | null>;
}
```

Use this for:

- integration HTTP/webhook handlers
- component webhook handlers
- trigger webhook handlers

### Integration Message Handler

```ts
interface IntegrationMessageHandlerContext<TConfiguration = RuntimeValue> {
  message: RuntimeValue;
  configuration: TConfiguration;
  runtime: RuntimeContext;
  findExecutionByKV?(key: string, value: string): Promise<RuntimeValue | null>;
}
```

Use this for:

- integration-aware components
- integration-aware triggers

### Integration Webhook Provisioner Handler

```ts
interface IntegrationWebhookHandlerContext {
  runtime: RuntimeContext;
  webhook: WebhookContext;
}
```

Use this for:

- integration `webhook.setup`
- integration `webhook.cleanup`

### Webhook Config Comparison

```ts
interface CompareWebhookConfigContext {
  current: RuntimeValue;
  requested: RuntimeValue;
}
```

### Webhook Config Merge

```ts
interface MergeWebhookConfigContext {
  current: RuntimeValue;
  requested: RuntimeValue;
}
```

## Context Objects

### Logger

```ts
interface RuntimeLogger {
  debug(message: string, fields?: Record<string, RuntimeValue>): void | Promise<void>;
  info(message: string, fields?: Record<string, RuntimeValue>): void | Promise<void>;
  warn(message: string, fields?: Record<string, RuntimeValue>): void | Promise<void>;
  error(message: string, fields?: Record<string, RuntimeValue>): void | Promise<void>;
}
```

### HTTP

```ts
interface HTTPContext {
  do(request: HTTPRequest): Promise<HTTPResponse>;
}
```

This mirrors the existing Go design: handlers should use the provided HTTP client abstraction rather than direct platform networking primitives.

### Metadata

```ts
interface MetadataContext {
  get(): Promise<RuntimeValue> | RuntimeValue;
  set(value: RuntimeValue): Promise<void> | void;
}
```

### Requests

```ts
interface RequestContext {
  scheduleActionCall(actionName: string, parameters: Record<string, RuntimeValue>, intervalMs: number): Promise<void> | void;
}
```

### Events

```ts
interface EventContext {
  emit(payloadType: string, payload: RuntimeValue): Promise<void> | void;
}
```

### Execution State

```ts
interface ExecutionStateContext {
  isFinished(): Promise<boolean> | boolean;
  setKV(key: string, value: string): Promise<void> | void;
  emit(channel: string, payloadType: string, payloads: RuntimeValue[]): Promise<void> | void;
  pass(): Promise<void> | void;
  fail(reason: string, message: string): Promise<void> | void;
}
```

### Integration

```ts
interface IntegrationContext {
  id(): Promise<string> | string;
  getMetadata(): Promise<RuntimeValue> | RuntimeValue;
  setMetadata(value: RuntimeValue): Promise<void> | void;
  getConfig(name: string): Promise<Uint8Array> | Uint8Array;
  ready(): Promise<void> | void;
  error(message: string): Promise<void> | void;
  newBrowserAction(action: BrowserAction): Promise<void> | void;
  removeBrowserAction(): Promise<void> | void;
  setSecret(name: string, value: Uint8Array): Promise<void> | void;
  getSecrets(): Promise<IntegrationSecret[]> | IntegrationSecret[];
  requestWebhook(configuration: RuntimeValue): Promise<void> | void;
  subscribe(configuration: RuntimeValue): Promise<string> | string;
  scheduleResync(intervalMs: number): Promise<void> | void;
  scheduleActionCall(actionName: string, parameters: RuntimeValue, intervalMs: number): Promise<void> | void;
  listSubscriptions(): Promise<IntegrationSubscription[]> | IntegrationSubscription[];
}
```

### Webhook

This is the provisioned webhook record used by integration webhook handlers.

```ts
interface WebhookContext {
  getID(): Promise<string> | string;
  getURL(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  getMetadata(): Promise<RuntimeValue> | RuntimeValue;
  getConfiguration(): Promise<RuntimeValue> | RuntimeValue;
  setSecret(secret: Uint8Array): Promise<void> | void;
}
```

### Node Webhook

This is the node-scoped webhook helper used by triggers/components that own their own webhook endpoint lifecycle.

```ts
interface NodeWebhookContext {
  setup(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  setSecret(secret: Uint8Array): Promise<void> | void;
  resetSecret(): Promise<{ previous: Uint8Array; current: Uint8Array }> | { previous: Uint8Array; current: Uint8Array };
  getBaseURL(): Promise<string> | string;
}
```

## Notes

- This is a shared context model, not a least-privilege design.
- Least-privilege or manifest-declared permissions can be added later without changing the extension block schema.
- The actual security boundary still depends on running the extension in a restricted execution backend with no ambient host access.
- `WebhookContext` and `NodeWebhookContext` are intentionally separate:
  - `WebhookContext` models the webhook provisioner record for integrations
  - `NodeWebhookContext` models node-level webhook helpers for triggers and components
