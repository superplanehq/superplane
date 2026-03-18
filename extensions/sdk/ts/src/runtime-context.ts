import type { ManifestJSONValue } from "./manifest-schema.js";

export type RuntimeValue = ManifestJSONValue;

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

export interface HTTPRequest {
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: string | Uint8Array;
}

export interface HTTPResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array;
}

export interface HTTPContext {
  do(request: HTTPRequest): Promise<HTTPResponse>;
}

export interface MetadataContext {
  get(): Promise<RuntimeValue> | RuntimeValue;
  set(value: RuntimeValue): Promise<void> | void;
}

export interface RequestContext {
  scheduleActionCall(
    actionName: string,
    parameters: Record<string, RuntimeValue>,
    intervalMs: number,
  ): Promise<void> | void;
}

export interface EventContext {
  emit(payloadType: string, payload: RuntimeValue): Promise<void> | void;
}

export interface ExecutionStateContext {
  isFinished(): Promise<boolean> | boolean;
  setKV(key: string, value: string): Promise<void> | void;
  emit(
    channel: string,
    payloadType: string,
    payloads: RuntimeValue[],
  ): Promise<void> | void;
  pass(): Promise<void> | void;
  fail(reason: string, message: string): Promise<void> | void;
}

export interface BrowserAction {
  description: string;
  url: string;
  method: string;
  formFields?: Record<string, string>;
}

export interface IntegrationSecret {
  name: string;
  value: Uint8Array;
}

export interface IntegrationSubscription {
  configuration(): Promise<RuntimeValue> | RuntimeValue;
  sendMessage(message: RuntimeValue): Promise<void> | void;
}

export interface IntegrationContext {
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
  scheduleActionCall(
    actionName: string,
    parameters: RuntimeValue,
    intervalMs: number,
  ): Promise<void> | void;
  listSubscriptions():
    | Promise<IntegrationSubscription[]>
    | IntegrationSubscription[];
}

export interface WebhookContext {
  getID(): Promise<string> | string;
  getURL(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  getMetadata(): Promise<RuntimeValue> | RuntimeValue;
  getConfiguration(): Promise<RuntimeValue> | RuntimeValue;
  setSecret(secret: Uint8Array): Promise<void> | void;
}

export interface NodeWebhookContext {
  setup(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  setSecret(secret: Uint8Array): Promise<void> | void;
  resetSecret():
    | Promise<{ previous: Uint8Array; current: Uint8Array }>
    | { previous: Uint8Array; current: Uint8Array };
  getBaseURL(): Promise<string> | string;
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
