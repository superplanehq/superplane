import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { findPlanningSessionByWorkOrder, sendWorkOrderPlanningMessage } from "../planningSessionClient";
import type { PlanningSessionPayload } from "../planningSessionView";
import { useAnalysisPlanningSession, workOrderPlanningSessionQueryKey } from "./useAnalysisPlanningSession";

vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(),
  sendPlanningSessionMessage: vi.fn(),
  sendWorkOrderPlanningMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

function wrapperWithClient(queryClient: QueryClient) {
  return function QueryWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const Wrapper = wrapperWithClient(new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return <Wrapper>{children}</Wrapper>;
}

describe("analysis planning recovery", () => {
  afterEach(() => {
    vi.mocked(findPlanningSessionByWorkOrder).mockReset();
    vi.mocked(sendWorkOrderPlanningMessage).mockReset();
  });

  it("lets a message recover analysis canceled before a session existed", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(null);
    vi.mocked(sendWorkOrderPlanningMessage).mockResolvedValue({
      id: "recovered-session",
      state: "running",
      messages: [{ role: "user", text: "Please analyze this task." }],
    });
    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
          refinementCanStart: true,
        }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.view.machineStatus).toBe("waiting");
    expect(result.current.canSend).toBe(true);
    await act(async () => {
      await result.current.onSend("Please analyze this task.");
    });
    expect(sendWorkOrderPlanningMessage).toHaveBeenCalledWith(
      "org-1",
      "factory-1",
      "order-1",
      "Please analyze this task.",
    );
    expect(result.current.sessionId).toBe("recovered-session");
    expect(result.current.view.machineStatus).toBe("starting");
  });

  it.each([
    { canUpdate: false, analysisActive: false },
    { canUpdate: true, analysisActive: true },
    { canUpdate: true, analysisQueued: true },
  ])("does not recover without permission or while analysis is queued: %j", async (state) => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(null);
    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          refinementCanStart: true,
          ...state,
        }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.canSend).toBe(false);
    await act(async () => {
      await result.current.onSend("Analyze this task.");
    });
    expect(sendWorkOrderPlanningMessage).not.toHaveBeenCalled();
  });

  it("keeps a recovery response on its original task after the popup changes tasks", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(null);
    let finishRecovery!: (session: PlanningSessionPayload) => void;
    vi.mocked(sendWorkOrderPlanningMessage).mockImplementation(
      () =>
        new Promise((resolve) => {
          finishRecovery = resolve;
        }),
    );
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result, rerender } = renderHook(
      ({ workOrderId }) =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId,
          enabled: true,
          canUpdate: true,
          refinementCanStart: true,
        }),
      { initialProps: { workOrderId: "order-1" }, wrapper: wrapperWithClient(queryClient) },
    );
    await waitFor(() => expect(result.current.canSend).toBe(true));
    act(() => result.current.onComposerChange("An unsent note for the first task."));
    let sending!: Promise<boolean>;
    act(() => {
      sending = result.current.onSend("Analyze the first task.");
    });
    await waitFor(() => expect(sendWorkOrderPlanningMessage).toHaveBeenCalledTimes(1));
    rerender({ workOrderId: "order-2" });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.composer).toBe("");
    act(() => result.current.onComposerChange("A note for the second task."));

    await act(async () => {
      finishRecovery({ id: "session-for-order-1", state: "running" });
      await sending;
    });

    expect(queryClient.getQueryData(workOrderPlanningSessionQueryKey("org-1", "factory-1", "order-1"))).toMatchObject({
      id: "session-for-order-1",
    });
    expect(result.current.session).toBeNull();
    expect(result.current.composer).toBe("A note for the second task.");
  });

  it("keeps the recovered session when an older missing-session poll completes", async () => {
    let finishLookup!: (session: PlanningSessionPayload | null) => void;
    vi.mocked(findPlanningSessionByWorkOrder)
      .mockResolvedValueOnce(null)
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishLookup = resolve;
          }),
      );
    vi.mocked(sendWorkOrderPlanningMessage).mockResolvedValue({ id: "recovered-session", state: "running" });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
          refinementCanStart: true,
        }),
      { wrapper: wrapperWithClient(queryClient) },
    );
    await waitFor(() => expect(result.current.canSend).toBe(true));
    let polling!: Promise<void>;
    act(() => {
      polling = queryClient.refetchQueries({
        queryKey: workOrderPlanningSessionQueryKey("org-1", "factory-1", "order-1"),
      });
    });
    await waitFor(() => expect(findPlanningSessionByWorkOrder).toHaveBeenCalledTimes(2));
    await act(async () => {
      await result.current.onSend("Analyze this task.");
    });

    await act(async () => {
      finishLookup(null);
      await polling;
    });

    expect(result.current.sessionId).toBe("recovered-session");
    expect(result.current.canSend).toBe(true);
  });
});
