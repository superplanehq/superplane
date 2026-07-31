import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import {
  factoryAppsKey,
  useDispatchWorkOrder,
  useFactory,
  useFactoryApps,
  useFactoryWorkOrders,
  type FactoriesFactoryLine,
} from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { appPath } from "@/lib/appPaths";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useMemo, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { CreateFactoryAppDialog } from "./CreateFactoryAppDialog";
import { DispatchWorkOrderDialog } from "./DispatchWorkOrderDialog";
import { FactoryAppsSidebar } from "./FactoryAppsSidebar";
import { FactoryDetailHeader } from "./FactoryDetailHeader";
import { FactoryLinesSidebar } from "./FactoryLinesSidebar";
import { FactoryPageShell } from "./FactoryPageShell";
import { WorkOrderCard } from "./WorkOrderCard";
import { WorkOrderFilters } from "./WorkOrderFilters";
import {
  factoryCountBadgeClassName,
  factoryDetailMainClassName,
  factoryPageContentClassName,
  factoryDetailPanelClassName,
  factoryDetailSidebarClassName,
} from "./factoryPageStyles";
import {
  countOpenWorkOrders,
  filterWorkOrdersByOwner,
  filterWorkOrdersByStatus,
  type WorkOrderOwnerFilter,
  type WorkOrderStatusFilter,
} from "./workOrderProgress";

function getPipelineLabels(lines: FactoriesFactoryLine[]): string[] {
  if (lines.length === 0) {
    return [];
  }

  const labels = lines.flatMap((line) => {
    const lineName = line.name?.trim();
    if (!lineName) {
      return [];
    }
    return [`${lineName} pipeline`];
  });

  return labels.slice(0, 2);
}

export function FactoryDetailPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const [createAppOpen, setCreateAppOpen] = useState(false);
  const [ownerFilter, setOwnerFilter] = useState<WorkOrderOwnerFilter>("all");
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatusFilter>("all");
  const [dispatchOrderId, setDispatchOrderId] = useState<string | null>(null);

  const {
    data: factory,
    isLoading: factoryLoading,
    error: factoryError,
  } = useFactory(organizationId ?? "", factoryId ?? "");
  const {
    data: workOrders = [],
    isLoading: ordersLoading,
    isFetching: ordersFetching,
    error: ordersError,
  } = useFactoryWorkOrders(organizationId ?? "", factoryId ?? "");
  const {
    data: factoryApps = [],
    isLoading: appsLoading,
    isFetching: appsFetching,
  } = useFactoryApps(organizationId ?? "", factoryId ?? "");

  const dispatchWorkOrder = useDispatchWorkOrder(organizationId ?? "", factoryId ?? "");
  const createCanvas = useCreateCanvas(organizationId ?? "");

  usePageTitle(factory?.name ? [factory.name, "Factories"] : ["Factory"]);

  const canCreateWork = canAct("factories", "create");
  const canUpdateFactory = canAct("factories", "update");
  const canCreateApps = canAct("canvases", "create");
  const canDispatch = canAct("factories", "update");
  const isLoading = factoryLoading || (ordersLoading && workOrders.length === 0);
  const isOrdersLoading = ordersLoading || (ordersFetching && workOrders.length === 0);
  const isAppsLoading = appsLoading || (appsFetching && factoryApps.length === 0);

  const factoryLines = factory?.lines ?? [];
  const openWorkOrders = useMemo(() => workOrders.filter((order) => order.state === "STATE_OPEN"), [workOrders]);
  const filteredWorkOrders = useMemo(() => {
    const byOwner = filterWorkOrdersByOwner(openWorkOrders, ownerFilter, me?.id);
    return filterWorkOrdersByStatus(byOwner, statusFilter);
  }, [openWorkOrders, ownerFilter, statusFilter, me?.id]);
  const pipelineLabels = useMemo(() => getPipelineLabels(factoryLines), [factoryLines]);
  const openWorkOrderCount = countOpenWorkOrders(workOrders);

  useReportPageReady(!isLoading && Boolean(factory), {
    work_order_count: workOrders.length,
    failed: Boolean(factoryError || ordersError),
  });

  if (!organizationId || !factoryId) {
    return null;
  }

  if (!factoryLoading && factoryError) {
    return <Navigate to={`/${organizationId}/factories`} replace />;
  }

  const factoryHref = `/${organizationId}/factories/${factoryId}`;
  const createWorkOrderHref = `${factoryHref}/orders/new`;

  const handleCreateApp = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateAppOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (canvasId) {
        showSuccessToast("Factory app created.");
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create factory app"));
      throw error;
    }
  };

  const handleDispatch = async (lineName: string) => {
    if (!dispatchOrderId) {
      return;
    }

    await dispatchWorkOrder.mutateAsync({ orderId: dispatchOrderId, lineName });
    setDispatchOrderId(null);
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  return (
    <FactoryPageShell backHref={`/${organizationId}/factories`} backLabel="Factories">
      {factoryLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading factory…</Text>
        </div>
      ) : factory ? (
        <>
          <div className={cn(factoryPageContentClassName, "pb-10")}>
            <FactoryDetailHeader
              factory={factory}
              workOrdersCount={openWorkOrderCount}
              canCreate={canCreateWork}
              permissionsLoading={permissionsLoading}
              createHref={createWorkOrderHref}
            />

            <div className={cn(factoryDetailPanelClassName, "mt-8 grid w-full lg:grid-cols-[minmax(0,1fr)_320px]")}>
              <section className={factoryDetailMainClassName}>
                <div className="mb-6">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Work Orders</h2>
                    <span className={factoryCountBadgeClassName}>{openWorkOrderCount}</span>
                  </div>
                </div>

                <WorkOrderFilters
                  ownerFilter={ownerFilter}
                  statusFilter={statusFilter}
                  onOwnerFilterChange={setOwnerFilter}
                  onStatusFilterChange={setStatusFilter}
                />

                <div className="mt-6">
                  {ordersError ? (
                    <div className="rounded-lg border border-red-300 px-4 py-3 text-red-500 dark:border-red-800 dark:text-red-400">
                      <Text>Failed to load work orders.</Text>
                    </div>
                  ) : isOrdersLoading ? (
                    <Text className="text-sm text-gray-500">Loading work orders…</Text>
                  ) : filteredWorkOrders.length === 0 ? (
                    <p className="text-sm font-medium text-gray-400 dark:text-gray-400 px-6 py-5 text-center">
                      No work orders match these filters.<br />
                      Create a work order or adjust filters to see more results.
                    </p>
                  ) : (
                    <div>
                      {filteredWorkOrders.map((order) => (
                        <WorkOrderCard
                          key={order.id}
                          order={order}
                          factoryHref={factoryHref}
                          pipelineLabels={pipelineLabels}
                          canDispatch={canDispatch}
                          onDispatch={(orderId) => setDispatchOrderId(orderId)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              </section>

              <div className={factoryDetailSidebarClassName}>
                <FactoryAppsSidebar
                  organizationId={organizationId}
                  apps={factoryApps}
                  isLoading={isAppsLoading}
                  canCreate={canCreateApps}
                  permissionsLoading={permissionsLoading}
                  onCreateClick={() => setCreateAppOpen(true)}
                >
                  <FactoryLinesSidebar factoryHref={factoryHref} lines={factoryLines} canUpdate={canUpdateFactory} />
                </FactoryAppsSidebar>
              </div>
            </div>
          </div>
        </>
      ) : null}

      <CreateFactoryAppDialog
        open={createAppOpen}
        isSaving={createCanvas.isPending}
        onClose={() => setCreateAppOpen(false)}
        onCreate={handleCreateApp}
      />

      <DispatchWorkOrderDialog
        open={dispatchOrderId !== null}
        lines={factoryLines}
        isSaving={dispatchWorkOrder.isPending}
        canDispatch={canDispatch || permissionsLoading}
        onClose={() => setDispatchOrderId(null)}
        onDispatch={handleDispatch}
      />
    </FactoryPageShell>
  );
}
