import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  DEFAULT_WORK_ORDERS,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_LINES,
} from "./__fixtures__/factoryPageResponses";
import { FactoryDetailWorkOrdersPanel } from "./FactoryDetailWorkOrdersPanel";
import { filterWorkOrdersByStatus, type WorkOrderOwnerFilter, type WorkOrderStatusFilter } from "./workOrderProgress";

/**
 * The Work Orders section on the factory detail page: header + filters + list.
 * Stories wire up local state so filter pills interact live.
 */
const meta = {
  title: "Factories/FactoryDetailWorkOrdersPanel",
  component: FactoryDetailWorkOrdersPanel,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen w-full bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof FactoryDetailWorkOrdersPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`;

function InteractivePanel({
  initialStatus = "all",
  workOrders = DEFAULT_WORK_ORDERS,
  isLoading = false,
  ordersError = null,
}: {
  initialStatus?: WorkOrderStatusFilter;
  workOrders?: typeof DEFAULT_WORK_ORDERS;
  isLoading?: boolean;
  ordersError?: Error | null;
}) {
  const [ownerFilter, setOwnerFilter] = useState<WorkOrderOwnerFilter>("all");
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatusFilter>(initialStatus);
  const filtered = filterWorkOrdersByStatus(workOrders, statusFilter);
  return (
    <FactoryDetailWorkOrdersPanel
      openWorkOrderCount={workOrders.filter((order) => order.state === "STATE_OPEN").length}
      ownerFilter={ownerFilter}
      statusFilter={statusFilter}
      onOwnerFilterChange={setOwnerFilter}
      onStatusFilterChange={setStatusFilter}
      ordersError={ordersError}
      isOrdersLoading={isLoading}
      filteredWorkOrders={filtered}
      factoryHref={factoryHref}
      organizationId={FACTORIES_ORGANIZATION_ID}
      factoryLines={REFUND_FACTORY_LINES}
      canDispatch
      permissionsLoading={false}
      isDispatching={false}
      onDispatch={async (orderId, lineName) => {
        console.log("dispatch", orderId, lineName);
      }}
    />
  );
}

/** Populated: all statuses visible. */
export const Populated: Story = {
  render: () => <InteractivePanel />,
};

/** Only failed orders — status filter set to `failed`. */
export const FailedOnly: Story = {
  name: "Failed Only",
  render: () => <InteractivePanel initialStatus="failed" />,
};

/** Loading state — skeleton copy while orders fetch. */
export const Loading: Story = {
  render: () => <InteractivePanel workOrders={[]} isLoading />,
};

/** Error state — banner instead of the list. */
export const Error: Story = {
  render: () => <InteractivePanel workOrders={[]} ordersError={new Error("boom")} />,
};

/** Empty state — no orders match; guidance to adjust filters. */
export const Empty: Story = {
  render: () => <InteractivePanel workOrders={[]} />,
};
