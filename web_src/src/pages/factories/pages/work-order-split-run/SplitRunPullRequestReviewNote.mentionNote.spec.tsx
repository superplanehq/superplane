import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type {
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestMergeability,
} from "@/api-client";

const mergeability = { current: undefined as FactoriesFactoryPullRequestMergeability | undefined };

vi.mock("@/hooks/useFactoryPullRequestMerge", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
  useMergeFactoryPullRequest: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => true, enabledExperimentalFeatures: [], isLoading: false }),
}));

import { pullRequestMentionNote } from "../../lib/pullRequestMentionNote";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { toFooterNote, type SplitRunFooterAction, type SplitRunFooterNote } from "./splitRunFooter";
import { trackedPullRequestReviewNote } from "./splitRunPullRequestReview";

const OPEN_PULL_REQUEST: FactoriesFactoryPullRequest = {
  id: "pr-42",
  workOrderId: "wo-1",
  repository: "acme/payments",
  url: "https://github.com/acme/payments/pull/42",
  number: "42",
  state: "STATE_OPEN",
};

const ACTIONS: SplitRunFooterAction[] = [
  { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" },
  { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" },
];

function handler(overrides: Partial<FactoriesFactoryPrFeedbackHandler> = {}): FactoriesFactoryPrFeedbackHandler {
  return {
    id: "handler-1",
    factoryId: "factory-1",
    source: "SOURCE_PULL_REQUEST_DISCUSSION",
    healthy: true,
    settings: { subject: { repository: "acme/payments" } },
    ...overrides,
  };
}

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

beforeEach(() => {
  mergeability.current = undefined;
});

/**
 * Renders the review step exactly as the popup assembles it: the
 * pull-request review note the waiting footer already carries, plus the
 * mention note built from the tracked pull request and the factory's PR
 * feedback handlers.
 */
function renderReview(handlers: FactoriesFactoryPrFeedbackHandler[] | undefined) {
  const note = trackedPullRequestReviewNote([OPEN_PULL_REQUEST], "wo-1") as SplitRunFooterNote;
  const mentionNote = pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", handlers);

  return render(
    <MemoryRouter>
      <SplitRunAttentionNote
        note={note}
        mentionNote={mentionNote ? toFooterNote(mentionNote) : undefined}
        tone="waiting"
        actions={ACTIONS}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
        canAct
      />
    </MemoryRouter>,
  );
}

describe("task popup review step: pull request mention note", () => {
  it("shows the note text, mention, and pull-request link for a matching healthy handler", () => {
    renderReview([handler()]);

    const note = screen.getByTestId("split-run-pull-request-mention-note");
    expect(note).toHaveTextContent("Mention @superplaneagent in a pull request comment or review to request changes.");

    const link = screen.getByTestId("split-run-pull-request-cta");
    expect(link).toHaveAccessibleName("Review PR #42");
    expect(link).toHaveAttribute("href", "https://github.com/acme/payments/pull/42");
  });

  it("is absent without a matching handler", () => {
    renderReview([]);

    expect(screen.queryByTestId("split-run-pull-request-mention-note")).not.toBeInTheDocument();
  });

  it("is absent when the only handler is unhealthy", () => {
    renderReview([handler({ healthy: false })]);

    expect(screen.queryByTestId("split-run-pull-request-mention-note")).not.toBeInTheDocument();
  });
});
