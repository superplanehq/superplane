import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { findPlanningSessionByWorkOrder, sendPlanningSessionMessage } from "../planningSessionClient";
import { useAutomationChatSessions } from "./useAutomationChatSessions";
import type { SplitRunPhase } from "./splitRunMocks";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

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

it("keeps separate automation conversations on the same task", async () => {
  vi.mocked(findPlanningSessionByWorkOrder).mockImplementation(async (_org, _factory, _order, runId) => ({
    id: `session-${runId}`,
    state: "ended",
    messages: [{ id: runId, role: "agent", text: `Reply for ${runId}` }],
  }));
  vi.mocked(sendPlanningSessionMessage).mockResolvedValue({ id: "session-first", state: "ended" });
  const { result, unmount } = renderHook(
    () => ({
      first: useAnalysisPlanningSession({
        organizationId: "org",
        factoryId: "factory",
        workOrderId: "task",
        runId: "first",
        enabled: true,
        canUpdate: true,
      }),
      second: useAnalysisPlanningSession({
        organizationId: "org",
        factoryId: "factory",
        workOrderId: "task",
        runId: "second",
        enabled: true,
        canUpdate: false,
      }),
    }),
    { wrapper },
  );
  await waitFor(() => {
    expect(result.current.first.sessionId).toBe("session-first");
    expect(result.current.second.sessionId).toBe("session-second");
  });
  await act(async () => {
    await result.current.first.onSend("Continue");
  });
  expect(sendPlanningSessionMessage).toHaveBeenCalledWith("org", "factory", "session-first", "Continue");
  expect(result.current.second.sessionId).toBe("session-second");
  expect(await result.current.second.onSend("Forbidden")).toBe(false);
  expect(sendPlanningSessionMessage).not.toHaveBeenCalledWith("org", "factory", "session-second", "Forbidden");
  unmount();
});

it("opens the latest chat by default, deduplicates restarts, and keeps Activity selectable", async () => {
  vi.mocked(findPlanningSessionByWorkOrder).mockImplementation(async (_org, _factory, _order, runId) =>
    runId === "plain" ? null : { id: runId === "older" ? "older-session" : "current-session", state: "ended" },
  );
  const phase = (runId: string, startedAt: string): SplitRunPhase => ({
    id: runId,
    runId,
    startedAt,
    name: "Custom",
    componentName: "Custom",
    status: "passed",
    duration: "1m",
    artifacts: [],
    stream: [],
    canvasSteps: [],
  });
  const { result, unmount } = renderHook(
    () =>
      useAutomationChatSessions("org", "factory", "task", [
        phase("older", "2026-11-24T10:00:00Z"),
        phase("original", "2026-11-24T11:00:00Z"),
        phase("restart", "2026-11-24T12:00:00Z"),
        phase("plain", "2026-11-24T13:00:00Z"),
      ]),
    { wrapper },
  );
  await waitFor(() => expect(result.current.selected?.runId).toBe("restart"));
  expect(result.current.chats.map((chat) => chat.runId)).toEqual(["restart", "older"]);
  act(() => result.current.setChoice("activity"));
  expect(result.current.selected).toBeUndefined();
  act(() => result.current.setChoice("older"));
  expect(result.current.selected?.runId).toBe("older");
  unmount();
});
