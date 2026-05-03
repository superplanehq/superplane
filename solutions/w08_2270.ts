// src/actions/github/get-repository-issues.ts
import { Action } from "../../types/action";
import { SolanaAgentKit } from "../../agent";
import { z } from "zod";
import { getRepositoryIssues } from "../../tools/github";

const getRepositoryIssuesAction: Action = {
  name: "GET_REPOSITORY_ISSUES",
  similes: [
    "list github issues",
    "search github issues",
    "find github issues",
    "get issues from repository",
  ],
  description: "List issues for a GitHub repository using GitHub's issue search API. Supports filtering by state, labels, assignee, author, and more.",
  examples: [
    [
      {
        input: {
          owner: "solana-labs",
          repo: "solana",
          state: "open",
          labels: ["bug", "enhancement"],
        },
        output: {
          issues: [
            {
              number: 123,
              title: "Example issue",
              state: "open",
              labels: ["bug"],
              assignee: "user1",
              author: "user2",
            },
          ],
          totalCount: 1,
        },
        explanation: "Lists open issues with 'bug' and 'enhancement' labels from solana-labs/solana",
      },
    ],
  ],
  schema: z.object({
    owner: z.string().min(1).describe("The GitHub repository owner (username or organization)"),
    repo: z.string().min(1).describe("The GitHub repository name"),
    state: z.enum(["open", "closed", "all"]).optional().describe("Filter by issue state"),
    labels: z.array(z.string()).optional().describe("Filter by labels (comma-separated in API)"),
    assignee: z.string().optional().describe("Filter by assignee username"),
    author: z.string().optional().describe("Filter by author username"),
    mentioned: z.string().optional().describe("Filter by mentioned user"),
    milestone: z.string().optional().describe("Filter by milestone number or title"),
    sort: z.enum(["created", "updated", "comments"]).optional().describe("Sort field"),
    direction: z.enum(["asc", "desc"]).optional().describe("Sort direction"),
    per_page: z.number().min(1).max(100).optional().describe("Results per page (max 100)"),
    page: z.number().min(1).optional().describe("Page number"),
  }),
  handler: async (agent: SolanaAgentKit, input: Record<string, any>) => {
    const result = await getRepositoryIssues(agent, input);
    return result;
  },
};

export default getRepositoryIssuesAction;
