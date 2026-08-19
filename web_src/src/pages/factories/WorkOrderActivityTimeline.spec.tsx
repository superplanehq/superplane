import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrder, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [] }),
}));

import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";

const ORDER: FactoriesWorkOrder = {
  id: "wo-1",
  title: "Refund retry",
  lineDispatches: [],
};

const ARTIFACT_ADDED_EVENT: FactoriesWorkOrderEvent = {
  type: "order.artifact.added",
  timestamp: "2026-08-01T12:00:00.000Z",
  event: {
    order: { id: ORDER.id, title: ORDER.title },
    artifact: {
      id: "art-pr-1",
      type: "pr",
      data: {
        url: "https://github.com/example/repo/pull/42",
        title: "Draft implementation",
        state: "open",
      },
    },
  },
};

function renderTimeline(artifacts?: FactoriesWorkOrderArtifact[]) {
  return render(
    <MemoryRouter>
      <WorkOrderActivityTimeline
        organizationId="org-1"
        factoryKey="factory-1"
        order={ORDER}
        events={[ARTIFACT_ADDED_EVENT]}
        artifacts={artifacts}
      />
    </MemoryRouter>,
  );
}

describe("WorkOrderActivityTimeline artifact chip", () => {
  it("renders the event's own snapshot when no live artifact data is provided", () => {
    renderTimeline();
    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });

  it("overlays live artifact data over the stale event-time snapshot", () => {
    renderTimeline([
      {
        id: "art-pr-1",
        type: "TYPE_PR",
        data: {
          url: "https://github.com/example/repo/pull/42",
          title: "Draft implementation",
          state: "merged",
        },
      },
    ]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-purple-600");
  });

  it("falls back to the snapshot when the artifact isn't in the current list", () => {
    renderTimeline([{ id: "some-other-artifact", type: "TYPE_PR", data: { url: "https://x", state: "merged" } }]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });
});
