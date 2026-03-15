import type {
  ActionHandlerContext,
  CompareWebhookConfigContext,
  HandlerContext,
  IntegrationWebhookHandlerContext,
  IntegrationMessageHandlerContext,
  MergeWebhookConfigContext,
  RuntimeValue,
  WebhookHandlerContext,
} from "./runtime-context.js";
import type {
  ActionDefinition,
  ConfigurationField,
  ExtensionMetadata,
  OutputChannel,
  RuntimeDescriptor,
} from "./manifest-schema.js";

export const DEFAULT_OUTPUT_CHANNEL: OutputChannel = {
  name: "default",
  label: "Default",
};

export interface IntegrationResource {
  type: string;
  name: string;
  id: string;
}

export interface WebhookResponse {
  status: number;
  headers?: Record<string, string>;
  body?: Uint8Array | string;
  contentType?: string;
}

export interface QueueProcessingResult {
  handled?: boolean;
  executionId?: string;
}

export interface ListResourcesInput {
  resourceType: string;
  parameters?: Record<string, string>;
}

export type VoidHandler<
  TConfiguration = RuntimeValue,
  TInput = RuntimeValue,
> = (context: HandlerContext<TConfiguration, TInput>) => Promise<void> | void;

export type ValueHandler<
  TOutput = RuntimeValue,
  TConfiguration = RuntimeValue,
  TInput = RuntimeValue,
> = (
  context: HandlerContext<TConfiguration, TInput>,
) => Promise<TOutput> | TOutput;

export type ActionHandler<
  TConfiguration = RuntimeValue,
  TParameters = Record<string, RuntimeValue>,
  TOutput = RuntimeValue,
> = (
  context: ActionHandlerContext<TConfiguration, TParameters>,
) => Promise<TOutput> | TOutput;

export type WebhookHandler<
  TConfiguration = RuntimeValue,
  TOutput = WebhookResponse | void,
> = (
  context: WebhookHandlerContext<TConfiguration>,
) => Promise<TOutput> | TOutput;

export type IntegrationMessageHandler<
  TConfiguration = RuntimeValue,
  TOutput = RuntimeValue,
> = (
  context: IntegrationMessageHandlerContext<TConfiguration>,
) => Promise<TOutput> | TOutput;

export interface IntegrationWebhookHandler<TConfiguration = RuntimeValue> {
  setup?(
    context: IntegrationWebhookHandlerContext,
  ): Promise<RuntimeValue> | RuntimeValue;
  cleanup?(context: IntegrationWebhookHandlerContext): Promise<void> | void;
  compareConfig?(
    context: CompareWebhookConfigContext,
  ): Promise<boolean> | boolean;
  merge?(
    context: MergeWebhookConfigContext,
  ):
    | Promise<{ merged: RuntimeValue; changed: boolean }>
    | { merged: RuntimeValue; changed: boolean };
}

export interface BaseBlockDefinition {
  name: string;
  label: string;
  description: string;
  icon: string;
}

export interface IntegrationDefinition<TConfiguration = RuntimeValue>
  extends BaseBlockDefinition {
  instructions?: string;
  configuration: readonly ConfigurationField[];
  actions?: readonly ActionDefinition[];
  resourceTypes?: readonly string[];
  sync?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  cleanup?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  handleAction?(
    context: ActionHandlerContext<TConfiguration, Record<string, RuntimeValue>>,
  ): Promise<void> | void;
  listResources?(
    context: HandlerContext<TConfiguration, ListResourcesInput>,
  ): Promise<IntegrationResource[]> | IntegrationResource[];
  handleRequest?(
    context: WebhookHandlerContext<TConfiguration>,
  ): Promise<WebhookResponse | void> | WebhookResponse | void;
  webhook?(): IntegrationWebhookHandler<TConfiguration>;
}

export interface ComponentDefinition<TConfiguration = RuntimeValue>
  extends BaseBlockDefinition {
  integration?: string;
  documentation?: string;
  color: string;
  outputChannels?: readonly OutputChannel[];
  configuration: readonly ConfigurationField[];
  actions?: readonly ActionDefinition[];
  setup?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  processQueueItem?(
    context: HandlerContext<TConfiguration>,
  ):
    | Promise<QueueProcessingResult | string | void>
    | QueueProcessingResult
    | string
    | void;
  execute(context: HandlerContext<TConfiguration>): Promise<void> | void;
  handleAction?(
    context: ActionHandlerContext<TConfiguration, Record<string, RuntimeValue>>,
  ): Promise<void> | void;
  handleWebhook?(
    context: WebhookHandlerContext<TConfiguration>,
  ): Promise<WebhookResponse | void> | WebhookResponse | void;
  cancel?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  cleanup?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  onIntegrationMessage?(
    context: IntegrationMessageHandlerContext<TConfiguration>,
  ): Promise<void> | void;
}

export interface TriggerDefinition<TConfiguration = RuntimeValue>
  extends BaseBlockDefinition {
  integration?: string;
  documentation?: string;
  color: string;
  configuration: readonly ConfigurationField[];
  actions?: readonly ActionDefinition[];
  setup?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  handleAction?(
    context: ActionHandlerContext<TConfiguration, Record<string, RuntimeValue>>,
  ): Promise<RuntimeValue> | RuntimeValue;
  handleWebhook?(
    context: WebhookHandlerContext<TConfiguration>,
  ): Promise<WebhookResponse | void> | WebhookResponse | void;
  cleanup?(context: HandlerContext<TConfiguration>): Promise<void> | void;
  onIntegrationMessage?(
    context: IntegrationMessageHandlerContext<TConfiguration>,
  ): Promise<void> | void;
}

export interface ExtensionDefinition {
  metadata: ExtensionMetadata;
  runtime?: RuntimeDescriptor;
  integrations?: readonly IntegrationDefinition<any>[];
  components?: readonly ComponentDefinition<any>[];
  triggers?: readonly TriggerDefinition<any>[];
}

export function defineExtension<TDefinition extends ExtensionDefinition>(
  definition: TDefinition,
): TDefinition {
  return definition;
}
