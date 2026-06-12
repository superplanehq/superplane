import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSidebarLayoutStore } from "@/stores/sidebarLayoutStore";

import { FilesOverlayLayer } from "./FilesOverlayLayer";

const repositoryFiles = [{ path: "README.md" }];
const repositoryFileContents: Record<string, string> = {
  "README.md": "# readme",
};

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasRepository: () => ({
    data: { status: { headSha: "abc123" } },
    isLoading: false,
    error: null,
  }),
  useCanvasRepositoryFiles: () => ({
    data: { files: repositoryFiles },
    isLoading: false,
    error: null,
  }),
  useCanvasRepositoryFile: (_canvasId: string, path: string | null) => ({
    data: path && repositoryFileContents[path] ? { path, content: repositoryFileContents[path] } : undefined,
    isLoading: false,
    error: null,
  }),
  useStageRepositoryFiles: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useDiscardRepositoryFilePaths: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useCanvasVersionStaging: () => ({
    data: { hasStaging: false, stagedPaths: [] },
    isLoading: false,
    error: null,
  }),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

let selectRepositoryPath: ((path: string) => void) | undefined;

vi.mock("@pierre/trees/react", () => ({
  FileTree: () => (
    <button type="button" onClick={() => selectRepositoryPath?.("README.md")}>
      README.md
    </button>
  ),
  useFileTree: ({
    paths,
    onSelectionChange,
  }: {
    paths: string[];
    onSelectionChange?: (selectedPaths: string[]) => void;
  }) => {
    selectRepositoryPath = (path: string) => {
      if (!paths.includes(path)) return;
      onSelectionChange?.([path]);
    };

    return {
      model: {
        resetPaths: vi.fn(),
        getSelectedPaths: () => [],
        getItem: () => ({
          select: vi.fn(),
          deselect: vi.fn(),
        }),
        scrollToPath: vi.fn(),
      },
    };
  },
}));

describe("FilesOverlayLayer", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarLayoutStore.getState().hydrateFromStorage();
  });

  it("keeps all editor tabs closed after closing the last tab", async () => {
    const user = userEvent.setup();

    render(
      <FilesOverlayLayer
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
      { wrapper: MemoryRouter },
    );

    expect(screen.getByRole("button", { name: "Close canvas.yaml" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close canvas.yaml" }));

    expect(screen.queryByRole("button", { name: "Close canvas.yaml" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("monaco-stub")).not.toBeInTheDocument();
  });

  it("keeps the first edit after switching away and back to a repository file", async () => {
    const user = userEvent.setup();

    render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="canvas-1"
        isEditing
        canWrite
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
      { wrapper: MemoryRouter },
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    await user.type(screen.getByTestId("monaco-stub"), "!");

    await user.click(screen.getByRole("button", { name: "canvas.yaml" }));
    await user.click(screen.getAllByRole("button", { name: "README.md" }).at(-1)!);

    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme!");
  });

  it("keeps repository file content when switching to and from generated files", async () => {
    const user = userEvent.setup();

    render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="canvas-1"
        isEditing
        canWrite
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
      { wrapper: MemoryRouter },
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");

    await user.click(screen.getByRole("button", { name: "canvas.yaml" }));
    expect(screen.getByTestId("monaco-stub")).toHaveValue("canvas: true");

    await user.click(screen.getAllByRole("button", { name: "README.md" }).at(-1)!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");
  });

  it("does not create a file when Escape is pressed in the new file input", async () => {
    const user = userEvent.setup();

    render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="test-canvas"
        isEditing
        canWrite
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
      { wrapper: MemoryRouter },
    );

    await user.click(screen.getByRole("button", { name: "New file" }));
    expect(screen.getByDisplayValue("untitled.txt")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByDisplayValue("untitled.txt")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close untitled.txt" })).not.toBeInTheDocument();
  });

  it("re-resolves the header actions portal host when entering edit mode", async () => {
    const user = userEvent.setup();
    const slotId = "canvas-files-header-actions-test-canvas";

    const { rerender } = render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="test-canvas"
        isEditing={false}
        canWrite
        headerActionsSlotId={slotId}
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
      { wrapper: MemoryRouter },
    );

    expect(document.getElementById(slotId)).toBeNull();

    const host = document.createElement("div");
    host.id = slotId;
    document.body.appendChild(host);

    rerender(
      <FilesOverlayLayer
        isFilesMode
        canvasId="test-canvas"
        isEditing
        canWrite
        headerActionsSlotId={slotId}
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    await user.clear(screen.getByTestId("monaco-stub"));
    await user.type(screen.getByTestId("monaco-stub"), "updated readme");

    expect(within(host).getByRole("button", { name: "Diff" })).toBeInTheDocument();

    host.remove();
  });

  it("offsets the overlay when the left tool sidebar is open", () => {
    useSidebarLayoutStore.setState({ leftWidth: 420, leftMountCount: 1 });

    render(
      <FilesOverlayLayer
        isFilesMode
        files={[
          {
            path: "canvas.yaml",
            content: "canvas: true",
            language: "yaml",
          },
        ]}
      />,
      { wrapper: MemoryRouter },
    );

    const overlay = screen.getByTestId("files-overlay");
    expect(overlay).toHaveStyle({ left: "420px", right: "0px" });
  });
});
