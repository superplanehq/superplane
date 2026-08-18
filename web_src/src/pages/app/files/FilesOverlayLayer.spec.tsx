import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useSidebarLayoutStore } from "@/stores/sidebarLayoutStore";

import { FilesOverlayLayer } from "./FilesOverlayLayer";

let repositoryFiles = [{ path: "README.md" }];
const repositoryFileContents: Record<string, string> = {
  "README.md": "# readme",
  "notes/scratchpad.json": '{ "hello": "agent" }',
};
let stagedPaths: string[] = [];
let repositoryFileQueryStages: Array<{ path: string; stage: boolean }> = [];

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

function Wrapper({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

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
  useCanvasRepositoryFile: (
    _canvasId: string,
    path: string | null,
    _enabled: boolean,
    _versionId: string | undefined,
    stage: boolean,
  ) => {
    if (path) repositoryFileQueryStages.push({ path, stage });
    return {
      data: path && repositoryFileContents[path] ? { path, content: repositoryFileContents[path] } : undefined,
      isLoading: false,
      error: null,
    };
  },
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
  useCanvasStaging: () => ({
    data: { hasStaging: stagedPaths.length > 0, stagedPaths },
    isLoading: false,
    error: null,
  }),
  fetchRepositoryFileContentCached: (_queryClient: unknown, _canvasId: string, path: string) =>
    Promise.resolve(repositoryFileContents[path] ?? ""),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@pierre/trees/react", () => ({
  FileTree: ({
    model,
    renderContextMenu,
  }: {
    model: {
      paths: string[];
      gitStatus?: Array<{ path: string; status: string }>;
      selectPath?: (path: string) => void;
    };
    renderContextMenu?: (item: { kind: "file"; path: string }, context: { close: () => void }) => ReactNode;
  }) => (
    <>
      {model.paths.map((path) => {
        const status = model.gitStatus?.find((entry) => entry.path === path)?.status;

        return (
          <div key={path} data-testid={`file-tree-row-${path}`}>
            <button type="button" data-git-status={status} onClick={() => model.selectPath?.(path)}>
              {path}
            </button>
            {renderContextMenu?.({ kind: "file", path }, { close: vi.fn() })}
          </div>
        );
      })}
    </>
  ),
  useFileTree: ({
    paths,
    gitStatus,
    onSelectionChange,
  }: {
    paths: string[];
    gitStatus?: Array<{ path: string; status: string }>;
    onSelectionChange?: (selectedPaths: string[]) => void;
  }) => {
    return {
      model: {
        paths,
        gitStatus,
        selectPath: (path: string) => onSelectionChange?.([path]),
        resetPaths: vi.fn(),
        setGitStatus: vi.fn(),
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
    repositoryFiles = [{ path: "README.md" }];
    stagedPaths = [];
    repositoryFileQueryStages = [];
    queryClient.clear();
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
      { wrapper: Wrapper },
    );

    expect(screen.getByRole("button", { name: "Close canvas.yaml" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close canvas.yaml" }));

    expect(screen.queryByRole("button", { name: "Close canvas.yaml" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("monaco-stub")).not.toBeInTheDocument();
  });

  it("keeps deleted files visible with committed content and a committed read", async () => {
    const user = userEvent.setup();

    const { rerender } = render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="canvas-1"
        versionId="version-1"
        isEditing
        canWrite
        files={[{ path: "canvas.yaml", content: "canvas: true", language: "yaml" }]}
      />,
      { wrapper: Wrapper },
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    await user.click(within(screen.getByTestId("file-tree-row-README.md")).getByRole("menuitem", { name: "Delete" }));

    expect(
      within(screen.getByTestId("file-tree-row-README.md")).getByRole("button", { name: "README.md" }),
    ).toHaveAttribute("data-git-status", "deleted");
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");
    expect(screen.queryByText("File marked for deletion")).not.toBeInTheDocument();
    expect(repositoryFileQueryStages.at(-1)).toEqual({ path: "README.md", stage: false });

    repositoryFiles = [];
    rerender(
      <FilesOverlayLayer
        isFilesMode
        canvasId="canvas-1"
        versionId="version-1"
        isEditing
        canWrite
        files={[{ path: "canvas.yaml", content: "canvas: true", language: "yaml" }]}
      />,
    );

    expect(
      within(screen.getByTestId("file-tree-row-README.md")).getByRole("button", { name: "README.md" }),
    ).toHaveAttribute("data-git-status", "deleted");
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
      { wrapper: Wrapper },
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    await user.type(screen.getByTestId("monaco-stub"), "!");
    expect(
      within(screen.getByTestId("file-tree-row-README.md")).getByRole("button", { name: "README.md" }),
    ).toHaveAttribute("data-git-status", "modified");

    await user.click(screen.getAllByRole("button", { name: "canvas.yaml" })[0]!);
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
      { wrapper: Wrapper },
    );

    await user.click(screen.getAllByRole("button", { name: "README.md" })[0]!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");

    await user.click(screen.getAllByRole("button", { name: "canvas.yaml" })[0]!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("canvas: true");

    await user.click(screen.getAllByRole("button", { name: "README.md" }).at(-1)!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");
  });

  it("shows and opens files that only exist in draft staging", async () => {
    const user = userEvent.setup();

    const props = {
      isFilesMode: true,
      canvasId: "canvas-1",
      versionId: "version-1",
      isEditing: true,
      canWrite: true,
      files: [
        {
          path: "canvas.yaml",
          content: "canvas: true",
          language: "yaml",
        },
      ],
    };

    const { rerender } = render(<FilesOverlayLayer {...props} />, { wrapper: Wrapper });

    expect(screen.queryByRole("button", { name: "notes/scratchpad.json" })).not.toBeInTheDocument();

    stagedPaths = ["notes/scratchpad.json"];
    rerender(
      <FilesOverlayLayer
        isFilesMode={props.isFilesMode}
        canvasId={props.canvasId}
        versionId={props.versionId}
        isEditing={props.isEditing}
        canWrite={props.canWrite}
        files={props.files}
      />,
    );

    await user.click(screen.getByRole("button", { name: "notes/scratchpad.json" }));

    expect(screen.getByTestId("monaco-stub")).toHaveValue('{ "hello": "agent" }');
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
      { wrapper: Wrapper },
    );

    await user.click(screen.getByRole("button", { name: "New file" }));
    expect(screen.getByDisplayValue("untitled.txt")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByDisplayValue("untitled.txt")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close untitled.txt" })).not.toBeInTheDocument();
  });

  it("marks a new file as added", async () => {
    const user = userEvent.setup();

    render(
      <FilesOverlayLayer
        isFilesMode
        canvasId="test-canvas"
        isEditing
        canWrite
        files={[{ path: "canvas.yaml", content: "canvas: true", language: "yaml" }]}
      />,
      { wrapper: Wrapper },
    );

    await user.click(screen.getByRole("button", { name: "New file" }));
    const newFileInput = screen.getByDisplayValue("untitled.txt");
    await user.clear(newFileInput);
    await user.type(newFileInput, "notes.txt{Enter}");

    expect(
      within(screen.getByTestId("file-tree-row-notes.txt")).getByRole("button", { name: "notes.txt" }),
    ).toHaveAttribute("data-git-status", "added");
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
      { wrapper: Wrapper },
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
      { wrapper: Wrapper },
    );

    const overlay = screen.getByTestId("files-overlay");
    expect(overlay).toHaveStyle({ left: "420px", right: "0px" });
  });
});
