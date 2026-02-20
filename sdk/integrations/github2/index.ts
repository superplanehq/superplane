const encoder = new TextEncoder();
const decoder = new TextDecoder();

type IntegrationRequest = {
  operation: string;
  integration: string;
  context: {
    configuration?: Record<string, unknown>;
    metadata?: Record<string, unknown>;
    actionName?: string;
    actionParameters?: unknown;
    resourceType?: string;
    resourceParameters?: Record<string, string>;
    request?: {
      method: string;
      path: string;
      query?: string;
      headers?: Record<string, string[]>;
      body?: number[];
    };
  };
};

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

  return decoder.decode(bytes);
}

function writeOutput(payload: unknown): void {
  Deno.stdout.writeSync(encoder.encode(JSON.stringify(payload)));
}

async function validateToken(apiToken: string): Promise<{ login: string }> {
  const response = await fetch("https://api.github.com/user", {
    method: "GET",
    headers: {
      Authorization: `Bearer ${apiToken}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "superplane-github2",
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(`GitHub token validation failed (${response.status}): ${message}`);
  }

  const data = (await response.json()) as { login?: string };
  return { login: data.login ?? "" };
}

const input = await readStdin();
const request = JSON.parse(input) as IntegrationRequest;

const config = request.context.configuration ?? {};
const apiTokenValue = config.apiToken;
const apiToken = typeof apiTokenValue === "string" ? apiTokenValue.trim() : "";

if (request.operation === "integration.sync") {
  if (!apiToken) {
    writeOutput({
      outcome: "fail",
      errorReason: "error",
      error: "apiToken is required",
      state: "error",
      stateDescription: "apiToken is required",
    });
  } else {
    try {
      const user = await validateToken(apiToken);
      writeOutput({
        outcome: "pass",
        state: "ready",
        metadata: {
          accountLogin: user.login,
        },
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Token validation failed";
      writeOutput({
        outcome: "fail",
        errorReason: "error",
        error: message,
        state: "error",
        stateDescription: message,
      });
    }
  }
} else if (request.operation === "integration.listResources") {
  writeOutput({
    outcome: "pass",
    resources: [],
  });
} else if (request.operation === "integration.handleAction") {
  writeOutput({
    outcome: "noop",
  });
} else if (request.operation === "integration.handleRequest") {
  writeOutput({
    outcome: "pass",
    http: {
      statusCode: 404
    },
  });
} else {
  writeOutput({
    outcome: "fail",
    errorReason: "error",
    error: `Unsupported operation: ${request.operation}`,
  });
}
