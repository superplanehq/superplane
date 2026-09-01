import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { createWithAgentViewFromSession, workspacePlanningRepository } from "./planningSessionView";

describe("workspacePlanningRepository", () => {
  it("uses the workspace app repository", () => {
    expect(workspacePlanningRepository({ onboarding: { appRepository: " semaphore/web " } })).toBe("semaphore/web");
  });

  it("returns empty when the workspace has no app repository", () => {
    expect(workspacePlanningRepository({ onboarding: { completedAt: "2026-08-31T12:00:00Z" } })).toBe("");
    expect(workspacePlanningRepository(null)).toBe("");
  });
});

describe("createWithAgentViewFromSession", () => {
  it("stays starting until the runner execution exists", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        canvasRunId: "run-1",
        messages: [{ id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting }],
        draft: { title: "Retry refunds", description: "Stop double charges." },
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.machineStatus).toBe("starting");
    expect(view.canvasId).toBe("canvas-1");
    expect(view.executionId).toBe("");
    expect(view.right).toEqual({
      kind: "draft",
      draft: { title: "Retry refunds", description: "Stop double charges." },
    });
  });

  it("marks the machine running when the runner execution exists", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        canvasRunId: "run-1",
        executionId: "exec-1",
        draft: { title: "Retry refunds", description: "Stop double charges." },
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.machineStatus).toBe("running");
    expect(view.executionId).toBe("exec-1");
  });
});
