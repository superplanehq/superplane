import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { FactoryHomeRedirect } from "./FactoryHomeRedirect";

function renderHome(factory: { lines?: Array<{ id?: string }> } | null) {
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
          <Route path="/org-1/workspaces/PAY" element={<FactoryHomeRedirect />} />
          <Route path="/org-1/workspaces/PAY/lines" element={<span>lines-list</span>} />
          <Route path="/org-1/workspaces/PAY/lines/:lineId" element={<span>line-board</span>} />
        </Routes>
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("FactoryHomeRedirect", () => {
  it("opens the first line board when the workspace has a line", () => {
    renderHome({ lines: [{ id: "line-plan" }, { id: "line-hotfix" }] });
    expect(screen.getByText("line-board")).toBeInTheDocument();
  });

  it("opens the lines list when the workspace has no line", () => {
    renderHome({ lines: [] });
    expect(screen.getByText("lines-list")).toBeInTheDocument();
  });
});
