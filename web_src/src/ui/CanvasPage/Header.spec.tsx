import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
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

vi.mock("./components/CanvasRunsSidebarTrigger", () => ({
  CanvasRunsSidebarTrigger: () => null,
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

const runsSidebarState = {
  isRunsSidebarOpen: true,
  showRunsSidebarToggle: true,
  handleRunsSidebarToggle: vi.fn(),
  openRunsSidebar: vi.fn(),
  closeRunsSidebar: vi.fn(),
} satisfies CanvasRunsSidebarState;

const versionsSidebarState = {
  isVersionsSidebarOpen: false,
  showVersionsSidebarToggle: true,
  handleVersionsSidebarToggle: vi.fn(),
  openVersionsSidebar: vi.fn(),
  closeVersionsSidebar: vi.fn(),
} satisfies CanvasVersionsSidebarState;

function renderHeader(
  mode: "version-live",
  options?: {
    isEditing?: boolean;
    isEditSessionActive?: boolean;
    onExitEditMode?: () => void;
    canvasName?: string;
  },
) {
  render(
    <MemoryRouter initialEntries={["/org/org-1/app/canvas-1"]}>
      <Routes>
        <Route
          path="/org/:organizationId/app/:appId"
          element={
            <Header
              canvasName={options && "canvasName" in options ? options.canvasName : "Test Canvas"}
              mode={mode}
              isEditing={options?.isEditing}
              isEditSessionActive={options?.isEditSessionActive ?? options?.isEditing}
              onEnterEditMode={vi.fn()}
              onExitEditMode={options?.onExitEditMode}
              toolSidebarState={toolSidebarState}
              runsSidebarState={runsSidebarState}
              versionsSidebarState={versionsSidebarState}
            />
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("Header", () => {
  it("shows enter edit actions on the live canvas tab", () => {
    renderHeader("version-live");

    expect(screen.getByTestId("canvas-edit-button")).toBeInTheDocument();
  });

  it("renders without crashing when canvasName is undefined", () => {
    expect(() => renderHeader("version-live", { canvasName: undefined })).not.toThrow();
  });

  it("shows the exit control in edit mode", () => {
    renderHeader("version-live", {
      isEditing: true,
      onExitEditMode: vi.fn(),
    });

    expect(screen.getByTestId("canvas-exit-edit-button")).toHaveAttribute("aria-label", "Exit edit");
    expect(screen.queryByTestId("canvas-edit-button")).not.toBeInTheDocument();
  });
});
