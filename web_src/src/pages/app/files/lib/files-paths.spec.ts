import { describe, expect, it } from "bun:test";

import { getPathValidationError, nextUntitledPath, normalizeFilePath } from "./files-paths";

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
});
