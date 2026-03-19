import type { RuntimeValue } from "../../context/runtime-value.js";
import type { InvocationEffects } from "../../effects/invoke-extension.js";

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

export interface InvokeExtensionJob {
  type: "invoke-extension";
  payload: InvocationPayload;
}

export type InvokeExtensionResult = InvocationOutput;
