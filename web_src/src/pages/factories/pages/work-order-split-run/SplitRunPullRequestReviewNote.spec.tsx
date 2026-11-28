import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryPullRequest, FactoriesFactoryPullRequestMergeability } from "@/api-client";
import { TooltipProvider } from "@/components/ui/tooltip";

const mergeability = { current: undefined as FactoriesFactoryPullRequestMergeability | undefined };
const experimentalFeatureHas = { current: (_id: string) => true };

vi.mock("@/hooks/useFactoryPullRequestMergeability", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => experimentalFeatureHas.current(id),
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { SplitRunPullRequestStatus } from "./SplitRunPullRequestStatus";
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

  it("reports unavailable merge status and keeps the review link", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_UNAVAILABLE",
      message: "Merge status is unavailable right now.",
    };
    renderNote({ pullRequests: [GITHUB_PR] });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "Merge status is unavailable right now." })).toBeInTheDocument();
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
    expect(screen.getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
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

  it("keeps the ready headline when the status flag is off", () => {
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

  for (const pullRequest of [GITHUB_PR, BITBUCKET_PR]) {
    for (const presentation of [{}, { compact: true }, { ctaOnly: true }]) {
      it(`links to ${pullRequest.provider} without merge controls in ${JSON.stringify(presentation)}`, () => {
        const note = pullRequest.provider === "PROVIDER_BITBUCKET" ? BITBUCKET_PR_NOTE : PR_NOTE;
        renderNote({ note, pullRequests: [pullRequest], ...presentation });

        expect(screen.getByRole("link", { name: note.cta!.label })).toHaveAttribute("href", pullRequest.url);
        expect(screen.queryByRole("button", { name: "Merge" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Merge method" })).not.toBeInTheDocument();
      });
    }
  }

  for (const flagEnabled of [true, false]) {
    for (const pullRequest of [GITHUB_PR, BITBUCKET_PR]) {
      it(`keeps ${pullRequest.provider} merged confirmation with status flag ${flagEnabled}`, () => {
        experimentalFeatureHas.current = () => flagEnabled;
        mergeability.current = { canMerge: false, message: "Checks are still running." };
        const note = pullRequest.provider === "PROVIDER_BITBUCKET" ? BITBUCKET_PR_NOTE : PR_NOTE;
        renderNote({ note, pullRequests: [{ ...pullRequest, state: "STATE_MERGED" }] });

        expect(screen.getByTestId("split-run-pr-merged").textContent).toBe("The pull request is merged.");
        expect(screen.queryByTestId("split-run-pr-status")).not.toBeInTheDocument();
      });
    }

    it(`keeps multiple-PR blockers gated by status flag ${flagEnabled}`, () => {
      experimentalFeatureHas.current = () => flagEnabled;
      mergeability.current = {
        canMerge: false,
        blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
        message: "Checks are still running.",
      };
      renderNote({ compact: true, pullRequests: [GITHUB_PR, BITBUCKET_PR] });

      expect(screen.getAllByTestId("split-run-pull-request-cta")).toHaveLength(2);
      expect(screen.queryAllByTestId("split-run-pr-status")).toHaveLength(flagEnabled ? 2 : 0);
      for (const status of screen.queryAllByTestId("split-run-pr-status")) {
        expect(status.textContent).toBe("Checks are still running.");
      }
    });
  }

  it("reports Bitbucket conflicts while keeping the provider review link", () => {
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_CONFLICTING",
      message: "The pull request has conflicts.",
    };
    renderNote({ note: BITBUCKET_PR_NOTE, pullRequests: [BITBUCKET_PR] });

    expect(screen.getByRole("heading", { name: "The pull request has conflicts" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Review PR #7" })).toHaveAttribute("href", BITBUCKET_PR.url);
  });

  it("lists each provider review link without merge controls when a task has several", () => {
    const docsPullRequest: FactoriesFactoryPullRequest = {
      id: "pr-5",
      provider: "PROVIDER_BITBUCKET",
      repository: "acme/docs",
      url: "https://bitbucket.org/acme/docs/pull-requests/5",
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
      "https://bitbucket.org/acme/docs/pull-requests/5",
    );

    expect(within(note).queryByRole("button", { name: "Merge" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Merge method" })).not.toBeInTheDocument();
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
});

describe("SplitRunPullRequestStatus message selection", () => {
  const cases: {
    name: string;
    data: FactoriesFactoryPullRequestMergeability | undefined;
    expectedMessage?: string;
  }[] = [
    { name: "hides a server message when mergeable", data: { canMerge: true, message: "Ready to merge." } },
    { name: "hides status with missing data", data: undefined },
    { name: "hides status with a missing message", data: { canMerge: false } },
    { name: "hides status with an empty message", data: { canMerge: false, message: "" } },
    {
      name: "shows a server message when canMerge is omitted",
      data: { message: "Merge status is unavailable right now." },
      expectedMessage: "Merge status is unavailable right now.",
    },
  ];

  for (const { name, data, expectedMessage } of cases) {
    it(name, () => {
      mergeability.current = data;
      render(<SplitRunPullRequestStatus organizationId="org-1" factoryId="factory-1" pullRequest={GITHUB_PR} />);

      if (expectedMessage) {
        expect(screen.getByTestId("split-run-pr-status").textContent).toBe(expectedMessage);
        return;
      }
      expect(screen.queryByTestId("split-run-pr-status")).not.toBeInTheDocument();
    });
  }
});
