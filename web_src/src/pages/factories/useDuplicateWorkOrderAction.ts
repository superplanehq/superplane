import { useDuplicateWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

export function useDuplicateWorkOrderAction(
  organizationId: string | undefined,
  factoryId: string | undefined,
  onCreated?: (orderNumber: string) => void,
) {
  const duplicateWorkOrder = useDuplicateWorkOrder(organizationId ?? "", factoryId ?? "");

  const handleDuplicate = async (orderId?: string) => {
    if (!orderId) {
      return;
    }
    try {
      const copied = await duplicateWorkOrder.mutateAsync(orderId);
      showSuccessToast("Task duplicated.");
      if (copied.number) {
        onCreated?.(String(copied.number));
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to duplicate task."));
    }
  };

  return { handleDuplicate, isDuplicating: duplicateWorkOrder.isPending };
}
