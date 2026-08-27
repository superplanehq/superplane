import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import {
  useAddWorkOrderComment,
  useCloseWorkOrder,
  useDispatchWorkOrder,
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

  const handleAssigneesSave = async (nextAssigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: nextAssigneeIds });
      showSuccessToast("Owner updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update the owner"));
      throw error;
    }
  };

  const handleDispatch = async ({ lineName }: { lineName: string }) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName });
    showSuccessToast(`Dispatched to ${lineName}.`);
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

  const handleAddComment = async (body: string, mentionedUserIds: string[]) => {
    try {
      await addComment.mutateAsync({ orderId, body, mentionedUserIds });
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
    isDispatching: dispatchWorkOrder.isPending,
    isCompleting,
    isRejecting,
    isClosing: closeWorkOrder.isPending,
    isAssigneesSaving: updateAssignees.isPending,
    isUpdatingStatus: updateStatus.isPending,
    isAddingComment: addComment.isPending,
  };
}
