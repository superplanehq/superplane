import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import {
  endPlanningSession,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
  endPlanningSession: vi.fn(),
}));

const running = {
  id: "session-1",
  state: "running",
  messages: [{ id: "user-1", role: "user", text: "Add a screenshot." }],
};

function planningTalk(message: CreateWithAgentMessage) {
  return { role: message.role, text: message.kind === "text" ? message.text : undefined };
}

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

function renderSession(canUpdate = true) {
  return renderHook(
    () =>
      useAnalysisPlanningSession({
        organizationId: "org-1",
        factoryId: "factory-1",
        workOrderId: "order-1",
        enabled: true,
        canUpdate,
      }),
    { wrapper },
  );
}

function hang<T>() {
  let finish: (value: T) => void = () => {};
  const promise = new Promise<T>((resolve) => {
    finish = resolve;
  });
  return { promise, finish: (value: T) => finish(value) };
}

describe("useAnalysisPlanningSession stop", () => {
  afterEach(() => {
    vi.mocked(findPlanningSessionByWorkOrder).mockReset();
    vi.mocked(sendPlanningSessionMessage).mockReset();
    vi.mocked(endPlanningSession).mockReset();
  });

  it("does not expose Stop when the user cannot update the work order", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(running);
    const { result } = renderSession(false);

    await waitFor(() => expect(result.current.showChat).toBe(true));
    expect(result.current.canStop).toBe(false);
    expect(result.current.onStop).toBeUndefined();
    expect(endPlanningSession).not.toHaveBeenCalled();
  });

  it("does not stop while a send is in flight", async () => {
    const pending = hang<typeof running>();
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(running);
    vi.mocked(sendPlanningSessionMessage).mockImplementation(() => pending.promise);
    const { result } = renderSession();

    await waitFor(() => expect(result.current.canSend).toBe(true));
    act(() => result.current.onComposerChange("Use this image."));
    act(() => {
      void result.current.onSend();
    });
    await waitFor(() => expect(result.current.canStop).toBe(false));
    await act(async () => {
      await result.current.onStop?.();
    });

    expect(endPlanningSession).not.toHaveBeenCalled();
    await act(async () => {
      pending.finish(running);
    });
  });

  it("does not send while a stop is in flight", async () => {
    const pending = hang<typeof running>();
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(running);
    vi.mocked(endPlanningSession).mockImplementation(() => pending.promise);
    const { result } = renderSession();

    await waitFor(() => expect(result.current.canStop).toBe(true));
    act(() => {
      void result.current.onStop?.();
    });
    await waitFor(() => expect(result.current.canSend).toBe(false));
    act(() => result.current.onComposerChange("Use this image."));
    await act(async () => {
      await result.current.onSend();
    });

    expect(sendPlanningSessionMessage).not.toHaveBeenCalled();
    await act(async () => {
      pending.finish({ ...running, state: "ended" });
    });
  });

  it("stops the current analysis turn without closing the chat", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(running);
    vi.mocked(endPlanningSession).mockResolvedValue({ ...running, state: "ended" });
    const { result } = renderSession();

    await waitFor(() => expect(result.current.showChat).toBe(true));
    act(() => result.current.onComposerChange("Keep this note."));
    await act(async () => {
      await result.current.onStop?.();
    });

    expect(endPlanningSession).toHaveBeenCalledWith("org-1", "factory-1", "session-1");
    expect(result.current.showChat).toBe(true);
    expect(result.current.composer).toBe("Keep this note.");
    expect(sendPlanningSessionMessage).not.toHaveBeenCalled();
  });

  it("lets the user send after stop so a new run can continue the chat", async () => {
    const ended = { ...running, state: "ended" };
    const continued = {
      ...ended,
      state: "running",
      messages: [...running.messages, { role: "user", text: "Use the existing form." }],
    };
    vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValue(running);
    vi.mocked(endPlanningSession).mockResolvedValue(ended);
    vi.mocked(sendPlanningSessionMessage).mockResolvedValue(continued);
    const { result } = renderSession();

    await waitFor(() => expect(result.current.showChat).toBe(true));
    await act(async () => {
      await result.current.onStop?.();
    });
    expect(result.current.canSend).toBe(true);
    act(() => result.current.onComposerChange("Use the existing form."));
    await act(async () => {
      await result.current.onSend();
    });

    expect(sendPlanningSessionMessage).toHaveBeenCalledWith(
      "org-1",
      "factory-1",
      "session-1",
      "Use the existing form.",
    );
    expect(result.current.view.messages.map(planningTalk)).toEqual([
      { role: "user", text: "Add a screenshot." },
      { role: "user", text: "Use the existing form." },
    ]);
  });
});
