import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { Text } from "@/components/Text/text";
import { WorkOrderCard } from "./WorkOrderCard";
import { WorkOrderFilters } from "./WorkOrderFilters";
import { factoryCountBadgeClassName, factoryDetailMainClassName } from "./lib/factoryPageStyles";
import type { WorkOrderOwnerFilter, WorkOrderStatusFilter } from "./lib/workOrderProgress";

interface FactoryDetailWorkOrdersPanelProps {
  // Work orders in an active state (matches the default status filter).
  activeWorkOrderCount: number;
  ownerFilter: WorkOrderOwnerFilter;
  statusFilter: WorkOrderStatusFilter;
  onOwnerFilterChange: (filter: WorkOrderOwnerFilter) => void;
  onStatusFilterChange: (filter: WorkOrderStatusFilter) => void;
  ordersError: Error | null;
  isOrdersLoading: boolean;
  filteredWorkOrders: FactoriesWorkOrder[];
  factoryHref: string;
  organizationId: string;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, lineName: string) => Promise<void>;
}

export function FactoryDetailWorkOrdersPanel({
  activeWorkOrderCount,
  ownerFilter,
  statusFilter,
  onOwnerFilterChange,
  onStatusFilterChange,
  ordersError,
  isOrdersLoading,
  filteredWorkOrders,
  factoryHref,
  organizationId,
  factoryLines,
  canDispatch,
  permissionsLoading,
  isDispatching,
  onDispatch,
}: FactoryDetailWorkOrdersPanelProps) {
  return (
    <section className={factoryDetailMainClassName}>
      <div className="mb-6">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Work Orders</h2>
          <span className={factoryCountBadgeClassName}>{activeWorkOrderCount}</span>
        </div>
      </div>

      <WorkOrderFilters
        ownerFilter={ownerFilter}
        statusFilter={statusFilter}
        onOwnerFilterChange={onOwnerFilterChange}
        onStatusFilterChange={onStatusFilterChange}
      />

      <div className="mt-6">
        {ordersError ? (
          <div className="rounded-lg border border-red-300 px-4 py-3 text-red-500 dark:border-red-800 dark:text-red-400">
            <Text>Failed to load work orders.</Text>
          </div>
        ) : isOrdersLoading ? (
          <Text className="text-sm text-gray-500">Loading work orders…</Text>
        ) : filteredWorkOrders.length === 0 ? (
          <p className="px-6 py-5 text-center text-sm font-medium text-gray-400 dark:text-gray-400">
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
                factoryHref={factoryHref}
                organizationId={organizationId}
                lines={factoryLines}
                canDispatch={canDispatch || permissionsLoading}
                isDispatching={isDispatching}
                onDispatch={(lineName) => onDispatch(order.id!, lineName)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
