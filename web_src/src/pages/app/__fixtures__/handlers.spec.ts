import { describe, expect, it, vi } from "vitest";

import { createFixtureFetch, type CanvasAppFixture } from "./handlers";

const baseFixture: CanvasAppFixture = {
  canvasId: "canvas-1",
  organizationId: "org-1",
  consoleYaml: "kind: Console\n",
  repositoryFileContents: {
    "README.md": "# Hello from fixture\n",
  },
};

async function fetchFixture(path: string, fixture: CanvasAppFixture = baseFixture): Promise<Response> {
  const fallback = vi.fn() as unknown as typeof fetch;
  const fixtureFetch = createFixtureFetch(fallback, fixture);
  return fixtureFetch(`http://localhost${path}`);
}

describe("createFixtureFetch repository routes", () => {
  it("returns a ready repository so the Files tab query has defined data", async () => {
    const response = await fetchFixture("/api/v1/canvases/canvas-1/repository");
    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      repository: {
        metadata: { canvasId: "canvas-1" },
        status: { state: "STATE_READY", headSha: "storybook-fixture-head" },
      },
    });
  });

  it("lists the default repository file paths plus fixture contents", async () => {
    const response = await fetchFixture("/api/v1/canvases/canvas-1/repository/files");
    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      files: [{ path: "README.md" }, { path: "canvas.yaml" }, { path: "console.yaml" }],
    });
  });

  it("serves README and console.yaml bodies from the fixture", async () => {
    const readme = await fetchFixture("/api/v1/canvases/canvas-1/repository/file?path=README.md");
    expect(await readme.text()).toBe("# Hello from fixture\n");

    const consoleYaml = await fetchFixture("/api/v1/canvases/canvas-1/repository/file?path=console.yaml");
    expect(await consoleYaml.text()).toBe("kind: Console\n");
  });

  it("seeds a non-empty default console.yaml for Live Canvas stories", async () => {
    const { createFixtureFetch: createDefaultFixtureFetch } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);
    const response = await fixtureFetch("http://localhost/api/v1/canvases/any/repository/file?path=console.yaml");
    const text = await response.text();
    expect(text).toContain("kind: Console");
    expect(text).toContain("markdown-showcase");
    expect(text).toContain("Markdown renderer showcase");
  });

  it("honors an explicit repositoryFilePaths override", async () => {
    const response = await fetchFixture("/api/v1/canvases/canvas-1/repository/files", {
      ...baseFixture,
      repositoryFilePaths: ["docs/guide.md", "canvas.yaml"],
    });
    await expect(response.json()).resolves.toEqual({
      files: [{ path: "docs/guide.md" }, { path: "canvas.yaml" }],
    });
  });
});
