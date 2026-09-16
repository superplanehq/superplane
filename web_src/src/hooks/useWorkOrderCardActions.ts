import { useDispatchWorkOrder, useUpdateWorkOrderAssignees, useUpdateWorkOrderStatus } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useState } from "react";

const NO_ORDERS: ReadonlySet<string> = new Set();

/** Shared mutation state and callbacks for every surface that renders task cards. */
export function useWorkOrderCardActions(organizationId: string, factoryId: string) {
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);
  const updateStatus = useUpdateWorkOrderStatus(organizationId, factoryId);
  // The mutation is shared by every card on the page, so its pending flag
  // cannot say which card the user clicked. Track the tasks in flight
  // instead, so only their controls show a busy state.
  const [dispatchingOrderIds, setDispatchingOrderIds] = useState<ReadonlySet<string>>(NO_ORDERS);
  const [cancelingOrderIds, setCancelingOrderIds] = useState<ReadonlySet<string>>(NO_ORDERS);

  const onDispatch = useCallback(
    async (orderId: string, input: { lineName: string; model?: string }) => {
      setDispatchingOrderIds((current) => withOrderId(current, orderId));
      try {
        await dispatchWorkOrder.mutateAsync({ orderId, lineName: input.lineName, model: input.model });
        showSuccessToast(`Dispatched to ${input.lineName}.`);
      } catch {
        showErrorToast("Failed to dispatch task.");
      } finally {
        setDispatchingOrderIds((current) => withoutOrderId(current, orderId));
      }
    },
    [dispatchWorkOrder],
  );

  const onCancelQueue = useCallback(
    async (orderId: string) => {
      setCancelingOrderIds((current) => withOrderId(current, orderId));
      try {
        await updateStatus.mutateAsync({ orderId, state: "STATE_DRAFT" });
        showSuccessToast("Removed from the queue.");
      } catch {
        showErrorToast("Failed to cancel the queue.");
      } finally {
        setCancelingOrderIds((current) => withoutOrderId(current, orderId));
      }
    },
    [updateStatus],
  );

  const onAssigneesSave = useCallback(
    async (orderId: string, assigneeIds: string[]) => {
      try {
        await updateAssignees.mutateAsync({ orderId, assigneeIds });
        showSuccessToast("Owner updated.");
      } catch {
        showErrorToast("Failed to update the owner.");
      }
    },
    [updateAssignees],
  );

  return {
    dispatchingOrderIds,
    cancelingOrderIds,
    isAssigneesSaving: updateAssignees.isPending,
    onDispatch,
    onCancelQueue,
    onAssigneesSave,
  };
}

function withOrderId(orderIds: ReadonlySet<string>, orderId: string): ReadonlySet<string> {
  const next = new Set(orderIds);
  next.add(orderId);
  return next;
}

function withoutOrderId(orderIds: ReadonlySet<string>, orderId: string): ReadonlySet<string> {
  if (!orderIds.has(orderId)) {
    return orderIds;
  }
  const next = new Set(orderIds);
  next.delete(orderId);
  return next;
}
