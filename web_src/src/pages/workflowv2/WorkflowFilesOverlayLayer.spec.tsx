import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkflowFilesOverlayLayer } from "./WorkflowFilesOverlayLayer";

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
  useCommitCanvasRepositoryFiles: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
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

describe("WorkflowFilesOverlayLayer", () => {
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

  it("keeps repository file content when switching to and from generated files", async () => {
    const user = userEvent.setup();

    render(
      <WorkflowFilesOverlayLayer
        isFilesMode
        canvasId="canvas-1"
        canWrite
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
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");

    await user.click(screen.getByRole("button", { name: "canvas.yaml" }));
    expect(screen.getByTestId("monaco-stub")).toHaveValue("canvas: true");

    await user.click(screen.getAllByRole("button", { name: "README.md" }).at(-1)!);
    expect(screen.getByTestId("monaco-stub")).toHaveValue("# readme");
  });
});
