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
import { WorkOrderArtifactsPanel } from "./WorkOrderArtifactsPanel";
import { WorkOrderAssigneesField } from "./WorkOrderAssigneesField";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

function WorkOrderDetailMainColumn({
  organizationId,
  factoryId,
  order,
  events,
  eventsError,
  isEventsLoading,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
  onRetryEvents,
  canManage,
  isAddingComment,
  onAddComment,
}: {
  organizationId: string;
  factoryId: string;
  order: FactoriesWorkOrder;
  events?: FactoriesWorkOrderEvent[];
  eventsError?: Error | null;
  isEventsLoading?: boolean;
  hasMoreEvents?: boolean;
  isLoadingMoreEvents?: boolean;
  onLoadMoreEvents?: () => void;
  onRetryEvents?: () => void;
  canManage: boolean;
  isAddingComment: boolean;
  onAddComment: (body: string) => Promise<void>;
}) {
  return (
    <>
      {order.description ? (
        <section className="mb-8 rounded-lg border border-gray-200 px-4 py-4 dark:border-gray-700/70">
          <p className="whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-gray-300">
            {order.description}
          </p>
        </section>
      ) : null}

      <section className="mb-8">
        <WorkOrderCommentComposer canComment={canManage} isSubmitting={isAddingComment} onSubmit={onAddComment} />
      </section>

      <section>
        <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Activity</h2>
        <div className="mt-5">
          <WorkOrderActivityTimeline
            organizationId={organizationId}
            factoryId={factoryId}
            order={order}
            events={events}
            eventsError={eventsError}
            isLoading={isEventsLoading}
            hasMoreEvents={hasMoreEvents}
            isLoadingMoreEvents={isLoadingMoreEvents}
            onLoadMoreEvents={onLoadMoreEvents}
            onRetryEvents={onRetryEvents}
          />
        </div>
      </section>
    </>
  );
}

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
  onDispatch: (lineName: string) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  onAddComment: (body: string) => Promise<void>;
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
  onDispatch,
  onClose,
  onAssigneesSave,
  onStatusChange,
  onAddComment,
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
        <div className="mt-8 grid lg:grid-cols-[minmax(0,1fr)_280px]">
          <div className="min-w-0 px-6 py-6 sm:px-8">
            <WorkOrderDetailMainColumn
              organizationId={organizationId}
              factoryId={factory.id ?? ""}
              order={order}
              events={events}
              eventsError={eventsError}
              isEventsLoading={isEventsLoading}
              hasMoreEvents={hasMoreEvents}
              isLoadingMoreEvents={isLoadingMoreEvents}
              onLoadMoreEvents={onLoadMoreEvents}
              onRetryEvents={onRetryEvents}
              canManage={canManage}
              isAddingComment={isAddingComment}
              onAddComment={onAddComment}
            />
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

            <div className="mt-6">
              <WorkOrderArtifactsPanel artifacts={artifacts} isLoading={isArtifactsLoading} error={artifactsError} />
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
