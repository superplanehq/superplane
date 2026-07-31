import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Link } from "@/components/Link/link";
import { usePermissions } from "@/contexts/usePermissions";
import { useDispatchWorkOrder, useFactory, useUpdateWorkOrderAssignees, useWorkOrder } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { formatTimeAgo } from "@/lib/date";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ArrowLeft, Loader2, Send } from "lucide-react";
import { useState } from "react";
import { Navigate, useParams } from "react-router-dom";
import { DispatchWorkOrderDialog } from "./DispatchWorkOrderDialog";
import { factoryCardClassName, factoryPageContentClassName } from "./factoryPageStyles";
import { FactoryPageShell } from "./FactoryPageShell";
import { formatWorkOrderResult, formatWorkOrderState } from "./workOrderPresentation";
import { deriveWorkOrderProgress, getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

export function WorkOrderDetailPage() {
  const { organizationId, factoryId, orderId } = useParams<{
    organizationId: string;
    factoryId: string;
    orderId: string;
  }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);
  const [dispatchOpen, setDispatchOpen] = useState(false);

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
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId ?? "", factoryId ?? "");

  usePageTitle([order?.title ?? "Work Order", factory?.name ?? "Factory"]);

  const isLoading = factoryLoading || orderLoading;
  const canDispatch = canAct("factories", "update");
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
  const progress = order ? deriveWorkOrderProgress(order) : null;
  const displayStatus = order ? getWorkOrderDisplayStatus(order) : null;
  const statusMeta = displayStatus ? getWorkOrderDisplayStatusMeta(displayStatus) : null;
  const updatedAt = order?.updatedAt ?? order?.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
  const showClaim = progress?.phase === "unassigned" && isOpen;

  const handleClaim = async () => {
    if (!me?.id || !orderId) {
      showErrorToast("Could not determine your user account.");
      return;
    }

    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: [me.id] });
      showSuccessToast("Work order assigned to you.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to claim work order"));
    }
  };

  const handleDispatch = async (lineName: string) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName });
    setDispatchOpen(false);
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  return (
    <FactoryPageShell backHref={factoryHref} backLabel={factory?.name ?? "Factory"}>
      {isLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading work order…</Text>
        </div>
      ) : factory && order && progress && statusMeta && displayStatus ? (
        <div className={factoryPageContentClassName}>
          <Link
            href={factoryHref}
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {factory.name}
          </Link>

          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-3">
                <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
                  {order.title}
                </h1>
                <Badge
                  variant="outline"
                  className={cn("rounded-full px-2.5 py-1 text-xs font-medium", statusMeta.className)}
                >
                  {displayStatus === "running" ? (
                    <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden />
                  ) : null}
                  {statusMeta.label}
                </Badge>
              </div>
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{progress.summary}</p>
            </div>

            <div className="flex flex-wrap gap-2">
              {showClaim ? (
                <PermissionTooltip allowed={canAssign} message="You don't have permission to claim work orders.">
                  <Button
                    type="button"
                    variant="outline"
                    disabled={!canAssign || updateAssignees.isPending}
                    onClick={() => void handleClaim()}
                    data-testid="work-order-claim-button"
                  >
                    Claim
                  </Button>
                </PermissionTooltip>
              ) : null}

              {isOpen ? (
                <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
                  <Button
                    type="button"
                    disabled={!canDispatch || factoryLines.length === 0}
                    onClick={() => setDispatchOpen(true)}
                    data-testid="work-order-dispatch-button"
                  >
                    <Send className="mr-1.5 h-4 w-4" aria-hidden />
                    Dispatch to line
                  </Button>
                </PermissionTooltip>
              ) : null}
            </div>
          </div>

          <div className={cn(factoryCardClassName, "mt-8 space-y-6 p-6")}>
            {order.description ? (
              <section>
                <h2 className="text-sm font-medium text-slate-700 dark:text-gray-300">Description</h2>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-gray-600 dark:text-gray-400">
                  {order.description}
                </p>
              </section>
            ) : null}

            <section className="grid gap-4 sm:grid-cols-2">
              <DetailField label="State" value={formatWorkOrderState(order.state)} />
              {order.result ? <DetailField label="Result" value={formatWorkOrderResult(order.result)} /> : null}
              <DetailField
                label="Assignees"
                value={
                  order.assignees?.length ? order.assignees.map((a) => a.name ?? "Unknown").join(", ") : "Unassigned"
                }
              />
              <DetailField label="Updated" value={timeLabel} />
            </section>

            {factoryLines.length > 0 ? (
              <section>
                <h2 className="text-sm font-medium text-slate-700 dark:text-gray-300">Available lines</h2>
                <ul className="mt-3 space-y-2">
                  {factoryLines.map((line) => (
                    <li
                      key={line.id ?? line.name}
                      className="flex items-center justify-between rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-gray-700"
                    >
                      <span className="font-medium text-slate-900 dark:text-gray-100">{line.name}</span>
                      <span className="text-gray-500 dark:text-gray-400">
                        {line.steps?.length ?? 0} step{(line.steps?.length ?? 0) === 1 ? "" : "s"}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            ) : null}
          </div>
        </div>
      ) : null}

      <DispatchWorkOrderDialog
        open={dispatchOpen}
        lines={factoryLines}
        isSaving={dispatchWorkOrder.isPending}
        canDispatch={canDispatch || permissionsLoading}
        onClose={() => setDispatchOpen(false)}
        onDispatch={handleDispatch}
      />
    </FactoryPageShell>
  );
}

function DetailField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{label}</p>
      <p className="mt-1 text-sm text-slate-900 dark:text-gray-100">{value}</p>
    </div>
  );
}
