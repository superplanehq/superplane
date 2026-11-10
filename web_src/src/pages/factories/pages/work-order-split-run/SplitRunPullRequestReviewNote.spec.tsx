import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryPullRequest, FactoriesFactoryPullRequestMergeability } from "@/api-client";
import { TooltipProvider } from "@/components/ui/tooltip";
import { FEATURE_FACTORY_STEP_ACTIONS } from "@/lib/experimentalFeatures";

const mergeability = { current: undefined as FactoriesFactoryPullRequestMergeability | undefined };
const mergePending = { current: false };
const mergeMutate = vi.fn();
const experimentalFeatureHas = { current: (_id: string) => true };

vi.mock("@/hooks/useFactoryPullRequestMerge", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
  useMergeFactoryPullRequest: () => ({ mutate: mergeMutate, isPending: mergePending.current }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => experimentalFeatureHas.current(id),
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { PullRequestChecksContext } from "./redesign/pullRequestChecksContext";
import { SplitRunPullRequestReviewNote } from "./SplitRunPullRequestReviewNote";
import type { SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";

const PR_NOTE: SplitRunFooterNote = {
  headline: "Waiting for user review",
  text: "The pull request is open and waiting for user review. Mention @superplaneagent in a pull request comment or review to request changes.",
  cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
};

const ACTIONS: SplitRunFooterAction[] = [
  { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" },
  { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" },
];

const GITHUB_PR: FactoriesFactoryPullRequest = {
  id: "pr-6812",
  provider: "PROVIDER_GITHUB",
  url: "https://github.com/acme/payments/pull/6812",
  number: "6812",
  state: "STATE_OPEN",
};

const BITBUCKET_PR_NOTE: SplitRunFooterNote = {
  ...PR_NOTE,
  cta: { label: "Review PR #7", href: "https://bitbucket.org/acme/payments/pull-requests/7" },
};

const BITBUCKET_PR: FactoriesFactoryPullRequest = {
  id: "pr-7",
  provider: "PROVIDER_BITBUCKET",
  url: "https://bitbucket.org/acme/payments/pull-requests/7",
  number: "7",
  state: "STATE_OPEN",
};

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

beforeEach(() => {
  mergeMutate.mockReset();
  mergePending.current = false;
  experimentalFeatureHas.current = () => true;
  mergeability.current = {
    canMerge: true,
    allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_MERGE"],
    headSha: "abc123",
  };
});

function renderNote(props: Partial<Parameters<typeof SplitRunAttentionNote>[0]> = {}) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <SplitRunAttentionNote
          note={PR_NOTE}
          tone="waiting"
          actions={ACTIONS}
          organizationId="org-1"
          factoryId="factory-1"
          orderId="wo-1"
          canAct
          {...props}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("SplitRunAttentionNote for a pull request", () => {
  it.each(
    [{ compact: false }, { compact: true }, { ctaOnly: true }].flatMap((variant) =>
      [false, true].flatMap((actionsEnabled) =>
        ([undefined, "activities", "runs"] as const).map((runSource) => ({ ...variant, actionsEnabled, runSource })),
      ),
    ),
  )(
    "keeps review evidence available until it can appear in Actions (%p)",
    async ({ actionsEnabled, runSource, ...variant }) => {
      experimentalFeatureHas.current = (id) => id !== FEATURE_FACTORY_STEP_ACTIONS || actionsEnabled;
      renderNote({
        ...variant,
        pullRequests: [
          {
            ...GITHUB_PR,
            activities: runSource === "activities" ? [{ run: { id: "review-run", canvasId: "review-app" } }] : [],
            runs: runSource === "runs" ? [{ run: { id: "review-run", canvasId: "review-app" } }] : [],
            humanReview: {
              runId: runSource ? "review-run" : undefined,
              status: "decided",
              reviewRequired: true,
              rationale: "A security finding requires review.",
              selectedReviewers: [{ login: "alex", reason: "Owns the affected files." }],
            },
          },
        ],
      });
      if (actionsEnabled && runSource) {
        expect(screen.queryByRole("region", { name: "Human review" })).not.toBeInTheDocument();
        return;
      }
      expect(screen.getByRole("region", { name: "Human review" })).toBeVisible();
      await userEvent.setup().click(screen.getByRole("button", { name: /Human review/ }));
      expect(screen.getByText("A security finding requires review.")).toBeVisible();
      expect(screen.getByText(/Owns the affected files/)).toBeVisible();
    },
  );

  it.each([
    { status: "waiting_assessment", blockingReason: "Merge confidence failed. Retry the assessment." },
    { status: "failed", blockingReason: "The review agent could not start. Retry Human review." },
  ])("keeps recovery instructions available without a review run (%p)", async (humanReview) => {
    renderNote({ pullRequests: [{ ...GITHUB_PR, humanReview }], ctaOnly: true });

    await userEvent.setup().click(screen.getByRole("button", { name: /Human review/ }));
    expect(screen.getByRole("status")).toHaveTextContent(humanReview.blockingReason);
  });

  it("keeps retained evidence available when the review run is absent from the activity log", async () => {
    const pullRequest = {
      ...GITHUB_PR,
      activities: [{ run: { id: "assessment-run", canvasId: "assessment-app" } }],
      humanReview: { runId: "deleted-review-run", status: "decided", rationale: "Retained review evidence." },
    };
    renderNote({ pullRequests: [pullRequest] });

    await userEvent.setup().click(screen.getByRole("button", { name: /Human review/ }));
    expect(screen.getByText("Retained review evidence.")).toBeVisible();
  });

  it("keeps the waiting state visible when Actions excludes the previous revision review", () => {
    const pullRequest = {
      ...GITHUB_PR,
      currentRevision: { sha: "current" },
      activities: [{ run: { id: "review-run", canvasId: "review-app" } }],
      humanReview: { runId: "review-run", status: "decided", assessedCommit: "previous" },
    };
    renderNote({ pullRequests: [pullRequest] });

    expect(screen.getByRole("region", { name: "Human review" })).toBeVisible();
    expect(screen.getByRole("status")).toHaveTextContent("Waiting for merge confidence");
  });

  it("says the pull request is ready without a numbered guide", () => {
    renderNote();

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
    expect(within(note).queryByRole("list")).not.toBeInTheDocument();
    expect(note).not.toHaveTextContent("Waiting for user review");
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
  });

  it("makes the pull request link the one large call to action", () => {
    renderNote();

    const cta = screen.getByTestId("split-run-pull-request-cta");
    expect(cta).toHaveAccessibleName("Review PR #6812");
    expect(cta).toHaveAttribute("href", "https://github.com/acme/payments/pull/6812");
    expect(cta).toHaveAttribute("target", "_blank");
    expect(cta).toHaveAttribute("rel", "noreferrer");

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getAllByRole("link")).toHaveLength(1);
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();
  });

  it("shows the close actions in an actions-only More menu", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    renderNote({ actionsOnly: true, onAction });

    await user.click(screen.getByRole("button", { name: "More actions" }));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu)
        .getAllByRole("menuitem")
        .map((item) => item.textContent),
    ).toEqual(["Reject", "Approve"]);

    await user.click(within(menu).getByRole("menuitem", { name: "Approve" }));
    expect(onAction).toHaveBeenCalledWith(ACTIONS[1]);
  });

  it("hides the More menu when there are no actions", () => {
    renderNote({ actions: [], actionsOnly: true });

    expect(screen.queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();
  });

  it("disables the More menu while an action is in flight", () => {
    renderNote({ actionBusy: true, actionsOnly: true });

    expect(screen.getByRole("button", { name: "More actions" })).toBeDisabled();
  });

  it("keeps the standard note for a link that is not a pull request", () => {
    renderNote({
      note: {
        headline: "Implement did not pass",
        text: "Fix the error.",
        cta: { label: "View run details", href: "/runs/1" },
      },
      tone: "failed",
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).not.toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "Implement did not pass" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });

  it("keeps no box of its own in the stacked panel section", () => {
    renderNote({ compact: true, stacked: true });

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(note.className || "").not.toContain("rounded-lg");
    expect(note.className || "").not.toContain("border");
  });

  it("keeps repository merge confidence available in the compact summary", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <PullRequestChecksContext.Provider
            value={
              new Map([
                ["acme/payments", [{ id: "security", name: "Security", score: 60, maxScore: 100, level: "caution" }]],
              ])
            }
          >
            <SplitRunPullRequestReviewNote
              ctaLabel={PR_NOTE.cta!.label}
              pullRequest={{ href: GITHUB_PR.url!, number: 6812 }}
              trackedPullRequest={{
                ...GITHUB_PR,
                repository: "acme/payments",
                humanReview: { status: "decided", reviewRequired: true, rationale: "Review the payment changes." },
              }}
              compact
            />
          </PullRequestChecksContext.Provider>
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByRole("button", { name: "Merge confidence" })).toBeVisible();
    expect(screen.getByText("Security")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Merge confidence" }));
    expect(screen.queryByText("Security")).not.toBeInTheDocument();
  });

  it("says the pull request waits for checks while checks run", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
      message: "Checks are still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "The pull request waits for checks" })).toBeInTheDocument();
    expect(note).toHaveTextContent("You can review it now.");
  });

  it("reports unavailable merge status and keeps merge disabled", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_UNAVAILABLE",
      message: "Merge status is unavailable right now.",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "Merge status is unavailable right now." })).toBeInTheDocument();
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.getByTestId("split-run-merge-method")).toBeDisabled();
  });

  it("says a check failed when a pull request check fails", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECK_FAILED",
      message: "A check failed.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR], compact: true });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "A pull request check failed" })).toBeInTheDocument();
    expect(note).toHaveTextContent("Review the failed check on the pull request.");
  });

  it("keeps the ready headline when the merge flag is off", () => {
    experimentalFeatureHas.current = () => false;
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECK_FAILED",
      message: "A check failed.",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
  });

  it("enables merge when mergeability is true", () => {
    renderNote({ pullRequests: [GITHUB_PR] });

    const button = screen.getByTestId("split-run-merge-button");
    expect(button).toBeEnabled();
    expect(button).toHaveAccessibleName("Merge");
    expect(screen.queryByTestId("split-run-merge-reason")).not.toBeInTheDocument();
  });

  it("shows Merging and blocks another merge while the request is pending", () => {
    mergePending.current = true;
    renderNote({ pullRequests: [GITHUB_PR] });

    const button = screen.getByTestId("split-run-merge-button");
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleName("Merging");
  });

  it("disables merge and shows the reason on hover when automation is running", async () => {
    const user = userEvent.setup();
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_ACTIVE_RUN",
      message: "Automation is still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(screen.getByTestId("split-run-merge-reason"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Automation is still running.");
  });

  it("disables merge and shows the reason on hover when checks are still running", async () => {
    const user = userEvent.setup();
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
      message: "Checks are still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR], compact: true });

    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(screen.getByTestId("split-run-merge-reason"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Checks are still running.");
  });

  it("lists only allowed merge methods", async () => {
    const user = userEvent.setup();
    renderNote({ pullRequests: [GITHUB_PR] });

    await user.click(screen.getByTestId("split-run-merge-method"));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu)
        .getAllByRole("menuitem")
        .map((item) => item.textContent),
    ).toEqual(["✓ Squash and merge", "Create a merge commit"]);
  });

  it("merges with the method the person picks in the menu", async () => {
    const user = userEvent.setup();
    renderNote({ pullRequests: [GITHUB_PR] });

    await user.click(screen.getByTestId("split-run-merge-method"));
    await user.click(await screen.findByTestId("split-run-merge-method-MERGE_METHOD_MERGE"));

    expect(mergeMutate).toHaveBeenCalledTimes(1);
    expect(mergeMutate.mock.calls[0]?.[0]).toEqual({
      pullRequestId: "pr-6812",
      mergeMethod: "MERGE_METHOD_MERGE",
      expectedHeadSha: "abc123",
    });
  });

  it("does not offer merge methods while merge is blocked", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECK_FAILED",
      message: "A check failed.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    expect(screen.getByTestId("split-run-merge-method")).toBeDisabled();
  });

  it("sends the selected method once on click", async () => {
    const user = userEvent.setup();
    renderNote({ pullRequests: [GITHUB_PR] });

    await user.click(screen.getByTestId("split-run-merge-button"));
    expect(mergeMutate).toHaveBeenCalledTimes(1);
    expect(mergeMutate.mock.calls[0]?.[0]).toEqual({
      pullRequestId: "pr-6812",
      mergeMethod: "MERGE_METHOD_SQUASH",
      expectedHeadSha: "abc123",
    });
  });

  it("hides merge for a merged pull request", () => {
    renderNote({
      pullRequests: [{ ...GITHUB_PR, state: "STATE_MERGED" }],
    });

    expect(screen.queryByTestId("split-run-merge-button")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-pr-merged")).toHaveTextContent("The pull request is merged.");
  });

  it("hides merge when the pull request merge flag is off", () => {
    experimentalFeatureHas.current = () => false;
    renderNote({ pullRequests: [GITHUB_PR] });

    expect(screen.queryByTestId("split-run-merge-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-merge-method")).not.toBeInTheDocument();
  });

  it("shows merged confirmation even when the flag is off", () => {
    experimentalFeatureHas.current = () => false;
    renderNote({
      pullRequests: [{ ...GITHUB_PR, state: "STATE_MERGED" }],
    });

    expect(screen.getByTestId("split-run-pr-merged")).toHaveTextContent("The pull request is merged.");
    expect(screen.queryByTestId("split-run-merge-button")).not.toBeInTheDocument();
  });

  it("shows the merge control for a Bitbucket pull request", () => {
    renderNote({ note: BITBUCKET_PR_NOTE, pullRequests: [BITBUCKET_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-merge-button")).toBeEnabled();
  });

  it("disables Bitbucket merge and explains the blocker", async () => {
    const user = userEvent.setup();
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CONFLICTING",
      message: "The pull request has conflicts.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ note: BITBUCKET_PR_NOTE, pullRequests: [BITBUCKET_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "The pull request has conflicts" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();

    await user.hover(screen.getByTestId("split-run-merge-reason"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("The pull request has conflicts.");
  });

  it("offers only the branch merge strategies for Bitbucket, never rebase", async () => {
    const user = userEvent.setup();
    renderNote({ note: BITBUCKET_PR_NOTE, pullRequests: [BITBUCKET_PR] });

    await user.click(screen.getByTestId("split-run-merge-method"));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu)
        .getAllByRole("menuitem")
        .map((item) => item.textContent),
    ).toEqual(["✓ Squash and merge", "Create a merge commit"]);
  });

  it("lists each open pull request with its repository when a task has several", async () => {
    const user = userEvent.setup();
    const docsPullRequest: FactoriesFactoryPullRequest = {
      id: "pr-5",
      provider: "PROVIDER_GITHUB",
      repository: "acme/docs",
      url: "https://github.com/acme/docs/pull/5",
      number: "5",
      title: "Add AGENTS.md",
      state: "STATE_OPEN",
    };
    renderNote({
      compact: true,
      pullRequests: [{ ...GITHUB_PR, repository: "acme/payments", title: "Refine payments" }, docsPullRequest],
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "2 pull requests are ready for review" })).toBeInTheDocument();
    expect(note).toHaveTextContent("This task closes when all pull requests are merged or closed.");
    const rows = within(note).getAllByTestId("split-run-review-pull-request");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]!).getByRole("link", { name: "Review acme/payments #6812" })).toHaveAttribute(
      "href",
      "https://github.com/acme/payments/pull/6812",
    );
    expect(within(rows[1]!).getByRole("link", { name: "Review acme/docs #5" })).toHaveAttribute(
      "href",
      "https://github.com/acme/docs/pull/5",
    );

    await user.click(within(rows[1]!).getByTestId("split-run-merge-button"));
    expect(mergeMutate.mock.calls[0]?.[0]).toMatchObject({ pullRequestId: "pr-5" });
  });

  it("keeps the single review action when only one tracked pull request is open", () => {
    renderNote({
      pullRequests: [
        GITHUB_PR,
        { ...GITHUB_PR, id: "pr-old", url: "https://github.com/acme/docs/pull/1", state: "STATE_MERGED" },
      ],
    });

    expect(screen.getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-pull-request-cta")).toHaveAccessibleName("Review PR #6812");
    expect(screen.queryByTestId("split-run-review-pull-request")).not.toBeInTheDocument();
  });

  it("merges a Bitbucket pull request with the selected method", async () => {
    const user = userEvent.setup();
    renderNote({ note: BITBUCKET_PR_NOTE, pullRequests: [BITBUCKET_PR] });

    await user.click(screen.getByTestId("split-run-merge-button"));
    expect(mergeMutate).toHaveBeenCalledTimes(1);
    expect(mergeMutate.mock.calls[0]?.[0]).toEqual({
      pullRequestId: "pr-7",
      mergeMethod: "MERGE_METHOD_SQUASH",
      expectedHeadSha: "abc123",
    });
  });
});
