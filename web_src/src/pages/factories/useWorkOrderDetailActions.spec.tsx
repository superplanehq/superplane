import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { addReactionMutate, removeReactionMutate } = vi.hoisted(() => ({
  addReactionMutate: vi.fn(),
  removeReactionMutate: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false, variables: undefined }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useAddWorkOrderComment: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderReactions", () => ({
  useAddWorkOrderReaction: () => ({ mutateAsync: addReactionMutate, isPending: false }),
  useRemoveWorkOrderReaction: () => ({ mutateAsync: removeReactionMutate, isPending: false }),
}));

const { showErrorToast } = vi.hoisted(() => ({ showErrorToast: vi.fn() }));
vi.mock("@/lib/toast", () => ({
  showErrorToast,
  showSuccessToast: vi.fn(),
}));

import { useWorkOrderDetailActions } from "./useWorkOrderDetailActions";

describe("useWorkOrderDetailActions - handleToggleReaction", () => {
  beforeEach(() => {
    addReactionMutate.mockReset();
    removeReactionMutate.mockReset();
    showErrorToast.mockReset();
  });

  it("adds a reaction when the caller hasn't reacted yet", async () => {
    addReactionMutate.mockResolvedValue([]);

    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "order-1"));

    await result.current.handleToggleReaction("+1", false);

    expect(addReactionMutate).toHaveBeenCalledWith({ orderId: "order-1", content: "+1" });
    expect(removeReactionMutate).not.toHaveBeenCalled();
  });

  it("removes a reaction when the caller already reacted", async () => {
    removeReactionMutate.mockResolvedValue([]);

    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "order-1"));

    await result.current.handleToggleReaction("+1", true);

    expect(removeReactionMutate).toHaveBeenCalledWith({ orderId: "order-1", content: "+1" });
    expect(addReactionMutate).not.toHaveBeenCalled();
  });

  it("shows an error toast and swallows the error when the mutation fails", async () => {
    addReactionMutate.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "order-1"));

    await expect(result.current.handleToggleReaction("heart", false)).resolves.toBeUndefined();

    await waitFor(() => expect(showErrorToast).toHaveBeenCalled());
  });
});
