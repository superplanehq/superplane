import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useCreateWithAgentSession } from "./useCreateWithAgentSession";

const startPlanningSession = vi.fn();
const describePlanningSession = vi.fn();
const endPlanningSession = vi.fn();
const sendPlanningSessionMessage = vi.fn();
const updatePlanningSessionDraft = vi.fn();
const createPlanningSessionWorkOrder = vi.fn();
const skipPlanningSessionDraft = vi.fn();

vi.mock("./planningSessionClient", () => ({
  startPlanningSession: (...args: unknown[]) => startPlanningSession(...args),
  describePlanningSession: (...args: unknown[]) => describePlanningSession(...args),
  endPlanningSession: (...args: unknown[]) => endPlanningSession(...args),
  sendPlanningSessionMessage: (...args: unknown[]) => sendPlanningSessionMessage(...args),
  updatePlanningSessionDraft: (...args: unknown[]) => updatePlanningSessionDraft(...args),
  createPlanningSessionWorkOrder: (...args: unknown[]) => createPlanningSessionWorkOrder(...args),
  skipPlanningSessionDraft: (...args: unknown[]) => skipPlanningSessionDraft(...args),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

const session = (id: string) => ({
  id,
  factoryId: "factory-1",
  canvasId: `canvas-${id}`,
  repository: "acme/payments",
  state: "running",
  messages: [],
  created: [],
});

describe("useCreateWithAgentSession", () => {
  beforeEach(() => {
    startPlanningSession.mockReset();
    describePlanningSession.mockReset();
    endPlanningSession.mockReset();
    sendPlanningSessionMessage.mockReset();
    updatePlanningSessionDraft.mockReset();
    createPlanningSessionWorkOrder.mockReset();
    skipPlanningSessionDraft.mockReset();
    endPlanningSession.mockResolvedValue(session("ended"));
    describePlanningSession.mockResolvedValue(session("session-1"));
  });

  it("ends the session when the dialog closes after Open task", async () => {
    startPlanningSession.mockResolvedValue(session("session-1"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.canvasId).toBe("canvas-session-1");
    });

    act(() => {
      result.current.close();
    });

    await waitFor(() => {
      expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "session-1");
    });
    expect(result.current.open).toBe(false);
  });

  it("ends the current session before a new start", async () => {
    startPlanningSession.mockResolvedValueOnce(session("session-1")).mockResolvedValueOnce(session("session-2"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.canvasId).toBe("canvas-session-1");
    });

    act(() => {
      result.current.start();
    });

    await waitFor(() => {
      expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "session-1");
      expect(startPlanningSession).toHaveBeenCalledTimes(2);
    });
  });

  it("ends the session after the screen unmounts", async () => {
    startPlanningSession.mockResolvedValue(session("session-1"));
    const { result, unmount } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.canvasId).toBe("canvas-session-1");
    });

    unmount();

    await waitFor(() => {
      expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "session-1", { keepalive: true });
    });
  });

  it("tells the agent when Refine further runs", async () => {
    startPlanningSession.mockResolvedValue({
      ...session("session-1"),
      created: [{ id: "wo-1", key: "NEW-1", title: "Retry refunds", description: "Stop double charges." }],
    });
    sendPlanningSessionMessage.mockResolvedValue(session("session-1"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.created).toHaveLength(1);
    });

    act(() => {
      result.current.onRefineCreated(result.current.view.created[0]);
    });

    await waitFor(() => {
      expect(sendPlanningSessionMessage).toHaveBeenCalledWith(
        "org-1",
        "factory-1",
        "session-1",
        "Refine NEW-1: Retry refunds.",
      );
    });
  });

  it("does not tell the agent when the title opens read-only", async () => {
    startPlanningSession.mockResolvedValue({
      ...session("session-1"),
      created: [{ id: "wo-1", key: "NEW-1", title: "Retry refunds", description: "Stop double charges." }],
    });
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.created).toHaveLength(1);
    });

    act(() => {
      result.current.onSelectCreated(result.current.view.created[0]);
    });

    expect(sendPlanningSessionMessage).not.toHaveBeenCalled();
    expect(result.current.view.right).toEqual({
      kind: "preview",
      order: { id: "wo-1", key: "NEW-1", title: "Retry refunds", description: "Stop double charges." },
    });
  });
});
