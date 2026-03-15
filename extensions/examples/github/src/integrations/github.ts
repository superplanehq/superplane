import type { IntegrationDefinition, RuntimeValue } from "@superplanehq/sdk";
import { githubWebhookHandler } from "./webhook-handler.js";
import { githubRequest, type GitHubRepository, readGitHubToken } from "../lib/github.js";

interface SyncResponse {
  login: string;
  id: number;
  html_url: string;
}

interface ListResourcesInput {
  resourceType?: RuntimeValue;
}

export const github = {
  name: "github",
  label: "GitHub",
  icon: "github",
  description: "Create and manage GitHub issues",
  instructions: `## Create a GitHub Personal Access Token

1. Open **Settings > Developer settings > Personal access tokens**
2. Create a token with repository issue permissions
3. Paste the token below`,
  configuration: [
    {
      name: "token",
      label: "Token",
      type: "string",
      required: true,
      sensitive: true,
      description: "GitHub personal access token with issue permissions",
    },
  ],
  resourceTypes: ["repository"],
  async sync({ runtime }) {
    const token = await readGitHubToken(runtime.integration);
    const user = await githubRequest<SyncResponse>(runtime.http, {
      method: "GET",
      path: "/user",
      token,
    });

    await runtime.integration.setMetadata({
      login: user.login,
      id: user.id,
      htmlUrl: user.html_url,
    });
    await runtime.integration.ready();
  },
  async listResources({ input, runtime }) {
    const payload = (input ?? {}) as ListResourcesInput;
    if (payload.resourceType !== "repository") {
      return [];
    }

    const token = await readGitHubToken(runtime.integration);
    const repositories = await githubRequest<GitHubRepository[]>(runtime.http, {
      method: "GET",
      path: "/user/repos?per_page=100&sort=updated",
      token,
    });

    return repositories.map((repository) => ({
      type: "repository",
      id: repository.full_name,
      name: repository.full_name,
    }));
  },
  webhook() {
    return githubWebhookHandler;
  },
} satisfies IntegrationDefinition;
