import grpc from "npm:@grpc/grpc-js@1.14.0";
import protoLoader from "npm:@grpc/proto-loader@0.8.0";
import type { ComponentExecutionResponse, IntegrationExecutionResponse } from "../types.ts";
import { ModuleRegistry } from "./registry.ts";

type OperationRequest = {
  request?: {
    requestId?: string;
    version?: string;
    timeoutMs?: number;
  };
  context?: {
    organizationId?: string;
    workspaceId?: string;
    userId?: string;
    canvasId?: string;
    nodeId?: string;
    labels?: Record<string, unknown>;
    metadata?: Record<string, unknown>;
  };
  input?: Record<string, unknown>;
};

type RunnerErrorCode = "INVALID_INPUT" | "NOT_FOUND" | "TIMEOUT" | "EXECUTION_ERROR" | "UNAVAILABLE";

type RunnerResponse = {
  ok: boolean;
  output?: Record<string, unknown>;
  logs?: Array<{ level: string; message: string; fields?: Record<string, unknown> }>;
  error?: {
    code: RunnerErrorCode;
    message: string;
    details?: Record<string, unknown>;
  };
  metrics?: Record<string, number>;
};

type ServerOptions = {
  authToken: string;
  version: string;
  httpHost: string;
  httpPort: number;
  grpcAddress: string;
  grpcEnabled: boolean;
  httpEnabled: boolean;
  protoPath: string;
};

type GrpcHandlerCall = {
  request: Record<string, unknown>;
  metadata?: {
    get(key: string): unknown[];
  };
};

function readOptionsFromEnv(): ServerOptions {
  return {
    authToken: (Deno.env.get("TYPESCRIPT_RUNNER_AUTH_TOKEN") ?? "").trim(),
    version: (Deno.env.get("TYPESCRIPT_RUNNER_VERSION") ?? "v1").trim(),
    httpHost: (Deno.env.get("TYPESCRIPT_RUNNER_HTTP_HOST") ?? "0.0.0.0").trim(),
    httpPort: Number.parseInt(Deno.env.get("TYPESCRIPT_RUNNER_HTTP_PORT") ?? "7761", 10),
    grpcAddress: (Deno.env.get("TYPESCRIPT_RUNNER_GRPC_ADDRESS") ?? "0.0.0.0:7762").trim(),
    grpcEnabled: parseBoolean(Deno.env.get("TYPESCRIPT_RUNNER_ENABLE_GRPC"), false),
    httpEnabled: parseBoolean(Deno.env.get("TYPESCRIPT_RUNNER_ENABLE_HTTP"), true),
    protoPath: (Deno.env.get("TYPESCRIPT_RUNNER_PROTO_PATH") ??
      "pkg/runtime/runner/proto/runtime_runner.proto")
      .trim(),
  };
}

function parseBoolean(value: string | undefined, fallback: boolean): boolean {
  if (!value) {
    return fallback;
  }

  const normalized = value.trim().toLowerCase();
  if (normalized === "1" || normalized === "true" || normalized === "yes") {
    return true;
  }
  if (normalized === "0" || normalized === "false" || normalized === "no") {
    return false;
  }

  return fallback;
}

function getAuthorizationHeader(request: Request): string {
  return request.headers.get("authorization") ?? "";
}

function authAllowed(options: ServerOptions, headerValue: string): boolean {
  if (!options.authToken) {
    return true;
  }

  return headerValue === `Bearer ${options.authToken}`;
}

function createLogger(logs: Array<{ level: string; message: string; fields?: Record<string, unknown> }>) {
  const write = (level: string, message: string, fields?: Record<string, unknown>) => {
    logs.push({
      level,
      message,
      fields,
    });
  };

  return {
    debug(message: string, fields?: Record<string, unknown>) {
      write("debug", message, fields);
    },
    info(message: string, fields?: Record<string, unknown>) {
      write("info", message, fields);
    },
    warn(message: string, fields?: Record<string, unknown>) {
      write("warn", message, fields);
    },
    error(message: string, fields?: Record<string, unknown>) {
      write("error", message, fields);
    },
  };
}

function ok(output: Record<string, unknown>, logs: RunnerResponse["logs"]): RunnerResponse {
  return {
    ok: true,
    output,
    logs: logs ?? [],
  };
}

function err(code: RunnerErrorCode, message: string, details?: Record<string, unknown>): RunnerResponse {
  return {
    ok: false,
    error: {
      code,
      message,
      details,
    },
  };
}

function jsonResponse(status: number, payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "content-type": "application/json",
    },
  });
}

function mapErrorCodeToHTTPStatus(code: RunnerErrorCode): number {
  switch (code) {
    case "INVALID_INPUT":
      return 400;
    case "NOT_FOUND":
      return 404;
    case "TIMEOUT":
      return 504;
    case "UNAVAILABLE":
      return 503;
    default:
      return 500;
  }
}

function mapErrorCodeToGRPC(code: RunnerErrorCode): string {
  switch (code) {
    case "INVALID_INPUT":
      return "RUNTIME_ERROR_CODE_INVALID_INPUT";
    case "NOT_FOUND":
      return "RUNTIME_ERROR_CODE_NOT_FOUND";
    case "TIMEOUT":
      return "RUNTIME_ERROR_CODE_TIMEOUT";
    case "UNAVAILABLE":
      return "RUNTIME_ERROR_CODE_UNAVAILABLE";
    default:
      return "RUNTIME_ERROR_CODE_EXECUTION_ERROR";
  }
}

function mapCapabilityKindToGRPC(kind: string): string {
  switch (kind) {
    case "component":
      return "CAPABILITY_KIND_COMPONENT";
    case "integration":
      return "CAPABILITY_KIND_INTEGRATION";
    case "trigger":
      return "CAPABILITY_KIND_TRIGGER";
    default:
      return "CAPABILITY_KIND_UNSPECIFIED";
  }
}

class RunnerService {
  constructor(
    private readonly registry: ModuleRegistry,
    private readonly options: ServerOptions,
  ) {}

  async setupTrigger(name: string, request: OperationRequest): Promise<RunnerResponse> {
    const implementation = this.registry.triggers.get(name);
    if (!implementation) {
      return err("NOT_FOUND", `Trigger ${name} not found`);
    }

    try {
      await implementation.setup?.({
        configuration: request.input?.configuration,
        metadata: asRecord(request.input?.metadata),
      });

      return ok({ outcome: "noop" }, []);
    } catch (error) {
      return err("EXECUTION_ERROR", errorMessage(error));
    }
  }

  async setupComponent(name: string, request: OperationRequest): Promise<RunnerResponse> {
    const implementation = this.registry.components.get(name);
    if (!implementation) {
      return err("NOT_FOUND", `Component ${name} not found`);
    }

    try {
      await implementation.setup?.({
        configuration: request.input?.configuration,
        integrationConfiguration: asRecord(request.input?.integrationConfiguration),
        metadata: asRecord(request.input?.metadata),
        nodeMetadata: asRecord(request.input?.nodeMetadata),
      });

      return ok({ outcome: "noop" }, []);
    } catch (error) {
      return err("EXECUTION_ERROR", errorMessage(error));
    }
  }

  async executeComponent(name: string, request: OperationRequest): Promise<RunnerResponse> {
    const implementation = this.registry.components.get(name);
    if (!implementation) {
      return err("NOT_FOUND", `Component ${name} not found`);
    }
    if (!implementation.execute) {
      return err("UNAVAILABLE", `Component ${name} does not implement execute()`);
    }

    const logs: Array<{ level: string; message: string; fields?: Record<string, unknown> }> = [];
    const logger = createLogger(logs);

    try {
      const result = (await implementation.execute({
        executionId: String(request.input?.executionId ?? ""),
        workflowId: String(request.input?.workflowId ?? ""),
        organizationId: String(request.input?.organizationId ?? ""),
        nodeId: String(request.input?.nodeId ?? ""),
        sourceNodeId: String(request.input?.sourceNodeId ?? ""),
        configuration: request.input?.configuration,
        integrationConfiguration: asRecord(request.input?.integrationConfiguration),
        data: request.input?.data,
        metadata: asRecord(request.input?.metadata),
        nodeMetadata: asRecord(request.input?.nodeMetadata),
        logger,
      })) as ComponentExecutionResponse;

      return ok((result ?? { outcome: "noop" }) as Record<string, unknown>, logs);
    } catch (error) {
      return err("EXECUTION_ERROR", errorMessage(error));
    }
  }

  async syncIntegration(name: string, request: OperationRequest): Promise<RunnerResponse> {
    const implementation = this.registry.integrations.get(name);
    if (!implementation) {
      return err("NOT_FOUND", `Integration ${name} not found`);
    }

    try {
      const result = (await implementation.sync?.({
        configuration: asRecord(request.input?.configuration),
        metadata: asRecord(request.input?.metadata),
        organizationId: stringOrUndefined(request.input?.organizationId),
        baseUrl: stringOrUndefined(request.input?.baseUrl),
        webhooksBaseUrl: stringOrUndefined(request.input?.webhooksBaseUrl),
      })) as IntegrationExecutionResponse;

      return ok((result ?? { outcome: "noop" }) as Record<string, unknown>, []);
    } catch (error) {
      return err("EXECUTION_ERROR", errorMessage(error));
    }
  }

  async cleanupIntegration(name: string, request: OperationRequest): Promise<RunnerResponse> {
    const implementation = this.registry.integrations.get(name);
    if (!implementation) {
      return err("NOT_FOUND", `Integration ${name} not found`);
    }

    try {
      const result = (await implementation.cleanup?.({
        configuration: asRecord(request.input?.configuration),
        metadata: asRecord(request.input?.metadata),
        organizationId: stringOrUndefined(request.input?.organizationId),
        baseUrl: stringOrUndefined(request.input?.baseUrl),
        webhooksBaseUrl: stringOrUndefined(request.input?.webhooksBaseUrl),
      })) as IntegrationExecutionResponse;

      return ok((result ?? { outcome: "noop" }) as Record<string, unknown>, []);
    } catch (error) {
      return err("EXECUTION_ERROR", errorMessage(error));
    }
  }

  listCapabilities() {
    return this.registry.listCapabilities();
  }
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }

  return undefined;
}

function stringOrUndefined(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }

  return "Unknown runtime runner error";
}

async function readRequestBody(request: Request): Promise<OperationRequest> {
  const content = await request.text();
  if (!content.trim()) {
    return {};
  }

  return JSON.parse(content) as OperationRequest;
}

async function handleHTTP(service: RunnerService, options: ServerOptions, request: Request): Promise<Response> {
  const url = new URL(request.url);
  if (request.method === "GET" && url.pathname === "/healthz") {
    return new Response("ok", { status: 200 });
  }
  if (request.method === "GET" && url.pathname === "/readyz") {
    return new Response("ready", { status: 200 });
  }

  if (!authAllowed(options, getAuthorizationHeader(request))) {
    return jsonResponse(401, err("UNAVAILABLE", "Unauthorized"));
  }

  if (request.method === "GET" && url.pathname === "/v1/capabilities") {
    return jsonResponse(200, { capabilities: service.listCapabilities() });
  }

  try {
    if (request.method !== "POST") {
      return jsonResponse(405, err("INVALID_INPUT", "Unsupported HTTP method"));
    }

    const payload = await readRequestBody(request);
    const path = url.pathname;

    let response: RunnerResponse;
    let match: RegExpMatchArray | null;

    match = path.match(/^\/v1\/triggers\/([^/]+)\/setup$/);
    if (match) {
      response = await service.setupTrigger(decodeURIComponent(match[1]), payload);
      return jsonResponse(response.ok ? 200 : mapErrorCodeToHTTPStatus(response.error?.code ?? "EXECUTION_ERROR"), response);
    }

    match = path.match(/^\/v1\/components\/([^/]+)\/setup$/);
    if (match) {
      response = await service.setupComponent(decodeURIComponent(match[1]), payload);
      return jsonResponse(response.ok ? 200 : mapErrorCodeToHTTPStatus(response.error?.code ?? "EXECUTION_ERROR"), response);
    }

    match = path.match(/^\/v1\/components\/([^/]+)\/execute$/);
    if (match) {
      response = await service.executeComponent(decodeURIComponent(match[1]), payload);
      return jsonResponse(response.ok ? 200 : mapErrorCodeToHTTPStatus(response.error?.code ?? "EXECUTION_ERROR"), response);
    }

    match = path.match(/^\/v1\/integrations\/([^/]+)\/sync$/);
    if (match) {
      response = await service.syncIntegration(decodeURIComponent(match[1]), payload);
      return jsonResponse(response.ok ? 200 : mapErrorCodeToHTTPStatus(response.error?.code ?? "EXECUTION_ERROR"), response);
    }

    match = path.match(/^\/v1\/integrations\/([^/]+)\/cleanup$/);
    if (match) {
      response = await service.cleanupIntegration(decodeURIComponent(match[1]), payload);
      return jsonResponse(response.ok ? 200 : mapErrorCodeToHTTPStatus(response.error?.code ?? "EXECUTION_ERROR"), response);
    }

    return jsonResponse(404, err("NOT_FOUND", "Route not found"));
  } catch (error) {
    return jsonResponse(500, err("EXECUTION_ERROR", errorMessage(error)));
  }
}

function grpcResponseFromRunner(response: RunnerResponse): Record<string, unknown> {
  const grpcResponse: Record<string, unknown> = {
    ok: response.ok,
    output: response.output ?? {},
    logs: (response.logs ?? []).map((log) => ({
      level: log.level,
      message: log.message,
      fields: log.fields ?? {},
    })),
    metrics: response.metrics ?? {},
  };

  if (response.error) {
    grpcResponse.error = {
      code: mapErrorCodeToGRPC(response.error.code),
      message: response.error.message,
      details: response.error.details ?? {},
    };
  }

  return grpcResponse;
}

async function startGRPC(service: RunnerService, options: ServerOptions): Promise<void> {
  const packageDefinition = protoLoader.loadSync(options.protoPath, {
    keepCase: false,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
  });
  const descriptor = grpc.loadPackageDefinition(packageDefinition) as Record<string, unknown>;
  const runtimeRunner = ((((descriptor.Superplane as Record<string, unknown>)?.RuntimeRunner as Record<string, unknown>)
    ?.RuntimeRunner as { service: grpc.ServiceDefinition<grpc.UntypedServiceImplementation> }) ??
    null);

  if (!runtimeRunner) {
    throw new Error(`Unable to load RuntimeRunner service from ${options.protoPath}`);
  }

  const server = new grpc.Server();
  server.addService(runtimeRunner.service, {
    SetupTrigger: (call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      if (!authAllowed(options, String(call.metadata?.get("authorization")?.[0] ?? ""))) {
        callback(null, grpcResponseFromRunner(err("UNAVAILABLE", "Unauthorized")));
        return;
      }
      service
        .setupTrigger(String(call.request.name ?? ""), call.request as OperationRequest)
        .then((response) => callback(null, grpcResponseFromRunner(response)))
        .catch((error) => callback(null, grpcResponseFromRunner(err("EXECUTION_ERROR", errorMessage(error)))));
    },
    SetupComponent: (call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      if (!authAllowed(options, String(call.metadata?.get("authorization")?.[0] ?? ""))) {
        callback(null, grpcResponseFromRunner(err("UNAVAILABLE", "Unauthorized")));
        return;
      }
      service
        .setupComponent(String(call.request.name ?? ""), call.request as OperationRequest)
        .then((response) => callback(null, grpcResponseFromRunner(response)))
        .catch((error) => callback(null, grpcResponseFromRunner(err("EXECUTION_ERROR", errorMessage(error)))));
    },
    ExecuteComponent: (call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      if (!authAllowed(options, String(call.metadata?.get("authorization")?.[0] ?? ""))) {
        callback(null, grpcResponseFromRunner(err("UNAVAILABLE", "Unauthorized")));
        return;
      }
      service
        .executeComponent(String(call.request.name ?? ""), call.request as OperationRequest)
        .then((response) => callback(null, grpcResponseFromRunner(response)))
        .catch((error) => callback(null, grpcResponseFromRunner(err("EXECUTION_ERROR", errorMessage(error)))));
    },
    SyncIntegration: (call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      if (!authAllowed(options, String(call.metadata?.get("authorization")?.[0] ?? ""))) {
        callback(null, grpcResponseFromRunner(err("UNAVAILABLE", "Unauthorized")));
        return;
      }
      service
        .syncIntegration(String(call.request.name ?? ""), call.request as OperationRequest)
        .then((response) => callback(null, grpcResponseFromRunner(response)))
        .catch((error) => callback(null, grpcResponseFromRunner(err("EXECUTION_ERROR", errorMessage(error)))));
    },
    CleanupIntegration: (call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      if (!authAllowed(options, String(call.metadata?.get("authorization")?.[0] ?? ""))) {
        callback(null, grpcResponseFromRunner(err("UNAVAILABLE", "Unauthorized")));
        return;
      }
      service
        .cleanupIntegration(String(call.request.name ?? ""), call.request as OperationRequest)
        .then((response) => callback(null, grpcResponseFromRunner(response)))
        .catch((error) => callback(null, grpcResponseFromRunner(err("EXECUTION_ERROR", errorMessage(error)))));
    },
    ListCapabilities: (_call: GrpcHandlerCall, callback: grpc.sendUnaryData<unknown>) => {
      callback(null, {
        capabilities: service.listCapabilities().map((capability) => ({
          kind: mapCapabilityKindToGRPC(capability.kind),
          name: capability.name,
          operations: capability.operations,
          schemaHash: capability.schemaHash,
        })),
      });
    },
  });

  await new Promise<void>((resolve, reject) => {
    server.bindAsync(options.grpcAddress, grpc.ServerCredentials.createInsecure(), (error) => {
      if (error) {
        reject(error);
        return;
      }

      server.start();
      resolve();
    });
  });
}

export async function startRunnerServer(): Promise<void> {
  const options = readOptionsFromEnv();
  const registry = await ModuleRegistry.fromEnv();
  const service = new RunnerService(registry, options);

  const tasks: Promise<unknown>[] = [];
  if (options.httpEnabled) {
    const server = Deno.serve(
      {
        hostname: options.httpHost,
        port: options.httpPort,
      },
      (request) => handleHTTP(service, options, request),
    );
    tasks.push(server.finished);
  }

  if (options.grpcEnabled) {
    tasks.push(startGRPC(service, options));
  }

  if (tasks.length === 0) {
    throw new Error("TypeScript runner has no enabled transports");
  }

  await Promise.all(tasks);
}

if (import.meta.main) {
  await startRunnerServer();
}
