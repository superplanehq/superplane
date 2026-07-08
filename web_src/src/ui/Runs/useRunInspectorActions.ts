import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import {
  canvasesCancelExecution,
  canvasesDeleteNodeQueueItem,
  canvasesListNodeQueueItems,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
} from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export function useRunInspectorActions({
  canvasId,
  run,
  sections,
  executionsLoading,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  sections: RunInspectorNodeSection[];
  executionsLoading: boolean;
}) {
  const queryClient = useQueryClient();
  const runningExecutionIds = useMemo(
    () =>
      sections
        .map((section) => section.execution)
        .filter((execution) => execution?.id && execution.state === "STATE_STARTED")
        .map((execution) => execution!.id!),
    [sections],
  );
  const hasActionSection = useMemo(() => sections.some((section) => !section.isTrigger), [sections]);
  const stoppableNodeIds = useMemo(() => [...new Set(sections.map((section) => section.nodeId))], [sections]);

  const refreshRunQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  }, [queryClient]);

  const rerunMutation = useMutation({
    mutationFn: async () => {
      if (!run.rootEvent?.nodeId || !run.rootEvent?.id) {
        throw new Error("Run root event is missing");
      }

      await canvasesReemitTriggerEvent(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId: run.rootEvent.nodeId,
            eventId: run.rootEvent.id,
          },
        }),
      );
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run restarted");
    },
    onError: (error) => {
      console.error("Failed to restart run", error);
      showErrorToast("Failed to restart run");
    },
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      const queuedItems = await listQueuedItemsForRun({
        canvasId,
        nodeIds: stoppableNodeIds,
        rootEventId: run.rootEvent?.id,
      });

      if (runningExecutionIds.length === 0 && queuedItems.length === 0) {
        throw new Error("No running or queued steps to stop");
      }

      await Promise.all([
        ...runningExecutionIds.map((executionId) =>
          canvasesCancelExecution(
            withOrganizationHeader({
              path: {
                canvasId,
                executionId,
              },
            }),
          ),
        ),
        ...queuedItems.map((item) =>
          canvasesDeleteNodeQueueItem(
            withOrganizationHeader({
              path: {
                canvasId,
                nodeId: item.nodeId,
                itemId: item.itemId,
              },
            }),
          ),
        ),
      ]);
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run stopped");
    },
    onError: (error) => {
      console.error("Failed to stop run", error);
      showErrorToast("Failed to stop run");
    },
  });

  return {
    rerun: () => rerunMutation.mutate(),
    rerunPending: rerunMutation.isPending,
    stop: () => stopMutation.mutate(),
    stopPending: stopMutation.isPending,
    stopDisabled:
      executionsLoading ||
      stopMutation.isPending ||
      (runningExecutionIds.length === 0 && (!run.rootEvent?.id || !hasActionSection)),
  };
}

async function listQueuedItemsForRun({
  canvasId,
  nodeIds,
  rootEventId,
}: {
  canvasId: string;
  nodeIds: string[];
  rootEventId?: string;
}) {
  if (!rootEventId || nodeIds.length === 0) {
    return [];
  }

  const responses = await Promise.all(
    nodeIds.map(async (nodeId) => {
      const response = await canvasesListNodeQueueItems(
        withOrganizationHeader({
          path: { canvasId, nodeId },
          query: { limit: 100 },
        }),
      );

      return (
        response.data?.items
          ?.filter((item) => item.id && item.rootEvent?.id === rootEventId)
          .map((item) => ({ nodeId, itemId: item.id! })) ?? []
      );
    }),
  );

  return responses.flat();
}
