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
    expect(text).toContain("submit-task");
    expect(text).toContain("how-it-works");
    expect(text).toContain("pipeline-board");
    expect(text).toContain("Create a task");
    expect(text).toContain("How it works");
    expect(text).toContain("Your Factory Pipeline");
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

describe("createFixtureFetch agent routes", () => {
  it("serves canvas agent chat and seeded messages", async () => {
    const { createFixtureFetch: createDefaultFixtureFetch, canvasAppIds } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);

    const chat = await fixtureFetch(`http://localhost/api/v1/agents/canvases/${canvasAppIds.canvasId}/chat`);
    await expect(chat.json()).resolves.toMatchObject({
      chat: expect.objectContaining({ id: "storybook-agent-chat", canvasId: canvasAppIds.canvasId, status: "idle" }),
    });

    const messages = await fixtureFetch("http://localhost/api/v1/agents/chats/storybook-agent-chat/messages");
    const body = await messages.json();
    expect(body.hasMore).toBe(false);
    expect(body.messages.length).toBeGreaterThanOrEqual(2);
    expect(body.messages[0]).toEqual(expect.objectContaining({ role: "user" }));
  });

  it("echoes POST message content for send acknowledgements", async () => {
    const { createFixtureFetch: createDefaultFixtureFetch } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);
    const response = await fixtureFetch("http://localhost/api/v1/agents/chats/storybook-agent-chat/messages", {
      method: "POST",
      body: JSON.stringify({ content: "Hello agent" }),
    });
    await expect(response.json()).resolves.toMatchObject({
      message: expect.objectContaining({ role: "user", content: "Hello agent" }),
    });
  });
});

describe("createFixtureFetch agent gates", () => {
  it("exposes agents permissions and managed-agents feature", async () => {
    const { createFixtureFetch: createDefaultFixtureFetch } = await import("./handlers");
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createDefaultFixtureFetch(fallback);

    const me = await fixtureFetch("http://localhost/api/v1/me");
    const meBody = await me.json();
    expect(meBody.user.permissions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ resource: "agents", action: "read" }),
        expect.objectContaining({ resource: "agents", action: "create" }),
      ]),
    );

    const features = await fixtureFetch("http://localhost/account/experimental-features");
    await expect(features.json()).resolves.toMatchObject({
      features: expect.arrayContaining([expect.objectContaining({ id: "claude_managed_agents", released: true })]),
    });
  });
});
