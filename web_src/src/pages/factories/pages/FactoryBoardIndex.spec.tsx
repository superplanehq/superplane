import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { FactoryBoardIndex } from "./FactoryBoardIndex";

const permissionsState = {
  canAct: (_resource: string, _action: string) => true,
  isLoading: false,
};

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => permissionsState,
}));

vi.mock("../mobile/MobileShellSwitch", () => ({
  LineBoardRouteSwitch: () => <span>line-board</span>,
}));

function renderIndex(factory: { lines?: Array<{ id?: string }> } | null) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY"]}>
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: "factory-1",
          factoryKey: "PAY",
          factory: factory as never,
          factories: [],
          openCreateWorkOrder: vi.fn(),
        }}
      >
        <Routes>
          <Route path="/org-1/workspaces/PAY" element={<FactoryBoardIndex />} />
          <Route path="/org-1/workspaces/PAY/lines/:lineId" element={<span>public-board</span>} />
          <Route path="/org-1/workspaces/PAY/overview" element={<span>overview</span>} />
        </Routes>
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("FactoryBoardIndex", () => {
  beforeEach(() => {
    permissionsState.isLoading = false;
    permissionsState.canAct = () => true;
  });

  it("shows the line board at the workspace index", () => {
    renderIndex({ lines: [{ id: "line-plan" }] });
    expect(screen.getByText("line-board")).toBeInTheDocument();
  });

  it("opens the public board at its line URL when the member cannot read work orders", () => {
    permissionsState.canAct = (resource, action) => !(resource === "work_orders" && action === "read");
    renderIndex({ lines: [{ id: "line-plan" }] });
    expect(screen.getByText("public-board")).toBeInTheDocument();
  });

  it("opens overview when the member cannot read work orders and the workspace has no line", () => {
    permissionsState.canAct = (resource, action) => !(resource === "work_orders" && action === "read");
    renderIndex({ lines: [] });
    expect(screen.getByText("overview")).toBeInTheDocument();
  });
});
