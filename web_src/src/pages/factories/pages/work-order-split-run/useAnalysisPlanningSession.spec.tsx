import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { factoryQueryKeys } from "@/hooks/useFactoryData";

import {
  answerPlanningSessionSurvey,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import { analysisSessionPollInterval, useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

function wrapperWithClient(queryClient: QueryClient) {
  return function QueryWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useAnalysisPlanningSession", () => {
  afterEach(() => {
    vi.mocked(findPlanningSessionByWorkOrder).mockReset();
    vi.mocked(sendPlanningSessionMessage).mockReset();
    vi.mocked(answerPlanningSessionSurvey).mockReset();
  });

  it("keeps chat closed when the draft has no analysis session", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(null);

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.showChat).toBe(false);
    });
    expect(result.current.organizationId).toBe("org-1");
    expect(result.current.canSend).toBe(false);
    expect(result.current.queryError).toBeNull();
    expect(findPlanningSessionByWorkOrder).toHaveBeenCalledTimes(1);
  });

  it("surfaces failures that are not a missing session", async () => {
    const failure = new Error("server unavailable");
    vi.mocked(findPlanningSessionByWorkOrder).mockRejectedValue(failure);

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.queryError).toBe(failure);
    });
    expect(result.current.showChat).toBe(false);
  });

  it("refreshes analysis results when the session response changes", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue({
      id: "session-1",
      state: "ended",
    });

    renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper: wrapperWithClient(queryClient) },
    );

    await waitFor(() => {
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: factoryQueryKeys.workOrderChecks("org-1", "factory-1", "order-1"),
      });
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: factoryQueryKeys.workOrderArtifacts("org-1", "factory-1", "order-1"),
      });
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"),
      });
    });
  });

  it("lets the user send after analysis stops so a new run can continue the chat", async () => {
    const stopped = {
      id: "session-1",
      state: "ended",
      canvasId: "",
      canvasRunId: "",
      messages: [{ role: "user", text: "Add a breed field." }],
    };
    const continued = {
      ...stopped,
      state: "running",
      messages: [
        { role: "user", text: "Add a breed field." },
        { role: "user", text: "Use the existing puppy form." },
      ],
    };
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(stopped);
    vi.mocked(sendPlanningSessionMessage).mockResolvedValue(continued);

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
          analysisDelivered: true,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.showChat).toBe(true);
      expect(result.current.canSend).toBe(true);
      expect(result.current.isLive).toBe(false);
    });
    expect(result.current.view.messages.map(({ role, text }) => ({ role, text }))).toEqual([
      { role: "user", text: "Add a breed field." },
    ]);

    act(() => {
      result.current.onComposerChange("Use the existing puppy form.");
    });
    await act(async () => {
      await result.current.onSend();
    });

    expect(sendPlanningSessionMessage).toHaveBeenCalledWith(
      "org-1",
      "factory-1",
      "session-1",
      "Use the existing puppy form.",
    );
    expect(result.current.view.machineStatus).toBe("starting");
    expect(result.current.showChat).toBe(true);
    expect(result.current.view.messages.map(({ role, text }) => ({ role, text }))).toEqual(continued.messages);
  });

  it("keeps prior messages when a restart response contains only the new turn", async () => {
    const stopped = {
      id: "session-1",
      state: "ended",
      messages: [
        { id: "user-1", role: "user", text: "Use the current form." },
        { id: "agent-1", role: "agent", text: "I updated the plan." },
      ],
    };
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(stopped);
    vi.mocked(sendPlanningSessionMessage).mockResolvedValue({
      id: "session-1",
      state: "running",
      messages: [{ id: "user-2", role: "user", text: "Also cover errors." }],
    });

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.canSend).toBe(true));
    act(() => result.current.onComposerChange("Also cover errors."));
    act(() => result.current.onSend());

    await waitFor(() => {
      expect(result.current.view.messages.map(({ role, text }) => ({ role, text }))).toEqual([
        { role: "user", text: "Use the current form." },
        { role: "agent", text: "I updated the plan." },
        { role: "user", text: "Also cover errors." },
      ]);
    });
  });

  it("answers a survey through its dedicated command", async () => {
    const session = {
      id: "session-1",
      state: "running",
      executionId: "execution-1",
      survey: { id: "survey-1", questions: [{ prompt: "Priority?", options: ["High", "Low"] }] },
    };
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(session);
    vi.mocked(answerPlanningSessionSurvey).mockResolvedValue({ ...session, survey: null });

    const { result } = renderHook(
      () =>
        useAnalysisPlanningSession({
          organizationId: "org-1",
          factoryId: "factory-1",
          workOrderId: "order-1",
          enabled: true,
          canUpdate: true,
        }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.canSend).toBe(true));
    act(() => result.current.onSubmitSurvey("Priority? High"));
    await waitFor(() => {
      expect(answerPlanningSessionSurvey).toHaveBeenCalledWith("org-1", "factory-1", "session-1", "Priority? High");
    });
    expect(sendPlanningSessionMessage).not.toHaveBeenCalled();
  });
});

describe("analysisSessionPollInterval", () => {
  it("polls while waiting for an enabled analysis session", () => {
    expect(analysisSessionPollInterval(true, null)).toBe(1500);
  });

  it("does not poll an absent session after the disabled-mode lookup", () => {
    expect(analysisSessionPollInterval(false, null)).toBe(false);
  });

  it("polls a live pinned session and stops after it ends", () => {
    const session = { id: "session-1", state: "running" };
    expect(analysisSessionPollInterval(false, session)).toBe(1500);
    expect(analysisSessionPollInterval(false, { ...session, state: "ended" })).toBe(false);
  });
});
