import { describe, expect, it } from "vitest";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/lib/canvas-staging";

import { resolveStagedBranchYaml } from "./resolve-staged-branch-yaml";

describe("resolveStagedBranchYaml", () => {
  it("falls back to git tip yaml when staging only has other repo files", () => {
    const resolved = resolveStagedBranchYaml(
      {
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          "README.md": "# updated readme",
        },
        updatedAt: Date.now(),
      },
      "name: from-git",
      "panels: []",
    );

    expect(resolved).toEqual({
      canvasYaml: "name: from-git",
      consoleYaml: "panels: []",
    });
  });

  it("prefers staged canvas and console yaml when present", () => {
    const resolved = resolveStagedBranchYaml(
      {
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc123",
        files: {
          [CANVAS_YAML_PATH]: "name: staged",
          [CONSOLE_YAML_PATH]: "panels: [staged]",
          "README.md": "# updated readme",
        },
        updatedAt: Date.now(),
      },
      "name: from-git",
      "panels: []",
    );

    expect(resolved).toEqual({
      canvasYaml: "name: staged",
      consoleYaml: "panels: [staged]",
    });
  });
});
