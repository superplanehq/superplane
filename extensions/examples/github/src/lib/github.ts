import type { HTTPContext, IntegrationContext, RuntimeValue } from "@superplanehq/sdk";

const decoder = new TextDecoder();

interface GitHubRequestOptions {
  method: string;
  path: string;
  token: string;
  body?: RuntimeValue;
}

export interface GitHubRepository {
  id: number;
  full_name: string;
  private: boolean;
  html_url: string;
}

export interface GitHubIssue {
  id: number;
  number: number;
  title: string;
  state: string;
  html_url: string;
  body: string | null;
}

export interface GitHubWebhook {
  id: number;
  name: string;
}

export type GitHubPushEvent = Record<string, RuntimeValue>;

export async function readGitHubToken(integration: IntegrationContext): Promise<string> {
  const value = await integration.getConfig("token");
  const token = decoder.decode(value).trim();
  if (!token) {
    throw new Error("GitHub token is required");
  }

  return token;
}

export async function githubRequest<T>(http: HTTPContext, options: GitHubRequestOptions): Promise<T> {
  const response = await http.do({
    method: options.method,
    url: `https://api.github.com${options.path}`,
    headers: {
      Accept: "application/vnd.github+json",
      Authorization: `Bearer ${options.token}`,
      "User-Agent": "superplane-extension-runner",
      "X-GitHub-Api-Version": "2022-11-28",
      ...(options.body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  const payloadText = decoder.decode(response.body);
  const payload = payloadText ? (JSON.parse(payloadText) as T) : (null as T);
  if (response.status < 200 || response.status >= 300) {
    const message = typeof payload === "object" && payload !== null && "message" in (payload as Record<string, unknown>)
      ? String((payload as Record<string, unknown>).message)
      : `GitHub API returned ${response.status}`;
    throw new Error(message);
  }

  return payload;
}

export function parseRepositoryReference(value: RuntimeValue): string {
  if (typeof value !== "string" || !value.trim()) {
    throw new Error("repository is required");
  }

  return value.trim();
}

export function parseIssueNumber(value: RuntimeValue): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value <= 0) {
    throw new Error("issueNumber must be a positive integer");
  }

  return value;
}

export function parseRepositoryOwnerAndName(value: RuntimeValue): { owner: string; repository: string } {
  const fullName = parseRepositoryReference(value);
  const parts = fullName.split("/");
  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    throw new Error("repository must be in the format owner/name");
  }

  return {
    owner: parts[0],
    repository: parts[1],
  };
}

export function getHeader(headers: Record<string, string[]>, name: string): string {
  const target = name.toLowerCase();
  for (const [key, values] of Object.entries(headers)) {
    if (key.toLowerCase() !== target || values.length === 0) {
      continue;
    }

    return values[0] ?? "";
  }

  return "";
}
