import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Link } from "@/components/Link/link";
import type { FactoriesWorkOrderResult } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useCloseWorkOrder, useDispatchWorkOrder, useFactory, useUpdateWorkOrderAssignees, useWorkOrder } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ArrowLeft, Forward, Loader2 } from "lucide-react";
import { Navigate, useParams } from "react-router-dom";
import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import {
  factoryDetailPanelClassName,
  factoryDetailSidebarClassName,
  factoryPageContentClassName,
} from "./factoryPageStyles";
import { FactoryPageShell } from "./FactoryPageShell";
import { WorkOrderAssigneesField } from "./WorkOrderAssigneesField";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

export function WorkOrderDetailPage() {
  const { organizationId, factoryId, orderId } = useParams<{
    organizationId: string;
    factoryId: string;
    orderId: string;
  }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const {
    data: factory,
    isLoading: factoryLoading,
    error: factoryError,
  } = useFactory(organizationId ?? "", factoryId ?? "");
  const {
    data: order,
    isLoading: orderLoading,
    error: orderError,
  } = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");

  const dispatchWorkOrder = useDispatchWorkOrder(organizationId ?? "", factoryId ?? "");
  const closeWorkOrder = useCloseWorkOrder(organizationId ?? "", factoryId ?? "");
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId ?? "", factoryId ?? "");

  usePageTitle([order?.title ?? "Work Order", factory?.name ?? "Factory"]);

  const isLoading = factoryLoading || orderLoading;
  const canDispatch = canAct("factories", "update");
  const canClose = canAct("factories", "update");
  const canAssign = canAct("factories", "update");
  const factoryLines = factory?.lines ?? [];
  const isOpen = order?.state === "STATE_OPEN";

  useReportPageReady(!isLoading && Boolean(factory && order), {
    failed: Boolean(factoryError || orderError),
  });

  if (!organizationId || !factoryId || !orderId) {
    return null;
  }

  if (!factoryLoading && factoryError) {
    return <Navigate to={`/${organizationId}/factories`} replace />;
  }

  if (!orderLoading && orderError) {
    return <Navigate to={`/${organizationId}/factories/${factoryId}`} replace />;
  }

  const factoryHref = `/${organizationId}/factories/${factoryId}`;
  const displayStatus = order ? getWorkOrderDisplayStatus(order) : null;
  const statusMeta = displayStatus ? getWorkOrderDisplayStatusMeta(displayStatus) : null;
  const assigneeIds = (order?.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id));
  const assigneeNames = (order?.assignees ?? []).map((assignee) => assignee.name ?? "Unknown");

  const handleAssigneesSave = async (nextAssigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: nextAssigneeIds });
      showSuccessToast("Assignees updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update assignees"));
      throw error;
    }
  };

  const handleDispatch = async (lineName: string) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName });
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  const handleClose = async (result: FactoriesWorkOrderResult) => {
    try {
      await closeWorkOrder.mutateAsync({ orderId, result });
      showSuccessToast(`Work order closed as ${formatWorkOrderResult(result).toLowerCase()}.`);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close work order"));
    }
  };

  const isCompleting =
    closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_COMPLETED";
  const isRejecting =
    closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_REJECTED";
  const isClosing = closeWorkOrder.isPending;

  return (
    <FactoryPageShell backHref={factoryHref} backLabel={factory?.name ?? "Factory"}>
      {isLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading work order…</Text>
        </div>
      ) : factory && order && statusMeta && displayStatus ? (
        <div className={factoryPageContentClassName}>
          <Link
            href={factoryHref}
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {factory.name}
          </Link>

          <header className="border-b border-gray-200 pb-6 dark:border-gray-700/70">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
                <Badge
                  variant="outline"
                  className={cn("inline-flex shrink-0 px-2.5 py-1 text-xs font-medium", statusMeta.className)}
                >
                  {displayStatus === "running" ? (
                    <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden />
                  ) : null}
                  {statusMeta.label}
                </Badge>
                <h1 className="min-w-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
                  {order.title}
                </h1>
              </div>

              {isOpen ? (
                <div className="flex flex-wrap items-center gap-2">
                  <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
                    <DispatchWorkOrderPopover
                      lines={factoryLines}
                      isSaving={dispatchWorkOrder.isPending}
                      canDispatch={canDispatch || permissionsLoading}
                      onDispatch={handleDispatch}
                    >
                      <Button
                        type="button"
                        disabled={!canDispatch || factoryLines.length === 0}
                        data-testid="work-order-dispatch-button"
                      >
                        Dispatch
                        <Forward className="ml-1.5 h-4 w-4" aria-hidden />
                      </Button>
                    </DispatchWorkOrderPopover>
                  </PermissionTooltip>

                  <PermissionTooltip allowed={canClose} message="You don't have permission to close work orders.">
                    <LoadingButton
                      type="button"
                      variant="ghost"
                      disabled={!canClose || isClosing}
                      loading={isCompleting}
                      loadingText="Completing..."
                      onClick={() => void handleClose("RESULT_COMPLETED")}
                      data-testid="work-order-complete-button"
                    >
                      Complete
                    </LoadingButton>
                  </PermissionTooltip>

                  <PermissionTooltip allowed={canClose} message="You don't have permission to close work orders.">
                    <LoadingButton
                      type="button"
                      variant="ghost"
                      disabled={!canClose || isClosing}
                      loading={isRejecting}
                      loadingText="Rejecting..."
                      onClick={() => void handleClose("RESULT_REJECTED")}
                      data-testid="work-order-reject-button"
                    >
                      Reject
                    </LoadingButton>
                  </PermissionTooltip>
                </div>
              ) : null}
            </div>
          </header>

          <div className={cn(factoryDetailPanelClassName, "mt-8 grid lg:grid-cols-[minmax(0,1fr)_280px]")}>
            <div className="min-w-0 px-6 py-6 sm:px-8">
              {order.description ? (
                <section className="mb-8 rounded-lg border border-gray-200 px-4 py-4 dark:border-gray-700/70">
                  <p className="whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-gray-300">
                    {order.description}
                  </p>
                </section>
              ) : null}

              <section>
                <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Activity</h2>
                <div className="mt-5">
                  <WorkOrderActivityTimeline organizationId={organizationId} order={order} />
                </div>
              </section>
            </div>

            <aside className={cn(factoryDetailSidebarClassName, "lg:min-h-full")}>
              <WorkOrderAssigneesField
                organizationId={organizationId}
                assigneeIds={assigneeIds}
                assigneeNames={assigneeNames}
                canEdit={canAssign}
                isSaving={updateAssignees.isPending}
                onSave={handleAssigneesSave}
              />
            </aside>
          </div>
        </div>
      ) : null}
    </FactoryPageShell>
  );
}
