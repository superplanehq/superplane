import type {
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
} from "@/api-client";
import { cn } from "@/lib/utils";
import { factoryHomePath, firstFactoryLineId } from "./lib/factoryPagePaths";
import { latestDispatchForLine } from "./lib/workOrderNumberResolution";
import { getWorkOrderDisplayKey, type WorkOrderDisplayStatus } from "./lib/workOrderProgress";
import { factoryContentBodyClassName } from "./pages/factoryPageLayoutStyles";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import type { WorkOrderCheckPresentation } from "./lib/workOrderChecks";
import { WorkOrderChecksSection } from "./WorkOrderChecksSection";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";
import { WorkOrderDescription } from "./WorkOrderDescription";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { WorkOrderDetailSidebar } from "./WorkOrderDetailSidebar";
import type { WorkOrderStatusNotePresentation } from "./lib/workOrderStatusNote";
import { buildWorkOrderStatusActions } from "./lib/workOrderStatusActions";
import { WorkOrderStatusNote } from "./WorkOrderStatusNote";

interface WorkOrderDetailLoadedViewProps {
  organizationId: string;
  factoryKey: string;
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
  pullRequests?: FactoriesFactoryPullRequest[];
  isPullRequestsLoading?: boolean;
  pullRequestsError?: Error | null;
  /** Why the order is waiting, announced by automations. */
  statusNotes?: WorkOrderStatusNotePresentation[];
  /** Scores reported by automations (risk review, coverage, …). */
  checks?: WorkOrderCheckPresentation[];
  isChecksLoading?: boolean;
  checksError?: Error | null;
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
  onDispatch: (input: { lineName: string }) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  onAddComment: (body: string, mentionedUserIds: string[]) => Promise<void>;
  /** Page chrome includes the back link. Dialog chrome is the card overlay. */
  chrome?: "page" | "dialog";
}

export function WorkOrderDetailLoadedView(props: WorkOrderDetailLoadedViewProps) {
  const identifier = getWorkOrderDisplayKey(props.order, props.factoryKey);
  const isDialog = props.chrome === "dialog";
  return (
    <>
      <WorkOrderDetailHeader
        orderTitle={props.order.title ?? "Task"}
        orderIdentifier={identifier === "—" ? undefined : identifier}
        backHref={isDialog ? undefined : workOrderBoardBackHref(props)}
        backLabel="Workspace"
        displayStatus={props.displayStatus}
        isOpen={props.isOpen}
        isDispatchable={props.isDispatchable}
        isClosed={props.isClosed}
        canClose={props.canClose}
        canManage={props.canManage}
        isCompleting={props.isCompleting}
        isRejecting={props.isRejecting}
        isClosing={props.isClosing}
        isUpdatingStatus={props.isUpdatingStatus}
        onClose={props.onClose}
        onStatusChange={props.onStatusChange}
        className={isDialog ? "max-w-none px-6 pt-4 pb-3 pr-12" : undefined}
      />
      <WorkOrderDetailBody {...props} />
    </>
  );
}

function WorkOrderDetailBody(props: WorkOrderDetailLoadedViewProps) {
  const isDialog = props.chrome === "dialog";
  return (
    <div className={cn(isDialog ? "px-6 pb-6 pt-2" : cn(factoryContentBodyClassName, "pt-2"))}>
      <div className="grid gap-x-[var(--workspace-column-gap)] gap-y-0 lg:grid-cols-[minmax(0,1fr)_var(--workspace-detail-sidebar-width)]">
        <WorkOrderDetailMainColumn {...props} />
        <WorkOrderDetailBodyAside {...props} />
      </div>
    </div>
  );
}

function WorkOrderDetailMainColumn({
  organizationId,
  factoryKey,
  order,
  events,
  eventsError,
  isEventsLoading,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
  onRetryEvents,
  artifacts,
  pullRequests,
  statusNotes,
  checks,
  isChecksLoading,
  checksError,
  displayStatus,
  isOpen,
  isDispatchable,
  isClosed,
  canClose,
  isCompleting,
  isRejecting,
  isClosing,
  isUpdatingStatus,
  onClose,
  onStatusChange,
  canManage,
  isAddingComment,
  onAddComment,
}: WorkOrderDetailLoadedViewProps) {
  const hasChecksSection = Boolean(checks?.length) || Boolean(isChecksLoading) || Boolean(checksError);
  const notesToShow = statusNotes ?? [];
  const showStatusNotes = notesToShow.length > 0;

  return (
    <div className="min-w-0">
      {order.description ? <WorkOrderDescription description={order.description} /> : null}

      {showStatusNotes ? (
        <div className={order.description ? "mt-10" : undefined}>
          <WorkOrderStatusNotesSection
            notes={notesToShow}
            organizationId={organizationId}
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
        </div>
      ) : null}

      {hasChecksSection ? (
        <WorkOrderChecksSection
          checks={checks ?? []}
          isLoading={isChecksLoading}
          error={checksError}
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={order.number}
          className={order.description || showStatusNotes ? "mt-10" : undefined}
        />
      ) : null}

      <section className={order.description || hasChecksSection || showStatusNotes ? "mt-10" : undefined}>
        <h2 className="workspace-section-title">Activity</h2>
        <p className="workspace-body-text mt-1 text-muted-foreground">
          Actions and comments on the task, plus factory line runs.
        </p>
        <div className="mt-4">
          <WorkOrderActivityTimeline
            organizationId={organizationId}
            factoryKey={factoryKey}
            order={order}
            events={events}
            eventsError={eventsError}
            isLoading={isEventsLoading}
            hasMoreEvents={hasMoreEvents}
            isLoadingMoreEvents={isLoadingMoreEvents}
            onLoadMoreEvents={onLoadMoreEvents}
            onRetryEvents={onRetryEvents}
            artifacts={artifacts}
            pullRequests={pullRequests}
            footer={
              <WorkOrderCommentComposer
                organizationId={organizationId}
                canComment={canManage}
                isSubmitting={isAddingComment}
                onSubmit={onAddComment}
              />
            }
          />
        </div>
      </section>
    </div>
  );
}

function WorkOrderDetailBodyAside({
  organizationId,
  factoryKey,
  order,
  artifacts,
  isArtifactsLoading,
  artifactsError,
  pullRequests,
  isPullRequestsLoading,
  pullRequestsError,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  canAssign,
  canDispatch,
  permissionsLoading,
  isAssigneesSaving,
  isDispatchable,
  isDispatching,
  onAssigneesSave,
  onDispatch,
}: WorkOrderDetailLoadedViewProps) {
  return (
    <aside className="mt-1 lg:sticky lg:top-16 lg:self-start">
      <WorkOrderDetailSidebar
        organizationId={organizationId}
        factoryKey={factoryKey}
        order={order}
        artifacts={artifacts}
        isArtifactsLoading={isArtifactsLoading}
        artifactsError={artifactsError}
        pullRequests={pullRequests}
        isPullRequestsLoading={isPullRequestsLoading}
        pullRequestsError={pullRequestsError}
        displayStatus={displayStatus}
        statusMeta={statusMeta}
        assigneeIds={assigneeIds}
        assigneeNames={assigneeNames}
        factoryLines={factoryLines}
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
  );
}

function WorkOrderStatusNotesSection({
  notes,
  organizationId,
  displayStatus,
  isOpen,
  isDispatchable,
  isClosed,
  canClose,
  canManage,
  isCompleting,
  isRejecting,
  isClosing,
  isUpdatingStatus,
  onClose,
  onStatusChange,
}: Pick<
  WorkOrderDetailLoadedViewProps,
  | "organizationId"
  | "displayStatus"
  | "isOpen"
  | "isDispatchable"
  | "isClosed"
  | "canClose"
  | "canManage"
  | "isCompleting"
  | "isRejecting"
  | "isClosing"
  | "isUpdatingStatus"
  | "onClose"
  | "onStatusChange"
> & { notes: WorkOrderStatusNotePresentation[] }) {
  const lastIndex = notes.length - 1;
  const statusActions = buildWorkOrderStatusActions({
    displayStatus,
    isOpen,
    isDispatchable,
    isClosed,
    canClose,
    canManage,
    isClosing,
    isUpdatingStatus,
  });

  return (
    <div className="flex flex-col gap-3">
      {notes.map((note, index) => (
        <WorkOrderStatusNote
          key={note.key}
          note={note}
          organizationId={organizationId}
          canClose={canClose}
          canManage={canManage}
          isBusy={isCompleting || isRejecting || isClosing || isUpdatingStatus}
          statusActions={index === lastIndex ? statusActions : []}
          onClose={onClose}
          onStatusChange={onStatusChange}
        />
      ))}
    </div>
  );
}

function workOrderBoardBackHref(
  props: Pick<WorkOrderDetailLoadedViewProps, "organizationId" | "factoryKey" | "order" | "factoryLines">,
) {
  const lineId = latestDispatchForLine(props.order)?.line?.id ?? firstFactoryLineId({ lines: props.factoryLines });
  return factoryHomePath(props.organizationId, props.factoryKey, lineId);
}
