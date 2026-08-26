import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { WorkOrdersErrorState, WorkOrdersLoadingState } from "../../workOrders/WorkOrdersEmptyStates";
import { factoryContentBodyClassName, factorySectionHeaderClassName } from "../factoryPageLayoutStyles";
import { useWorkOrderListState } from "../../lib/useWorkOrderListState";
import { MissionsWorkOrdersLoadedView } from "./MissionsWorkOrdersLoadedView";

/** Storybook-only Work Orders page with a Missions rail. */
export function MissionsWorkOrdersPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const state = useWorkOrderListState(factoryId);

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
        <WorkspacePageHeader className={factorySectionHeaderClassName} title="Work Orders" />
        <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
          <WorkOrdersErrorState onRetry={() => void refetch()} />
        </div>
      </>
    );
  }

  if (isOrdersLoading || !factory) {
    return (
      <>
        <WorkspacePageHeader className={factorySectionHeaderClassName} title="Work Orders" />
        <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
          <WorkOrdersLoadingState />
        </div>
      </>
    );
  }

  return (
    <MissionsWorkOrdersLoadedView
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
