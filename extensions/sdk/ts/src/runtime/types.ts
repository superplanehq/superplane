import type { ManifestJSONValue, ManifestV1 } from "../manifest-schema.js";
import type { RuntimeContext, RuntimeValue } from "../runtime-context.js";

export interface InvocationIntegrationContext {
  id?: string;
  configuration?: Record<string, RuntimeValue>;
  metadata?: RuntimeValue;
}

export interface InvocationContext {
  configuration?: RuntimeValue;
  integration?: InvocationIntegrationContext;
  metadata?: RuntimeValue;
}

export type ComponentOperation =
  | "setup"
  | "execute"
  | "handleAction"
  | "handleWebhook"
  | "cancel"
  | "onIntegrationMessage";

export interface ComponentInvocationTarget<
  TOperation extends ComponentOperation = ComponentOperation,
> {
  blockType: "components";
  blockName: string;
  operation: TOperation;
}

export type InvocationTarget = ComponentInvocationTarget;

export interface ComponentSetupInvocation {}

export interface ComponentExecutionInvocation {
  data?: RuntimeValue;
}

export interface ComponentHandleActionInvocation {
  name: string;
  parameters?: Record<string, RuntimeValue>;
}

export interface ComponentHandleWebhookInvocation {
  headers?: Record<string, string[]>;
  body?: RuntimeValue;
}

export interface ComponentOnIntegrationMessageInvocation {
  message?: RuntimeValue;
}

interface BaseInvocationPayload<
  TOperation extends ComponentOperation,
  TInvocation,
> {
  target: ComponentInvocationTarget<TOperation>;
  context?: InvocationContext;
  invocation?: TInvocation;
}

export interface ComponentSetupInvocationPayload
  extends BaseInvocationPayload<"setup", ComponentSetupInvocation> {}

export interface ComponentExecuteInvocationPayload
  extends BaseInvocationPayload<"execute", ComponentExecutionInvocation> {}

export interface ComponentCancelInvocationPayload
  extends BaseInvocationPayload<"cancel", ComponentExecutionInvocation> {}

export interface ComponentHandleActionInvocationPayload
  extends Omit<
    BaseInvocationPayload<"handleAction", ComponentHandleActionInvocation>,
    "invocation"
  > {
  invocation: ComponentHandleActionInvocation;
}

export interface ComponentHandleWebhookInvocationPayload
  extends BaseInvocationPayload<
    "handleWebhook",
    ComponentHandleWebhookInvocation
  > {}

export interface ComponentOnIntegrationMessageInvocationPayload
  extends BaseInvocationPayload<
    "onIntegrationMessage",
    ComponentOnIntegrationMessageInvocation
  > {}

export type InvocationPayload =
  | ComponentSetupInvocationPayload
  | ComponentExecuteInvocationPayload
  | ComponentCancelInvocationPayload
  | ComponentHandleActionInvocationPayload
  | ComponentHandleWebhookInvocationPayload
  | ComponentOnIntegrationMessageInvocationPayload;

export interface NormalizedInvocationIntegrationContext {
  id: string;
  configuration: Record<string, RuntimeValue>;
  metadata: RuntimeValue;
}

export interface NormalizedInvocationContext {
  configuration: RuntimeValue;
  integration: NormalizedInvocationIntegrationContext;
  metadata: RuntimeValue;
}

interface BaseInvocationEnvelope<
  TOperation extends ComponentOperation,
  TInvocation,
> {
  target: ComponentInvocationTarget<TOperation>;
  context: NormalizedInvocationContext;
  invocation: TInvocation;
}

export interface ComponentSetupInvocationEnvelope
  extends BaseInvocationEnvelope<"setup", ComponentSetupInvocation> {}

export interface ComponentExecuteInvocationEnvelope
  extends BaseInvocationEnvelope<"execute", { data: RuntimeValue }> {}

export interface ComponentCancelInvocationEnvelope
  extends BaseInvocationEnvelope<"cancel", { data: RuntimeValue }> {}

export interface ComponentHandleActionInvocationEnvelope
  extends BaseInvocationEnvelope<
    "handleAction",
    { name: string; parameters: Record<string, RuntimeValue> }
  > {}

export interface ComponentHandleWebhookInvocationEnvelope
  extends BaseInvocationEnvelope<
    "handleWebhook",
    { headers: Record<string, string[]>; body: Uint8Array }
  > {}

export interface ComponentOnIntegrationMessageInvocationEnvelope
  extends BaseInvocationEnvelope<
    "onIntegrationMessage",
    { message: RuntimeValue }
  > {}

export type InvocationEnvelope =
  | ComponentSetupInvocationEnvelope
  | ComponentExecuteInvocationEnvelope
  | ComponentCancelInvocationEnvelope
  | ComponentHandleActionInvocationEnvelope
  | ComponentHandleWebhookInvocationEnvelope
  | ComponentOnIntegrationMessageInvocationEnvelope;

export interface InvocationScheduledAction {
  actionName: string;
  parameters: Record<string, RuntimeValue>;
  intervalMs: number;
}

export interface InvocationEvent {
  payloadType: string;
  payload: RuntimeValue;
}

export interface InvocationExecutionEmission {
  channel: string;
  payloadType: string;
  payloads: RuntimeValue[];
}

export interface InvocationExecutionStateEffects {
  finished: boolean;
  passed: boolean;
  failed: { reason: string; message: string } | null;
  kv: Record<string, string>;
  emissions: InvocationExecutionEmission[];
}

export interface InvocationIntegrationSecret {
  name: string;
  value: RuntimeValue;
}

export interface InvocationIntegrationSubscription {
  id: string;
  configuration: RuntimeValue;
  messages: RuntimeValue[];
}

export interface InvocationIntegrationEffects {
  id: string;
  ready: boolean;
  error?: string;
  metadata: RuntimeValue;
  browserAction: RuntimeValue;
  requestedWebhooks: RuntimeValue[];
  scheduledResyncIntervalMs: number | null;
  scheduledActions: Array<{
    actionName: string;
    parameters: RuntimeValue;
    intervalMs: number;
  }>;
  secrets: InvocationIntegrationSecret[];
  subscriptions: InvocationIntegrationSubscription[];
}

export interface InvocationWebhookEffects {
  url: string;
  baseURL: string;
  secret: RuntimeValue;
}

export interface InvocationEffects {
  metadata: RuntimeValue;
  requests: {
    scheduledActions: InvocationScheduledAction[];
  };
  events: InvocationEvent[];
  executionState: InvocationExecutionStateEffects;
  integration: InvocationIntegrationEffects;
  webhook: InvocationWebhookEffects;
}

interface BaseInvocationOutput<TOperation extends ComponentOperation> {
  target: ComponentInvocationTarget<TOperation>;
  effects: InvocationEffects;
}

export interface ComponentSetupInvocationOutput
  extends BaseInvocationOutput<"setup"> {}

export interface ComponentExecuteInvocationOutput
  extends BaseInvocationOutput<"execute"> {}

export interface ComponentCancelInvocationOutput
  extends BaseInvocationOutput<"cancel"> {}

export interface ComponentHandleActionInvocationOutput
  extends BaseInvocationOutput<"handleAction"> {}

export interface ComponentHandleWebhookInvocationOutput
  extends BaseInvocationOutput<"handleWebhook"> {
  response: RuntimeValue;
}

export interface ComponentOnIntegrationMessageInvocationOutput
  extends BaseInvocationOutput<"onIntegrationMessage"> {}

export type InvocationOutput =
  | ComponentSetupInvocationOutput
  | ComponentExecuteInvocationOutput
  | ComponentCancelInvocationOutput
  | ComponentHandleActionInvocationOutput
  | ComponentHandleWebhookInvocationOutput
  | ComponentOnIntegrationMessageInvocationOutput;

export interface DiscoveredOperation {
  name: string;
  description?: string;
  inputSchema?: ManifestJSONValue;
  outputSchema?: ManifestJSONValue;
}

export interface ExtensionDiscovery {
  manifest: ManifestV1;
  operations: DiscoveredOperation[];
}

export type OperationHandler = (
  envelope: InvocationEnvelope,
) => Promise<InvocationOutput>;

export interface OperationDescriptor {
  target: InvocationTarget;
  name: string;
  description: string;
  invoke: OperationHandler;
}

export interface RuntimeHarness {
  context: RuntimeContext;
  snapshot(): InvocationEffects;
}
