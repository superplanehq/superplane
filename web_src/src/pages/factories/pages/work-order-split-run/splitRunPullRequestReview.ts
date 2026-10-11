import type {
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestMergeability,
  FactoryPullRequestMergeabilityMergeMethod,
} from "@/api-client";

import { selectWorkOrderCardPullRequest } from "../../lib/workOrderCardPullRequest";
import { pullRequestLabel, pullRequestState } from "../../lib/workOrderPullRequest";

import type { SplitRunFooter, SplitRunFooterNote } from "./splitRunFooter";

/**
 * `https://github.com/<owner>/<repo>/pull/<number>` or
 * `https://bitbucket.org/<workspace>/<repo>/pull-requests/<number>`,
 * each with an optional tail.
 */
const PULL_REQUEST_URL = /^https?:\/\/[^/]+\/[^/]+\/[^/]+\/(?:pull\/(\d+)|pull-requests\/(\d+))(?:[/?#]|$)/;

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
  return { href, number: Number(match[1] ?? match[2]) };
}

export function isPullRequestReviewFooter(footer: SplitRunFooter): boolean {
  return Boolean(
    footer.kind === "waiting" && footer.attentionCard && footer.note && pullRequestReviewNote(footer.note),
  );
}

export function trackedPullRequestReviewNote(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
  workOrderId: string | undefined,
): SplitRunFooterNote | undefined {
  const pullRequest = selectWorkOrderCardPullRequest(pullRequests, workOrderId ?? "")?.pullRequest;
  const href = pullRequest?.url?.trim();
  if (!pullRequest || pullRequestState(pullRequest.state) !== "open" || !href) {
    return undefined;
  }

  return {
    headline: PULL_REQUEST_REVIEW_COPY.headline,
    text: PULL_REQUEST_REVIEW_COPY.closing,
    cta: {
      label: `Review PR ${pullRequestLabel(pullRequest)}`,
      href,
    },
  };
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

/** Pull requests SuperPlane can merge: GitHub and Bitbucket. */
export function supportsPullRequestMerge(pullRequest: FactoriesFactoryPullRequest | undefined): boolean {
  switch (pullRequest?.provider ?? "PROVIDER_GITHUB") {
    case "PROVIDER_GITHUB":
    case "PROVIDER_BITBUCKET":
      return true;
    default:
      return false;
  }
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

export interface PullRequestReviewCopy {
  headline: string;
  closing: string;
}

/**
 * Headline and closing line for the review strip. The merge gate knows the
 * checks state, so while checks run or fail the strip says that instead of
 * calling the pull request ready.
 */
export function pullRequestReviewCopy(mergeability?: FactoriesFactoryPullRequestMergeability): PullRequestReviewCopy {
  if (mergeability?.blockedReason === "BLOCKED_REASON_UNAVAILABLE") {
    return {
      headline: "Merge status is unavailable right now.",
      closing: PULL_REQUEST_REVIEW_COPY.closing,
    };
  }
  if (mergeability?.blockedReason === "BLOCKED_REASON_CHECKS_UNFINISHED") {
    return {
      headline: "The pull request waits for checks",
      closing: "You can review it now. This task closes when the pull request is merged or closed.",
    };
  }
  if (mergeability?.blockedReason === "BLOCKED_REASON_CHECK_FAILED") {
    return {
      headline: "A pull request check failed",
      closing: "Review the failed check on the pull request.",
    };
  }
  if (mergeability?.blockedReason === "BLOCKED_REASON_CONFLICTING") {
    return {
      headline: "The pull request has conflicts",
      closing: "Resolve the conflicts on the pull request before merge.",
    };
  }
  return { headline: PULL_REQUEST_REVIEW_COPY.headline, closing: PULL_REQUEST_REVIEW_COPY.closing };
}

export const PULL_REQUEST_REVIEW_COPY = {
  headline: "The pull request is ready for review",
  closing: "This task closes when the pull request is merged or closed.",
  moreActions: "More actions",
  merge: "Merge",
  mergeMethod: "Merge method",
  merged: "The pull request is merged.",
  permission: "You do not have permission to manage this task.",
  methods: {
    MERGE_METHOD_SQUASH: "Squash and merge",
    MERGE_METHOD_MERGE: "Create a merge commit",
    MERGE_METHOD_REBASE: "Rebase and merge",
  },
} as const;
