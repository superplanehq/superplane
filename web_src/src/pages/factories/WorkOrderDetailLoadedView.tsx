import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
} from "@/api-client";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import {
  factoryDetailPanelClassName,
  factoryDetailSidebarClassName,
  factoryPageContentClassName,
} from "./lib/factoryPageStyles";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";
import { WorkOrderDescription } from "./WorkOrderDescription";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { WorkOrderDetailSidebar } from "./WorkOrderDetailSidebar";
import type { FactoriesWorkOrderApprovalStatus } from "@/api-client";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailLoadedViewProps {
  factory: FactoriesFactory;
  factoryHref: string;
  backLabel?: string;
  organizationId: string;
  order: FactoriesWorkOrder;
  events?: FactoriesWorkOrderEvent[];
  eventsError?: Error | null;
  isEventsLoading?: boolean;
  hasMoreEvents?: boolean;
  isLoadingMoreEvents?: boolean;
  onLoadMoreEvents?: () => void;
  onRetryEvents?: () => void;
  artifacts: FactoriesWorkOrderArtifact[];
  isArtifactsLoading: boolean;
  artifactsError?: Error | null;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  factoryLines: FactoriesFactoryLine[];
  isOpen: boolean;
  isDispatchable: boolean;
  isClosed: boolean;
  canDispatch: boolean;
  canClose: boolean;
  canAssign: boolean;
  canManage: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  isAssigneesSaving: boolean;
  isUpdatingStatus: boolean;
  isAddingComment: boolean;
  isResolvingApproval: boolean;
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  onAddComment: (body: string) => Promise<void>;
  onResolveApproval: (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => Promise<void>;
}

export function WorkOrderDetailLoadedView({
  factory,
  factoryHref,
  backLabel,
  organizationId,
  order,
  events,
  eventsError,
  isEventsLoading,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
  onRetryEvents,
  artifacts,
  isArtifactsLoading,
  artifactsError,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  isOpen,
  isDispatchable,
  isClosed,
  canDispatch,
  canClose,
  canAssign,
  canManage,
  permissionsLoading,
  isDispatching,
  isCompleting,
  isRejecting,
  isClosing,
  isAssigneesSaving,
  isUpdatingStatus,
  isAddingComment,
  isResolvingApproval,
  onDispatch,
  onClose,
  onAssigneesSave,
  onStatusChange,
  onAddComment,
  onResolveApproval,
}: WorkOrderDetailLoadedViewProps) {
  return (
    <div className={factoryPageContentClassName}>
      <Link
        href={factoryHref}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        {backLabel ?? factory.name}
      </Link>

      <WorkOrderDetailHeader
        orderTitle={order.title ?? "Work Order"}
        statusMeta={statusMeta}
        displayStatus={displayStatus}
        isOpen={isOpen}
        isDispatchable={isDispatchable}
        isClosed={isClosed}
        factoryLines={factoryLines}
        canDispatch={canDispatch}
        canClose={canClose}
        canManage={canManage}
        permissionsLoading={permissionsLoading}
        isDispatching={isDispatching}
        isCompleting={isCompleting}
        isRejecting={isRejecting}
        isClosing={isClosing}
        isUpdatingStatus={isUpdatingStatus}
        onDispatch={onDispatch}
        onClose={onClose}
        onStatusChange={onStatusChange}
      />

      <div className={factoryDetailPanelClassName}>
        <div className="mt-8 grid lg:grid-cols-[minmax(0,1fr)_320px]">
          <div className="min-w-0 px-6 py-6 sm:px-8">
            {order.description ? <WorkOrderDescription description={order.description} className="mb-8" /> : null}

            <section>
              <div className="flex items-baseline justify-between gap-3">
                <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Activity</h2>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Actions and comments on the work order, plus factory line runs.
                </p>
              </div>
              <div className="mt-5">
                <WorkOrderActivityTimeline
                  organizationId={organizationId}
                  order={order}
                  events={events}
                  eventsError={eventsError}
                  isLoading={isEventsLoading}
                  hasMoreEvents={hasMoreEvents}
                  isLoadingMoreEvents={isLoadingMoreEvents}
                  onLoadMoreEvents={onLoadMoreEvents}
                  onRetryEvents={onRetryEvents}
                  canResolveApproval={canManage}
                  isResolvingApproval={isResolvingApproval}
                  onResolveApproval={onResolveApproval}
                />
              </div>
            </section>

            <section className="mt-8">
              <WorkOrderCommentComposer canComment={canManage} isSubmitting={isAddingComment} onSubmit={onAddComment} />
            </section>
          </div>

          <aside className={cn(factoryDetailSidebarClassName, "lg:min-h-full")}>
            <WorkOrderDetailSidebar
              organizationId={organizationId}
              order={order}
              artifacts={artifacts}
              isArtifactsLoading={isArtifactsLoading}
              artifactsError={artifactsError}
              displayStatus={displayStatus}
              statusMeta={statusMeta}
              assigneeIds={assigneeIds}
              assigneeNames={assigneeNames}
              factoryLines={factoryLines}
              factoryId={factory.id ?? ""}
              canAssign={canAssign}
              canDispatch={canDispatch}
              permissionsLoading={permissionsLoading}
              isAssigneesSaving={isAssigneesSaving}
              isDispatchable={isDispatchable}
              isDispatching={isDispatching}
              onAssigneesSave={onAssigneesSave}
              onDispatch={onDispatch}
            />
          </aside>
        </div>
      </div>
    </div>
  );
}
