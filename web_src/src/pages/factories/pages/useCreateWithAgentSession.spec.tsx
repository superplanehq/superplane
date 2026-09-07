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

const showErrorToast = vi.fn();

vi.mock("@/lib/toast", () => ({
  showErrorToast: (...args: unknown[]) => showErrorToast(...args),
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
    showErrorToast.mockReset();
    vi.useRealTimers();
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

  it("warns before unload and ignores a persisted pagehide", async () => {
    startPlanningSession.mockResolvedValue(session("session-1"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await waitFor(() => {
      expect(result.current.view.canvasId).toBe("canvas-session-1");
    });

    const beforeUnload = new Event("beforeunload", { cancelable: true }) as BeforeUnloadEvent;
    Object.defineProperty(beforeUnload, "returnValue", { writable: true, value: "" });
    window.dispatchEvent(beforeUnload);
    expect(beforeUnload.defaultPrevented).toBe(true);

    endPlanningSession.mockClear();
    window.dispatchEvent(Object.assign(new Event("pagehide"), { persisted: true }));
    expect(endPlanningSession).not.toHaveBeenCalled();

    window.dispatchEvent(Object.assign(new Event("pagehide"), { persisted: false }));
    await waitFor(() => {
      expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "session-1", { keepalive: true });
    });
  });

  it("closes the dialog when start fails", async () => {
    startPlanningSession.mockRejectedValue(new Error("no machines"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });

    await waitFor(() => {
      expect(result.current.open).toBe(false);
    });
    expect(result.current.view.canvasId).toBe("");
    expect(showErrorToast).toHaveBeenCalled();
  });

  it("does not save a draft after close", async () => {
    vi.useFakeTimers();
    startPlanningSession.mockResolvedValue(session("session-1"));
    updatePlanningSessionDraft.mockResolvedValue(session("session-1"));
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current.view.canvasId).toBe("canvas-session-1");

    act(() => {
      result.current.onDraftTitleChange("Retry refunds");
    });
    act(() => {
      result.current.close();
    });
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(updatePlanningSessionDraft).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it("does not save a draft after unmount", async () => {
    vi.useFakeTimers();
    startPlanningSession.mockResolvedValue(session("session-1"));
    updatePlanningSessionDraft.mockResolvedValue(session("session-1"));
    const { result, unmount } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current.view.canvasId).toBe("canvas-session-1");

    act(() => {
      result.current.onDraftTitleChange("Retry refunds");
    });
    unmount();
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(updatePlanningSessionDraft).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it("ends a session that arrives after close", async () => {
    let resolveStart: ((value: ReturnType<typeof session>) => void) | undefined;
    startPlanningSession.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveStart = resolve;
        }),
    );
    const { result } = renderHook(() => useCreateWithAgentSession("acme/payments", "org-1", "factory-1"));

    act(() => {
      result.current.start();
    });
    act(() => {
      result.current.close();
    });

    await act(async () => {
      resolveStart?.(session("late-1"));
    });

    await waitFor(() => {
      expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "late-1");
    });
    expect(result.current.open).toBe(false);
    expect(result.current.view.canvasId).toBe("");
  });
});
