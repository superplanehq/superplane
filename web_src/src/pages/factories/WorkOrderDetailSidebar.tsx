import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";

import { WorkOrderSidebarFactoryLines } from "./sidebar/WorkOrderSidebarFactoryLines";
import { WorkOrderSidebarOverview } from "./sidebar/WorkOrderSidebarOverview";
import { WorkOrderArtifactsList } from "./WorkOrderArtifactsList";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailSidebarProps {
  organizationId: string;
  factoryId: string;
  order: FactoriesWorkOrder;
  artifacts: FactoriesWorkOrderArtifact[];
  isArtifactsLoading: boolean;
  artifactsError?: Error | null;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  factoryLines: FactoriesFactoryLine[];
  canEditFactoryLines: boolean;
  canAssign: boolean;
  canDispatch: boolean;
  permissionsLoading: boolean;
  isAssigneesSaving: boolean;
  isDispatchable: boolean;
  isDispatching: boolean;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onDispatch: (input: { lineName: string }) => Promise<void>;
}

/** Work order overview, factory lines, and artifacts. */
export function WorkOrderDetailSidebar({
  organizationId,
  factoryId,
  order,
  artifacts,
  isArtifactsLoading,
  artifactsError,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  canEditFactoryLines,
  canAssign,
  canDispatch,
  permissionsLoading,
  isAssigneesSaving,
  isDispatchable,
  isDispatching,
  onAssigneesSave,
  onDispatch,
}: WorkOrderDetailSidebarProps) {
  return (
    <div className="flex flex-col gap-6">
      <WorkOrderSidebarOverview
        organizationId={organizationId}
        order={order}
        displayStatus={displayStatus}
        statusMeta={statusMeta}
        assigneeIds={assigneeIds}
        assigneeNames={assigneeNames}
        canAssign={canAssign}
        isAssigneesSaving={isAssigneesSaving}
        onAssigneesSave={onAssigneesSave}
      />

      <WorkOrderSidebarFactoryLines
        organizationId={organizationId}
        factoryId={factoryId}
        canEditFactoryLines={canEditFactoryLines}
        executions={order.executions ?? []}
        factoryLines={factoryLines}
        canDispatch={canDispatch}
        permissionsLoading={permissionsLoading}
        isDispatchable={isDispatchable}
        isDispatching={isDispatching}
        onDispatch={onDispatch}
      />

      <WorkOrderArtifactsList artifacts={artifacts} isLoading={isArtifactsLoading} error={artifactsError} />
    </div>
  );
}
