import { DEFAULT_OUTPUT_CHANNEL, type ComponentDefinition } from "@superplanehq/sdk";
import { githubRequest, type GitHubIssue, parseRepositoryReference, readGitHubToken } from "../lib/github.js";

export interface CreateIssueConfiguration {
  repository?: string;
  title?: string;
  body?: string;
}

export const createIssue = {
  name: "github.createIssue",
  integration: "github",
  label: "Create Issue",
  description: "Create a GitHub issue in a selected repository",
  icon: "github",
  color: "gray",
  configuration: [
    {
      name: "repository",
      label: "Repository",
      type: "integration-resource",
      required: true,
      description: "Repository that will receive the issue",
      typeOptions: {
        resource: {
          type: "repository",
          useNameAsValue: true,
        },
      },
    },
    {
      name: "title",
      label: "Title",
      type: "string",
      required: true,
      description: "Issue title",
    },
    {
      name: "body",
      label: "Body",
      type: "text",
      required: false,
      description: "Optional issue body",
      typeOptions: {
        text: {
          minLength: 0,
        },
      },
    },
  ],
  outputChannels: [DEFAULT_OUTPUT_CHANNEL],
  async execute({ configuration, runtime }) {
    const config = configuration as CreateIssueConfiguration;
    const repository = parseRepositoryReference(config.repository ?? null);
    if (!config.title || !config.title.trim()) {
      throw new Error("title is required");
    }

    const token = await readGitHubToken(runtime.integration);
    const requestBody: Record<string, string> = {
      title: config.title,
    };
    if (config.body?.trim()) {
      requestBody.body = config.body.trim();
    }

    const issue = await githubRequest<GitHubIssue>(runtime.http, {
      method: "POST",
      path: `/repos/${repository}/issues`,
      token,
      body: requestBody,
    });

    await runtime.executionState.emit(DEFAULT_OUTPUT_CHANNEL.name, "github.issue", [
      {
        id: issue.id,
        number: issue.number,
        title: issue.title,
        state: issue.state,
        url: issue.html_url,
        body: issue.body,
        repository,
      },
    ]);
  },
} satisfies ComponentDefinition<CreateIssueConfiguration>;
