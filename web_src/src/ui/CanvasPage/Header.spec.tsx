import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { Header } from "./Header";

vi.mock("@/components/OrganizationMenuButton", () => ({
  OrganizationMenuButton: () => null,
}));

vi.mock("./components/CanvasProjectSwitcher", () => ({
  CanvasProjectSwitcher: () => null,
}));

vi.mock("./components/CanvasToolSidebarTrigger", () => ({
  CanvasToolSidebarTrigger: () => null,
}));

vi.mock("./components/CanvasModeToggle", () => ({
  CanvasModeToggle: () => null,
}));

const toolSidebarState = {
  canvasId: "canvas-1",
  organizationId: "org-1",
  isEditing: false,
  readOnly: false,
  isToolSidebarOpen: true,
  showToolSidebarToggle: true,
  isAgentEnabled: false,
  handleToolSidebarToggle: vi.fn(),
  openToolSidebar: vi.fn(),
  closeToolSidebar: vi.fn(),
  agentMode: "operator" as const,
  switchAgentMode: vi.fn(),
} satisfies CanvasToolSidebarState;

function renderHeader(mode: "runs" | "version-live") {
  render(
    <MemoryRouter initialEntries={["/org/org-1/app/canvas-1"]}>
      <Routes>
        <Route
          path="/org/:organizationId/app/:appId"
          element={
            <Header
              canvasName="Test Canvas"
              mode={mode}
              onEnterEditMode={vi.fn()}
              toolSidebarState={toolSidebarState}
            />
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("Header", () => {
  it("hides enter edit actions in runs mode", () => {
    renderHeader("runs");

    expect(screen.queryByTestId("canvas-edit-button")).not.toBeInTheDocument();
  });

  it("shows enter edit actions outside runs mode", () => {
    renderHeader("version-live");

    expect(screen.getByTestId("canvas-edit-button")).toBeInTheDocument();
  });
});
