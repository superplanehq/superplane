import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { useAssignWorkOrder, useCreateWorkOrder, useFactory, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { Navigate, useParams } from "react-router-dom";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoryDetailHeader, type FactoryDetailTab } from "./FactoryDetailHeader";
import { FactoryPageShell } from "./FactoryPageShell";
import { WorkOrderBoard } from "./WorkOrderBoard";
import {
  MY_WORK_SECTIONS,
  WORK_ORDER_TAB_SECTIONS,
  countNeedsAttention,
  countOpenWorkOrders,
  filterMyWorkOrders,
} from "./workOrderProgress";

export function FactoryDetailPage() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<FactoryDetailTab>("my-work");
  const [claimingOrderId, setClaimingOrderId] = useState<string | null>(null);

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
  const createWorkOrder = useCreateWorkOrder(organizationId ?? "", factoryId ?? "");
  const assignWorkOrder = useAssignWorkOrder(organizationId ?? "", factoryId ?? "");

  usePageTitle(factory?.name ? [factory.name, "Factories"] : ["Factory"]);

  const canCreate = canAct("factories", "create");
  const canAssign = canAct("factories", "update");
  const isLoading = factoryLoading || (ordersLoading && workOrders.length === 0);
  const isOrdersLoading = ordersLoading || (ordersFetching && workOrders.length === 0);

  const myWorkOrders = useMemo(() => filterMyWorkOrders(workOrders, me?.id), [workOrders, me?.id]);
  const openWorkOrders = useMemo(() => workOrders.filter((order) => order.state === "STATE_OPEN"), [workOrders]);
  const myWorkCount = myWorkOrders.length;
  const openWorkOrderCount = countOpenWorkOrders(workOrders);
  const myNeedsAttentionCount = countNeedsAttention(myWorkOrders);

  const tabOrders = activeTab === "my-work" ? myWorkOrders : openWorkOrders;
  const tabSections = activeTab === "my-work" ? MY_WORK_SECTIONS : WORK_ORDER_TAB_SECTIONS;

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

  const handleCreateWorkOrder = async (input: { title: string; description: string }) => {
    await createWorkOrder.mutateAsync(input);
    setCreateOpen(false);
  };

  const handleClaimWorkOrder = async (orderId: string) => {
    if (!me?.id) {
      showErrorToast("Could not determine your user account.");
      return;
    }

    setClaimingOrderId(orderId);
    try {
      await assignWorkOrder.mutateAsync({ orderId, assigneeIds: [me.id] });
      showSuccessToast("Work order assigned to you.");
      setActiveTab("my-work");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to claim work order"));
    } finally {
      setClaimingOrderId(null);
    }
  };

  return (
    <FactoryPageShell backHref={`/${organizationId}/factories`} backLabel="Factories">
      {factoryLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading factory…</Text>
        </div>
      ) : factory ? (
        <>
          <FactoryDetailHeader
            factory={factory}
            backHref={`/${organizationId}/factories`}
            activeTab={activeTab}
            onTabChange={setActiveTab}
            myWorkCount={myWorkCount}
            workOrdersCount={openWorkOrderCount}
            needsAttentionCount={myNeedsAttentionCount}
            canCreate={canCreate}
            permissionsLoading={permissionsLoading}
            onCreateClick={() => setCreateOpen(true)}
          />

          <div className="mx-auto w-full max-w-5xl px-6 py-6 sm:px-8">
            {ordersError ? (
              <div className="rounded border border-red-300 bg-white px-4 py-2 text-red-500 dark:border-red-800 dark:bg-gray-800 dark:text-red-400">
                <Text>Failed to load work orders.</Text>
              </div>
            ) : isOrdersLoading ? (
              <Text className="text-sm text-gray-500">Loading work orders…</Text>
            ) : (
              <WorkOrderBoard
                orders={tabOrders}
                sections={tabSections}
                emptyTitle={activeTab === "my-work" ? "Nothing assigned to you" : "No work orders yet"}
                emptyDescription={
                  activeTab === "my-work"
                    ? "Open the Work orders tab and claim unassigned work, or create new work to get started."
                    : "Create work manually, or connect a source later to ingest from GitHub, Sentry, and more."
                }
                canCreate={canCreate}
                permissionsLoading={permissionsLoading}
                onCreateClick={() => setCreateOpen(true)}
                canClaim={canAssign}
                claimingOrderId={claimingOrderId}
                onClaim={activeTab === "work-orders" ? handleClaimWorkOrder : undefined}
                onBrowseWorkOrders={activeTab === "my-work" ? () => setActiveTab("work-orders") : undefined}
              />
            )}
          </div>
        </>
      ) : null}

      <CreateWorkOrderDialog
        open={createOpen}
        isSaving={createWorkOrder.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreateWorkOrder}
      />
    </FactoryPageShell>
  );
}
