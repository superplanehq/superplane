import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryPullRequest, FactoriesFactoryPullRequestMergeability } from "@/api-client";
import { TooltipProvider } from "@/components/ui/tooltip";

const mergeability = { current: undefined as FactoriesFactoryPullRequestMergeability | undefined };
const mergeMutate = vi.fn();

vi.mock("@/hooks/useFactoryPullRequestMerge", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
  useMergeFactoryPullRequest: () => ({ mutate: mergeMutate, isPending: false }),
}));

import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
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

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

beforeEach(() => {
  mergeMutate.mockReset();
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
      note: { headline: "Implement did not pass", text: "Fix the error.", cta: { label: "Debug", href: "/runs/1" } },
      tone: "failed",
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).not.toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "Implement did not pass" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });

  it("enables merge when mergeability is true", () => {
    renderNote({ pullRequests: [GITHUB_PR] });

    expect(screen.getByTestId("split-run-merge-button")).toBeEnabled();
    expect(screen.queryByTestId("split-run-merge-reason")).not.toBeInTheDocument();
  });

  it("disables merge and shows the reason when automation is running", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_ACTIVE_RUN",
      message: "Automation is still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.getByTestId("split-run-merge-reason")).toHaveTextContent("Automation is still running.");
  });

  it("disables merge and shows the reason when checks are still running", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
      message: "Checks are still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderNote({ pullRequests: [GITHUB_PR], compact: true });

    expect(screen.getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.getByTestId("split-run-merge-reason")).toHaveTextContent("Checks are still running.");
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
});
