import type { FactoriesWorkOrder } from "@/api-client";
import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AlertCircle, ClipboardList } from "lucide-react";
import { WorkOrderListItem } from "./WorkOrderListItem";
import { WORK_ORDER_SECTIONS, groupWorkOrdersBySection, type WorkOrderSectionDefinition } from "./workOrderProgress";

interface WorkOrderBoardProps {
  orders: FactoriesWorkOrder[];
  emptyTitle: string;
  emptyDescription: string;
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
  sections?: WorkOrderSectionDefinition[];
  onBrowseWorkOrders?: () => void;
}

export function WorkOrderBoard({
  orders,
  emptyTitle,
  emptyDescription,
  canCreate,
  permissionsLoading,
  onCreateClick,
  sections = WORK_ORDER_SECTIONS,
  onBrowseWorkOrders,
}: WorkOrderBoardProps) {
  const grouped = groupWorkOrdersBySection(orders, sections);

  if (orders.length === 0) {
    return (
      <WorkOrderBoardEmptyState
        title={emptyTitle}
        description={emptyDescription}
        canCreate={canCreate}
        permissionsLoading={permissionsLoading}
        onCreateClick={onCreateClick}
        onBrowseWorkOrders={onBrowseWorkOrders}
      />
    );
  }

  if (grouped.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center dark:border-gray-700 dark:bg-gray-900">
        <p className="text-sm text-gray-500 dark:text-gray-400">No work orders match this view.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {grouped.map(({ section, orders: sectionOrders }) => (
        <WorkOrderSection key={section.id} section={section} orders={sectionOrders} />
      ))}
    </div>
  );
}

function WorkOrderSection({
  section,
  orders,
}: {
  section: WorkOrderSectionDefinition;
  orders: FactoriesWorkOrder[];
}) {
  const isAttention = section.tone === "attention";

  return (
    <section
      className={cn(
        "overflow-hidden rounded-lg border bg-white dark:bg-gray-900",
        isAttention ? "border-amber-500/30 dark:border-amber-500/25" : "border-slate-200 dark:border-gray-700/70",
      )}
      data-testid={`work-order-section-${section.id}`}
    >
      <header
        className={cn(
          "flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3",
          isAttention
            ? "border-amber-500/20 bg-amber-500/5 dark:border-amber-500/15 dark:bg-amber-500/10"
            : "border-slate-200 bg-slate-50/80 dark:border-gray-700/70 dark:bg-gray-800/50",
        )}
      >
        <div className="flex items-start gap-2.5">
          {isAttention ? <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" aria-hidden /> : null}
          <div>
            <h3 className="text-sm font-medium text-slate-900 dark:text-gray-100">{section.title}</h3>
            <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">{section.description}</p>
          </div>
        </div>
        <span
          className={cn(
            "rounded-full px-2.5 py-0.5 text-xs font-medium",
            isAttention
              ? "bg-amber-500/15 text-amber-700 dark:bg-amber-500/20 dark:text-amber-200"
              : "bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300",
          )}
        >
          {orders.length} {orders.length === 1 ? "item" : "items"}
        </span>
      </header>

      <ul className="divide-y divide-slate-200 dark:divide-gray-700/70">
        {orders.map((order) => (
          <li key={order.id}>
            <WorkOrderListItem order={order} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function WorkOrderBoardEmptyState({
  title,
  description,
  canCreate,
  permissionsLoading,
  onCreateClick,
  onBrowseWorkOrders,
}: {
  title: string;
  description: string;
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
  onBrowseWorkOrders?: () => void;
}) {
  return (
    <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center dark:border-gray-700 dark:bg-gray-900">
      <ClipboardList className="h-9 w-9 text-slate-400 dark:text-gray-500" aria-hidden />
      <p className="mt-4 text-base font-medium text-slate-900 dark:text-gray-100">{title}</p>
      <p className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">{description}</p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
        {onBrowseWorkOrders ? (
          <Button type="button" variant="outline" onClick={onBrowseWorkOrders} data-testid="browse-work-orders-button">
            Browse work orders
          </Button>
        ) : null}
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create work orders."
        >
          <Button type="button" onClick={onCreateClick} disabled={!canCreate}>
            <Icon name="plus" />
            New work
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
