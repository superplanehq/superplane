import type {
  HTTPContext,
  HTTPRequest,
  HTTPResponse,
} from "../../context/http.js";
import type { ExecutionStateContext } from "../../context/execution.js";
import type { RuntimeValue } from "../../context/runtime-value.js";
import type { ExecuteCodeEffects } from "../../effects/execute-code.js";
import type {
  ExecuteCodeContext,
  ExecuteCodeJob,
  ExecuteCodeModule,
  ExecuteCodeResult,
} from "./types.js";

export interface ExecuteCodeRuntime {
  context: ExecuteCodeContext;
  snapshot(): ExecuteCodeEffects;
}

export function createExecuteCodeRuntime(
  _job: ExecuteCodeJob,
): ExecuteCodeRuntime {
  const executionKV = new Map<string, string>();
  const emissions: Array<{
    channel: string;
    payloadType: string;
    payloads: RuntimeValue[];
  }> = [];
  let executionPassed = false;
  let executionFailed: { reason: string; message: string } | null = null;
  let executionFinished = false;

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

  return {
    context: {
      http,
      executionState,
    },
    snapshot() {
      return {
        executionState: {
          finished: executionFinished,
          passed: executionPassed,
          failed: executionFailed,
          kv: Object.fromEntries(executionKV.entries()),
          emissions,
        },
      };
    },
  };
}

export async function runExecuteCodeModule(
  module: ExecuteCodeModule,
  job: ExecuteCodeJob,
): Promise<ExecuteCodeResult> {
  if (typeof module.default !== "function") {
    throw new Error("execute-code module does not export a default function");
  }

  const runtime = createExecuteCodeRuntime(job);
  await module.default(runtime.context);

  if (!runtime.context.executionState.isFinished()) {
    runtime.context.executionState.pass();
  }

  return {
    effects: runtime.snapshot(),
  };
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

function readResponseHeaders(headers: Headers): Record<string, string> {
  const values: Record<string, string> = {};
  headers.forEach((value, key) => {
    values[key] = value;
  });
  return values;
}
