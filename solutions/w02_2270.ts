// src/actions/github-get-issues.ts
import { Action } from "@superplane/actions";
import { z } from "zod";

const InputSchema = z.object({
  repo: z.string().describe("Repository in format owner/repo"),
  state: z.enum(["open", "closed", "all"]).optional().default("open"),
  labels: z.string().optional().describe("Comma-separated list of labels"),
  assignee: z.string().optional().describe("GitHub username or 'none'"),
  author: z.string().optional().describe("GitHub username"),
  mentioned: z.string().optional().describe("GitHub username"),
  search: z.string().optional().describe("Free text search filter"),
  per_page: z.number().min(1).max(100).optional().default(30),
  page: z.number().min(1).optional().default(1),
});

const OutputSchema = z.object({
  issues: z.array(z.object({
    number: z.number(),
    title: z.string(),
    state: z.string(),
    html_url: z.string(),
    created_at: z.string(),
    updated_at: z.string(),
    labels: z.array(z.object({
      name: z.string(),
      color: z.string(),
    })),
    assignees: z.array(z.object({
      login: z.string(),
    })),
    user: z.object({
      login: z.string(),
    }),
  })),
  total_count: z.number(),
});

export default new Action({
  id: "github-get-issues",
  name: "Get Repository Issues",
  description: "List issues for a GitHub repository with advanced filtering",
  input: InputSchema,
  output: OutputSchema,
  async run(ctx, input) {
    const { repo, state, labels, assignee, author, mentioned, search, per_page, page } = input;

    // Build query parameters
    const params = new URLSearchParams();
    params.set("state", state);
    params.set("per_page", per_page.toString());
    params.set("page", page.toString());

    // Build search qualifiers
    const qualifiers: string[] = [];
    qualifiers.push(`repo:${repo}`);
    qualifiers.push(`is:issue`);
    qualifiers.push(`state:${state}`);

    if (labels) {
      const labelList = labels.split(",").map(l => l.trim()).filter(Boolean);
      if (labelList.length > 0) {
        qualifiers.push(`label:${labelList.join(",")}`);
      }
    }

    if (assignee) {
      qualifiers.push(`assignee:${assignee}`);
    }

    if (author) {
      qualifiers.push(`author:${author}`);
    }

    if (mentioned) {
      qualifiers.push(`mentions:${mentioned}`);
    }

    // Combine search query
    let query = qualifiers.join(" ");
    if (search) {
      query = `${search} ${query}`;
    }

    params.set("q", query);

    // Make API request
    const response = await ctx.fetch(
      `https://api.github.com/search/issues?${params.toString()}`,
      {
        headers: {
          "Accept": "application/vnd.github.v3+json",
          "Authorization": `token ${ctx.secrets.GITHUB_TOKEN}`,
        },
      }
    );

    if (!response.ok) {
      throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();

    return {
      issues: data.items.map((item: any) => ({
        number: item.number,
        title: item.title,
        state: item.state,
        html_url: item.html_url,
        created_at: item.created_at,
        updated_at: item.updated_at,
        labels: item.labels.map((label: any) => ({
          name: label.name,
          color: label.color,
        })),
        assignees: item.assignees.map((assignee: any) => ({
          login: assignee.login,
        })),
        user: {
          login: item.user.login,
        },
      })),
      total_count: data.total_count,
    };
  },
});
