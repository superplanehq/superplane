import type {
  BrowserAction,
  EventContext,
  ExecutionStateContext,
  HTTPContext,
  HTTPRequest,
  HTTPResponse,
  IntegrationContext,
  IntegrationSecret,
  IntegrationSubscription,
  MetadataContext,
  NodeWebhookContext,
  RequestContext,
  RuntimeContext,
  RuntimeLogger,
  RuntimeValue,
} from "../runtime-context.js";
import type { InvocationEnvelope, InvocationPayload, RuntimeHarness } from "../runtime/types.js";

export function normalizeInvocationEnvelope(payload: InvocationPayload): InvocationEnvelope {
  const configuration = payload.configuration ?? {};
  const input = payload.input ?? {};
  const current = payload.current ?? {};
  const requested = payload.requested ?? {};
  const parameters = payload.parameters ?? {};
  const actionName = payload.actionName ?? "";
  const headers = payload.headers ?? {};
  const body = normalizeBody(payload.body);
  const message = payload.message ?? input;
  const integrationRecord = payload.integration ?? null;
  const webhookRecord = payload.webhook ?? null;
  const integrationConfig = asRecord(integrationRecord?.configuration) ?? {};
  const integrationMetadata = integrationRecord?.metadata ?? null;
  const integrationID = typeof integrationRecord?.id === "string" ? integrationRecord.id : "";
  const webhookID = typeof webhookRecord?.id === "string" ? webhookRecord.id : "";
  const webhookURL = typeof webhookRecord?.url === "string" ? webhookRecord.url : "";
  const webhookSecret = normalizeBody(webhookRecord?.secret);
  const webhookMetadata = webhookRecord?.metadata ?? null;
  const webhookConfiguration = webhookRecord?.configuration ?? null;

  return {
    target: payload.target,
    configuration,
    input,
    current,
    requested,
    parameters,
    actionName,
    headers,
    body,
    message,
    integration: {
      id: integrationID,
      configuration: integrationConfig,
      metadata: integrationMetadata,
    },
    webhook: {
      id: webhookID,
      url: webhookURL,
      secret: webhookSecret,
      metadata: webhookMetadata,
      configuration: webhookConfiguration,
    },
    metadata: payload.metadata ?? null,
  };
}

export function createRuntimeHarness(invocation: InvocationEnvelope): RuntimeHarness {
  let metadataState: RuntimeValue = invocation.metadata;
  let integrationMetadataState: RuntimeValue = invocation.integration.metadata;
  let integrationReady = false;
  let integrationErrorMessage = "";
  let browserAction: BrowserAction | null = null;
  let nodeWebhookSecret: Uint8Array = new Uint8Array(0);
  let provisionedWebhookSecret: Uint8Array = invocation.webhook.secret;
  let provisionedWebhookMetadata: RuntimeValue = invocation.webhook.metadata;
  let provisionedWebhookConfiguration: RuntimeValue = invocation.webhook.configuration;
  const requestedWebhooks: RuntimeValue[] = [];
  const scheduledRequests: Array<{ actionName: string; parameters: Record<string, RuntimeValue>; intervalMs: number }> = [];
  const emittedEvents: Array<{ payloadType: string; payload: RuntimeValue }> = [];
  const integrationSecrets = new Map<string, Uint8Array>();
  const integrationSubscriptions = new Map<string, { configuration: RuntimeValue; messages: RuntimeValue[] }>();
  const integrationScheduledActions: Array<{ actionName: string; parameters: RuntimeValue; intervalMs: number }> = [];
  let integrationResyncIntervalMs: number | null = null;
  const executionKV = new Map<string, string>();
  const emissions: Array<{ channel: string; payloadType: string; payloads: RuntimeValue[] }> = [];
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
      return invocation.integration.id;
    },
    getMetadata() {
      return integrationMetadataState;
    },
    setMetadata(value) {
      integrationMetadataState = value;
    },
    getConfig(name) {
      const value = invocation.integration.configuration[name];
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
      return typeof asRecord(invocation.metadata)?.webhookURL === "string" ? String(asRecord(invocation.metadata)?.webhookURL) : "";
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
      return typeof asRecord(invocation.metadata)?.webhookBaseURL === "string"
        ? String(asRecord(invocation.metadata)?.webhookBaseURL)
        : "";
    },
  };

  const provisionedWebhook = {
    getID() {
      return invocation.webhook.id;
    },
    getURL() {
      return invocation.webhook.url;
    },
    getSecret() {
      return provisionedWebhookSecret;
    },
    getMetadata() {
      return provisionedWebhookMetadata;
    },
    getConfiguration() {
      return provisionedWebhookConfiguration;
    },
    setSecret(secret: Uint8Array) {
      provisionedWebhookSecret = secret;
    },
  };

  const runtime: RuntimeContext = {
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
    runtime,
    webhook: provisionedWebhook,
    snapshot() {
      return normalizeForJSON({
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
          id: invocation.integration.id,
          ready: integrationReady,
          error: integrationErrorMessage || undefined,
          metadata: integrationMetadataState,
          browserAction,
          requestedWebhooks,
          scheduledResyncIntervalMs: integrationResyncIntervalMs,
          scheduledActions: integrationScheduledActions,
          secrets: Array.from(integrationSecrets.entries()).map(([name, value]) => ({ name, value })),
          subscriptions: Array.from(integrationSubscriptions.entries()).map(([id, entry]) => ({
            id,
            configuration: entry.configuration,
            messages: entry.messages,
          })),
        },
        webhook: {
          id: invocation.webhook.id,
          url: invocation.webhook.url,
          metadata: provisionedWebhookMetadata,
          configuration: provisionedWebhookConfiguration,
          provisionedSecret: provisionedWebhookSecret,
          baseURL: typeof asRecord(invocation.metadata)?.webhookBaseURL === "string"
            ? String(asRecord(invocation.metadata)?.webhookBaseURL)
            : "",
          secret: nodeWebhookSecret,
        },
      });
    },
  };
}

export function normalizeForJSON(value: unknown): RuntimeValue {
  if (value === undefined) {
    return null;
  }

  if (value === null || typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
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
    const entries = Object.entries(value as Record<string, unknown>).map(([key, entryValue]) => [key, normalizeForJSON(entryValue)]);
    return Object.fromEntries(entries) as Record<string, RuntimeValue>;
  }

  return String(value);
}

function asRecord(value: unknown): Record<string, RuntimeValue> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  return value as Record<string, RuntimeValue>;
}

function normalizeHeaders(value: Record<string, RuntimeValue> | null): Record<string, string[]> {
  if (!value) {
    return {};
  }

  const headers: Record<string, string[]> = {};
  for (const [key, headerValue] of Object.entries(value)) {
    if (typeof headerValue === "string") {
      headers[key] = [headerValue];
      continue;
    }

    if (Array.isArray(headerValue)) {
      headers[key] = headerValue.map((item) => String(item));
    }
  }

  return headers;
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

function normalizeRequestBody(body: string | Uint8Array | undefined): BodyInit | undefined {
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

function writeLog(level: string, message: string, fields?: Record<string, RuntimeValue>): void {
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
