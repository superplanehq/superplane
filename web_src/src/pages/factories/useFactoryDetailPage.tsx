import { usePermissions } from "@/contexts/usePermissions";
import { useFactory, useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useMemo, useState } from "react";
import { getFactoryDetailLoadingState } from "./lib/factoryDetailLoading";
import { createWorkOrderPath, factoryDetailPath } from "./lib/factoryPagePaths";
import { resolveFactoryLoadRedirect } from "./factoryPageRedirects";
import { useFactoryDetailActions } from "./useFactoryDetailActions";
import {
  countOpenWorkOrders,
  filterWorkOrdersByOwner,
  filterWorkOrdersByStatus,
  type WorkOrderOwnerFilter,
  type WorkOrderStatusFilter,
} from "./lib/workOrderProgress";

export function useFactoryDetailPage(organizationId: string, factoryId: string) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const [createAppOpen, setCreateAppOpen] = useState(false);
  const [ownerFilter, setOwnerFilter] = useState<WorkOrderOwnerFilter>("mine");
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatusFilter>("open");

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const {
    data: workOrders = [],
    isLoading: ordersLoading,
    isFetching: ordersFetching,
    error: ordersError,
  } = useFactoryWorkOrders(organizationId, factoryId);
  const {
    data: factoryApps = [],
    isLoading: appsLoading,
    isFetching: appsFetching,
  } = useFactoryApps(organizationId, factoryId);

  const actions = useFactoryDetailActions(organizationId, factoryId, setCreateAppOpen);

  usePageTitle(factory?.name ? [factory.name, "Factories"] : ["Factory"]);

  const { isLoading, isOrdersLoading, isAppsLoading } = getFactoryDetailLoadingState({
    factoryLoading,
    ordersLoading,
    ordersFetching,
    workOrderCount: workOrders.length,
    appsLoading,
    appsFetching,
    factoryAppCount: factoryApps.length,
  });

  const filteredWorkOrders = useMemo(() => {
    const byOwner = filterWorkOrdersByOwner(workOrders, ownerFilter, me?.id);
    return filterWorkOrdersByStatus(byOwner, statusFilter);
  }, [workOrders, ownerFilter, statusFilter, me?.id]);

  useReportPageReady(!isLoading && Boolean(factory), {
    work_order_count: workOrders.length,
    failed: Boolean(factoryError || ordersError),
  });

  const redirect = resolveFactoryLoadRedirect(factoryLoading, factoryError, organizationId);
  if (redirect) {
    return { kind: "redirect" as const, element: redirect };
  }

  const factoryHref = factoryDetailPath(organizationId, factoryId);

  return {
    kind: "ready" as const,
    organizationId,
    factoryId,
    factory,
    factoryLoading,
    factoryHref,
    createWorkOrderHref: createWorkOrderPath(organizationId, factoryId),
    factoryApps,
    filteredWorkOrders,
    openWorkOrderCount: countOpenWorkOrders(workOrders),
    ownerFilter,
    statusFilter,
    setOwnerFilter,
    setStatusFilter,
    ordersError,
    isOrdersLoading,
    isAppsLoading,
    canCreateWork: canAct("factories", "create"),
    canUpdateFactory: canAct("factories", "update"),
    canCreateApps: canAct("canvases", "create"),
    canDispatch: canAct("factories", "update"),
    permissionsLoading,
    createAppOpen,
    setCreateAppOpen,
    ...actions,
  };
}
