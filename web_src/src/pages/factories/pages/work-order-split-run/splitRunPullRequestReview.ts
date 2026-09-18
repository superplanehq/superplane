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

export const PULL_REQUEST_REVIEW_COPY = {
  headline: "The pull request is ready for review",
  closing: "This task closes when the pull request is merged or closed.",
  moreActions: "More actions",
} as const;
