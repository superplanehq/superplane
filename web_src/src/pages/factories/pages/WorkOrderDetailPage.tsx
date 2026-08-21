import { usePermissions } from "@/contexts/usePermissions";
import {
  useFactory,
  useFactoryWorkOrders,
  useWorkOrder,
  useWorkOrderArtifacts,
  useWorkOrderEvents,
} from "@/hooks/useFactoryData";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { usePageTitle } from "@/hooks/usePageTitle";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { useMemo } from "react";
import { Navigate, useLocation, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";
import { flattenWorkOrderEventsPages } from "../lib/workOrderEventsPagination";
import { getWorkOrderDetailDerived } from "../lib/workOrderProgress";
import {
  resolveWorkOrderByNumber,
  workOrderRouteNeedsCanonicalRedirect,
  type WorkOrderResolution,
} from "../lib/workOrderNumberResolution";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../lib/workOrderChecks";
import { useWorkOrderDetailActions } from "../useWorkOrderDetailActions";
import { WorkOrderDetailLoadedView } from "../WorkOrderDetailLoadedView";
import { presentWorkOrderStatusNotes } from "../lib/workOrderStatusNote";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

export function WorkOrderDetailPage() {
  const { orderNumber } = useParams<{ orderNumber: string }>();
  const { organizationId, factoryId, factoryKey } = useFactoriesLayout();
  const location = useLocation();
  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
  } = useFactoryWorkOrders(organizationId, factoryId);

  if (!orderNumber) {
    return null;
  }

  // `isFetching` (not just `isLoading`) so a just-created work order — whose
  // list invalidation is still in flight when we navigate to its permalink —
  // shows the loading state instead of bouncing back to the list.
  const resolution = resolveWorkOrderByNumber(workOrders, orderNumber, workOrdersLoading || workOrdersFetching);

  if (workOrderRouteNeedsCanonicalRedirect(resolution, orderNumber) && resolution.order?.number !== undefined) {
    const canonicalHref = workOrderDetailPath(organizationId, factoryKey, String(Number(resolution.order.number)));
    return <Navigate to={`${canonicalHref}${location.search}`} replace />;
  }

  if (resolution.status === "not-found") {
    return <Navigate to={workOrdersPath(organizationId, factoryKey)} replace />;
  }

  if (resolution.status === "loading" || !resolution.order?.id) {
    return (
      <div className={factoryContentBodyClassName}>
        <p className="text-[13px] text-muted-foreground">Loading work order…</p>
      </div>
    );
  }

  return (
    <WorkOrderDetailPanel
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      orderId={resolution.order.id}
    />
  );
}

/** Legacy `/work-orders/:orderId` bookmarks redirect to the canonical `/work-order/:number` permalink. */
export function LegacyWorkOrderDetailRedirect() {
  const { orderId } = useParams<{ orderId: string }>();
  const { organizationId, factoryId, factoryKey } = useFactoriesLayout();
  const location = useLocation();
  const { data: workOrders = [], isLoading, isFetching } = useFactoryWorkOrders(organizationId, factoryId);

  if (!orderId) {
    return <Navigate to={workOrdersPath(organizationId, factoryKey)} replace />;
  }

  const resolution: WorkOrderResolution = resolveWorkOrderByNumber(workOrders, orderId, isLoading || isFetching);

  if (resolution.status === "loading") {
    return (
      <div className={factoryContentBodyClassName}>
        <p className="text-[13px] text-muted-foreground">Loading work order…</p>
      </div>
    );
  }

  if (resolution.status === "not-found" || resolution.order?.number === undefined) {
    return <Navigate to={workOrdersPath(organizationId, factoryKey)} replace />;
  }

  const canonicalHref = workOrderDetailPath(organizationId, factoryKey, String(Number(resolution.order.number)));
  return <Navigate to={`${canonicalHref}${location.search}`} replace />;
}

export function WorkOrderDetailPanel({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  chrome = "page",
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  orderId: string;
  chrome?: "page" | "dialog";
}) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const { data: order, isLoading: orderLoading, error: orderError } = useWorkOrder(organizationId, factoryId, orderId);
  const eventsQuery = useWorkOrderEvents(organizationId, factoryId, orderId);
  const events = useMemo(() => flattenWorkOrderEventsPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);
  const artifactsQuery = useWorkOrderArtifacts(organizationId, factoryId, orderId);
  const checksQuery = useWorkOrderChecks(organizationId, factoryId, orderId);
  const checks = useMemo(() => presentWorkOrderChecks(checksQuery.data ?? []), [checksQuery.data]);

  const actions = useWorkOrderDetailActions(organizationId, factoryId, orderId);
  // Memoize so derived arrays (e.g. `assigneeIds`) keep a stable reference
  // across re-renders/refetches that don't actually change `order`.
  const derived = useMemo(() => getWorkOrderDetailDerived(order), [order]);

  usePageTitle([order?.title ?? "Work Order", factory?.name ?? "Workspace"], { enabled: chrome === "page" });

  const workOrdersHref = workOrdersPath(organizationId, factoryKey);

  if (shouldRedirectAfterError({ factoryLoading, factoryError, orderLoading, orderError })) {
    if (chrome === "dialog") {
      return <p className="px-6 py-8 text-[13px] text-muted-foreground">This work order cannot be opened.</p>;
    }
    return <Navigate to={workOrdersHref} replace />;
  }

  if (factoryLoading || orderLoading) {
    return (
      <div className={chrome === "dialog" ? "px-6 py-8" : factoryContentBodyClassName}>
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
      factoryKey={factoryKey}
      chrome={chrome}
      events={events}
      eventsQuery={eventsQuery}
      artifactsQuery={artifactsQuery}
      checks={checks}
      isChecksLoading={checksQuery.isLoading}
      checksError={checksQuery.error ?? null}
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
  order: FactoriesWorkOrder;
  derived: ReturnType<typeof getWorkOrderDetailDerived>;
  factoryLines: FactoriesFactoryLine[];
  organizationId: string;
  factoryKey: string;
  chrome?: "page" | "dialog";
  events: ReturnType<typeof flattenWorkOrderEventsPages>;
  eventsQuery: ReturnType<typeof useWorkOrderEvents>;
  artifactsQuery: ReturnType<typeof useWorkOrderArtifacts>;
  checks: WorkOrderCheckPresentation[];
  isChecksLoading: boolean;
  checksError: Error | null;
  canManageWorkOrders: boolean;
  permissionsLoading: boolean;
  actions: ReturnType<typeof useWorkOrderDetailActions>;
}

function LoadedWorkOrderDetail({
  order,
  derived,
  factoryLines,
  organizationId,
  factoryKey,
  chrome = "page",
  events,
  eventsQuery,
  artifactsQuery,
  checks,
  isChecksLoading,
  checksError,
  canManageWorkOrders,
  permissionsLoading,
  actions,
}: LoadedWorkOrderDetailProps) {
  return (
    <WorkOrderDetailLoadedView
      statusNotes={presentWorkOrderStatusNotes(order.statusNotes, derived.displayStatus ?? undefined)}
      organizationId={organizationId}
      factoryKey={factoryKey}
      chrome={chrome}
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
      checks={checks}
      isChecksLoading={isChecksLoading}
      checksError={checksError}
      displayStatus={derived.displayStatus!}
      statusMeta={derived.statusMeta!}
      assigneeIds={derived.assigneeIds}
      assigneeNames={derived.assigneeNames}
      factoryLines={factoryLines}
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
