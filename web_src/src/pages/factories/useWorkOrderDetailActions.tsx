import type { FactoriesWorkOrderApprovalStatus, FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import {
  useAddWorkOrderComment,
  useCloseWorkOrder,
  useDispatchWorkOrder,
  useResolveWorkOrderApproval,
  useUpdateWorkOrderAssignees,
  useUpdateWorkOrderStatus,
} from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { formatWorkOrderResult, formatWorkOrderState } from "./lib/workOrderPresentation";

export function useWorkOrderDetailActions(organizationId: string, factoryId: string, orderId: string) {
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const closeWorkOrder = useCloseWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);
  const updateStatus = useUpdateWorkOrderStatus(organizationId, factoryId);
  const addComment = useAddWorkOrderComment(organizationId, factoryId);
  const resolveApproval = useResolveWorkOrderApproval(organizationId, factoryId);

  const handleAssigneesSave = async (nextAssigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: nextAssigneeIds });
      showSuccessToast("Assignees updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update assignees"));
      throw error;
    }
  };

  const handleDispatch = async ({ lineName, note }: { lineName: string; note?: string }) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName, note });
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  const handleResolveApproval = async (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => {
    try {
      await resolveApproval.mutateAsync({ orderId, ...input });
      showSuccessToast(input.status === "STATUS_APPROVED" ? "Approval granted." : "Approval rejected.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to resolve approval"));
      throw error;
    }
  };

  const handleClose = async (result: FactoriesWorkOrderResult) => {
    try {
      await closeWorkOrder.mutateAsync({ orderId, result });
      showSuccessToast(`Work order closed as ${formatWorkOrderResult(result).toLowerCase()}.`);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close work order"));
    }
  };

  const handleStatusChange = async (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => {
    try {
      await updateStatus.mutateAsync({ orderId, state, result });
      showSuccessToast(`Work order moved to ${formatWorkOrderState(state).toLowerCase()}.`);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update status"));
      throw error;
    }
  };

  const handleAddComment = async (body: string) => {
    try {
      await addComment.mutateAsync({ orderId, body });
      showSuccessToast("Comment added.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add comment"));
      throw error;
    }
  };

  const isCompleting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_COMPLETED";
  const isRejecting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_REJECTED";

  return {
    handleAssigneesSave,
    handleDispatch,
    handleClose,
    handleStatusChange,
    handleAddComment,
    handleResolveApproval,
    isDispatching: dispatchWorkOrder.isPending,
    isCompleting,
    isRejecting,
    isClosing: closeWorkOrder.isPending,
    isAssigneesSaving: updateAssignees.isPending,
    isUpdatingStatus: updateStatus.isPending,
    isAddingComment: addComment.isPending,
    isResolvingApproval: resolveApproval.isPending,
  };
}
