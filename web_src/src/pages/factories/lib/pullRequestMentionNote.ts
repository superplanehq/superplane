import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPullRequest } from "@/api-client";

import { DISCUSSION_MENTION } from "../pages/useDiscussionPRFeedbackSetup";
import { selectWorkOrderCardPullRequest } from "./workOrderCardPullRequest";
import { pullRequestState } from "./workOrderPullRequest";
import type { WorkOrderStatusNotePresentation } from "./workOrderStatusNote";

export const PULL_REQUEST_MENTION_NOTE_KEY = "pull-request-mention";

function normalizedRepository(value: string | undefined): string {
  return value?.trim().toLowerCase() ?? "";
}

/**
 * The healthy pull-request-discussion PR feedback handler that covers a
 * repository, if the factory declared one. A handler with no repository set
 * covers every repository. An unhealthy handler is skipped: its canvas can
 * no longer react to a mention, so promising one would mislead the reviewer.
 */
export function findPullRequestDiscussionHandler(
  repository: string | undefined,
  handlers: FactoriesFactoryPrFeedbackHandler[] | undefined,
): FactoriesFactoryPrFeedbackHandler | undefined {
  const target = normalizedRepository(repository);
  return (handlers ?? []).find((handler) => {
    if (!handler.healthy || handler.source !== "SOURCE_PULL_REQUEST_DISCUSSION") {
      return false;
    }
    const handlerRepository = normalizedRepository(handler.settings?.subject?.repository);
    return handlerRepository === "" || handlerRepository === target;
  });
}

/** The text a pull request comment or review must carry to restart the handler. */
export function pullRequestFeedbackMention(handler: FactoriesFactoryPrFeedbackHandler): string {
  return handler.settings?.discussion?.mention?.trim() || DISCUSSION_MENTION;
}

/**
 * "Ask for changes" note for a task waiting on an open pull request, shown
 * only when the repository already has a healthy PR feedback handler
 * listening for a mention. Undefined otherwise, so the task never promises
 * an automation that will not run.
 */
export function pullRequestMentionNote(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
  workOrderId: string | undefined,
  handlers: FactoriesFactoryPrFeedbackHandler[] | undefined,
): WorkOrderStatusNotePresentation | undefined {
  const pullRequest = selectWorkOrderCardPullRequest(pullRequests, workOrderId ?? "")?.pullRequest;
  const href = pullRequest?.url?.trim();
  if (!pullRequest || pullRequestState(pullRequest.state) !== "open" || !href) {
    return undefined;
  }

  const handler = findPullRequestDiscussionHandler(pullRequest.repository, handlers);
  if (!handler) {
    return undefined;
  }

  const mention = pullRequestFeedbackMention(handler);
  return {
    key: PULL_REQUEST_MENTION_NOTE_KEY,
    headline: "Ask for changes in the pull request",
    text: `Mention ${mention} in a pull request comment or review to request changes.`,
    cta: { label: "Open pull request", href },
  };
}
