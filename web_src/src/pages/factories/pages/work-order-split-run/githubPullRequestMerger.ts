import type { FactoriesFactoryPullRequest } from "@/api-client";
import { githubApiOrigin, githubProfileUrl, githubWebHostFromUrl, isGitHubWebHostname } from "@/lib/githubHost";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { pullRequestState } from "../../lib/workOrderPullRequest";
import type { SplitRunFooterCloser } from "./splitRunFooterActor";

export function githubPullRequestApiHref(url: string | undefined): string | undefined {
  const parts = githubPullRequestParts(url);
  if (!parts) {
    return undefined;
  }
  return `${githubApiOrigin(parts.host)}/repos/${parts.owner}/${parts.repo}/pulls/${parts.number}`;
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
  host?: string | null,
): SplitRunFooterCloser {
  const login = user?.login?.trim();
  if (!login) {
    return {};
  }
  const name = user?.name?.trim() || login;
  const href = safeExternalUrl(user?.html_url) ?? githubProfileUrl(login, host);
  return { automationName: name, automationHref: href };
}

function githubPullRequestParts(
  url: string | undefined,
): { host: string; owner: string; repo: string; number: string } | undefined {
  const safe = safeExternalUrl(url);
  if (!safe) {
    return undefined;
  }
  let parsed: URL;
  try {
    parsed = new URL(safe);
  } catch {
    return undefined;
  }
  if (!isGitHubWebHostname(parsed.hostname)) {
    return undefined;
  }
  const [, owner, repo, kind, number] = parsed.pathname.split("/");
  if (!owner || !repo || kind !== "pull" || !/^\d+$/.test(number ?? "")) {
    return undefined;
  }
  return { host: githubWebHostFromUrl(safe), owner, repo, number };
}
