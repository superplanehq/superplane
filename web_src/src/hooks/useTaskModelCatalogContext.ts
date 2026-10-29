import { usePermissions } from "@/contexts/usePermissions";
import { useWorkOrder } from "./useFactoryData";

export type TaskModelCatalogContext = { orderId?: string; ownerId?: string; memberId?: string };

export function useTaskModelCatalogContext(organizationId = "", factoryId = "", orderId = ""): TaskModelCatalogContext {
  const { currentUserId } = usePermissions();
  const order = useWorkOrder(orderId ? organizationId : "", factoryId, orderId);
  return { orderId, ownerId: order.data?.assignees?.map((owner) => owner.id).join(","), memberId: currentUserId };
}
