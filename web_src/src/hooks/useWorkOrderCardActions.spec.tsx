import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { showErrorToast, showSuccessToast } from "@/lib/toast";
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
  beforeEach(() => {
    vi.mocked(showErrorToast).mockClear();
    vi.mocked(showSuccessToast).mockClear();
  });

  it("reports only the task that dispatches as in flight", async () => {
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
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it.each([
    {
      name: "shows the model availability error from the server",
      error: {
        code: 9,
        message: "the selected model is unavailable for this task owner; select another model or clear the override",
        details: [],
      },
      expectedMessage:
        "the selected model is unavailable for this task owner; select another model or clear the override",
    },
    {
      name: "uses a fallback for network failures",
      error: new Error("Failed to fetch"),
      expectedMessage: "Failed to dispatch task.",
    },
  ])("$name and clears the in-flight task", async ({ error, expectedMessage }) => {
    mockMutations();
    vi.mocked(useDispatchWorkOrder).mockReturnValue({
      mutateAsync: vi.fn().mockRejectedValue(error),
    } as unknown as DispatchMutation);
    const { result } = renderHook(() => useWorkOrderCardActions("org-1", "factory-1"));

    await act(async () => {
      await result.current.onDispatch("wo-1", { lineName: "hotfix" });
    });

    expect(result.current.dispatchingOrderIds.has("wo-1")).toBe(false);
    expect(showErrorToast).toHaveBeenCalledWith(expectedMessage);
    expect(showSuccessToast).not.toHaveBeenCalled();
  });
});
