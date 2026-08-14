import { usePermissions } from "@/contexts/usePermissions";
import { useDispatchWorkOrder, useFactoryWorkOrders, useUpdateWorkOrderAssignees } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { WorkOrdersLoadedView } from "../workOrders/WorkOrdersLoadedView";
import { WorkOrdersErrorState, WorkOrdersLoadingState } from "../workOrders/WorkOrdersEmptyStates";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

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

  const state = useWorkOrderListState();

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
    refetch,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);

  const canCreate = canAct("work_orders", "create");
  const canDispatch = canAct("work_orders", "update");
  const canAssign = canAct("work_orders", "update");

  const isOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);

  const handleDispatch = async (orderId: string, input: { lineName: string }) => {
    try {
      await dispatchWorkOrder.mutateAsync({ orderId, lineName: input.lineName });
      showSuccessToast(`Dispatched to ${input.lineName}.`);
    } catch {
      showErrorToast("Failed to dispatch work order.");
    }
  };

  const handleAssigneesSave = async (orderId: string, assigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds });
      showSuccessToast("Owners updated.");
    } catch {
      showErrorToast("Failed to update owners.");
    }
  };

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
      isDispatching={dispatchWorkOrder.isPending}
      isAssigneesSaving={updateAssignees.isPending}
      onDispatch={handleDispatch}
      onAssigneesSave={handleAssigneesSave}
    />
  );
}

/** Title-only header shown while work orders load or fail to load. */
function WorkOrdersHeaderStub() {
  return <WorkspacePageHeader title="Work Orders" />;
}
