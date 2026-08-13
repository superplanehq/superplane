import { usePermissions } from "@/contexts/usePermissions";
import { useFactory, useWorkOrder, useWorkOrderArtifacts, useWorkOrderEvents } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
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
  // `order` comes from a query with `staleTime: 0` and is invalidated on
  // nearly every websocket event, so it's refetched (and its object
  // identity changes) far more often than its actual contents. Memoize the
  // derived value so consumers like `assigneeIds` don't see a brand-new
  // array reference on every unrelated re-render/refetch.
  const derived = useMemo(() => getWorkOrderDetailDerived(order), [order]);

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
      order={order}
      derived={derived}
      factoryLines={factory.lines ?? []}
      organizationId={organizationId}
      factoryId={factoryId}
      events={events}
      eventsQuery={eventsQuery}
      artifactsQuery={artifactsQuery}
      canManageWorkOrders={canAct("work_orders", "update")}
      canEditFactoryLines={canAct("factories", "update")}
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
  order: FactoriesWorkOrder;
  derived: ReturnType<typeof getWorkOrderDetailDerived>;
  factoryLines: FactoriesFactoryLine[];
  organizationId: string;
  factoryId: string;
  events: ReturnType<typeof flattenWorkOrderEventsPages>;
  eventsQuery: ReturnType<typeof useWorkOrderEvents>;
  artifactsQuery: ReturnType<typeof useWorkOrderArtifacts>;
  canManageWorkOrders: boolean;
  canEditFactoryLines: boolean;
  permissionsLoading: boolean;
  actions: ReturnType<typeof useWorkOrderDetailActions>;
}

function LoadedWorkOrderDetail({
  order,
  derived,
  factoryLines,
  organizationId,
  factoryId,
  events,
  eventsQuery,
  artifactsQuery,
  canManageWorkOrders,
  canEditFactoryLines,
  permissionsLoading,
  actions,
}: LoadedWorkOrderDetailProps) {
  return (
    <WorkOrderDetailLoadedView
      organizationId={organizationId}
      factoryId={factoryId}
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
      factoryLines={factoryLines}
      canEditFactoryLines={canEditFactoryLines}
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
