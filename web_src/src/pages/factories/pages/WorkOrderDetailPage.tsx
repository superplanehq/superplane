import { usePermissions } from "@/contexts/usePermissions";
import {
  useFactory,
  useFactoryPullRequests,
  useWorkOrder,
  useWorkOrderArtifacts,
  useWorkOrderEvents,
} from "@/hooks/useFactoryData";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { usePageTitle } from "@/hooks/usePageTitle";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { useMemo } from "react";
import { Navigate } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { flattenWorkOrderEventsPages } from "../lib/workOrderEventsPagination";
import { getWorkOrderDetailDerived } from "../lib/workOrderProgress";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../lib/workOrderChecks";
import { useWorkOrderDetailActions } from "../useWorkOrderDetailActions";
import { WorkOrderDetailLoadedView } from "../WorkOrderDetailLoadedView";
import { presentWorkOrderStatusNotes } from "../lib/workOrderStatusNote";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

export function WorkOrderDetailPage() {
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  return <Navigate to={factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory))} replace />;
}

/** Legacy `/work-orders/:orderId` bookmarks go to the workspace line board. */
export function LegacyWorkOrderDetailRedirect() {
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  return <Navigate to={factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory))} replace />;
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
  const pullRequestsQuery = useFactoryPullRequests(organizationId, factoryId, { workOrderIds: [orderId] });
  const checksQuery = useWorkOrderChecks(organizationId, factoryId, orderId);
  const checks = useMemo(() => presentWorkOrderChecks(checksQuery.data ?? []), [checksQuery.data]);

  const actions = useWorkOrderDetailActions(organizationId, factoryId, orderId);
  // Memoize so derived arrays (e.g. `assigneeIds`) keep a stable reference
  // across re-renders/refetches that don't actually change `order`.
  const derived = useMemo(() => getWorkOrderDetailDerived(order), [order]);

  usePageTitle(workOrderDetailTitle(order, factory), { enabled: chrome === "page" });

  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const unavailable = shouldRedirectAfterError({ factoryLoading, factoryError, orderLoading, orderError });
  const ready = isWorkOrderDetailReady(factory, order, derived);

  if (unavailable) {
    return <WorkOrderDetailUnavailable chrome={chrome} boardHref={boardHref} />;
  }

  if (factoryLoading || orderLoading) {
    return <WorkOrderDetailLoading chrome={chrome} />;
  }

  if (!ready) {
    return null;
  }

  return (
    <LoadedWorkOrderDetail
      order={order!}
      derived={derived}
      factoryLines={factory!.lines ?? []}
      organizationId={organizationId}
      factoryKey={factoryKey}
      chrome={chrome}
      events={events}
      eventsQuery={eventsQuery}
      artifactsQuery={artifactsQuery}
      pullRequestsQuery={pullRequestsQuery}
      checks={checks}
      isChecksLoading={checksQuery.isLoading}
      checksError={checksQuery.error ?? null}
      canManageWorkOrders={canAct("work_orders", "update")}
      permissionsLoading={permissionsLoading}
      actions={actions}
    />
  );
}

function workOrderDetailTitle(
  order: ReturnType<typeof useWorkOrder>["data"],
  factory: ReturnType<typeof useFactory>["data"],
) {
  return [order?.title ?? "Work Order", factory?.name ?? "Workspace"];
}

function isWorkOrderDetailReady(
  factory: ReturnType<typeof useFactory>["data"],
  order: ReturnType<typeof useWorkOrder>["data"],
  derived: ReturnType<typeof getWorkOrderDetailDerived>,
): derived is ReturnType<typeof getWorkOrderDetailDerived> & {
  statusMeta: NonNullable<ReturnType<typeof getWorkOrderDetailDerived>["statusMeta"]>;
  displayStatus: NonNullable<ReturnType<typeof getWorkOrderDetailDerived>["displayStatus"]>;
} {
  return Boolean(factory && order && derived.statusMeta && derived.displayStatus);
}

function WorkOrderDetailLoading({ chrome }: { chrome: "page" | "dialog" }) {
  return (
    <div className={chrome === "dialog" ? "px-6 py-8" : factoryContentBodyClassName}>
      <p className="text-[13px] text-muted-foreground">Loading work order…</p>
    </div>
  );
}

function WorkOrderDetailUnavailable({ chrome, boardHref }: { chrome: "page" | "dialog"; boardHref: string }) {
  if (chrome === "dialog") {
    return <p className="px-6 py-8 text-[13px] text-muted-foreground">This work order cannot be opened.</p>;
  }
  return <Navigate to={boardHref} replace />;
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
  pullRequestsQuery: ReturnType<typeof useFactoryPullRequests>;
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
  pullRequestsQuery,
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
      pullRequests={pullRequestsQuery.data ?? []}
      isPullRequestsLoading={pullRequestsQuery.isLoading}
      pullRequestsError={pullRequestsQuery.error ?? null}
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
