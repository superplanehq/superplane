import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSidebarLayoutStore } from "@/stores/sidebarLayoutStore";

import { WorkflowFilesOverlayLayer } from "./WorkflowFilesOverlayLayer";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@pierre/trees/react", () => ({
  FileTree: () => null,
  useFileTree: () => ({
    model: {
      resetPaths: vi.fn(),
      getSelectedPaths: () => [],
      getItem: () => undefined,
      scrollToPath: vi.fn(),
    },
  }),
}));

describe("WorkflowFilesOverlayLayer", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarLayoutStore.getState().hydrateFromStorage();
  });

  it("keeps all editor tabs closed after closing the last tab", async () => {
    const user = userEvent.setup();

    render(
      <WorkflowFilesOverlayLayer
        isFilesMode
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
          {
            path: "console.yaml",
            content: "console: true",
            language: "yaml",
          },
        ]}
      />,
    );

    expect(screen.getByRole("button", { name: "Close canvas.yaml" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close canvas.yaml" }));

    expect(screen.queryByRole("button", { name: "Close canvas.yaml" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("monaco-stub")).not.toBeInTheDocument();
  });

  it("offsets the overlay when the left tool sidebar is open", () => {
    useSidebarLayoutStore.setState({ leftWidth: 420, leftMountCount: 1 });

    render(
      <WorkflowFilesOverlayLayer
        isFilesMode
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
    );

    const overlay = screen.getByTestId("workflow-files-overlay");
    expect(overlay).toHaveStyle({ left: "420px", right: "0px" });
  });
});
