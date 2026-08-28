import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDispatchWorkOrder, useUpdateWorkOrderAssignees } from "./useFactoryData";
import { useWorkOrderCardActions } from "./useWorkOrderCardActions";

vi.mock("./useFactoryData", () => ({
  useDispatchWorkOrder: vi.fn(),
  useUpdateWorkOrderAssignees: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

type DispatchMutation = ReturnType<typeof useDispatchWorkOrder>;

function mockMutations() {
  let resolveDispatch: (() => void) | undefined;
  const mutateAsync = vi.fn(
    () =>
      new Promise<void>((resolve) => {
        resolveDispatch = resolve;
      }),
  );
  vi.mocked(useDispatchWorkOrder).mockReturnValue({ mutateAsync } as unknown as DispatchMutation);
  vi.mocked(useUpdateWorkOrderAssignees).mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
  } as unknown as ReturnType<typeof useUpdateWorkOrderAssignees>);
  return { finishDispatch: () => resolveDispatch?.() };
}

describe("useWorkOrderCardActions", () => {
  it("reports only the work order that dispatches as in flight", async () => {
    const { finishDispatch } = mockMutations();
    const { result } = renderHook(() => useWorkOrderCardActions("org-1", "factory-1"));

    let dispatched: Promise<void> | undefined;
    act(() => {
      dispatched = result.current.onDispatch("wo-1", { lineName: "hotfix" });
    });

    await waitFor(() => expect(result.current.dispatchingOrderIds.has("wo-1")).toBe(true));
    expect(result.current.dispatchingOrderIds.has("wo-2")).toBe(false);

    await act(async () => {
      finishDispatch();
      await dispatched;
    });

    expect(result.current.dispatchingOrderIds.has("wo-1")).toBe(false);
  });

  it("clears the in-flight work order when the dispatch fails", async () => {
    mockMutations();
    vi.mocked(useDispatchWorkOrder).mockReturnValue({
      mutateAsync: vi.fn().mockRejectedValue(new Error("nope")),
    } as unknown as DispatchMutation);
    const { result } = renderHook(() => useWorkOrderCardActions("org-1", "factory-1"));

    await act(async () => {
      await result.current.onDispatch("wo-1", { lineName: "hotfix" });
    });

    expect(result.current.dispatchingOrderIds.has("wo-1")).toBe(false);
  });
});
