import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { WorkOrderFilters } from "./WorkOrderFilters";
import type { WorkOrderOwnerFilter, WorkOrderStatusFilter } from "./workOrderProgress";

/**
 * Row of interactive pills: "My Work / Unassigned / All" on the left and
 * status filters ("All / Open / Running / Failed / Completed / Rejected")
 * on the right. Stories manage local state so the pills toggle live.
 */
const meta = {
  title: "Factories/WorkOrderFilters",
  component: WorkOrderFilters,
  parameters: { layout: "padded" },
} satisfies Meta<typeof WorkOrderFilters>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveFilters({
  initialOwner = "mine",
  initialStatus = "open",
}: {
  initialOwner?: WorkOrderOwnerFilter;
  initialStatus?: WorkOrderStatusFilter;
}) {
  const [ownerFilter, setOwnerFilter] = useState<WorkOrderOwnerFilter>(initialOwner);
  const [statusFilter, setStatusFilter] = useState<WorkOrderStatusFilter>(initialStatus);
  return (
    <WorkOrderFilters
      ownerFilter={ownerFilter}
      statusFilter={statusFilter}
      onOwnerFilterChange={(value) => {
        console.log("owner", value);
        setOwnerFilter(value);
      }}
      onStatusFilterChange={(value) => {
        console.log("status", value);
        setStatusFilter(value);
      }}
    />
  );
}

/** Default: "My Work" + "Open" active. */
export const Default: Story = {
  render: () => <InteractiveFilters />,
};

/** All owners + Running status — shows the running badge as active. */
export const AllRunning: Story = {
  name: "All / Running",
  render: () => <InteractiveFilters initialOwner="all" initialStatus="running" />,
};

/** Unassigned + Failed — used to triage stuck work. */
export const UnassignedFailed: Story = {
  name: "Unassigned / Failed",
  render: () => <InteractiveFilters initialOwner="unassigned" initialStatus="failed" />,
};
