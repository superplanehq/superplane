import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type * as ApiClient from "@/api-client";

const { closeMutateAsync, updateMutateAsync, dispatchMutateAsync, cancelRunMock } = vi.hoisted(() => ({
  closeMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  dispatchMutateAsync: vi.fn(),
  cancelRunMock: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesCancelRun: (...args: unknown[]) => cancelRunMock(...args),
  };
});

vi.mock("@/hooks/useFactoryData", () => ({
  useCloseWorkOrder: () => ({ mutateAsync: closeMutateAsync, isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: dispatchMutateAsync, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useSplitRunFooterActions } from "./useSplitRunFooterActions";

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>;
}

describe("useSplitRunFooterActions", () => {
  beforeEach(() => {
    closeMutateAsync.mockReset().mockResolvedValue({});
    updateMutateAsync.mockReset().mockResolvedValue({});
    dispatchMutateAsync.mockReset().mockResolvedValue({});
    cancelRunMock.mockReset().mockResolvedValue({});
    vi.mocked(showSuccessToast).mockReset();
    vi.mocked(showErrorToast).mockReset();
  });

  it("closes as rejected for Reject", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("canceled", { kind: "waiting" });

    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_REJECTED" });
    expect(cancelRunMock).not.toHaveBeenCalled();
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as failed.");
  });

  it("closes as completed for Stop and Complete", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("completed", { kind: "failed" });

    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_COMPLETED" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as completed.");
  });

  it("reruns the current step", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("rerun-step", {
      kind: "waiting",
      lineName: "Software delivery",
      stepIndex: 1,
    });

    expect(dispatchMutateAsync).toHaveBeenCalledWith({
      orderId: "wo-1",
      lineName: "Software delivery",
      startStepIndex: 1,
      replaceActive: true,
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order step started again.");
  });

  it("reruns from the first step", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("rerun-start", {
      kind: "waiting",
      lineName: "Software delivery",
      stepIndex: 1,
    });

    expect(dispatchMutateAsync).toHaveBeenCalledWith({
      orderId: "wo-1",
      lineName: "Software delivery",
      startStepIndex: 0,
      replaceActive: true,
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order started from the first step.");
  });

  it("reopens a closed work order", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("reopen", { kind: "failed", status: "failed" });

    expect(updateMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", state: "STATE_OPEN" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order reopened.");
  });

  it("cancels the canvas run before closing a running order", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("canceled", {
      kind: "running",
      run: { appId: "app-implement", runId: "run-9" },
    });

    expect(cancelRunMock).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "app-implement", runId: "run-9" },
        headers: expect.objectContaining({ "x-organization-id": "org-1" }),
      }),
    );
    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_REJECTED" });
  });

  it("closes a draft from Reject", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    const deleted = await result.current.handleReject();

    expect(deleted).toBe(true);
    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_REJECTED" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as failed.");
  });

  it("does not mutate when the popup has no live order", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions(), { wrapper });

    await result.current.handleStop("canceled", { kind: "running" });
    await result.current.handleReject();

    expect(closeMutateAsync).not.toHaveBeenCalled();
    expect(updateMutateAsync).not.toHaveBeenCalled();
    expect(cancelRunMock).not.toHaveBeenCalled();
  });

  it("keeps the work order open when cancel fails", async () => {
    cancelRunMock.mockRejectedValue(new Error("run still busy"));
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("canceled", {
      kind: "running",
      run: { appId: "app-implement", runId: "run-9" },
    });

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalled();
    });
    expect(closeMutateAsync).not.toHaveBeenCalled();
  });
});
