import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderLineDispatch } from "@/api-client";

import { WorkOrderSidebarFactoryLines } from "./WorkOrderSidebarFactoryLines";

const LINE_DISPATCHES: FactoriesWorkOrderLineDispatch[] = [
  {
    id: "dispatch-1",
    line: { id: "line-1", name: "Refund Line" },
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2024-01-01T00:00:00.000Z",
  },
];

function renderSidebar() {
  return render(
    <MemoryRouter>
      <WorkOrderSidebarFactoryLines
        organizationId="org"
        factoryKey="factory"
        lineDispatches={LINE_DISPATCHES}
        factoryLines={[]}
        canDispatch={false}
        permissionsLoading={false}
        isDispatchable={false}
        isDispatching={false}
        onDispatch={async () => {}}
      />
    </MemoryRouter>,
  );
}

describe("WorkOrderSidebarFactoryLines", () => {
  it("links a factory line row to the read-only line view page, not the editor", () => {
    renderSidebar();

    const link = screen.getByRole("link", { name: "View Refund Line" });
    expect(link).toHaveAttribute("href", "/org/workspaces/factory/lines/line-1");
  });

  it("does not render a link for a run without a resolved line id", () => {
    render(
      <MemoryRouter>
        <WorkOrderSidebarFactoryLines
          organizationId="org"
          factoryKey="factory"
          lineDispatches={[
            {
              id: "dispatch-2",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2024-01-01T00:00:00.000Z",
            },
          ]}
          factoryLines={[]}
          canDispatch={false}
          permissionsLoading={false}
          isDispatchable={false}
          isDispatching={false}
          onDispatch={async () => {}}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("Unnamed line")).toBeInTheDocument();
  });

  it("shows two rows when the same line has two separate dispatches, tone from the most recent", () => {
    render(
      <MemoryRouter>
        <WorkOrderSidebarFactoryLines
          organizationId="org"
          factoryKey="factory"
          lineDispatches={[
            {
              id: "dispatch-old",
              line: { id: "line-1", name: "Refund Line" },
              state: "STATE_FINISHED",
              result: "RESULT_FAILED",
              createdAt: "2024-01-01T00:00:00.000Z",
            },
            {
              id: "dispatch-new",
              line: { id: "line-1", name: "Refund Line" },
              state: "STATE_ACTIVE",
              result: "RESULT_UNKNOWN",
              createdAt: "2024-01-02T00:00:00.000Z",
            },
          ]}
          factoryLines={[]}
          canDispatch={false}
          permissionsLoading={false}
          isDispatchable={false}
          isDispatching={false}
          onDispatch={async () => {}}
        />
      </MemoryRouter>,
    );

    // The sidebar collapses to one row per line, summarizing the latest dispatch.
    expect(screen.getAllByRole("link", { name: "View Refund Line" })).toHaveLength(1);
  });
});
