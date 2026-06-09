import { describe, expect, it } from "vitest";

import { CANVAS_YAML_PATH } from "../../lib/workflow-spec-paths";
import type { AppFile } from "../types";

import { getSelectedFileViewState } from "./files-view-state";

describe("getSelectedFileViewState", () => {
  const canvasFile: AppFile = {
    path: CANVAS_YAML_PATH,
    content: "apiVersion: v1\nkind: Canvas\nspec:\n  nodes: []\n  edges: []",
    language: "yaml",
  };

  it("allows editing workflow spec files while keeping repository files read-only", () => {
    const specView = getSelectedFileViewState({
      selectedPath: CANVAS_YAML_PATH,
      selectedGeneratedFile: canvasFile,
      loadedContentByPath: {},
      selectedPathExistsInRepository: false,
      selectedFileQuery: { isLoading: false, error: null },
      canManageRepositoryFiles: true,
    });

    expect(specView.editorDisabled).toBe(false);

    const repoView = getSelectedFileViewState({
      selectedPath: "README.md",
      selectedGeneratedFile: undefined,
      loadedContentByPath: { "README.md": "# readme" },
      selectedPathExistsInRepository: true,
      selectedFileQuery: { isLoading: false, error: null },
      canManageRepositoryFiles: true,
    });

    expect(repoView.editorDisabled).toBe(false);
  });

  it("prefers pending yaml edits over generated file content", () => {
    const view = getSelectedFileViewState({
      selectedPath: CANVAS_YAML_PATH,
      selectedGeneratedFile: canvasFile,
      selectedChange: { type: "modified", path: CANVAS_YAML_PATH, content: "name: edited" },
      loadedContentByPath: {},
      selectedPathExistsInRepository: false,
      selectedFileQuery: { isLoading: false, error: null },
      canManageRepositoryFiles: true,
    });

    expect(view.selectedContent).toBe("name: edited");
  });
});
