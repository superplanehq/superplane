import { useDispatchWorkOrder, useUpdateWorkOrderAssignees } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback } from "react";

/** Shared mutation state and callbacks for every surface that renders work order cards. */
export function useWorkOrderCardActions(organizationId: string, factoryId: string) {
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);

  const onDispatch = useCallback(
    async (orderId: string, input: { lineName: string }) => {
      try {
        await dispatchWorkOrder.mutateAsync({ orderId, lineName: input.lineName });
        showSuccessToast(`Dispatched to ${input.lineName}.`);
      } catch {
        showErrorToast("Failed to dispatch work order.");
      }
    },
    [dispatchWorkOrder],
  );

  const onAssigneesSave = useCallback(
    async (orderId: string, assigneeIds: string[]) => {
      try {
        await updateAssignees.mutateAsync({ orderId, assigneeIds });
        showSuccessToast("Owners updated.");
      } catch {
        showErrorToast("Failed to update owners.");
      }
    },
    [updateAssignees],
  );

  return {
    isDispatching: dispatchWorkOrder.isPending,
    isAssigneesSaving: updateAssignees.isPending,
    onDispatch,
    onAssigneesSave,
  };
}
