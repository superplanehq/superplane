import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

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
