import type { FactoriesFactoryPullRequest, FactoryPullRequestMergeabilityMergeMethod } from "@/api-client";

import { pullRequestState } from "../../lib/workOrderPullRequest";

import type { SplitRunFooter, SplitRunFooterNote } from "./splitRunFooter";

/** `https://github.com/<owner>/<repo>/pull/<number>` with an optional tail. */
const PULL_REQUEST_URL = /^https?:\/\/[^/]+\/[^/]+\/[^/]+\/pull\/(\d+)(?:[/?#]|$)/;

export interface PullRequestReviewTarget {
  href: string;
  number: number;
}

/**
 * A status note whose call to action opens a pull request. The Implement
 * line sets one when it opens or updates the pull request for a task.
 * The popup renders such a note as the pull-request review strip.
 */
export function pullRequestReviewNote(note: SplitRunFooterNote): PullRequestReviewTarget | undefined {
  const href = note.cta?.href;
  if (!href) {
    return undefined;
  }
  const match = PULL_REQUEST_URL.exec(href);
  if (!match) {
    return undefined;
  }
  return { href, number: Number(match[1]) };
}

export function isPullRequestReviewFooter(footer: SplitRunFooter): boolean {
  return Boolean(
    footer.kind === "waiting" && footer.attentionCard && footer.note && pullRequestReviewNote(footer.note),
  );
}

export function pullRequestForReviewHref(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
  href: string,
): FactoriesFactoryPullRequest | undefined {
  const normalized = href.replace(/\/$/, "");
  return pullRequests?.find((pullRequest) => (pullRequest.url ?? "").replace(/\/$/, "") === normalized);
}

export function isGitHubPullRequest(pullRequest: FactoriesFactoryPullRequest | undefined): boolean {
  return (pullRequest?.provider ?? "PROVIDER_GITHUB") === "PROVIDER_GITHUB";
}

export function isMergedPullRequest(pullRequest: FactoriesFactoryPullRequest | undefined): boolean {
  return pullRequestState(pullRequest?.state) === "merged";
}

export type FactoryPullRequestMergeMethodChoice = Exclude<
  FactoryPullRequestMergeabilityMergeMethod,
  "MERGE_METHOD_UNSPECIFIED"
>;

const MERGE_METHOD_PREFERENCE: FactoryPullRequestMergeMethodChoice[] = [
  "MERGE_METHOD_SQUASH",
  "MERGE_METHOD_MERGE",
  "MERGE_METHOD_REBASE",
];

export function defaultMergeMethod(
  methods: FactoryPullRequestMergeabilityMergeMethod[] | undefined,
): FactoryPullRequestMergeMethodChoice | undefined {
  const allowed = new Set(methods ?? []);
  return MERGE_METHOD_PREFERENCE.find((method) => allowed.has(method));
}

export const PULL_REQUEST_REVIEW_COPY = {
  headline: "The pull request is ready for review",
  closing: "This task closes when the pull request is merged or closed.",
  moreActions: "More actions",
  merge: "Merge",
  merging: "Merging",
  mergeMethod: "Merge method",
  merged: "The pull request is merged.",
  permission: "You do not have permission to manage this task.",
  methods: {
    MERGE_METHOD_SQUASH: "Squash and merge",
    MERGE_METHOD_MERGE: "Create a merge commit",
    MERGE_METHOD_REBASE: "Rebase and merge",
  },
} as const;
