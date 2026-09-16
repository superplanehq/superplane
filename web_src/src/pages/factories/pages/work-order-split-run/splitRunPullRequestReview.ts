import type { SplitRunFooterNote } from "./splitRunFooter";

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

export interface PullRequestReviewStep {
  title: string;
  text: string;
}

export const PULL_REQUEST_REVIEW_COPY = {
  headline: "The pull request is ready for review",
  steps: [
    { title: "Review the pull request.", text: "Open it on GitHub and read the changes." },
    { title: "Leave comments.", text: "Mention @superplaneagent in a comment or review to request changes." },
    { title: "SuperPlane addresses them.", text: "It updates the pull request and asks you to review again." },
  ] satisfies PullRequestReviewStep[],
  stepsLabel: "Next steps",
  closing: "This task closes when the pull request is merged or closed.",
} as const;
