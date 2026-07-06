import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchRepositorySpecFileContent } from "./repository-spec-files";

describe("fetchRepositorySpecFileContent", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("file-body", { status: 200 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads live files without version_id or stage", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "README.md");

    expect(fetch).toHaveBeenCalledWith("/api/v1/canvases/canvas-1/repository/file?path=README.md", expect.any(Object));
  });

  it("reads a specific version with version_id only", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "README.md", "version-1", false);

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/repository/file?path=README.md&version_id=version-1",
      expect.any(Object),
    );
  });

  it("reads staging with stage=true and ignores version_id", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "README.md", "version-1", true);

    const url = vi.mocked(fetch).mock.calls[0]?.[0];
    expect(String(url)).toContain("stage=true");
    expect(String(url)).not.toContain("version_id");
  });
});
