import type { HTTPContext } from "./http.js";
import type {
  EventContext,
  ExecutionStateContext,
  MetadataContext,
  RequestContext,
} from "./execution.js";
import type { IntegrationContext } from "./integration.js";
import type { RuntimeValue } from "./runtime-value.js";
import type { NodeWebhookContext, WebhookContext } from "./webhook.js";

export interface RuntimeLogger {
  debug(
    message: string,
    fields?: Record<string, RuntimeValue>,
  ): void | Promise<void>;
  info(
    message: string,
    fields?: Record<string, RuntimeValue>,
  ): void | Promise<void>;
  warn(
    message: string,
    fields?: Record<string, RuntimeValue>,
  ): void | Promise<void>;
  error(
    message: string,
    fields?: Record<string, RuntimeValue>,
  ): void | Promise<void>;
}

export interface RuntimeContext {
  logger: RuntimeLogger;
  http: HTTPContext;
  metadata: MetadataContext;
  requests: RequestContext;
  events: EventContext;
  executionState: ExecutionStateContext;
  integration: IntegrationContext;
  webhook: NodeWebhookContext;
}

export interface SetupHandlerContext<TConfiguration = RuntimeValue> {
  configuration: TConfiguration;
  context: RuntimeContext;
}

export interface ExecutionHandlerContext<
  TConfiguration = RuntimeValue,
  TData = RuntimeValue,
> {
  configuration: TConfiguration;
  data: TData;
  context: RuntimeContext;
}

export interface HandlerContext<
  TConfiguration = RuntimeValue,
  TInput = RuntimeValue,
> {
  configuration: TConfiguration;
  input: TInput;
  context: RuntimeContext;
}

export interface IntegrationWebhookHandlerContext {
  context: RuntimeContext;
  webhook: WebhookContext;
}

export interface CompareWebhookConfigContext {
  current: RuntimeValue;
  requested: RuntimeValue;
}

export interface MergeWebhookConfigContext {
  current: RuntimeValue;
  requested: RuntimeValue;
}

export interface WebhookHandlerContext<TConfiguration = RuntimeValue> {
  configuration: TConfiguration;
  body: Uint8Array;
  headers: Record<string, string[]>;
  context: RuntimeContext;
  findExecutionByKV?(key: string, value: string): Promise<RuntimeValue | null>;
}

export interface ActionHandlerContext<
  TConfiguration = RuntimeValue,
  TParameters = Record<string, RuntimeValue>,
> {
  name: string;
  configuration: TConfiguration;
  parameters: TParameters;
  context: RuntimeContext;
}

export interface IntegrationMessageHandlerContext<
  TConfiguration = RuntimeValue,
> {
  message: RuntimeValue;
  configuration: TConfiguration;
  context: RuntimeContext;
  findExecutionByKV?(key: string, value: string): Promise<RuntimeValue | null>;
}
