import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  useAddWorkOrderComment,
  useCloseWorkOrder,
  useDispatchWorkOrder,
  useUpdateWorkOrderAssignees,
  useUpdateWorkOrderStatus,
} from "@/hooks/useFactoryData";
import { showSuccessToast } from "@/lib/toast";
import { useWorkOrderDetailActions } from "./useWorkOrderDetailActions";

vi.mock("@/hooks/useFactoryData", () => ({
  useAddWorkOrderComment: vi.fn(),
  useCloseWorkOrder: vi.fn(),
  useDispatchWorkOrder: vi.fn(),
  useUpdateWorkOrderAssignees: vi.fn(),
  useUpdateWorkOrderStatus: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

function idleMutation(mutateAsync = vi.fn().mockResolvedValue(undefined)) {
  return { mutateAsync, isPending: false, variables: undefined };
}

function mockMutations() {
  const dispatch = vi.fn().mockResolvedValue(undefined);
  const close = vi.fn().mockResolvedValue(undefined);
  const updateStatus = vi.fn().mockResolvedValue(undefined);
  const addComment = vi.fn().mockResolvedValue(undefined);

  vi.mocked(useDispatchWorkOrder).mockReturnValue(
    idleMutation(dispatch) as unknown as ReturnType<typeof useDispatchWorkOrder>,
  );
  vi.mocked(useCloseWorkOrder).mockReturnValue(idleMutation(close) as unknown as ReturnType<typeof useCloseWorkOrder>);
  vi.mocked(useUpdateWorkOrderAssignees).mockReturnValue(
    idleMutation() as unknown as ReturnType<typeof useUpdateWorkOrderAssignees>,
  );
  vi.mocked(useUpdateWorkOrderStatus).mockReturnValue(
    idleMutation(updateStatus) as unknown as ReturnType<typeof useUpdateWorkOrderStatus>,
  );
  vi.mocked(useAddWorkOrderComment).mockReturnValue(
    idleMutation(addComment) as unknown as ReturnType<typeof useAddWorkOrderComment>,
  );

  return { dispatch, close, updateStatus, addComment };
}

describe("useWorkOrderDetailActions", () => {
  beforeEach(() => {
    vi.mocked(showSuccessToast).mockClear();
  });

  it("dispatches the task without a success toast", async () => {
    const { dispatch } = mockMutations();
    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "wo-1"));

    await act(async () => {
      await result.current.handleDispatch({ lineName: "hotfix" });
    });

    expect(dispatch).toHaveBeenCalledWith({ orderId: "wo-1", lineName: "hotfix" });
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it("shows a success toast when the task is closed", async () => {
    const { close } = mockMutations();
    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "wo-1"));

    await act(async () => {
      await result.current.handleClose("RESULT_COMPLETED");
    });

    expect(close).toHaveBeenCalledWith({ orderId: "wo-1", result: "RESULT_COMPLETED" });
    expect(showSuccessToast).toHaveBeenCalledWith("Task closed as completed.");
  });

  it("shows a success toast when the task status changes", async () => {
    const { updateStatus } = mockMutations();
    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "wo-1"));

    await act(async () => {
      await result.current.handleStatusChange("STATE_OPEN");
    });

    expect(updateStatus).toHaveBeenCalledWith({ orderId: "wo-1", state: "STATE_OPEN", result: undefined });
    expect(showSuccessToast).toHaveBeenCalledWith("Task moved to open.");
  });

  it("shows a success toast when a comment is added", async () => {
    const { addComment } = mockMutations();
    const { result } = renderHook(() => useWorkOrderDetailActions("org-1", "factory-1", "wo-1"));

    await act(async () => {
      await result.current.handleAddComment("Ship it", ["user-1"]);
    });

    expect(addComment).toHaveBeenCalledWith({ orderId: "wo-1", body: "Ship it", mentionedUserIds: ["user-1"] });
    expect(showSuccessToast).toHaveBeenCalledWith("Comment added.");
  });
});
