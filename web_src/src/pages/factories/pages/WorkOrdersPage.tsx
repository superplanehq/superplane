import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useDispatchWorkOrder, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { createWorkOrderPath } from "../lib/factoryPagePaths";
import {
  countActiveWorkOrders,
  filterWorkOrdersByOwner,
  filterWorkOrdersByStatus,
  type WorkOrderOwnerFilter,
  type WorkOrderStatusFilter,
} from "../lib/workOrderProgress";
import { WorkOrderCard } from "../WorkOrderCard";
import { WorkOrderFilters } from "../WorkOrderFilters";
import { factoryContentBodyClassName, factoryContentHeaderClassName } from "./factoryPageLayoutStyles";

export function WorkOrdersPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
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
  const canCreate = canAct("factories", "create");
  const canDispatch = canAct("factories", "update");

  const isOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);
  const factoryLines = factory?.lines ?? [];

  const handleDispatch = async (orderId: string, lineName: string) => {
    try {
      await dispatchWorkOrder.mutateAsync({ orderId, lineName });
      showSuccessToast(`Dispatched to ${lineName}.`);
    } catch {
      showErrorToast("Failed to dispatch work order.");
    }
  };

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Heading level={1} className="!text-xl text-gray-900 dark:text-gray-100">
              Work Orders
            </Heading>
            <span
              className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300"
              data-testid="work-orders-active-count"
            >
              {activeCount}
            </span>
          </div>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Track intake, planning, and delivery for this workspace.
          </Text>
        </div>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create work orders."
        >
          <Button
            type="button"
            asChild
            disabled={!canCreate}
            className={cn(appDarkModeClasses.primaryAction)}
            data-testid="work-order-list-create-button"
          >
            <Link href={canCreate ? createWorkOrderPath(organizationId, factoryId) : "#"}>
              <Icon name="plus" />
              New Work Order
            </Link>
          </Button>
        </PermissionTooltip>
      </header>

      <div className={factoryContentBodyClassName}>
        <div
          className="rounded-lg border border-slate-950/10 bg-white p-6 dark:border-gray-700/70 dark:bg-gray-900"
          data-testid="work-orders-panel"
        >
          <WorkOrderFilters
            ownerFilter={ownerFilter}
            statusFilter={statusFilter}
            onOwnerFilterChange={setOwnerFilter}
            onStatusFilterChange={setStatusFilter}
          />

          <div className="mt-6">
            {workOrdersError ? (
              <div className="rounded-md border border-red-300 px-4 py-3 text-sm text-red-600 dark:border-red-800 dark:text-red-400">
                Failed to load work orders.
              </div>
            ) : isOrdersLoading ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">Loading work orders…</p>
            ) : filteredWorkOrders.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
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
                    onDispatch={(lineName) => handleDispatch(order.id!, lineName)}
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
