import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderExecution } from "@/api-client";

import { WorkOrderSidebarFactoryLines } from "./WorkOrderSidebarFactoryLines";

const EXECUTIONS: FactoriesWorkOrderExecution[] = [
  {
    id: "execution-1",
    line: { id: "line-1", name: "Refund Line" },
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
  },
];

function renderSidebar() {
  return render(
    <MemoryRouter>
      <WorkOrderSidebarFactoryLines
        organizationId="org"
        factoryId="factory"
        executions={EXECUTIONS}
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
          factoryId="factory"
          executions={[{ id: "execution-2", state: "STATE_FINISHED", result: "RESULT_PASSED" }]}
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
});
