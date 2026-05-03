import { Octokit } from "@octokit/rest";
import { z } from "zod";

// Configuration schema
const IssueSearchConfig = z.object({
  owner: z.string().min(1, "Repository owner is required"),
  repo: z.string().min(1, "Repository name is required"),
  searchFilter: z.string().optional(),
  state: z.enum(["open", "closed", "all"]).optional().default("open"),
  labels: z.array(z.string()).optional(),
  assignee: z.string().optional(),
  author: z.string().optional(),
  involved: z.string().optional(),
  sort: z.enum(["created", "updated", "comments"]).optional().default("created"),
  direction: z.enum(["asc", "desc"]).optional().default("desc"),
  perPage: z.number().min(1).max(100).optional().default(30),
  page: z.number().min(1).optional().default(1),
});

type IssueSearchConfigType = z.infer<typeof IssueSearchConfig>;

export class GitHubIssueFetcher {
  private octokit: Octokit;

  constructor(token?: string) {
    this.octokit = new Octokit({
      auth: token || process.env.GITHUB_TOKEN,
    });
  }

  async getIssues(config: IssueSearchConfigType) {
    const validatedConfig = IssueSearchConfig.parse(config);
    
    // Build search query
    const queryParts: string[] = [
      `repo:${validatedConfig.owner}/${validatedConfig.repo}`,
      `is:issue`,
      `state:${validatedConfig.state}`,
    ];

    if (validatedConfig.searchFilter) {
      queryParts.push(validatedConfig.searchFilter);
    }

    if (validatedConfig.labels && validatedConfig.labels.length > 0) {
      queryParts.push(`label:${validatedConfig.labels.join(",")}`);
    }

    if (validatedConfig.assignee) {
      queryParts.push(`assignee:${validatedConfig.assignee}`);
    }

    if (validatedConfig.author) {
      queryParts.push(`author:${validatedConfig.author}`);
    }

    if (validatedConfig.involved) {
      queryParts.push(`involves:${validatedConfig.involved}`);
    }

    const query = queryParts.join(" ");

    try {
      const response = await this.octokit.rest.search.issuesAndPullRequests({
        q: query,
        sort: validatedConfig.sort,
        order: validatedConfig.direction,
        per_page: validatedConfig.perPage,
        page: validatedConfig.page,
      });

      return {
        totalCount: response.data.total_count,
        incompleteResults: response.data.incomplete_results,
        issues: response.data.items.map((item) => ({
          id: item.id,
          number: item.number,
          title: item.title,
          state: item.state,
          body: item.body,
          htmlUrl: item.html_url,
          labels: item.labels.map((label) => ({
            name: typeof label === "string" ? label : label.name,
            color: typeof label === "string" ? undefined : label.color,
          })),
          assignees: item.assignees?.map((assignee) => ({
            login: assignee.login,
            avatarUrl: assignee.avatar_url,
          })),
          author: item.user?.login,
          createdAt: item.created_at,
          updatedAt: item.updated_at,
          closedAt: item.closed_at,
          commentsCount: item.comments,
          milestone: item.milestone?.title,
        })),
      };
    } catch (error) {
      if (error instanceof Error) {
        throw new Error(`Failed to fetch issues: ${error.message}`);
      }
      throw error;
    }
  }

  // Helper method to get issues with pagination support
  async *getIssuesGenerator(config: IssueSearchConfigType) {
    let currentPage = 1;
    let hasMore = true;

    while (hasMore) {
      const result = await this.getIssues({
        ...config,
        page: currentPage,
      });

      yield result;

      hasMore = result.issues.length === config.perPage;
      currentPage++;
    }
  }
}

// Export for use in other modules
export default GitHubIssueFetcher;
