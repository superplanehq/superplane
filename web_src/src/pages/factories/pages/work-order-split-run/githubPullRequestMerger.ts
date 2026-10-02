import type { FactoriesFactoryPullRequest } from "@/api-client";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { pullRequestState } from "../../lib/workOrderPullRequest";
import type { SplitRunFooterCloser } from "./splitRunFooterActor";

const GITHUB_PULL = /^https?:\/\/(?:www\.)?github\.com\/([^/]+)\/([^/]+)\/pull\/(\d+)(?:[/?#]|$)/i;

export function githubPullRequestApiHref(url: string | undefined): string | undefined {
  const safe = safeExternalUrl(url);
  if (!safe) {
    return undefined;
  }
  const match = GITHUB_PULL.exec(safe);
  if (!match) {
    return undefined;
  }
  return `https://api.github.com/repos/${match[1]}/${match[2]}/pulls/${match[3]}`;
}

export function latestMergedGitHubPullRequest(
  pullRequests: FactoriesFactoryPullRequest[] = [],
): FactoriesFactoryPullRequest | undefined {
  return [...pullRequests]
    .filter(
      (pullRequest) => pullRequestState(pullRequest.state) === "merged" && githubPullRequestApiHref(pullRequest.url),
    )
    .sort((left, right) => Date.parse(right.mergedAt ?? "") - Date.parse(left.mergedAt ?? ""))[0];
}

export function closerFromGitHubMergedBy(
  user?: {
    login?: string;
    name?: string;
    html_url?: string;
  } | null,
): SplitRunFooterCloser {
  const login = user?.login?.trim();
  if (!login) {
    return {};
  }
  const name = user?.name?.trim() || login;
  const href = safeExternalUrl(user?.html_url) ?? `https://github.com/${login}`;
  return { automationName: name, automationHref: href };
}
