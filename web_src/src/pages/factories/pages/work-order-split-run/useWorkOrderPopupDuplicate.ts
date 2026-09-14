import type { FactoriesWorkOrder } from "@/api-client";
import { useDuplicateWorkOrder } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

export function useWorkOrderPopupDuplicate({
  organizationId,
  factoryId,
  orderId,
  canCreate = false,
  onOpenWorkOrder,
}: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canCreate?: boolean;
  onOpenWorkOrder?: (orderId: string, order: FactoriesWorkOrder) => void;
}) {
  const refinementEnabled = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_CREATE_WITH_AGENT);
  const duplicate = useDuplicateWorkOrder(organizationId ?? "", factoryId ?? "");
  const canDuplicate = canCreate && refinementEnabled && Boolean(organizationId && factoryId && orderId);

  const onDuplicate = async () => {
    if (!orderId || !canDuplicate) {
      return;
    }
    try {
      const order = await duplicate.mutateAsync(orderId);
      if (!order.id) {
        return;
      }
      showSuccessToast("Task duplicated.");
      onOpenWorkOrder?.(order.id, order);
    } catch {
      showErrorToast("Failed to duplicate task.");
    }
  };

  return { canDuplicate, onDuplicate, busy: duplicate.isPending };
}
