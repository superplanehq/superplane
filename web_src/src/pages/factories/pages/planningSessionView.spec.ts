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
  it("maps a running session with a greeting and a draft", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasRunId: "run-1",
        messages: [{ id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting }],
        draft: { title: "Retry refunds", description: "Stop double charges." },
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.machineStatus).toBe("running");
    expect(view.messages[0]).toMatchObject({ kind: "text", text: CREATE_WITH_AGENT_COPY.greeting });
    expect(view.right).toEqual({
      kind: "draft",
      draft: { title: "Retry refunds", description: "Stop double charges." },
    });
  });
});
