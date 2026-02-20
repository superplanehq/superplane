import type {
  ActionImplementation,
  ComponentExecutionRequest,
  ComponentExecutionResponse,
  RuntimeLogger,
} from "./types.ts";

const textDecoder = new TextDecoder();
const textEncoder = new TextEncoder();

function createLogger(executionId: string): RuntimeLogger {
  const write = (level: string, message: string, fields?: Record<string, unknown>) => {
    const payload: Record<string, unknown> = {
      level,
      message,
      executionId,
      timestamp: new Date().toISOString(),
    };

    if (fields && Object.keys(fields).length > 0) {
      payload.fields = fields;
    }

    console.error(JSON.stringify(payload));
  };

  return {
    debug(message, fields) {
      write("debug", message, fields);
    },
    info(message, fields) {
      write("info", message, fields);
    },
    warn(message, fields) {
      write("warn", message, fields);
    },
    error(message, fields) {
      write("error", message, fields);
    },
  };
}

async function readStdin(): Promise<string> {
  const chunks: Uint8Array[] = [];
  const reader = Deno.stdin.readable.getReader();

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }

  const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
  const bytes = new Uint8Array(totalLength);
  let offset = 0;

  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.length;
  }

  return textDecoder.decode(bytes);
}

function writeStdout(payload: ComponentExecutionResponse): void {
  Deno.stdout.writeSync(textEncoder.encode(JSON.stringify(payload)));
}

export async function runComponentCLI(implementation: ActionImplementation): Promise<void> {
  try {
    const input = await readStdin();
    const request = JSON.parse(input) as ComponentExecutionRequest;

    if (request.operation === "component.setup") {
      if (implementation.setup) {
        await implementation.setup({
          configuration: request.context.configuration,
          integrationConfiguration: request.context.integrationConfiguration,
          metadata: request.context.metadata,
          nodeMetadata: request.context.nodeMetadata,
        });
      }

      writeStdout({ outcome: "noop" });
      return;
    }

    if (request.operation === "component.execute") {
      const logger = createLogger(request.context.executionId);
      const result = await implementation.execute({
        ...request.context,
        logger,
      });
      writeStdout(result);
      return;
    }

    writeStdout({
      outcome: "fail",
      errorReason: "error",
      error: `Unsupported operation: ${request.operation}`,
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown runtime error";
    writeStdout({
      outcome: "fail",
      errorReason: "error",
      error: message,
    });
  }
}
