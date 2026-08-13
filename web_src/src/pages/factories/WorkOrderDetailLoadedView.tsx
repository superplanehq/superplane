import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
} from "@/api-client";
import { factoryContentBodyClassName } from "./pages/factoryPageLayoutStyles";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";
import { WorkOrderDescription } from "./WorkOrderDescription";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { WorkOrderDetailSidebar } from "./WorkOrderDetailSidebar";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailLoadedViewProps {
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
  artifacts: FactoriesWorkOrderArtifact[];
  isArtifactsLoading: boolean;
  artifactsError?: Error | null;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  factoryLines: FactoriesFactoryLine[];
  canEditFactoryLines: boolean;
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
  onDispatch: (input: { lineName: string }) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  onAddComment: (body: string) => Promise<void>;
}

export function WorkOrderDetailLoadedView({
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
  artifacts,
  isArtifactsLoading,
  artifactsError,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  canEditFactoryLines,
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
    <div className={factoryContentBodyClassName}>
      <div className="grid gap-x-[var(--workspace-column-gap)] gap-y-0 lg:grid-cols-[minmax(0,1fr)_var(--workspace-detail-sidebar-width)]">
        <WorkOrderDetailHeader
          orderTitle={order.title ?? "Work Order"}
          displayStatus={displayStatus}
          isOpen={isOpen}
          isDispatchable={isDispatchable}
          isClosed={isClosed}
          canClose={canClose}
          canManage={canManage}
          isCompleting={isCompleting}
          isRejecting={isRejecting}
          isClosing={isClosing}
          isUpdatingStatus={isUpdatingStatus}
          onClose={onClose}
          onStatusChange={onStatusChange}
        />

        <div className="min-w-0">
          {order.description ? <WorkOrderDescription description={order.description} /> : null}

          <section className={order.description ? "mt-10" : undefined}>
            <h2 className="workspace-section-title">Activity</h2>
            <p className="workspace-body-text mt-1 text-muted-foreground">
              Actions and comments on the work order, plus factory line runs.
            </p>
            <div className="mt-4">
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
                artifacts={artifacts}
                footer={
                  <WorkOrderCommentComposer
                    canComment={canManage}
                    isSubmitting={isAddingComment}
                    onSubmit={onAddComment}
                  />
                }
              />
            </div>
          </section>
        </div>

        <aside className="mt-1 lg:sticky lg:top-16 lg:self-start">
          <WorkOrderDetailSidebar
            organizationId={organizationId}
            factoryId={factoryId}
            order={order}
            artifacts={artifacts}
            isArtifactsLoading={isArtifactsLoading}
            artifactsError={artifactsError}
            displayStatus={displayStatus}
            statusMeta={statusMeta}
            assigneeIds={assigneeIds}
            assigneeNames={assigneeNames}
            factoryLines={factoryLines}
            canEditFactoryLines={canEditFactoryLines}
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
  );
}
