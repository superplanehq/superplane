import { usePermissions } from "@/contexts/usePermissions";
import { useFactory, useWorkOrder, useWorkOrderArtifacts, useWorkOrderEvents } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { useMemo } from "react";
import { Navigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { workOrdersPath } from "../lib/factoryPagePaths";
import { flattenWorkOrderEventsPages } from "../lib/workOrderEventsPagination";
import { getWorkOrderDetailDerived } from "../lib/workOrderProgress";
import { useWorkOrderDetailActions } from "../useWorkOrderDetailActions";
import { WorkOrderDetailLoadedView } from "../WorkOrderDetailLoadedView";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

export function WorkOrderDetailPage() {
  const { orderId } = useParams<{ orderId: string }>();
  const { organizationId, factoryId } = useFactoriesLayout();

  if (!orderId) {
    return null;
  }

  return <WorkOrderDetailPageContent organizationId={organizationId} factoryId={factoryId} orderId={orderId} />;
}

function WorkOrderDetailPageContent({
  organizationId,
  factoryId,
  orderId,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
}) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const { data: order, isLoading: orderLoading, error: orderError } = useWorkOrder(organizationId, factoryId, orderId);
  const eventsQuery = useWorkOrderEvents(organizationId, factoryId, orderId);
  const events = useMemo(() => flattenWorkOrderEventsPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);
  const artifactsQuery = useWorkOrderArtifacts(organizationId, factoryId, orderId);

  const actions = useWorkOrderDetailActions(organizationId, factoryId, orderId);
  const derived = getWorkOrderDetailDerived(order);

  usePageTitle([order?.title ?? "Work Order", factory?.name ?? "Workspace"]);

  const workOrdersHref = workOrdersPath(organizationId, factoryId);

  if (shouldRedirectAfterError({ factoryLoading, factoryError, orderLoading, orderError })) {
    return <Navigate to={workOrdersHref} replace />;
  }

  if (factoryLoading || orderLoading) {
    return (
      <div className={factoryContentBodyClassName}>
        <p className="text-[13px] text-muted-foreground">Loading work order…</p>
      </div>
    );
  }

  if (!factory || !order || !derived.statusMeta || !derived.displayStatus) {
    return null;
  }

  return (
    <LoadedWorkOrderDetail
      factory={factory}
      order={order}
      derived={derived}
      workOrdersHref={workOrdersHref}
      organizationId={organizationId}
      events={events}
      eventsQuery={eventsQuery}
      artifactsQuery={artifactsQuery}
      canManageWorkOrders={canAct("work_orders", "update")}
      permissionsLoading={permissionsLoading}
      actions={actions}
    />
  );
}

function shouldRedirectAfterError(state: {
  factoryLoading: boolean;
  factoryError: unknown;
  orderLoading: boolean;
  orderError: unknown;
}) {
  const factoryFailed = !state.factoryLoading && state.factoryError;
  const orderFailed = !state.orderLoading && state.orderError;
  return Boolean(factoryFailed || orderFailed);
}

interface LoadedWorkOrderDetailProps {
  factory: FactoriesFactory;
  order: FactoriesWorkOrder;
  derived: ReturnType<typeof getWorkOrderDetailDerived>;
  workOrdersHref: string;
  organizationId: string;
  events: ReturnType<typeof flattenWorkOrderEventsPages>;
  eventsQuery: ReturnType<typeof useWorkOrderEvents>;
  artifactsQuery: ReturnType<typeof useWorkOrderArtifacts>;
  canManageWorkOrders: boolean;
  permissionsLoading: boolean;
  actions: ReturnType<typeof useWorkOrderDetailActions>;
}

function LoadedWorkOrderDetail({
  factory,
  order,
  derived,
  workOrdersHref,
  organizationId,
  events,
  eventsQuery,
  artifactsQuery,
  canManageWorkOrders,
  permissionsLoading,
  actions,
}: LoadedWorkOrderDetailProps) {
  return (
    <WorkOrderDetailLoadedView
      factory={factory}
      factoryHref={workOrdersHref}
      backLabel="Work Orders"
      organizationId={organizationId}
      order={order}
      events={events}
      eventsError={eventsQuery.error ?? null}
      isEventsLoading={eventsQuery.isLoading}
      hasMoreEvents={eventsQuery.hasNextPage ?? false}
      isLoadingMoreEvents={eventsQuery.isFetchingNextPage}
      onLoadMoreEvents={() => {
        void eventsQuery.fetchNextPage();
      }}
      onRetryEvents={() => {
        void eventsQuery.refetch();
      }}
      artifacts={artifactsQuery.data ?? []}
      isArtifactsLoading={artifactsQuery.isLoading}
      artifactsError={artifactsQuery.error ?? null}
      displayStatus={derived.displayStatus!}
      statusMeta={derived.statusMeta!}
      assigneeIds={derived.assigneeIds}
      assigneeNames={derived.assigneeNames}
      factoryLines={factory.lines ?? []}
      isOpen={derived.isOpen}
      isDispatchable={derived.isDispatchable}
      isClosed={derived.isClosed}
      canDispatch={canManageWorkOrders}
      canClose={canManageWorkOrders}
      canAssign={canManageWorkOrders}
      canManage={canManageWorkOrders}
      permissionsLoading={permissionsLoading}
      isDispatching={actions.isDispatching}
      isCompleting={actions.isCompleting}
      isRejecting={actions.isRejecting}
      isClosing={actions.isClosing}
      isAssigneesSaving={actions.isAssigneesSaving}
      isUpdatingStatus={actions.isUpdatingStatus}
      isAddingComment={actions.isAddingComment}
      onDispatch={actions.handleDispatch}
      onClose={actions.handleClose}
      onAssigneesSave={actions.handleAssigneesSave}
      onStatusChange={actions.handleStatusChange}
      onAddComment={actions.handleAddComment}
    />
  );
}
