import { Octokit } from "@octokit/rest";
import { createAppAuth } from "@octokit/auth-app";

interface IssueSearchParams {
  q?: string;
  repo?: string;
  owner?: string;
  state?: "open" | "closed" | "all";
  labels?: string[];
  assignee?: string;
  author?: string;
  mentioned?: string;
  sort?: "created" | "updated" | "comments";
  direction?: "asc" | "desc";
  per_page?: number;
  page?: number;
}

interface GitHubIssue {
  id: number;
  number: number;
  title: string;
  state: string;
  labels: { name: string; color: string }[];
  assignees: { login: string }[];
  user: { login: string };
  created_at: string;
  updated_at: string;
  html_url: string;
  body: string;
}

export class GitHubIssueFetcher {
  private octokit: Octokit;

  constructor(private token: string, private appId?: string, private privateKey?: string) {
    if (appId && privateKey) {
      this.octokit = new Octokit({
        authStrategy: createAppAuth,
        auth: {
          appId,
          privateKey,
          installationId: "",
        },
      });
    } else {
      this.octokit = new Octokit({ auth: token });
    }
  }

  async searchIssues(params: IssueSearchParams): Promise<{ issues: GitHubIssue[]; totalCount: number }> {
    const queryParts: string[] = [];

    if (params.q) {
      queryParts.push(params.q);
    }

    if (params.owner && params.repo) {
      queryParts.push(`repo:${params.owner}/${params.repo}`);
    }

    if (params.state && params.state !== "all") {
      queryParts.push(`state:${params.state}`);
    }

    if (params.labels && params.labels.length > 0) {
      params.labels.forEach((label) => {
        queryParts.push(`label:"${label}"`);
      });
    }

    if (params.assignee) {
      queryParts.push(`assignee:${params.assignee}`);
    }

    if (params.author) {
      queryParts.push(`author:${params.author}`);
    }

    if (params.mentioned) {
      queryParts.push(`mentions:${params.mentioned}`);
    }

    const query = queryParts.join(" ");

    const response = await this.octokit.rest.search.issuesAndPullRequests({
      q: query,
      sort: params.sort || "created",
      order: params.direction || "desc",
      per_page: params.per_page || 30,
      page: params.page || 1,
    });

    const issues = response.data.items
      .filter((item) => !item.pull_request)
      .map((item) => ({
        id: item.id,
        number: item.number,
        title: item.title,
        state: item.state,
        labels: item.labels.map((label: any) => ({
          name: label.name,
          color: label.color,
        })),
        assignees: item.assignees.map((assignee: any) => ({
          login: assignee.login,
        })),
        user: { login: item.user.login },
        created_at: item.created_at,
        updated_at: item.updated_at,
        html_url: item.html_url,
        body: item.body || "",
      }));

    return {
      issues,
      totalCount: response.data.total_count,
    };
  }

  async getRepositoryIssues(
    owner: string,
    repo: string,
    options?: {
      state?: "open" | "closed" | "all";
      labels?: string[];
      assignee?: string;
      creator?: string;
      sort?: "created" | "updated" | "comments";
      direction?: "asc" | "desc";
      per_page?: number;
      page?: number;
    }
  ): Promise<GitHubIssue[]> {
    const response = await this.octokit.rest.issues.listForRepo({
      owner,
      repo,
      state: options?.state || "open",
      labels: options?.labels?.join(","),
      assignee: options?.assignee,
      creator: options?.creator,
      sort: options?.sort || "created",
      direction: options?.direction || "desc",
      per_page: options?.per_page || 30,
      page: options?.page || 1,
    });

    return response.data.map((item) => ({
      id: item.id,
      number: item.number,
      title: item.title,
      state: item.state,
      labels: item.labels.map((label: any) => ({
        name: label.name,
        color: label.color,
      })),
      assignees: item.assignees.map((assignee: any) => ({
        login: assignee.login,
      })),
      user: { login: item.user.login },
      created_at: item.created_at,
      updated_at: item.updated_at,
      html_url: item.html_url,
      body: item.body || "",
    }));
  }
}

// Example usage:
// const fetcher = new GitHubIssueFetcher("your-github-token");
// 
// // Search with filters
// const result = await fetcher.searchIssues({
//   q: "bug",
//   owner: "octocat",
//   repo: "Hello-World",
//   state: "open",
//   labels: ["bug", "help wanted"],
//   sort: "updated",
//   direction: "desc",
// });
// 
// // Get repository issues directly
// const issues = await fetcher.getRepositoryIssues("octocat", "Hello-World", {
//   state: "open",
//   labels: ["bug"],
// });
