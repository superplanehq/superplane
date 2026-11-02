import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "bun:test";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { Header } from "./Header";

vi.mock("@/components/OrganizationMenuButton", () => ({
  OrganizationMenuButton: () => null,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => featureId === "factories",
    enabledExperimentalFeatures: ["factories"],
    isLoading: false,
  }),
}));

vi.mock("./components/CanvasProjectSwitcher", () => ({
  CanvasProjectSwitcher: () => null,
}));

vi.mock("./components/CanvasRunsSidebarTrigger", () => ({
  CanvasRunsSidebarTrigger: () => null,
}));

vi.mock("./components/CanvasModeToggle", () => ({
  CanvasModeToggle: () => null,
}));

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
    activeDraftBranchLabel?: string;
    onExitEditMode?: () => void;
    canvasName?: string;
    factoryId?: string;
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
              organizationId="org-1"
              factoryId={options?.factoryId}
              mode={mode}
              isEditing={options?.isEditing}
              activeDraftBranchLabel={options?.activeDraftBranchLabel}
              onEnterEditMode={vi.fn()}
              onExitEditMode={options?.onExitEditMode}
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

  it("shows return to factory link when factoryId is set", () => {
    renderHeader("version-live", { factoryId: "factory-123" });

    const link = screen.getByTestId("return-to-factory-link");
    expect(link).toHaveTextContent("Return to workspace");
    expect(link).toHaveAttribute("href", "/org-1/workspaces/factory-123/overview");
  });

  it("hides return to factory link when factoryId is absent", () => {
    renderHeader("version-live");

    expect(screen.queryByTestId("return-to-factory-link")).not.toBeInTheDocument();
  });

  it("shows the active draft label and exit control in edit mode", () => {
    renderHeader("version-live", {
      isEditing: true,
      activeDraftBranchLabel: "Draft #1",
      onExitEditMode: vi.fn(),
    });

    expect(screen.getByTestId("active-draft-branch-chip")).toHaveTextContent("Editing: Draft #1");
    expect(screen.getByText("Finish Editing")).toBeInTheDocument();
    expect(screen.getByTestId("canvas-exit-edit-button")).toHaveAttribute("aria-label", "Finish editing");
    expect(screen.queryByTestId("canvas-edit-button")).not.toBeInTheDocument();
  });
});
