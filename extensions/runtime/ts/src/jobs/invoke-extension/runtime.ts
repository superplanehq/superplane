import type {
  RuntimeContext,
  RuntimeLogger,
} from "../../context/runtime-context.js";
import type {
  HTTPContext,
  HTTPRequest,
  HTTPResponse,
} from "../../context/http.js";
import type {
  EventContext,
  ExecutionStateContext,
  MetadataContext,
  RequestContext,
} from "../../context/execution.js";
import type {
  BrowserAction,
  IntegrationContext,
  IntegrationSecret,
  IntegrationSubscription,
} from "../../context/integration.js";
import type { NodeWebhookContext } from "../../context/webhook.js";
import type { RuntimeValue } from "../../context/runtime-value.js";
import type { InvocationEffects } from "../../effects/invoke-extension.js";
import type {
  ComponentCancelInvocationPayload,
  ComponentExecuteInvocationPayload,
  ComponentHandleActionInvocationPayload,
  ComponentHandleWebhookInvocationPayload,
  ComponentOnIntegrationMessageInvocationPayload,
  ComponentSetupInvocationPayload,
  InvocationContext,
  InvocationEnvelope,
  InvocationIntegrationContext,
  InvocationPayload,
  NormalizedInvocationContext,
} from "./types.js";

export interface InvokeExtensionRuntime {
  context: RuntimeContext;
  snapshot(): InvocationEffects;
}

export function normalizeInvocationEnvelope(
  payload: InvocationPayload,
): InvocationEnvelope {
  const context = normalizeInvocationContext(payload.context);

  switch (payload.target.operation) {
    case "setup":
      payload = payload as ComponentSetupInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: payload.invocation ?? {},
      };
    case "execute":
      payload = payload as ComponentExecuteInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: {
          data: payload.invocation?.data ?? null,
        },
      };
    case "cancel":
      payload = payload as ComponentCancelInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: {
          data: payload.invocation?.data ?? null,
        },
      };
    case "handleAction":
      payload = payload as ComponentHandleActionInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: {
          name: payload.invocation.name,
          parameters: payload.invocation.parameters ?? {},
        },
      };
    case "handleWebhook":
      payload = payload as ComponentHandleWebhookInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: {
          headers: payload.invocation?.headers ?? {},
          body: normalizeBody(payload.invocation?.body),
        },
      };
    case "onIntegrationMessage":
      payload = payload as ComponentOnIntegrationMessageInvocationPayload;
      return {
        target: payload.target,
        context,
        invocation: {
          message: payload.invocation?.message ?? null,
        },
      };
  }
}

export function createInvokeExtensionRuntime(
  invocation: InvocationEnvelope,
): InvokeExtensionRuntime {
  let metadataState: RuntimeValue = invocation.context.metadata;
  let integrationMetadataState: RuntimeValue =
    invocation.context.integration.metadata;
  let integrationReady = false;
  let integrationErrorMessage = "";
  let browserAction: BrowserAction | null = null;
  let nodeWebhookSecret: Uint8Array = new Uint8Array(0);
  const requestedWebhooks: RuntimeValue[] = [];
  const scheduledRequests: Array<{
    actionName: string;
    parameters: Record<string, RuntimeValue>;
    intervalMs: number;
  }> = [];
  const emittedEvents: Array<{ payloadType: string; payload: RuntimeValue }> =
    [];
  const integrationSecrets = new Map<string, Uint8Array>();
  const integrationSubscriptions = new Map<
    string,
    { configuration: RuntimeValue; messages: RuntimeValue[] }
  >();
  const integrationScheduledActions: Array<{
    actionName: string;
    parameters: RuntimeValue;
    intervalMs: number;
  }> = [];
  let integrationResyncIntervalMs: number | null = null;
  const executionKV = new Map<string, string>();
  const emissions: Array<{
    channel: string;
    payloadType: string;
    payloads: RuntimeValue[];
  }> = [];
  let executionPassed = false;
  let executionFailed: { reason: string; message: string } | null = null;
  let executionFinished = false;

  const logger: RuntimeLogger = {
    debug(message, fields) {
      writeLog("debug", message, fields);
    },
    info(message, fields) {
      writeLog("info", message, fields);
    },
    warn(message, fields) {
      writeLog("warn", message, fields);
    },
    error(message, fields) {
      writeLog("error", message, fields);
    },
  };

  const http: HTTPContext = {
    async do(request: HTTPRequest): Promise<HTTPResponse> {
      const response = await fetch(request.url, {
        method: request.method,
        headers: request.headers,
        body: normalizeRequestBody(request.body),
      });

      return {
        status: response.status,
        headers: readResponseHeaders(response.headers),
        body: new Uint8Array(await response.arrayBuffer()),
      };
    },
  };

  const metadata: MetadataContext = {
    get() {
      return metadataState;
    },
    set(value) {
      metadataState = value;
    },
  };

  const requests: RequestContext = {
    scheduleActionCall(actionName, parameters, intervalMs) {
      scheduledRequests.push({ actionName, parameters, intervalMs });
    },
  };

  const events: EventContext = {
    emit(payloadType, payload) {
      emittedEvents.push({ payloadType, payload });
    },
  };

  const executionState: ExecutionStateContext = {
    isFinished() {
      return executionFinished;
    },
    setKV(key, value) {
      executionKV.set(key, value);
    },
    emit(channel, payloadType, payloads) {
      emissions.push({ channel, payloadType, payloads });
      executionFinished = true;
      executionPassed = true;
    },
    pass() {
      executionPassed = true;
      executionFinished = true;
    },
    fail(reason, message) {
      executionFailed = { reason, message };
      executionFinished = true;
    },
  };

  const integration: IntegrationContext = {
    id() {
      return invocation.context.integration.id;
    },
    getMetadata() {
      return integrationMetadataState;
    },
    setMetadata(value) {
      integrationMetadataState = value;
    },
    getConfig(name) {
      const value = invocation.context.integration.configuration[name];
      if (typeof value === "string") {
        return new TextEncoder().encode(value);
      }

      return new TextEncoder().encode(JSON.stringify(value ?? null));
    },
    ready() {
      integrationReady = true;
      integrationErrorMessage = "";
    },
    error(message) {
      integrationReady = false;
      integrationErrorMessage = message;
    },
    newBrowserAction(action) {
      browserAction = action;
    },
    removeBrowserAction() {
      browserAction = null;
    },
    setSecret(name, value) {
      integrationSecrets.set(name, value);
    },
    getSecrets() {
      return Array.from(integrationSecrets.entries()).map(
        ([name, value]): IntegrationSecret => ({
          name,
          value,
        }),
      );
    },
    requestWebhook(configuration) {
      requestedWebhooks.push(configuration);
    },
    subscribe(configuration) {
      const id = crypto.randomUUID();
      integrationSubscriptions.set(id, { configuration, messages: [] });
      return id;
    },
    scheduleResync(intervalMs) {
      integrationResyncIntervalMs = intervalMs;
    },
    scheduleActionCall(actionName, parameters, intervalMs) {
      integrationScheduledActions.push({ actionName, parameters, intervalMs });
    },
    listSubscriptions() {
      return Array.from(integrationSubscriptions.entries()).map(
        ([, entry]): IntegrationSubscription => ({
          configuration() {
            return entry.configuration;
          },
          sendMessage(message) {
            entry.messages.push(message);
          },
        }),
      );
    },
  };

  const webhook: NodeWebhookContext = {
    setup() {
      return typeof asRecord(invocation.context.metadata)?.webhookURL ===
        "string"
        ? String(asRecord(invocation.context.metadata)?.webhookURL)
        : "";
    },
    getSecret() {
      return nodeWebhookSecret;
    },
    setSecret(secret) {
      nodeWebhookSecret = secret;
    },
    resetSecret() {
      const previous = nodeWebhookSecret;
      const nextSecret = new Uint8Array(24);
      crypto.getRandomValues(nextSecret);
      nodeWebhookSecret = nextSecret;
      return { previous, current: nodeWebhookSecret };
    },
    getBaseURL() {
      return typeof asRecord(invocation.context.metadata)?.webhookBaseURL ===
        "string"
        ? String(asRecord(invocation.context.metadata)?.webhookBaseURL)
        : "";
    },
  };

  const context: RuntimeContext = {
    logger,
    http,
    metadata,
    requests,
    events,
    executionState,
    integration,
    webhook,
  };

  return {
    context,
    snapshot() {
      return {
        metadata: metadataState,
        requests: {
          scheduledActions: scheduledRequests,
        },
        events: emittedEvents,
        executionState: {
          finished: executionFinished,
          passed: executionPassed,
          failed: executionFailed,
          kv: Object.fromEntries(executionKV.entries()),
          emissions,
        },
        integration: {
          id: invocation.context.integration.id,
          ready: integrationReady,
          error: integrationErrorMessage || undefined,
          metadata: integrationMetadataState,
          browserAction: normalizeForJSON(browserAction),
          requestedWebhooks,
          scheduledResyncIntervalMs: integrationResyncIntervalMs,
          scheduledActions: integrationScheduledActions,
          secrets: Array.from(integrationSecrets.entries()).map(
            ([name, value]) => ({ name, value: normalizeForJSON(value) }),
          ),
          subscriptions: Array.from(integrationSubscriptions.entries()).map(
            ([id, entry]) => ({
              id,
              configuration: entry.configuration,
              messages: entry.messages,
            }),
          ),
        },
        webhook: {
          url:
            typeof asRecord(invocation.context.metadata)?.webhookURL ===
            "string"
              ? String(asRecord(invocation.context.metadata)?.webhookURL)
              : "",
          baseURL:
            typeof asRecord(invocation.context.metadata)?.webhookBaseURL ===
            "string"
              ? String(asRecord(invocation.context.metadata)?.webhookBaseURL)
              : "",
          secret: normalizeForJSON(nodeWebhookSecret),
        },
      };
    },
  };
}

export function normalizeForJSON(value: unknown): RuntimeValue {
  if (value === undefined) {
    return null;
  }

  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return value;
  }

  if (value instanceof Uint8Array) {
    return {
      type: "bytes",
      base64: encodeBase64(value),
    };
  }

  if (Array.isArray(value)) {
    return value.map((item) => normalizeForJSON(item));
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>).map(
      ([key, entryValue]) => [key, normalizeForJSON(entryValue)],
    );
    return Object.fromEntries(entries) as Record<string, RuntimeValue>;
  }

  return String(value);
}

function normalizeInvocationContext(
  context?: InvocationContext,
): NormalizedInvocationContext {
  const integration = normalizeInvocationIntegrationContext(
    context?.integration,
  );

  return {
    configuration: context?.configuration ?? {},
    integration,
    metadata: context?.metadata ?? null,
  };
}

function normalizeInvocationIntegrationContext(
  integration?: InvocationIntegrationContext,
): NormalizedInvocationContext["integration"] {
  return {
    id: typeof integration?.id === "string" ? integration.id : "",
    configuration: asRecord(integration?.configuration) ?? {},
    metadata: integration?.metadata ?? null,
  };
}

function asRecord(value: unknown): Record<string, RuntimeValue> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  return value as Record<string, RuntimeValue>;
}

function normalizeBody(value: RuntimeValue | undefined): Uint8Array {
  if (typeof value === "string") {
    return new TextEncoder().encode(value);
  }

  if (Array.isArray(value)) {
    return Uint8Array.from(value.map((item) => Number(item)));
  }

  return new Uint8Array(0);
}

function normalizeRequestBody(
  body: string | Uint8Array | undefined,
): BodyInit | undefined {
  if (body === undefined) {
    return undefined;
  }

  if (typeof body === "string") {
    return body;
  }

  const copy = new Uint8Array(body.length);
  copy.set(body);
  return new Blob([copy]);
}

function writeLog(
  level: string,
  message: string,
  fields?: Record<string, RuntimeValue>,
): void {
  const payload = fields ? ` ${JSON.stringify(fields)}` : "";
  console.error(`[${level}] ${message}${payload}`);
}

function readResponseHeaders(headers: Headers): Record<string, string> {
  const values: Record<string, string> = {};
  headers.forEach((value, key) => {
    values[key] = value;
  });
  return values;
}

function encodeBase64(value: Uint8Array): string {
  let binary = "";
  for (const byte of value) {
    binary += String.fromCharCode(byte);
  }

  return btoa(binary);
}
