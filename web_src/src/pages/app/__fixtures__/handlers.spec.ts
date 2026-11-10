import { describe, expect, it, vi } from "bun:test";

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
  it("serves README and console.yaml bodies from the fixture", async () => {
    const readme = await fetchFixture("/api/v1/canvases/canvas-1/file?path=README.md");
    expect(await readme.text()).toBe("# Hello from fixture\n");

    const consoleYaml = await fetchFixture("/api/v1/canvases/canvas-1/file?path=console.yaml");
    expect(await consoleYaml.text()).toBe("kind: Console\n");
  });

  it("seeds a non-empty default console.yaml for Live Canvas stories", async () => {
    const { createFixtureFetch: createDefaultFixtureFetch } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);
    const response = await fixtureFetch("http://localhost/api/v1/canvases/any/file?path=console.yaml");
    const text = await response.text();
    expect(text).toContain("kind: Console");
    expect(text).toContain("submit-task");
    expect(text).toContain("how-it-works");
    expect(text).toContain("pipeline-board");
    expect(text).toContain("Create a task");
    expect(text).toContain("How it works");
    expect(text).toContain("Your Factory Pipeline");
  });
});

describe("createFixtureFetch run routes", () => {
  const runsFixture: CanvasAppFixture = {
    canvasId: "canvas-1",
    organizationId: "org-1",
    rootEventId: "event-passed",
    runs: {
      runs: [
        { id: "run-running", state: "STATE_STARTED", result: "RESULT_UNKNOWN", rootEvent: { id: "event-running" } },
        { id: "run-failed", state: "STATE_FINISHED", result: "RESULT_FAILED", rootEvent: { id: "event-failed" } },
        { id: "run-passed", state: "STATE_FINISHED", result: "RESULT_PASSED", rootEvent: { id: "event-passed" } },
      ],
      totalCount: 3,
      hasNextPage: false,
    },
    runDetailsById: {
      "run-failed": {
        run: {
          id: "run-failed",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          queueItems: [],
        },
      },
    },
    executionsByEventId: {
      "event-failed": {
        executions: [{ id: "exec-1", nodeId: "implement", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
      },
      "event-passed": {
        executions: [{ id: "exec-2", nodeId: "implement", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
      },
    },
  };

  it("filters runs by states so the running-runs badge count is accurate", async () => {
    const response = await fetchFixture("/api/v1/canvases/canvas-1/runs?states=STATE_STARTED", runsFixture);
    const body = await response.json();
    expect(body.totalCount).toBe(1);
    expect(body.runs).toEqual([expect.objectContaining({ id: "run-running", state: "STATE_STARTED" })]);
  });

  it("returns per-run describe payloads and per-event executions", async () => {
    const detail = await fetchFixture("/api/v1/canvases/canvas-1/runs/run-failed", runsFixture);
    await expect(detail.json()).resolves.toEqual({
      run: expect.objectContaining({ id: "run-failed", result: "RESULT_FAILED" }),
    });

    const failedExecs = await fetchFixture("/api/v1/canvases/canvas-1/events/event-failed/executions", runsFixture);
    await expect(failedExecs.json()).resolves.toEqual({
      executions: [expect.objectContaining({ result: "RESULT_FAILED" })],
    });

    const passedExecs = await fetchFixture("/api/v1/canvases/canvas-1/events/event-passed/executions", runsFixture);
    await expect(passedExecs.json()).resolves.toEqual({
      executions: [expect.objectContaining({ result: "RESULT_PASSED" })],
    });
  });
});

describe("createFixtureFetch canvas detail", () => {
  it("exposes a live version so edit sessions can start", async () => {
    const { canvasAppIds, createFixtureFetch: createDefaultFixtureFetch } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);

    const response = await fixtureFetch(`http://localhost/api/v1/canvases/${canvasAppIds.canvasId}`);
    const body = await response.json();

    expect(body.canvas.metadata.liveVersionId).toBe(canvasAppIds.versionId);
  });
});

describe("createFixtureFetch integrations", () => {
  it("defaults githubAppConfigured to true", async () => {
    const response = await fetchFixture("/api/v1/integrations");
    await expect(response.json()).resolves.toMatchObject({ githubAppConfigured: true });
  });

  it("keeps an explicit githubAppConfigured false from the fixture", async () => {
    const response = await fetchFixture("/api/v1/integrations", {
      ...baseFixture,
      integrations: { integrations: [], githubAppConfigured: false },
    });
    await expect(response.json()).resolves.toMatchObject({ githubAppConfigured: false });
  });
});
