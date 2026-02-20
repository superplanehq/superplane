import type { RpcTransport } from "./rpc";

export interface Disposable {
  dispose(): void;
}

export interface Logger {
  info(message: string, ...args: any[]): void;
  warn(message: string, ...args: any[]): void;
  error(message: string, ...args: any[]): void;
  debug(message: string, ...args: any[]): void;
}

export interface MetadataAccessor {
  get(): Promise<any>;
  set(value: any): Promise<void>;
}

export interface HTTPOptions {
  headers?: Record<string, string>;
  body?: string;
  timeout?: number;
}

export interface HTTPResponse {
  status: number;
  headers: Record<string, string>;
  body: any;
}

export interface HTTPClient {
  request(
    method: string,
    url: string,
    options?: HTTPOptions
  ): Promise<HTTPResponse>;
}

export interface SecretsAccessor {
  getKey(secretName: string, keyName: string): Promise<string>;
}

export interface WebhookAccessor {
  setup(): Promise<string>;
  getSecret(): Promise<string>;
}

export interface EventEmitter {
  emit(payloadType: string, payload: any): void;
}

// Component handler interfaces

export interface SetupContext {
  configuration: Record<string, any>;
  http: HTTPClient;
  metadata: MetadataAccessor;
  secrets: SecretsAccessor;
  webhook: WebhookAccessor;
  log: Logger;
}

export interface ExecutionContext {
  id: string;
  workflowId: string;
  organizationId: string;
  nodeId: string;
  sourceNodeId: string;
  baseUrl: string;
  input: any;
  configuration: Record<string, any>;
  eval(expression: string): Promise<Record<string, any>>;
  emit(channel: string, payloadType: string, data: any | any[]): void;
  pass(): void;
  fail(reason: string, message: string): void;
  setKV(key: string, value: string): void;
  metadata: MetadataAccessor;
  nodeMetadata: MetadataAccessor;
  http: HTTPClient;
  secrets: SecretsAccessor;
  integration: any;
  log: Logger;
}

export interface WebhookContext {
  body: string;
  headers: Record<string, string | string[]>;
  workflowId: string;
  nodeId: string;
  configuration: Record<string, any>;
  metadata: MetadataAccessor;
  webhook: WebhookAccessor;
  events: EventEmitter;
  http: HTTPClient;
  secrets: SecretsAccessor;
  log: Logger;
}

export interface TriggerSetupContext {
  configuration: Record<string, any>;
  http: HTTPClient;
  metadata: MetadataAccessor;
  webhook: WebhookAccessor;
  events: EventEmitter;
  secrets: SecretsAccessor;
  log: Logger;
}

export interface ComponentHandler {
  setup?(ctx: SetupContext): void | Promise<void>;
  execute(ctx: ExecutionContext): void | Promise<void>;
  cancel?(ctx: ExecutionContext): void | Promise<void>;
  cleanup?(ctx: SetupContext): void | Promise<void>;
}

export interface TriggerHandler {
  setup?(ctx: TriggerSetupContext): void | Promise<void>;
  cleanup?(ctx: TriggerSetupContext): void | Promise<void>;
  handleWebhook?(
    ctx: WebhookContext
  ): { status: number } | Promise<{ status: number }>;
}

export interface ComponentRegistry {
  register(name: string, handler: ComponentHandler): Disposable;
}

export interface TriggerRegistry {
  register(name: string, handler: TriggerHandler): Disposable;
}

export interface IntegrationRegistry {
  register(name: string, handler: any): Disposable;
}

export interface PluginContext {
  components: ComponentRegistry;
  triggers: TriggerRegistry;
  integrations: IntegrationRegistry;
  subscriptions: Disposable[];
  log: Logger;
}

// Internal implementation

export class PluginContextImpl implements PluginContext {
  components: ComponentRegistryImpl;
  triggers: TriggerRegistryImpl;
  integrations: IntegrationRegistryImpl;
  subscriptions: Disposable[] = [];
  log: Logger;

  constructor(
    private pluginId: string,
    private rpc: RpcTransport
  ) {
    this.components = new ComponentRegistryImpl();
    this.triggers = new TriggerRegistryImpl();
    this.integrations = new IntegrationRegistryImpl();
    this.log = createLogger(pluginId, rpc);
  }
}

export class ComponentRegistryImpl implements ComponentRegistry {
  private handlers = new Map<string, ComponentHandler>();

  register(name: string, handler: ComponentHandler): Disposable {
    this.handlers.set(name, handler);
    return {
      dispose: () => this.handlers.delete(name),
    };
  }

  getHandler(name: string): ComponentHandler | undefined {
    return this.handlers.get(name);
  }
}

export class TriggerRegistryImpl implements TriggerRegistry {
  private handlers = new Map<string, TriggerHandler>();

  register(name: string, handler: TriggerHandler): Disposable {
    this.handlers.set(name, handler);
    return {
      dispose: () => this.handlers.delete(name),
    };
  }

  getHandler(name: string): TriggerHandler | undefined {
    return this.handlers.get(name);
  }
}

export class IntegrationRegistryImpl implements IntegrationRegistry {
  private handlers = new Map<string, any>();

  register(name: string, handler: any): Disposable {
    this.handlers.set(name, handler);
    return {
      dispose: () => this.handlers.delete(name),
    };
  }

  getHandler(name: string): any {
    return this.handlers.get(name);
  }
}

function createLogger(prefix: string, rpc: RpcTransport): Logger {
  const log = (level: string, message: string) => {
    rpc.call("ctx/log", { level, message: `[${prefix}] ${message}` }).catch(
      () => {}
    );
  };

  return {
    info: (msg) => log("info", msg),
    warn: (msg) => log("warn", msg),
    error: (msg) => log("error", msg),
    debug: (msg) => log("debug", msg),
  };
}

/**
 * Build an ExecutionContext that records the plugin's action (emit/pass/fail)
 * and proxies context operations back to Go via RPC.
 */
export function buildExecutionContext(
  params: any,
  rpc: RpcTransport,
  pluginId: string
): { ctx: ExecutionContext; getResult: () => any } {
  let result: any = null;

  const executionId = params.id;

  const ctx: ExecutionContext = {
    id: params.id || "",
    workflowId: params.workflowId || "",
    organizationId: params.organizationId || "",
    nodeId: params.nodeId || "",
    sourceNodeId: params.sourceNodeId || "",
    baseUrl: params.baseUrl || "",
    input: params.input,
    configuration: params.configuration || {},

    emit(channel: string, payloadType: string, data: any) {
      result = { action: "emit", channel, payloadType, data };
    },

    pass() {
      result = { action: "pass" };
    },

    fail(reason: string, message: string) {
      result = { action: "fail", reason, message };
    },

    setKV(key: string, value: string) {
      result = { action: "setKV", key, value };
    },

    async eval(expression: string) {
      return rpc.call("ctx/eval", { executionId, expression });
    },

    metadata: buildMetadataAccessor(rpc, executionId, "execution"),
    nodeMetadata: buildMetadataAccessor(rpc, executionId, "node"),
    http: buildHTTPClient(rpc, executionId),
    secrets: buildSecretsAccessor(rpc, executionId),
    integration: {},
    log: createLogger(pluginId, rpc),
  };

  return { ctx, getResult: () => result };
}

export function buildSetupContext(
  params: any,
  rpc: RpcTransport,
  pluginId: string
): SetupContext {
  return {
    configuration: params.configuration || {},
    http: buildHTTPClient(rpc, "setup"),
    metadata: buildMetadataAccessor(rpc, "setup", "node"),
    secrets: buildSecretsAccessor(rpc, "setup"),
    webhook: buildWebhookAccessor(rpc, "setup"),
    log: createLogger(pluginId, rpc),
  };
}

export function buildTriggerSetupContext(
  params: any,
  rpc: RpcTransport,
  pluginId: string
): TriggerSetupContext {
  return {
    configuration: params.configuration || {},
    http: buildHTTPClient(rpc, "setup"),
    metadata: buildMetadataAccessor(rpc, "setup", "node"),
    webhook: buildWebhookAccessor(rpc, "setup"),
    events: buildEventEmitter(rpc, "setup"),
    secrets: buildSecretsAccessor(rpc, "setup"),
    log: createLogger(pluginId, rpc),
  };
}

export function buildWebhookContext(
  params: any,
  rpc: RpcTransport,
  pluginId: string
): WebhookContext {
  return {
    body: params.body || "",
    headers: params.headers || {},
    workflowId: params.workflowId || "",
    nodeId: params.nodeId || "",
    configuration: params.configuration || {},
    metadata: buildMetadataAccessor(rpc, "webhook", "node"),
    webhook: buildWebhookAccessor(rpc, "webhook"),
    events: buildEventEmitter(rpc, "webhook"),
    http: buildHTTPClient(rpc, "webhook"),
    secrets: buildSecretsAccessor(rpc, "webhook"),
    log: createLogger(pluginId, rpc),
  };
}

function buildMetadataAccessor(
  rpc: RpcTransport,
  contextId: string,
  scope: string
): MetadataAccessor {
  return {
    async get() {
      return rpc.call("ctx/metadata.get", { contextId, scope });
    },
    async set(value: any) {
      await rpc.call("ctx/metadata.set", { contextId, scope, value });
    },
  };
}

function buildHTTPClient(rpc: RpcTransport, contextId: string): HTTPClient {
  return {
    async request(method, url, options) {
      return rpc.call("ctx/http.request", {
        contextId,
        method,
        url,
        options,
      });
    },
  };
}

function buildSecretsAccessor(
  rpc: RpcTransport,
  contextId: string
): SecretsAccessor {
  return {
    async getKey(secretName, keyName) {
      return rpc.call("ctx/secrets.getKey", {
        contextId,
        secretName,
        keyName,
      });
    },
  };
}

function buildWebhookAccessor(
  rpc: RpcTransport,
  contextId: string
): WebhookAccessor {
  return {
    async setup() {
      return rpc.call("ctx/webhook.setup", { contextId });
    },
    async getSecret() {
      return rpc.call("ctx/webhook.getSecret", { contextId });
    },
  };
}

function buildEventEmitter(
  rpc: RpcTransport,
  contextId: string
): EventEmitter {
  return {
    emit(payloadType: string, payload: any) {
      rpc
        .call("ctx/events.emit", { contextId, payloadType, payload })
        .catch(() => {});
    },
  };
}
