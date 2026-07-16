import { act, renderHook } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { usePendingState } from "./usePendingState";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

function renderPendingState(overrides: Partial<Parameters<typeof usePendingState>[0]> = {}) {
  const openFile = vi.fn();
  const closeFile = vi.fn();
  const options = {
    generatedPathSet: new Set<string>(["canvas.yaml"]),
    generatedPaths: ["canvas.yaml"],
    finalRepositoryPathsRef: createRef<string[]>(),
    allPathsRef: createRef<string[]>(),
    loadedContentByPathRef: createRef<Record<string, string>>(),
    committedContentByPathRef: createRef<Record<string, string>>(),
    openFile,
    closeFile,
    ...overrides,
  };
  options.finalRepositoryPathsRef.current = ["docs/readme.md"];
  options.allPathsRef.current = ["canvas.yaml", "docs/readme.md"];
  options.loadedContentByPathRef.current = {};
  options.committedContentByPathRef.current = {};

  const view = renderHook(() => usePendingState(options));
  return { ...view, openFile, closeFile };
}

describe("usePendingState", () => {
  it("closes the tab and stages a deletion when a repository file is deleted", () => {
    const { result, closeFile } = renderPendingState();

    act(() => {
      result.current.deleteFile("docs/readme.md");
    });

    expect(closeFile).toHaveBeenCalledWith("docs/readme.md");
    expect(result.current.pendingChangesByPath["docs/readme.md"]).toEqual({
      type: "deleted",
      path: "docs/readme.md",
    });
  });

  it("closes the tab when deleting a newly added file without leaving a pending change", () => {
    const { result, closeFile } = renderPendingState();

    act(() => {
      result.current.setPendingChangesByPath({
        "new.txt": { type: "added", path: "new.txt", content: "" },
      });
    });

    act(() => {
      result.current.deleteFile("new.txt");
    });

    expect(closeFile).toHaveBeenCalledWith("new.txt");
    expect(result.current.pendingChangesByPath["new.txt"]).toBeUndefined();
  });

  it("does not delete or close generated (spec) files", () => {
    const { result, closeFile } = renderPendingState();

    act(() => {
      result.current.deleteFile("canvas.yaml");
    });

    expect(closeFile).not.toHaveBeenCalled();
    expect(result.current.pendingChangesByPath["canvas.yaml"]).toBeUndefined();
  });
});
