import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchRepositorySpecFileContent, fetchStagedCanvasVersionWithSpec } from "./repository-spec-files";

describe("fetchRepositorySpecFileContent", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("content", { status: 200 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads live committed content without version_id or stage", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "canvas.yaml");

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/repository/file?path=canvas.yaml",
      expect.any(Object),
    );
  });

  it("reads a historical version with version_id only", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "canvas.yaml", "version-1", false);

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/repository/file?path=canvas.yaml&version_id=version-1",
      expect.any(Object),
    );
  });

  it("reads staged content with stage only, not version_id", async () => {
    await fetchRepositorySpecFileContent("canvas-1", "canvas.yaml", "version-1", true);

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/repository/file?path=canvas.yaml&stage=true",
      expect.any(Object),
    );
  });
});

describe("fetchStagedCanvasVersionWithSpec", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("nodes: []", { status: 200 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("only reads staged canvas.yaml and reuses provided version metadata", async () => {
    const version = await fetchStagedCanvasVersionWithSpec("canvas-1", {
      metadata: { id: "version-1" },
      spec: { nodes: [], edges: [] },
    });

    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/canvases/canvas-1/repository/file?path=canvas.yaml&stage=true",
      expect.any(Object),
    );
    expect(version?.metadata?.id).toBe("version-1");
    expect(version?.spec?.nodes).toEqual([]);
  });
});
