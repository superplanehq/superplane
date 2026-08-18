import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { WorkOrdersLoadedView } from "../workOrders/WorkOrdersLoadedView";
import { WorkOrdersErrorState, WorkOrdersLoadingState } from "../workOrders/WorkOrdersEmptyStates";
import { factoryContentBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";

/**
 * Data + action shell for the Work Orders list. Fetches work orders and
 * permissions, wires mutations, and hands everything to the display-only
 * `WorkOrdersLoadedView`. Errors and loading states live here so the
 * loaded view can assume a populated payload.
 */
export function WorkOrdersPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);

  usePageTitle(["Work Orders", factory?.name ?? "Workspace"]);

  const state = useWorkOrderListState();

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
    refetch,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const cardActions = useWorkOrderCardActions(organizationId, factoryId);

  const canCreate = canAct("work_orders", "create");
  const canDispatch = canAct("work_orders", "update");
  const canAssign = canAct("work_orders", "update");

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
      state={state}
      currentUserId={me?.id}
      canCreate={canCreate}
      onCreateWorkOrder={openCreateWorkOrder}
      canDispatch={canDispatch}
      canAssign={canAssign}
      permissionsLoading={permissionsLoading}
      {...cardActions}
    />
  );
}

/** Title-only header shown while work orders load or fail to load. */
function WorkOrdersHeaderStub() {
  return <WorkspacePageHeader className={factorySectionHeaderClassName} title="Work Orders" />;
}
