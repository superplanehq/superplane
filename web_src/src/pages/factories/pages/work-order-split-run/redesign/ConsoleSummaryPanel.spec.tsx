import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, mock } from "bun:test";
import type { ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderCheck } from "@/api-client";
import { FEATURE_FACTORY_MULTI_REPO_TASKS } from "@/lib/experimentalFeatures";

import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";
import { outcomeSummary } from "./automationsViewModel";
import { ConsoleSummaryPanel } from "./ConsoleSummaryPanel";

const multiRepoTasks = { enabled: true };

mock.module("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => id === FEATURE_FACTORY_MULTI_REPO_TASKS && multiRepoTasks.enabled,
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

beforeEach(() => {
  multiRepoTasks.enabled = true;
});

const pullRequest: FactoriesFactoryPullRequest = {
  id: "pr-482",
  workOrderId: OPEN_WORK_ORDER.id,
  number: "482",
  url: "https://github.com/acme/payments/pull/482",
  title: "Fix duplicate refund on retry",
  state: "STATE_OPEN",
  mergeable: true,
};

function check(key: string, score?: number): FactoriesWorkOrderCheck {
  return { id: key, key, name: key, score, maxScore: 5 };
}

function renderPanel(checks: FactoriesWorkOrderCheck[], pr = pullRequest) {
  const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks, demoArtifacts: false });
  render(<ConsoleSummaryPanel fixture={fixture} outcome={outcomeSummary(fixture)} stages={[]} pullRequests={[pr]} />);
}

describe("ConsoleSummaryPanel mergeable chip", () => {
  it("hides Mergeable when a scored merge confidence metric is critical", () => {
    renderPanel([check("risk-review", 4), check("security-review", 5)]);

    expect(screen.getByRole("link", { name: "Review pull request #482." })).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-mergeable-chip")).not.toBeInTheDocument();
  });

  it("shows Mergeable when scores are caution and other checks do not count", () => {
    renderPanel([check("risk-review", 3), check("performance-review", 3), check("planning-review", 1)]);

    expect(screen.getByTestId("work-order-mergeable-chip")).toHaveTextContent("Mergeable");
  });

  it("shows Mergeable when a higher-is-better metric has no score", () => {
    renderPanel([check("security-review"), check("performance-review"), check("reversibility-review")]);

    expect(screen.getByTestId("work-order-mergeable-chip")).toHaveTextContent("Mergeable");
  });
});

describe("ConsoleSummaryPanel pull requests in several repositories", () => {
  const paymentsPullRequest = { ...pullRequest, repository: "acme/payments" };
  const docsPullRequest: FactoriesFactoryPullRequest = {
    id: "pr-5",
    workOrderId: OPEN_WORK_ORDER.id,
    repository: "acme/docs",
    number: "5",
    url: "https://github.com/acme/docs/pull/5",
    title: "Add AGENTS.md",
    state: "STATE_OPEN",
  };

  function renderTwoRepositoryPanel(panelReview?: ReactNode) {
    const pullRequests = [paymentsPullRequest, docsPullRequest];
    const fixture = splitRunFixtureForWorkOrder({ ...OPEN_WORK_ORDER, pullRequests }, { demoArtifacts: false });
    render(
      <ConsoleSummaryPanel
        fixture={fixture}
        outcome={outcomeSummary(fixture)}
        stages={[]}
        pullRequests={pullRequests}
        panelReview={panelReview}
      />,
    );
  }

  it("does not list again the pull requests that the review strip shows", () => {
    renderTwoRepositoryPanel(<div>Review strip</div>);

    expect(screen.queryByTestId("redesign-console-pull-requests")).not.toBeInTheDocument();
  });

  it("hides Mergeable only on the pull request whose repository has a critical check", () => {
    const pullRequests = [
      { ...paymentsPullRequest, mergeable: true },
      { ...docsPullRequest, mergeable: true },
    ];
    const checks = [
      { ...check("risk-review", 4), id: "payments-risk", repository: "acme/payments" },
      { ...check("risk-review", 1), id: "docs-risk", repository: "acme/docs" },
    ];
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks, demoArtifacts: false });
    render(
      <ConsoleSummaryPanel
        fixture={fixture}
        outcome={outcomeSummary(fixture)}
        stages={[]}
        pullRequests={pullRequests}
      />,
    );

    const chips = screen.getAllByTestId("work-order-mergeable-chip");
    expect(chips).toHaveLength(1);
    expect(screen.getByText("acme/docs #5").parentElement).toContainElement(chips[0]!);
  });

  it("names the repository of each listed pull request", () => {
    renderTwoRepositoryPanel();

    const section = within(screen.getByTestId("redesign-console-pull-requests"));
    expect(section.getByText("acme/payments #482")).toBeInTheDocument();
    expect(section.getByText("acme/docs #5")).toBeInTheDocument();
  });
});

describe("ConsoleSummaryPanel merge confidence per pull request", () => {
  const paymentsPullRequest = { ...pullRequest, repository: "acme/payments" };
  const docsPullRequest: FactoriesFactoryPullRequest = {
    id: "pr-5",
    workOrderId: OPEN_WORK_ORDER.id,
    repository: "acme/docs",
    number: "5",
    url: "https://github.com/acme/docs/pull/5",
    title: "Add AGENTS.md",
    state: "STATE_OPEN",
  };

  function renderWith(checks: FactoriesWorkOrderCheck[], pullRequests: FactoriesFactoryPullRequest[]) {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks, demoArtifacts: false });
    render(
      <ConsoleSummaryPanel
        fixture={fixture}
        outcome={outcomeSummary(fixture)}
        stages={[]}
        pullRequests={pullRequests}
      />,
    );
  }

  function confidenceFor(repository: string) {
    const row = screen
      .getAllByTestId("redesign-console-pull-request-checks")
      .find((element) => element.dataset.repository === repository);
    if (!row) throw new Error(`No merge confidence for ${repository}`);
    return row;
  }

  it("shows the checks of each repository under its pull request", () => {
    renderWith(
      [
        { ...check("risk-review", 4), id: "payments-risk", repository: "acme/payments", level: "LEVEL_CRITICAL" },
        { ...check("risk-review", 1), id: "docs-risk", repository: "acme/docs", level: "LEVEL_POSITIVE" },
      ],
      [paymentsPullRequest, docsPullRequest],
    );

    expect(screen.queryByTestId("redesign-console-checks")).not.toBeInTheDocument();
    const payments = confidenceFor("acme/payments");
    const docs = confidenceFor("acme/docs");
    // The critical check opens its pull request. The passing one stays one line.
    expect(within(payments).getByRole("button", { name: "Merge confidence" })).toHaveAttribute("aria-expanded", "true");
    expect(within(payments).getByTestId("split-run-check-payments-risk")).toBeInTheDocument();
    expect(within(payments).queryByTestId("split-run-check-docs-risk")).not.toBeInTheDocument();
    expect(within(docs).getByRole("button", { name: "Merge confidence" })).toHaveAttribute("aria-expanded", "false");
  });

  it("keeps current assessments separate when repositories share an automation", () => {
    const automation = { appId: "merge-app" };
    renderWith(
      [
        {
          ...check("risk-review", 1),
          id: "payments-risk",
          repository: "acme/payments",
          level: "LEVEL_POSITIVE",
          runId: "payments-run",
          automation,
        },
        {
          ...check("security-review", 5),
          id: "payments-security",
          repository: "acme/payments",
          level: "LEVEL_POSITIVE",
          runId: "old-run",
          automation,
        },
        {
          ...check("security-review", 5),
          id: "docs-security",
          repository: "acme/docs",
          level: "LEVEL_POSITIVE",
          runId: "docs-run",
          automation,
        },
      ],
      [
        {
          ...paymentsPullRequest,
          activities: [
            {
              run: {
                id: "payments-run",
                canvasId: "merge-app",
                result: "RESULT_FAILED",
                createdAt: "2026-11-06T14:24:00Z",
              },
            },
          ],
        },
        {
          ...docsPullRequest,
          activities: [
            {
              run: {
                id: "docs-run",
                canvasId: "merge-app",
                result: "RESULT_PASSED",
                createdAt: "2026-11-06T14:25:00Z",
              },
            },
          ],
        },
      ],
    );
    const payments = confidenceFor("acme/payments");
    const docs = confidenceFor("acme/docs");
    expect(within(payments).getByText(/Assessment failed/)).toBeInTheDocument();
    expect(within(docs).getByText(/All checks passed/)).toBeInTheDocument();
    fireEvent.click(within(payments).getByRole("button", { name: "Merge confidence" }));
    expect(
      within(screen.getByTestId("split-run-check-payments-risk")).getByRole("img", { name: "Passed" }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-check-payments-security")).getByRole("img", { name: "Not assessed" }),
    ).toBeInTheDocument();
  });

  it("keeps a check without a repository in the merge confidence list when several repositories have pull requests", () => {
    renderWith(
      [
        { ...check("risk-review", 1), id: "payments-risk", repository: "acme/payments" },
        { ...check("drift-review", 1), id: "task-drift" },
      ],
      [paymentsPullRequest, docsPullRequest],
    );

    const list = screen.getByTestId("redesign-console-checks");
    fireEvent.click(within(list).getByRole("button", { name: "Merge confidence" }));
    expect(within(list).getByTestId("split-run-check-task-drift")).toBeInTheDocument();
    expect(within(list).queryByTestId("split-run-check-payments-risk")).not.toBeInTheDocument();
  });

  it("puts the checks of the only pull request under it", () => {
    renderWith([check("risk-review", 1), check("security-review", 5)], [pullRequest]);

    expect(screen.queryByTestId("redesign-console-checks")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("redesign-console-pull-request-checks")).toHaveLength(1);
  });

  it("keeps the merge confidence list apart without multi-repo tasks", () => {
    multiRepoTasks.enabled = false;
    renderWith([check("risk-review", 1), check("security-review", 5)], [pullRequest]);

    expect(screen.getByTestId("redesign-console-checks")).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-pull-request-checks")).not.toBeInTheDocument();
  });
});

describe("ConsoleSummaryPanel merge confidence", () => {
  beforeEach(() => {
    multiRepoTasks.enabled = false;
  });
  it("does not reuse scores from an older run of the same revision", async () => {
    const user = userEvent.setup();
    renderPanel(
      [
        {
          ...check("risk-review", 1),
          level: "LEVEL_POSITIVE",
          runId: "current-run",
          automation: { appId: "merge-app" },
        },
        {
          ...check("security-review", 5),
          level: "LEVEL_POSITIVE",
          runId: "old-run",
          automation: { appId: "merge-app" },
        },
      ],
      {
        ...pullRequest,
        currentRevision: { id: "current-revision" },
        activities: [
          {
            revision: { id: "current-revision" },
            run: { id: "old-run", canvasId: "merge-app", createdAt: "2026-11-06T14:22:00Z", result: "RESULT_PASSED" },
          },
          {
            revision: { id: "current-revision" },
            run: {
              id: "current-run",
              canvasId: "merge-app",
              createdAt: "2026-11-06T14:23:00Z",
              result: "RESULT_FAILED",
            },
          },
        ],
      },
    );

    expect(screen.queryByText("All checks passed")).not.toBeInTheDocument();
    expect(screen.getByText("Assessment failed")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Merge confidence" }));
    expect(
      within(screen.getByTestId("split-run-check-security-review")).getByRole("img", { name: "Not assessed" }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-check-risk-review")).getByRole("img", { name: "Passed" }),
    ).toBeInTheDocument();
  });

  it("does not reuse an assessment after the pull request head changes", async () => {
    const user = userEvent.setup();
    renderPanel(
      [
        {
          ...check("security-review", 5),
          level: "LEVEL_POSITIVE",
          runId: "old-run",
          automation: { appId: "merge-app" },
        },
      ],
      {
        ...pullRequest,
        currentRevision: { id: "new-revision" },
        activities: [
          { revision: { id: "old-revision" }, run: { id: "old-run", canvasId: "merge-app", result: "RESULT_PASSED" } },
        ],
      },
    );

    expect(screen.getByText("Checks not fully assessed")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Merge confidence" }));
    expect(
      within(screen.getByTestId("split-run-check-security-review")).getByRole("img", { name: "Not assessed" }),
    ).toBeInTheDocument();
  });

  it("shows assessment failure when the first run reports only favorable scores", () => {
    renderPanel(
      [
        {
          ...check("risk-review", 1),
          level: "LEVEL_POSITIVE",
          runId: "current-run",
          automation: { appId: "merge-app" },
        },
      ],
      { ...pullRequest, activities: [{ run: { id: "current-run", canvasId: "merge-app", result: "RESULT_FAILED" } }] },
    );

    expect(screen.getByText("Assessment failed")).toBeInTheDocument();
    expect(screen.queryByText("All checks passed")).not.toBeInTheDocument();
  });
});
