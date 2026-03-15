# Invocation Envelope Draft

## Purpose

This document defines the stable execution envelope sent from the engine to a packaged extension operation.

This sits between:

- operation discovery
- engine dispatch
- runtime-context population inside the worker

## Design Rules

- The engine sends a structured envelope, not an ad hoc JSON blob.
- The envelope names the exact target operation explicitly.
- Runtime metadata belongs inside the envelope.
- The envelope should be generic enough to support integrations, components, triggers, actions, webhooks, and integration messages.

## Envelope Shape

```ts
interface InvocationTarget {
  blockType: "integrations" | "components" | "triggers";
  blockName: string;
  operation: string;
}

interface InvocationPayload {
  target: InvocationTarget;
  configuration?: RuntimeValue;
  input?: RuntimeValue;
  current?: RuntimeValue;
  requested?: RuntimeValue;
  parameters?: Record<string, RuntimeValue>;
  actionName?: string;
  headers?: Record<string, string[]>;
  body?: RuntimeValue;
  message?: RuntimeValue;
  integration?: {
    id?: string;
    configuration?: Record<string, RuntimeValue>;
    metadata?: RuntimeValue;
  };
  webhook?: {
    id?: string;
    url?: string;
    secret?: RuntimeValue;
    metadata?: RuntimeValue;
    configuration?: RuntimeValue;
  };
  metadata?: RuntimeValue;
}
```

## Meaning

- `target`
  - identifies the block type, block name, and operation being invoked
- `configuration`
  - block configuration for the current invocation
- `input`
  - generic runtime input payload
- `parameters`
  - action parameters when the operation is an action handler
- `actionName`
  - action identifier for integration/component/trigger action handlers
- `current`
  - current webhook configuration for integration `webhook.compareConfig` / `webhook.merge`
- `requested`
  - requested webhook configuration for integration `webhook.compareConfig` / `webhook.merge`
- `headers`
  - HTTP headers for request or webhook-style handlers
- `body`
  - request body for request or webhook-style handlers
- `message`
  - integration message payload for integration-aware blocks
- `integration`
  - integration-scoped state passed into the invocation
- `webhook`
  - provisioned webhook record state passed into integration webhook setup/cleanup
- `metadata`
  - engine-provided runtime metadata

## Normalization Rules

Inside the worker runtime, the payload is normalized into a runtime-facing envelope:

- missing `configuration` becomes `{}`
- missing `input` becomes `{}`
- missing `current` becomes `{}`
- missing `requested` becomes `{}`
- missing `parameters` becomes `{}`
- missing `actionName` becomes `""`
- missing `headers` becomes `{}`
- missing `body` becomes an empty byte array
- missing `message` becomes `input`
- missing `integration` becomes an empty integration context
- missing `webhook` becomes an empty provisioned webhook record
- missing `metadata` becomes `null`

The normalized envelope is what powers the runtime harness and handler contexts.

## Current CLI Behavior

The CLI still accepts:

- `--operation`
- `--input`
- `--metadata`

For now:

- `--operation` is parsed into `envelope.target`
- `--input` is parsed as a partial invocation envelope
- `--metadata` is written into `envelope.metadata`

That keeps the current CLI usable while the later CLI/API cleanup is still pending.
