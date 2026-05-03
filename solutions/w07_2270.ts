// src/actions/github/get-repository-issues.ts
import { Action } from "../../types/action";
import { SolanaAgentKit } from "../../agent";
import { z } from "zod";
import { get_github_issues } from "../../tools/github";

const getRepositoryIssuesAction: Action = {
  name: "GET_REPOSITORY_ISSUES",
  similes: [
    "list github issues",
    "fetch repository issues",
    "search github issues",
    "get issues from repo",
  ],
  description: "List issues for a GitHub repository using GitHub's issue search. Supports a top-level search filter input that prefills the form, plus full control via individual fields aligned with GitHub issue search qualifiers.",
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
              created_at: "2024-01-01T00:00:00Z",
              updated_at: "2024-01-02T00:00:00Z",
            },
          ],
          total_count: 1,
        },
        explanation: "List open issues with labels 'bug' and 'enhancement' from solana-labs/solana",
      },
    ],
  ],
  schema: z.object({
    owner: z.string().min(1).describe("The owner (user or organization) of the repository"),
    repo: z.string().min(1).describe("The name of the repository"),
    state: z.enum(["open", "closed", "all"]).optional().describe("Filter by state: open, closed, or all"),
    labels: z.array(z.string()).optional().describe("List of label names to filter by"),
    assignee: z.string().optional().describe("Filter by assignee username"),
    author: z.string().optional().describe("Filter by author username"),
    mentioned: z.string().optional().describe("Filter by mentioned username"),
    milestone: z.string().optional().describe("Filter by milestone number or title"),
    sort: z.enum(["created", "updated", "comments"]).optional().describe("Sort field"),
    direction: z.enum(["asc", "desc"]).optional().describe("Sort direction"),
    per_page: z.number().min(1).max(100).optional().describe("Results per page (max 100)"),
    page: z.number().min(1).optional().describe("Page number"),
    search_filter: z.string().optional().describe("Top-level search filter that prefills the form"),
  }),
  handler: async (agent: SolanaAgentKit, input: Record<string, any>) => {
    const result = await get_github_issues(agent, input);
    return result;
  },
};

export default getRepositoryIssuesAction;
