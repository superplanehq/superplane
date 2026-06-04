import { describe, expect, it } from "vitest";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/lib/canvas-staging";

import { hasStagedRepositoryFileChanges, pendingChangesFromStaging } from "./workflow-files-staging";

describe("pendingChangesFromStaging", () => {
  it("returns empty pending changes when staging is missing", () => {
    expect(pendingChangesFromStaging(null, { "README.md": "# readme" })).toEqual({});
  });

  it("detects added, modified, and deleted repository files", () => {
    const pending = pendingChangesFromStaging(
      {
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          [CANVAS_YAML_PATH]: "name: updated",
          [CONSOLE_YAML_PATH]: "panels: []",
          "notes.txt": "new file",
        },
        deletedPaths: ["README.md"],
        updatedAt: Date.now(),
      },
      {
        [CANVAS_YAML_PATH]: "name: original",
        "README.md": "# readme",
      },
    );

    expect(pending).toEqual({
      "notes.txt": { type: "added", path: "notes.txt", content: "new file" },
      "README.md": { type: "deleted", path: "README.md" },
    });
  });

  it("shows staged repository files before git baselines are loaded", () => {
    const pending = pendingChangesFromStaging(
      {
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          "README.md": "test",
        },
        updatedAt: Date.now(),
      },
      {},
      new Set(["README.md"]),
    );

    expect(pending).toEqual({
      "README.md": { type: "modified", path: "README.md", content: "test" },
    });
  });

  it("ignores staged files that match the git baseline", () => {
    const pending = pendingChangesFromStaging(
      {
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          "README.md": "# readme",
        },
        updatedAt: Date.now(),
      },
      {
        "README.md": "# readme",
      },
    );

    expect(pending).toEqual({});
  });

  it("detects staged repository files outside canvas.yaml and console.yaml", () => {
    expect(
      hasStagedRepositoryFileChanges({
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          "README.md": "test",
        },
        updatedAt: Date.now(),
      }),
    ).toBe(true);

    expect(
      hasStagedRepositoryFileChanges({
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          [CANVAS_YAML_PATH]: "name: test",
        },
        updatedAt: Date.now(),
      }),
    ).toBe(false);
  });
});
