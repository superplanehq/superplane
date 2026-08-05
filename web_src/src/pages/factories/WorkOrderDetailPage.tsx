import { Text } from "@/components/Text/text";
import { useParams } from "react-router-dom";
import { FactoryPageShell } from "./FactoryPageShell";
import { useWorkOrderDetailPage } from "./useWorkOrderDetailPage";
import { WorkOrderDetailLoadedView } from "./WorkOrderDetailLoadedView";

export function WorkOrderDetailPage() {
  const { organizationId, factoryId, orderId } = useParams<{
    organizationId: string;
    factoryId: string;
    orderId: string;
  }>();

  if (!organizationId || !factoryId || !orderId) {
    return null;
  }

  return <WorkOrderDetailPageContent organizationId={organizationId} factoryId={factoryId} orderId={orderId} />;
}

function WorkOrderDetailPageContent({
  organizationId,
  factoryId,
  orderId,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
}) {
  const page = useWorkOrderDetailPage(organizationId, factoryId, orderId);

  if (page.kind === "redirect") {
    return page.element;
  }

  return (
    <FactoryPageShell backHref={page.factoryHref} backLabel={page.factory?.name ?? "Factory"}>
      {page.isLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading work order…</Text>
        </div>
      ) : page.factory && page.order && page.statusMeta && page.displayStatus ? (
        <WorkOrderDetailLoadedView
          factory={page.factory}
          factoryHref={page.factoryHref}
          organizationId={page.organizationId}
          order={page.order}
          events={page.events}
          eventsError={page.eventsError}
          isEventsLoading={page.isEventsLoading}
          hasMoreEvents={page.hasMoreEvents}
          isLoadingMoreEvents={page.isLoadingMoreEvents}
          onLoadMoreEvents={page.onLoadMoreEvents}
          onRetryEvents={page.onRetryEvents}
          artifacts={page.artifacts}
          isArtifactsLoading={page.isArtifactsLoading}
          artifactsError={page.artifactsError}
          displayStatus={page.displayStatus}
          statusMeta={page.statusMeta}
          assigneeIds={page.assigneeIds}
          assigneeNames={page.assigneeNames}
          factoryLines={page.factoryLines}
          isOpen={page.isOpen}
          isDispatchable={page.isDispatchable}
          isClosed={page.isClosed}
          isDraft={page.isDraft}
          isReady={page.isReady}
          canDispatch={page.canDispatch}
          canClose={page.canClose}
          canAssign={page.canAssign}
          canManage={page.canManage}
          permissionsLoading={page.permissionsLoading}
          isDispatching={page.isDispatching}
          isCompleting={page.isCompleting}
          isRejecting={page.isRejecting}
          isClosing={page.isClosing}
          isAssigneesSaving={page.isAssigneesSaving}
          isUpdatingStatus={page.isUpdatingStatus}
          isAddingComment={page.isAddingComment}
          isAddingArtifact={page.isAddingArtifact}
          onDispatch={page.handleDispatch}
          onClose={page.handleClose}
          onAssigneesSave={page.handleAssigneesSave}
          onStatusChange={page.handleStatusChange}
          onAddComment={(body) => page.handleAddComment({ body, authorKind: "KIND_USER" })}
          onAddArtifact={page.handleAddArtifact}
        />
      ) : null}
    </FactoryPageShell>
  );
}
