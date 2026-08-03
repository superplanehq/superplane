import { usePermissions } from "@/contexts/usePermissions";
import { useFactory, useWorkOrder } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { factoryDetailPath } from "./factoryPagePaths";
import { resolveWorkOrderDetailRedirect } from "./factoryPageRedirects";
import { useWorkOrderDetailActions } from "./useWorkOrderDetailActions";
import { getWorkOrderDetailDerived } from "./workOrderProgress";

export function useWorkOrderDetailPage(organizationId: string, factoryId: string, orderId: string) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const { data: order, isLoading: orderLoading, error: orderError } = useWorkOrder(organizationId, factoryId, orderId);

  const actions = useWorkOrderDetailActions(organizationId, factoryId, orderId);
  const derived = getWorkOrderDetailDerived(order);

  usePageTitle([order?.title ?? "Work Order", factory?.name ?? "Factory"]);

  const isLoading = factoryLoading || orderLoading;
  const canManageWorkOrders = canAct("factories", "update");

  useReportPageReady(!isLoading && Boolean(factory && order), {
    failed: Boolean(factoryError || orderError),
  });

  const redirect = resolveWorkOrderDetailRedirect({
    factoryLoading,
    factoryError,
    orderLoading,
    orderError,
    organizationId,
    factoryId,
  });
  if (redirect) {
    return { kind: "redirect" as const, element: redirect };
  }

  return {
    kind: "ready" as const,
    factory,
    order,
    factoryHref: factoryDetailPath(organizationId, factoryId),
    organizationId,
    factoryLines: factory?.lines ?? [],
    isLoading,
    canDispatch: canManageWorkOrders,
    canClose: canManageWorkOrders,
    canAssign: canManageWorkOrders,
    permissionsLoading,
    ...derived,
    ...actions,
  };
}
