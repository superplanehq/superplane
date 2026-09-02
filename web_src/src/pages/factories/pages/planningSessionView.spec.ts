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

  it("marks the machine waiting when SuperPlane holds for the next message", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        canvasRunId: "run-1",
        executionId: "exec-1",
        waitState: "pending",
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.machineStatus).toBe("waiting");
  });

  it("exposes a pending survey and keeps it out of the chat messages", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        messages: [
          {
            id: "pending-survey",
            kind: "survey",
            role: "agent",
            text: JSON.stringify({
              questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
            }),
          },
          { id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting },
        ],
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.survey).toEqual({
      questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
    });
    expect(view.messages).toEqual([
      { id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting },
    ]);
  });

  it("marks the user text after a survey as a survey reply", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        messages: [
          {
            id: "pending-survey",
            kind: "survey",
            role: "agent",
            text: JSON.stringify({
              questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
            }),
          },
          { id: "reply", kind: "text", role: "user", text: "What is the priority? High" },
        ],
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.messages).toEqual([
      { id: "reply", kind: "text", role: "user", text: "What is the priority? High", origin: "survey" },
    ]);
  });

  it("keeps the work area empty after create so the session list can sit above it", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        created: [{ id: "wo-1", key: "NEW-1", title: "Retry refunds", description: "Stop double charges." }],
      },
      { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
    );

    expect(view.created).toEqual([
      { id: "wo-1", key: "NEW-1", title: "Retry refunds", description: "Stop double charges." },
    ]);
    expect(view.right).toEqual({ kind: "empty" });
  });

  it("keeps a read-only task when the user selected it and there is no new draft", () => {
    const selected = {
      id: "wo-1",
      key: "NEW-1",
      title: "Retry refunds",
      description: "Stop double charges.",
    };
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        created: [selected],
      },
      { composer: "", right: { kind: "preview", order: selected }, endConfirmOpen: false },
    );

    expect(view.right).toEqual({ kind: "preview", order: selected });
  });
});
