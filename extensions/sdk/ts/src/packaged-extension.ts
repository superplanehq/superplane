import type { ExtensionDefinition } from "./block-definitions.js";
import type { RuntimeValue } from "./runtime-context.js";
import { normalizeInvocationEnvelope } from "./contexts/runtime-harness.js";
import { deriveManifest, validateExtensionDefinition } from "./runtime/manifest.js";
import { collectOperations, deriveOperations } from "./runtime/operations.js";
import type { ExtensionDiscovery, InvocationPayload, OperationDescriptor } from "./runtime/types.js";

interface ExecuteRequest {
  type: "engine.execute";
  requestId: string;
  envelope: InvocationPayload;
}

type WorkerMessage =
  | {
      type: "worker.hello";
      workerId: string;
      manifest: ExtensionDiscovery["manifest"];
      operations: ExtensionDiscovery["operations"];
    }
  | {
      type: "worker.result";
      requestId: string;
      output: RuntimeValue;
    }
  | {
      type: "worker.error";
      requestId: string;
      code: string;
      message: string;
    };

type RuntimeMode = "manifest" | "worker";

export { deriveManifest, deriveOperations, validateExtensionDefinition };

export function discoverExtension(definition: ExtensionDefinition): ExtensionDiscovery {
  return {
    manifest: deriveManifest(definition),
    operations: deriveOperations(definition),
  };
}

export async function startPackagedExtension(definition: ExtensionDefinition): Promise<void> {
  const discovery = discoverExtension(definition);
  const operations = collectOperations(definition);
  const args = parseArgs(process.argv.slice(2));
  const mode = readMode(args);

  if (mode === "manifest") {
    process.stdout.write(`${JSON.stringify(discovery, null, 2)}\n`);
    return;
  }

  const engineURL = readRequired(args, "engine-url", "EXTENSION_RUNNER_ENGINE_URL");
  const workerId = readRequired(args, "worker-id", "EXTENSION_RUNNER_WORKER_ID");
  const socket = new WebSocket(engineURL);
  const operationIndex = new Map(operations.map((operation) => [operation.name, operation]));

  await waitForOpen(socket);
  send(socket, {
    type: "worker.hello",
    workerId,
    manifest: discovery.manifest,
    operations: discovery.operations,
  });

  socket.addEventListener("message", (event) => {
    void handleMessage(socket, operationIndex, event.data);
  });

  await waitForClose(socket);
}

async function handleMessage(socket: WebSocket, operationIndex: Map<string, OperationDescriptor>, rawData: unknown): Promise<void> {
  const message = JSON.parse(await readMessageData(rawData)) as ExecuteRequest;
  if (message.type !== "engine.execute") {
    return;
  }

  const operationName = formatOperationName(message.envelope.target);
  const operation = operationIndex.get(operationName);
  if (!operation) {
    send(socket, {
      type: "worker.error",
      requestId: message.requestId,
      code: "operation_not_found",
      message: `Operation ${operationName} is not registered`,
    });
    return;
  }

  try {
    const output = await operation.invoke(normalizeInvocationEnvelope(message.envelope));
    send(socket, {
      type: "worker.result",
      requestId: message.requestId,
      output,
    });
  } catch (error) {
    send(socket, {
      type: "worker.error",
      requestId: message.requestId,
      code: "execution_failed",
      message: error instanceof Error ? error.message : String(error),
    });
  }
}

function parseArgs(argv: string[]): Record<string, string> {
  const args: Record<string, string> = {};

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith("--")) {
      continue;
    }

    const body = token.slice(2);
    const [key, inlineValue] = body.split("=", 2);
    if (inlineValue !== undefined) {
      args[key] = inlineValue;
      continue;
    }

    args[key] = argv[index + 1] ?? "";
    index += 1;
  }

  return args;
}

function readMode(args: Record<string, string>): RuntimeMode {
  const mode = args.mode ?? process.env.EXTENSION_RUNNER_MODE ?? "worker";
  if (mode !== "manifest" && mode !== "worker") {
    throw new Error(`Unsupported mode ${mode}`);
  }

  return mode;
}

function readRequired(args: Record<string, string>, key: string, envKey: string): string {
  const value = args[key] ?? process.env[envKey];
  if (!value) {
    throw new Error(`Missing ${key}. Provide --${key} or ${envKey}`);
  }

  return value;
}

function send(socket: WebSocket, message: WorkerMessage): void {
  socket.send(JSON.stringify(message));
}

async function readMessageData(rawData: unknown): Promise<string> {
  if (typeof rawData === "string") {
    return rawData;
  }

  if (rawData instanceof ArrayBuffer) {
    return Buffer.from(rawData).toString("utf8");
  }

  if (ArrayBuffer.isView(rawData)) {
    return Buffer.from(rawData.buffer, rawData.byteOffset, rawData.byteLength).toString("utf8");
  }

  if (rawData instanceof Blob) {
    return await rawData.text();
  }

  return String(rawData);
}

function waitForOpen(socket: WebSocket): Promise<void> {
  return new Promise((resolve, reject) => {
    socket.addEventListener("open", () => resolve(), { once: true });
    socket.addEventListener("error", () => reject(new Error("websocket error")), { once: true });
  });
}

function waitForClose(socket: WebSocket): Promise<void> {
  return new Promise((resolve, reject) => {
    socket.addEventListener("close", () => resolve(), { once: true });
    socket.addEventListener("error", () => reject(new Error("websocket error")), { once: true });
  });
}

function formatOperationName(target: InvocationPayload["target"]): string {
  return `${target.blockType}.${target.blockName}.${target.operation}`;
}
