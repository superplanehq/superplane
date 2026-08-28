import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrder, FactoriesWorkOrderEvent } from "@/api-client";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [] }),
}));

import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";

const ORDER: FactoriesWorkOrder = {
  id: "wo-1",
  title: "Refund retry",
  lineDispatches: [],
};

const PULL_REQUEST_ADDED_EVENT: FactoriesWorkOrderEvent = {
  type: "order.pull_request.added",
  timestamp: "2026-08-01T12:00:00.000Z",
  event: {
    order: { id: ORDER.id, title: ORDER.title },
    pullRequest: {
      id: "pr-1",
      url: "https://github.com/example/repo/pull/42",
      title: "Draft implementation",
      number: 42,
      state: "open",
    },
  },
};

function renderTimeline(pullRequests?: FactoriesFactoryPullRequest[]) {
  return render(
    <MemoryRouter>
      <WorkOrderActivityTimeline
        organizationId="org-1"
        factoryKey="factory-1"
        order={ORDER}
        events={[PULL_REQUEST_ADDED_EVENT]}
        pullRequests={pullRequests}
      />
    </MemoryRouter>,
  );
}

describe("WorkOrderActivityTimeline pull request chip", () => {
  it("renders the event's own snapshot when no live pull request data is provided", () => {
    renderTimeline();
    const link = screen.getByRole("link");
    expect(link).toHaveTextContent("#42");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });

  it("overlays live pull request data over the stale event-time snapshot", () => {
    renderTimeline([
      {
        id: "pr-1",
        number: "42",
        url: "https://github.com/example/repo/pull/42",
        title: "Draft implementation",
        state: "STATE_MERGED",
      },
    ]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-purple-600");
  });

  it("falls back to the snapshot when the pull request isn't in the current list", () => {
    renderTimeline([
      {
        id: "some-other-pr",
        number: "99",
        url: "https://x",
        state: "STATE_MERGED",
      },
    ]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });
});
