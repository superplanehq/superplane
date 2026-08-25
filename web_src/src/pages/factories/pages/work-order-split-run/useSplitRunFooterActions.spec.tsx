import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type * as ApiClient from "@/api-client";

const { closeMutateAsync, updateMutateAsync, cancelRunMock } = vi.hoisted(() => ({
  closeMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
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
    cancelRunMock.mockReset().mockResolvedValue({});
    vi.mocked(showSuccessToast).mockReset();
    vi.mocked(showErrorToast).mockReset();
  });

  it("closes as rejected for Stop as Canceled", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("canceled", { kind: "waiting" });

    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_REJECTED" });
    expect(cancelRunMock).not.toHaveBeenCalled();
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as canceled.");
  });

  it("closes as completed for Stop as Completed", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("completed", { kind: "failed" });

    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_COMPLETED" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as completed.");
  });

  it("moves to draft for Stop and return to Draft", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleStop("draft", { kind: "waiting" });

    expect(updateMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", state: "STATE_DRAFT" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order moved to draft.");
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

  it("closes a draft as canceled from Reject", async () => {
    const { result } = renderHook(() => useSplitRunFooterActions("org-1", "factory-1", "wo-1"), { wrapper });

    await result.current.handleReject();

    expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_REJECTED" });
    expect(showSuccessToast).toHaveBeenCalledWith("Work order closed as canceled.");
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
