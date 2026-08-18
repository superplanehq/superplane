import { describe, expect, it } from "vitest";

import {
  buildFinalRepositoryPaths,
  buildRenderableTreePaths,
  getPathValidationError,
  nextUntitledPath,
  normalizeFilePath,
} from "./files-paths";

describe("files-paths", () => {
  it("normalizes file paths", () => {
    expect(normalizeFilePath("  foo/bar.txt  ")).toBe("foo/bar.txt");
    expect(normalizeFilePath("\\foo\\bar.txt")).toBe("foo/bar.txt");
  });

  it("picks the next untitled path", () => {
    expect(nextUntitledPath(new Set())).toBe("untitled.txt");
    expect(nextUntitledPath(new Set(["untitled.txt"]))).toBe("untitled-1.txt");
  });

  it("detects duplicate paths", () => {
    expect(getPathValidationError(["a.txt", "a.txt"])).toContain("already used");
  });

  it("keeps deleted files renderable while excluding them from final repository paths", () => {
    const deletedChange = { type: "deleted" as const, path: "README.md" };

    expect(buildRenderableTreePaths(["README.md"], [deletedChange])).toEqual(["README.md"]);
    expect(buildFinalRepositoryPaths(["README.md"], [deletedChange])).toEqual([]);
  });
});
