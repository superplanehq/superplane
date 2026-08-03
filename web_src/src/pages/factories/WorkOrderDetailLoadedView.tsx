import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderResult,
} from "@/api-client";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import {
  factoryDetailPanelClassName,
  factoryDetailSidebarClassName,
  factoryPageContentClassName,
} from "./factoryPageStyles";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import { WorkOrderAssigneesField } from "./WorkOrderAssigneesField";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

interface WorkOrderDetailLoadedViewProps {
  factory: FactoriesFactory;
  factoryHref: string;
  organizationId: string;
  order: FactoriesWorkOrder;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  factoryLines: FactoriesFactoryLine[];
  isOpen: boolean;
  canDispatch: boolean;
  canClose: boolean;
  canAssign: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  isAssigneesSaving: boolean;
  onDispatch: (lineName: string) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
}

export function WorkOrderDetailLoadedView({
  factory,
  factoryHref,
  organizationId,
  order,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  isOpen,
  canDispatch,
  canClose,
  canAssign,
  permissionsLoading,
  isDispatching,
  isCompleting,
  isRejecting,
  isClosing,
  isAssigneesSaving,
  onDispatch,
  onClose,
  onAssigneesSave,
}: WorkOrderDetailLoadedViewProps) {
  return (
    <div className={factoryPageContentClassName}>
      <Link
        href={factoryHref}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        {factory.name}
      </Link>

      <WorkOrderDetailHeader
        orderTitle={order.title ?? "Work Order"}
        statusMeta={statusMeta}
        displayStatus={displayStatus}
        isOpen={isOpen}
        factoryLines={factoryLines}
        canDispatch={canDispatch}
        canClose={canClose}
        permissionsLoading={permissionsLoading}
        isDispatching={isDispatching}
        isCompleting={isCompleting}
        isRejecting={isRejecting}
        isClosing={isClosing}
        onDispatch={onDispatch}
        onClose={onClose}
      />

      <div className={factoryDetailPanelClassName}>
        <div className="mt-8 grid lg:grid-cols-[minmax(0,1fr)_280px]">
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
              isSaving={isAssigneesSaving}
              onSave={onAssigneesSave}
            />
          </aside>
        </div>
      </div>
    </div>
  );
}
