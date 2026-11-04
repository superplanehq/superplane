import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import type { ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderCheck } from "@/api-client";

import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";
import { outcomeSummary } from "./automationsViewModel";
import { ConsoleSummaryPanel } from "./ConsoleSummaryPanel";

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

function renderPanel(checks: FactoriesWorkOrderCheck[]) {
  const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks, demoArtifacts: false });
  render(
    <ConsoleSummaryPanel
      fixture={fixture}
      outcome={outcomeSummary(fixture)}
      stages={[]}
      pullRequests={[pullRequest]}
    />,
  );
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
