import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryPullRequests, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { useBrokenIntegrationsBanner } from "../lib/useBrokenIntegrationsBanner";
import { useHostedCreditEmptyBanner } from "../lib/useHostedCreditEmptyBanner";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { WorkOrdersLoadedView } from "../workOrders/WorkOrdersLoadedView";
import { WorkOrdersErrorState, WorkOrdersLoadingState } from "../workOrders/WorkOrdersEmptyStates";
import { factoryContentBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";

/**
 * Data + action shell for the Tasks list. Fetches tasks and
 * permissions, wires mutations, and hands everything to the display-only
 * `WorkOrdersLoadedView`. Errors and loading states live here so the
 * loaded view can assume a populated payload.
 */
export function WorkOrdersPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);

  usePageTitle(["Tasks", factory?.name ?? "Workspace"]);

  const state = useWorkOrderListState(factoryId);

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
    refetch,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const { data: pullRequests = [] } = useFactoryPullRequests(organizationId, factoryId);

  const canCreate = canAct("work_orders", "create");
  const canDispatch = canAct("work_orders", "update");
  const canAssign = canAct("work_orders", "update");
  const hostedCreditEmptyBanner = useHostedCreditEmptyBanner(organizationId, factoryKey);
  const brokenIntegrationsBanner = useBrokenIntegrationsBanner(organizationId, factoryKey);

  const isOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);

  if (workOrdersError) {
    return (
      <>
        <WorkOrdersHeaderStub />
        <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
          <WorkOrdersErrorState onRetry={() => void refetch()} />
        </div>
      </>
    );
  }

  if (isOrdersLoading || !factory) {
    return (
      <>
        <WorkOrdersHeaderStub />
        <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
          <WorkOrdersLoadingState />
        </div>
      </>
    );
  }

  return (
    <WorkOrdersLoadedView
      organizationId={organizationId}
      factoryKey={factoryKey}
      factory={factory}
      factoryLines={factory.lines ?? []}
      workOrders={workOrders}
      pullRequests={pullRequests}
      state={state}
      currentUserId={me?.id}
      canCreate={canCreate}
      onCreateWorkOrder={openCreateWorkOrder}
      canDispatch={canDispatch}
      canAssign={canAssign}
      permissionsLoading={permissionsLoading}
      hostedCreditEmptyBanner={hostedCreditEmptyBanner}
      brokenIntegrationsBanner={brokenIntegrationsBanner}
      {...cardActions}
    />
  );
}

/** Title-only header shown while tasks load or fail to load. */
function WorkOrdersHeaderStub() {
  return <WorkspacePageHeader className={factorySectionHeaderClassName} title="Tasks" />;
}
