import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryPullRequests, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { WorkOrdersErrorState, WorkOrdersLoadingState } from "../../workOrders/WorkOrdersEmptyStates";
import { WorkOrdersLoadedView } from "../../workOrders/WorkOrdersLoadedView";
import { factoryContentBodyClassName, factorySectionHeaderClassName } from "../factoryPageLayoutStyles";
import { useBrokenIntegrationsBanner } from "../../lib/useBrokenIntegrationsBanner";
import { useHostedCreditChrome } from "../../lib/useHostedCreditEmptyBanner";
import { useWorkOrderListState } from "../../lib/useWorkOrderListState";

/** Storybook-only Tasks page with a Missions rail. */
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
  const { data: pullRequests = [] } = useFactoryPullRequests(organizationId, factoryId);

  const canCreate = canAct("work_orders", "create");
  const canDispatch = canAct("work_orders", "update");
  const canAssign = canAct("work_orders", "update");
  const { headerKicker: hostedCreditHeaderKicker, banner: hostedCreditEmptyBanner } = useHostedCreditChrome(
    organizationId,
    factoryKey,
  );
  const brokenIntegrationsBanner = useBrokenIntegrationsBanner(organizationId, factoryKey);
  const isOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);

  if (workOrdersError) {
    return (
      <>
        <WorkspacePageHeader className={factorySectionHeaderClassName} title="Tasks" />
        <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
          <WorkOrdersErrorState onRetry={() => void refetch()} />
        </div>
      </>
    );
  }

  if (isOrdersLoading || !factory) {
    return (
      <>
        <WorkspacePageHeader className={factorySectionHeaderClassName} title="Tasks" />
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
      hostedCreditHeaderKicker={hostedCreditHeaderKicker}
      hostedCreditEmptyBanner={hostedCreditEmptyBanner}
      brokenIntegrationsBanner={brokenIntegrationsBanner}
      {...cardActions}
    />
  );
}
