import { runIntegrationCLI } from "../../typescript/mod.ts";
import type { IntegrationImplementation } from "../../typescript/mod.ts";

type IntegrationConfiguration = {
  apiToken?: string;
};

function asObject(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object") {
    return {};
  }

  return value as Record<string, unknown>;
}

function parseConfiguration(value: unknown): IntegrationConfiguration {
  const input = asObject(value);
  return {
    apiToken: typeof input.apiToken === "string" ? input.apiToken.trim() : "",
  };
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

export const integration: IntegrationImplementation = {
  async sync(ctx) {
    const config = parseConfiguration(ctx.configuration);
    if (!config.apiToken) {
      return {
        outcome: "fail",
        errorReason: "error",
        error: "apiToken is required",
        state: "error",
        stateDescription: "apiToken is required",
      };
    }

    try {
      const user = await validateToken(config.apiToken);
      return {
        outcome: "pass",
        state: "ready",
        metadata: {
          accountLogin: user.login,
        },
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : "Token validation failed";
      return {
        outcome: "fail",
        errorReason: "error",
        error: message,
        state: "error",
        stateDescription: message,
      };
    }
  },

  cleanup() {
    return {
      outcome: "noop",
    };
  },
};

if (import.meta.main) {
  await runIntegrationCLI(integration);
}
