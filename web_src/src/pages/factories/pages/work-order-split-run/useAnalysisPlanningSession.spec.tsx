import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import {
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(),
  sendPlanningSessionMessage: vi.fn(),
  endPlanningSession: vi.fn(),
}));

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

describe("useAnalysisPlanningSession", () => {
  afterEach(() => {
    vi.mocked(findPlanningSessionByWorkOrder).mockReset();
    vi.mocked(sendPlanningSessionMessage).mockReset();
  });

  it("keeps chat closed when the draft has no analysis session", async () => {
    vi.mocked(findPlanningSessionByWorkOrder).mockRejectedValue(new Error("missing"));

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
  });
});
