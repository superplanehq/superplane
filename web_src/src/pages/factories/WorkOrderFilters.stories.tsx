import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { WorkOrderFilters } from "./WorkOrderFilters";
import type { WorkOrderOwnerFilter, WorkOrderStatusFilter } from "./lib/workOrderProgress";

/**
 * Row of interactive pills: "My Work / Unassigned / All" on the left and
 * status filters ("All / Active / Draft / Open / Running / Failed /
 * Completed / Rejected") on the right. Stories manage local state so the
 * pills toggle live.
 */
const meta = {
  title: "Factories/Components/WorkOrderFilters",
  component: WorkOrderFilters,
  parameters: { layout: "padded" },
} satisfies Meta<typeof WorkOrderFilters>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveFilters({
  initialOwner = "mine",
  initialStatus = "active",
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

/** Default: "My Work" + "Active" (draft/open/running/failed) active. */
export const Default: Story = {
  render: () => <InteractiveFilters />,
};

/** All owners + Running status — shows the running badge as active. */
export const AllRunning: Story = {
  name: "All / Running",
  render: () => <InteractiveFilters initialOwner="all" initialStatus="running" />,
};

/** Unassigned + Failed — triage view (includes closed-as-failed orders). */
export const UnassignedFailed: Story = {
  name: "Unassigned / Failed",
  render: () => <InteractiveFilters initialOwner="unassigned" initialStatus="failed" />,
};

/** Draft-only — surface work orders still being scoped. */
export const Draft: Story = {
  render: () => <InteractiveFilters initialOwner="all" initialStatus="draft" />,
};
