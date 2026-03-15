# SDK Authoring Draft

## Purpose

This document defines the preferred extension authoring model for the TypeScript SDK.

Current direction:

- one file per integration, component, or trigger
- explicit composition in `index.ts`
- block objects implement TypeScript interfaces
- the SDK derives internal discovery and runtime wiring from those objects
- extension authors do not write operation registries or handler strings

## File Layout

Recommended structure:

```text
integrations/github.ts
components/create-issue.ts
components/close-issue.ts
triggers/...
index.ts
```

The folders are for organization only. The SDK should not rely on filesystem scanning. `index.ts` should explicitly import and compose the extension.

## Import Specifiers

Use `.js` file extensions in local imports inside `.ts` source files:

```ts
import { github } from "./integrations/github.js";
```

This repository uses Node ESM semantics with TypeScript `module` and `moduleResolution` set to `NodeNext`.
In that mode, source import specifiers should match the emitted runtime module specifiers.
Since TypeScript emits `.js` files, the imports in `.ts` source should also use `.js`.

Do not switch these imports to `.ts` unless the runtime/tooling changes to a TypeScript-aware execution model.

## Top-Level Extension Definition

```ts
import { defineExtension } from "@superplanehq/extension-sdk";
import { github } from "./integrations/github.js";
import { createIssue } from "./components/create-issue.js";
import { closeIssue } from "./components/close-issue.js";

export default defineExtension({
  metadata: {
    id: "github",
    name: "GitHub Extension",
    version: "0.1.0",
    description: "Create and close GitHub issues",
  },
  integrations: [github],
  components: [createIssue, closeIssue],
  triggers: [],
});
```

## Integration Interface

```ts
interface IntegrationDefinition {
  name: string;
  label: string;
  icon: string;
  description: string;
  instructions?: string;
  configuration: readonly ConfigurationField[];
  actions?: readonly ActionDefinition[];
  resourceTypes?: readonly string[];
  sync?(context: HandlerContext): Promise<void> | void;
  cleanup?(context: HandlerContext): Promise<void> | void;
  handleAction?(context: ActionHandlerContext): Promise<void> | void;
  listResources?(context: HandlerContext<ListResourcesInput>): Promise<IntegrationResource[]> | IntegrationResource[];
  handleRequest?(context: WebhookHandlerContext): Promise<WebhookResponse | void> | WebhookResponse | void;
  webhook?(): IntegrationWebhookHandler;
}
```

```ts
interface IntegrationWebhookHandler {
  setup?(context: IntegrationWebhookHandlerContext): Promise<RuntimeValue> | RuntimeValue;
  cleanup?(context: IntegrationWebhookHandlerContext): Promise<void> | void;
  compareConfig?(context: CompareWebhookConfigContext): Promise<boolean> | boolean;
  merge?(context: MergeWebhookConfigContext): Promise<{ merged: RuntimeValue; changed: boolean }> | { merged: RuntimeValue; changed: boolean };
}
```

Example:

```ts
import type { IntegrationDefinition } from "@superplanehq/extension-sdk";

export const github = {
  name: "github",
  label: "GitHub",
  icon: "github",
  description: "Create and manage GitHub issues",
  instructions: "## Create a GitHub Personal Access Token",
  configuration: [
    {
      name: "token",
      label: "Token",
      type: "string",
      required: true,
      sensitive: true,
    },
  ],
  resourceTypes: ["repository"],
  async sync({ runtime }) {
    runtime.integration.ready();
  },
} satisfies IntegrationDefinition;
```

## Component Interface

```ts
interface ComponentDefinition {
  name: string;
  label: string;
  description: string;
  documentation?: string;
  icon: string;
  color: string;
  integration?: string;
  configuration: readonly ConfigurationField[];
  outputChannels?: readonly OutputChannel[];
  actions?: readonly ActionDefinition[];
  setup?(context: HandlerContext): Promise<void> | void;
  processQueueItem?(context: HandlerContext): Promise<QueueProcessingResult | string | void> | QueueProcessingResult | string | void;
  execute(context: HandlerContext): Promise<void> | void;
  handleAction?(context: ActionHandlerContext): Promise<void> | void;
  handleWebhook?(context: WebhookHandlerContext): Promise<WebhookResponse | void> | WebhookResponse | void;
  cancel?(context: HandlerContext): Promise<void> | void;
  cleanup?(context: HandlerContext): Promise<void> | void;
  onIntegrationMessage?(context: IntegrationMessageHandlerContext): Promise<void> | void;
}
```

Example:

```ts
import type { ComponentDefinition } from "@superplanehq/extension-sdk";

export const createDnsRecord = {
  name: "github.createIssue",
  integration: "github",
  label: "Create Issue",
  description: "Create a GitHub issue in a repository",
  icon: "github",
  color: "gray",
  configuration: [],
  async setup() {},
  async execute({ runtime }) {
    runtime.executionState.pass();
  },
} satisfies ComponentDefinition;
```

## Trigger Interface

```ts
interface TriggerDefinition {
  name: string;
  label: string;
  description: string;
  documentation?: string;
  icon: string;
  color: string;
  integration?: string;
  configuration: readonly ConfigurationField[];
  exampleData?: RuntimeValue;
  actions?: readonly ActionDefinition[];
  setup?(context: HandlerContext): Promise<void> | void;
  handleAction?(context: ActionHandlerContext): Promise<RuntimeValue> | RuntimeValue;
  handleWebhook?(context: WebhookHandlerContext): Promise<WebhookResponse | void> | WebhookResponse | void;
  cleanup?(context: HandlerContext): Promise<void> | void;
  onIntegrationMessage?(context: IntegrationMessageHandlerContext): Promise<void> | void;
}
```

## Derived Data

The SDK should derive the following internally from the exported block objects:

- serialized manifest blocks
- which optional lifecycle methods exist
- runtime dispatch tables
- integration linkage from component/trigger `integration` references

Defaults and validation:

- `actions` defaults to `[]` when omitted
- `outputChannels` defaults to `DEFAULT_OUTPUT_CHANNEL` when omitted
- static metadata arrays should be treated as readonly authoring data
- `integration` on components and triggers may refer to an integration defined by the same extension or by another installed extension
- blocks using `integration-resource` fields should declare `integration`

The author should only provide block objects that satisfy the interfaces.

## Non-Goals

- implicit discovery by scanning directories
- author-written handler strings
- author-written operation registries

## Notes

- This model is intentionally close to the existing Go interfaces.
- The manifest remains the static description of the extension.
- Runtime dispatch remains an SDK/runtime concern, not something authors write manually.
