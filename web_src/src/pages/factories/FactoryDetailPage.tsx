import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import {
  factoryAppsKey,
  useCreateFactoryLine,
  useCreateWorkOrder,
  useFactory,
  useFactoryApps,
  useFactoryWorkOrders,
  useUpdateFactoryLine,
  useUpdateWorkOrderAssignees,
  type FactoriesFactoryLine,
  type FactoryLineStep,
} from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { appPath } from "@/lib/appPaths";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { CreateFactoryAppDialog } from "./CreateFactoryAppDialog";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoryAppsPanel } from "./FactoryAppsPanel";
import { FactoryDetailHeader, type FactoryDetailTab } from "./FactoryDetailHeader";
import { FactoryLineDialog } from "./FactoryLineDialog";
import { FactoryLinesPanel } from "./FactoryLinesPanel";
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
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const [createWorkOpen, setCreateWorkOpen] = useState(false);
  const [createAppOpen, setCreateAppOpen] = useState(false);
  const [lineDialogOpen, setLineDialogOpen] = useState(false);
  const [editingLine, setEditingLine] = useState<FactoriesFactoryLine | null>(null);
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
  const {
    data: factoryApps = [],
    isLoading: appsLoading,
    isFetching: appsFetching,
  } = useFactoryApps(organizationId ?? "", factoryId ?? "");

  const createWorkOrder = useCreateWorkOrder(organizationId ?? "", factoryId ?? "");
  const updateWorkOrderAssignees = useUpdateWorkOrderAssignees(organizationId ?? "", factoryId ?? "");
  const createFactoryLine = useCreateFactoryLine(organizationId ?? "", factoryId ?? "");
  const updateFactoryLine = useUpdateFactoryLine(organizationId ?? "", factoryId ?? "");
  const createCanvas = useCreateCanvas(organizationId ?? "");

  usePageTitle(factory?.name ? [factory.name, "Factories"] : ["Factory"]);

  const canCreateWork = canAct("factories", "create");
  const canUpdateFactory = canAct("factories", "update");
  const canCreateApps = canAct("canvases", "create");
  const canAssign = canAct("factories", "update");
  const isLoading = factoryLoading || (ordersLoading && workOrders.length === 0);
  const isOrdersLoading = ordersLoading || (ordersFetching && workOrders.length === 0);
  const isAppsLoading = appsLoading || (appsFetching && factoryApps.length === 0);

  const factoryLines = factory?.lines ?? [];
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
    setCreateWorkOpen(false);
  };

  const handleCreateApp = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateAppOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (canvasId) {
        showSuccessToast("Factory app created.");
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create factory app"));
      throw error;
    }
  };

  const handleSaveLine = async (input: { name: string; steps: FactoryLineStep[] }) => {
    if (editingLine?.id) {
      await updateFactoryLine.mutateAsync({
        lineId: editingLine.id,
        name: input.name,
        steps: input.steps,
      });
      showSuccessToast("Line updated.");
    } else {
      await createFactoryLine.mutateAsync(input);
      showSuccessToast("Line created.");
    }
    setLineDialogOpen(false);
    setEditingLine(null);
  };

  const handleClaimWorkOrder = async (orderId: string) => {
    if (!me?.id) {
      showErrorToast("Could not determine your user account.");
      return;
    }

    setClaimingOrderId(orderId);
    try {
      await updateWorkOrderAssignees.mutateAsync({ orderId, assigneeIds: [me.id] });
      showSuccessToast("Work order assigned to you.");
      setActiveTab("my-work");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to claim work order"));
    } finally {
      setClaimingOrderId(null);
    }
  };

  const openCreateLineDialog = () => {
    setEditingLine(null);
    setLineDialogOpen(true);
  };

  const openEditLineDialog = (line: FactoriesFactoryLine) => {
    setEditingLine(line);
    setLineDialogOpen(true);
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
            linesCount={factoryLines.length}
            appsCount={factoryApps.length}
            needsAttentionCount={myNeedsAttentionCount}
            canCreate={canCreateWork}
            permissionsLoading={permissionsLoading}
            onCreateClick={() => setCreateWorkOpen(true)}
          />

          <div className="mx-auto w-full max-w-5xl px-6 py-6 sm:px-8">
            {activeTab === "lines" ? (
              <FactoryLinesPanel
                lines={factoryLines}
                apps={factoryApps}
                isLoading={factoryLoading}
                canUpdate={canUpdateFactory}
                permissionsLoading={permissionsLoading}
                onCreateClick={openCreateLineDialog}
                onEditLine={openEditLineDialog}
              />
            ) : null}

            {activeTab === "apps" ? (
              <FactoryAppsPanel
                organizationId={organizationId}
                apps={factoryApps}
                isLoading={isAppsLoading}
                canCreate={canCreateApps}
                permissionsLoading={permissionsLoading}
                onCreateClick={() => setCreateAppOpen(true)}
              />
            ) : null}

            {activeTab === "my-work" || activeTab === "work-orders" ? (
              ordersError ? (
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
                      : "Create work manually, or dispatch work to a factory line once lines are configured."
                  }
                  canCreate={canCreateWork}
                  permissionsLoading={permissionsLoading}
                  onCreateClick={() => setCreateWorkOpen(true)}
                  canClaim={canAssign}
                  claimingOrderId={claimingOrderId}
                  onClaim={activeTab === "work-orders" ? handleClaimWorkOrder : undefined}
                  onBrowseWorkOrders={activeTab === "my-work" ? () => setActiveTab("work-orders") : undefined}
                />
              )
            ) : null}
          </div>
        </>
      ) : null}

      <CreateWorkOrderDialog
        open={createWorkOpen}
        isSaving={createWorkOrder.isPending}
        onClose={() => setCreateWorkOpen(false)}
        onCreate={handleCreateWorkOrder}
      />

      <CreateFactoryAppDialog
        open={createAppOpen}
        isSaving={createCanvas.isPending}
        onClose={() => setCreateAppOpen(false)}
        onCreate={handleCreateApp}
      />

      <FactoryLineDialog
        open={lineDialogOpen}
        mode={editingLine ? "edit" : "create"}
        organizationId={organizationId}
        apps={factoryApps}
        initialName={editingLine?.name}
        initialSteps={editingLine?.steps}
        isSaving={createFactoryLine.isPending || updateFactoryLine.isPending}
        onClose={() => {
          setLineDialogOpen(false);
          setEditingLine(null);
        }}
        onSave={handleSaveLine}
      />
    </FactoryPageShell>
  );
}
