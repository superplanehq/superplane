import { runComponentCLI } from "../../../../typescript/mod.ts";

type GetIssueConfig = {
  owner?: string;
  repository?: string;
  issueNumber?: number;
};

type IntegrationConfig = {
  apiToken?: string;
};

function asObject(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object") {
    return {};
  }

  return value as Record<string, unknown>;
}

function parseComponentConfig(value: unknown): GetIssueConfig {
  const input = asObject(value);
  const issueNumber = typeof input.issueNumber === "number" ? input.issueNumber : Number(input.issueNumber ?? 0);

  return {
    owner: typeof input.owner === "string" ? input.owner.trim() : "",
    repository: typeof input.repository === "string" ? input.repository.trim() : "",
    issueNumber: Number.isFinite(issueNumber) ? issueNumber : 0,
  };
}

function parseIntegrationConfig(value: unknown): IntegrationConfig {
  const input = asObject(value);
  return {
    apiToken: typeof input.apiToken === "string" ? input.apiToken.trim() : "",
  };
}

await runComponentCLI({
  setup(ctx) {
    const config = parseComponentConfig(ctx.configuration);
    if (!config.owner) {
      throw new Error("owner is required");
    }
    if (!config.repository) {
      throw new Error("repository is required");
    }
    if (!config.issueNumber || config.issueNumber < 1) {
      throw new Error("issueNumber must be >= 1");
    }

    const integration = parseIntegrationConfig(ctx.integrationConfiguration);
    if (!integration.apiToken) {
      throw new Error("integration apiToken is required");
    }
  },

  async execute(ctx) {
    const config = parseComponentConfig(ctx.configuration);
    const integration = parseIntegrationConfig(ctx.integrationConfiguration);

    if (!integration.apiToken) {
      return {
        outcome: "fail",
        errorReason: "error",
        error: "integration apiToken is required",
      };
    }

    const url = `https://api.github.com/repos/${encodeURIComponent(config.owner ?? "")}/${encodeURIComponent(config.repository ?? "")}/issues/${config.issueNumber}`;
    const response = await fetch(url, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${integration.apiToken}`,
        Accept: "application/vnd.github+json",
        "User-Agent": "superplane-github2",
      },
    });

    if (!response.ok) {
      const message = await response.text();
      return {
        outcome: "fail",
        errorReason: "error",
        error: `GitHub API failed (${response.status}): ${message}`,
      };
    }

    const issue = (await response.json()) as Record<string, unknown>;

    return {
      outcome: "pass",
      outputs: [
        {
          channel: "default",
          payloadType: "github2.issue",
          payload: issue,
        },
      ],
    };
  },
});
