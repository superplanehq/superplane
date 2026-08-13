import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useDispatchWorkOrder, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  countActiveWorkOrders,
  filterWorkOrdersByOwner,
  filterWorkOrdersByStatus,
  type WorkOrderOwnerFilter,
  type WorkOrderStatusFilter,
} from "../lib/workOrderProgress";
import { WorkOrderCard } from "../WorkOrderCard";
import { WorkOrderFilters } from "../WorkOrderFilters";
import {
  factoryCardClassName,
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

export function WorkOrdersPage() {
  const { organizationId, factoryId, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: me } = useMe(false);

  const [ownerFilter, setOwnerFilter] = useState<WorkOrderOwnerFilter>("mine");
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatusFilter>("active");

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);

  const filteredWorkOrders = useMemo(() => {
    const byOwner = filterWorkOrdersByOwner(workOrders, ownerFilter, me?.id);
    return filterWorkOrdersByStatus(byOwner, statusFilter);
  }, [workOrders, ownerFilter, statusFilter, me?.id]);

  const activeCount = countActiveWorkOrders(workOrders);
  const canCreate = canAct("work_orders", "create");
  const canDispatch = canAct("work_orders", "update");

  const isOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);
  const factoryLines = factory?.lines ?? [];

  const handleDispatch = async (orderId: string, input: { lineName: string }) => {
    try {
      await dispatchWorkOrder.mutateAsync({ orderId, ...input });
      showSuccessToast(`Dispatched to ${input.lineName}.`);
    } catch {
      showErrorToast("Failed to dispatch work order.");
    }
  };

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
              Work Orders
            </Heading>
            <span
              className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground"
              data-testid="work-orders-active-count"
            >
              {activeCount}
            </span>
          </div>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Track intake, planning, and delivery for this workspace.
          </p>
        </div>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create work orders."
        >
          <Button
            type="button"
            disabled={!canCreate}
            onClick={() => openCreateWorkOrder()}
            data-testid="work-order-list-create-button"
          >
            <Icon name="plus" />
            New Work Order
          </Button>
        </PermissionTooltip>
      </header>

      <div className={factoryContentBodyClassName}>
        <div className={cn("p-6", factoryCardClassName)} data-testid="work-orders-panel">
          <WorkOrderFilters
            ownerFilter={ownerFilter}
            statusFilter={statusFilter}
            onOwnerFilterChange={setOwnerFilter}
            onStatusFilterChange={setStatusFilter}
          />

          <div className="mt-6">
            {workOrdersError ? (
              <div className="rounded-md border border-destructive/50 px-4 py-3 text-[13px] text-destructive">
                Failed to load work orders.
              </div>
            ) : isOrdersLoading ? (
              <p className="text-[13px] text-muted-foreground">Loading work orders…</p>
            ) : filteredWorkOrders.length === 0 ? (
              <p className="px-4 py-8 text-center text-[13px] text-muted-foreground">
                No work orders match these filters.
                <br />
                Create a work order or adjust filters to see more results.
              </p>
            ) : (
              <div>
                {filteredWorkOrders.map((order) => (
                  <WorkOrderCard
                    key={order.id}
                    order={order}
                    organizationId={organizationId}
                    factoryId={factoryId}
                    lines={factoryLines}
                    canDispatch={canDispatch || permissionsLoading}
                    isDispatching={dispatchWorkOrder.isPending}
                    onDispatch={(input) => handleDispatch(order.id!, input)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
