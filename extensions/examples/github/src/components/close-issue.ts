import { DEFAULT_OUTPUT_CHANNEL, type ComponentDefinition } from "@superplanehq/sdk";
import { githubRequest, type GitHubIssue, parseIssueNumber, parseRepositoryReference, readGitHubToken } from "../lib/github.js";

export interface CloseIssueConfiguration {
  repository?: string;
  issueNumber?: number;
}

export const closeIssue = {
  name: "github.closeIssue",
  integration: "github",
  label: "Close Issue",
  description: "Close a GitHub issue in a selected repository",
  icon: "github",
  color: "gray",
  configuration: [
    {
      name: "repository",
      label: "Repository",
      type: "integration-resource",
      required: true,
      description: "Repository that owns the issue",
      typeOptions: {
        resource: {
          type: "repository",
          useNameAsValue: true,
        },
      },
    },
    {
      name: "issueNumber",
      label: "Issue Number",
      type: "number",
      required: true,
      description: "Issue number to close",
      typeOptions: {
        number: {
          min: 1,
        },
      },
    },
  ],
  outputChannels: [DEFAULT_OUTPUT_CHANNEL],
  async execute({ configuration, runtime }) {
    const config = configuration as CloseIssueConfiguration;
    const repository = parseRepositoryReference(config.repository ?? null);
    const issueNumber = parseIssueNumber(config.issueNumber ?? null);
    const token = await readGitHubToken(runtime.integration);

    const issue = await githubRequest<GitHubIssue>(runtime.http, {
      method: "PATCH",
      path: `/repos/${repository}/issues/${issueNumber}`,
      token,
      body: {
        state: "closed",
      },
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
        operation: "closed",
      },
    ]);
  },
} satisfies ComponentDefinition<CloseIssueConfiguration>;
