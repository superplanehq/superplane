import type { FactoriesWorkOrderResult } from "@/api-client";
import { useCloseWorkOrder, useDispatchWorkOrder, useUpdateWorkOrderAssignees } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { formatWorkOrderResult } from "./workOrderPresentation";

export function useWorkOrderDetailActions(organizationId: string, factoryId: string, orderId: string) {
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const closeWorkOrder = useCloseWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);

  const handleAssigneesSave = async (nextAssigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: nextAssigneeIds });
      showSuccessToast("Assignees updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update assignees"));
      throw error;
    }
  };

  const handleDispatch = async (lineName: string) => {
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

  const isCompleting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_COMPLETED";
  const isRejecting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_REJECTED";

  return {
    handleAssigneesSave,
    handleDispatch,
    handleClose,
    isDispatching: dispatchWorkOrder.isPending,
    isCompleting,
    isRejecting,
    isClosing: closeWorkOrder.isPending,
    isAssigneesSaving: updateAssignees.isPending,
  };
}
